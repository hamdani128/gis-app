"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { useUserProfile } from "../../../components/auth/user-profile-context";
import "./dashboard.css";

type Trend = { month: number; created: number; grtt: number };
type Status = { key: string; count: number; area: number };
type Stage = { status: number; count: number };
type Breakdown = { id: number; name: string; land_count: number; area: number };
type OfficerLand = { id: number; company_id: number; company_name?: string; estate_name?: string; owner?: string; village?: string; land_wide?: number; wet_land_wide?: number; dry_land_wide?: number; admin_price?: number; plant_price?: number };
type OfficerPage = { lands: OfficerLand[] };
type Summary = { generated_at: string; year: number; scope: string; company_count: number; estate_count: number; owner_count: number; land_count: number; active_area: number; estimated_price: number; completed_count: number; completed_area: number; trend: Trend[]; statuses: Status[]; stages: Stage[]; breakdown: Breakdown[]; breakdown_kind: "company" | "estate" };
type Envelope<T> = { data?: T; message?: string };
const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");
const fmt = (value: number, digits = 0) => new Intl.NumberFormat("id-ID", { minimumFractionDigits: digits, maximumFractionDigits: digits }).format(value || 0);
const months = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
const statusMeta: Record<string, { label: string; color: string }> = {
  proses: { label: "Approval in progress", color: "#2979cc" }, payment: { label: "Payment", color: "#e6a733" },
  payment_owner: { label: "Payment to Owner", color: "#9a69d5" }, selesai: { label: "Selesai", color: "#23a276" },
  batal: { label: "Canceled", color: "#dd6871" }, lainnya: { label: "Other statuses", color: "#a2aebd" },
};
const stages = [
  { name: "Submit Memo Land", href: "/admin/acquisition/submit-memo-land" },
  { name: "RM Approval", href: "/admin/acquisition/rm-approval" },
  { name: "GM Approval", href: "/admin/acquisition/gm-approval" },
  { name: "GIS / Legal Approval", href: "/admin/acquisition/gis-approval" },
  { name: "Operation Control", href: "/admin/acquisition/operation-control-approval" },
  { name: "PD Approval", href: "/admin/acquisition/pd-approval" },
  { name: "Payment", href: "/admin/acquisition/payment" },
  { name: "Payment to Owner", href: "/admin/acquisition/payment-to-owner" },
];

async function apiGet<T>(path: string): Promise<T> {
  const request = () => fetch(`${apiBase}${path}`, { credentials: "include", cache: "no-store" });
  let response = await request();
  if (response.status === 401) {
    const renewal = await fetch(`${apiBase}/api/auth/refresh`, { method: "POST", credentials: "include" });
    if (renewal.ok) response = await request();
  }
  const body = await response.json().catch(() => null) as Envelope<T> | null;
  if (!response.ok) throw new Error(body?.message || "Unable to load dashboard data.");
  if (!body?.data) throw new Error("Dashboard data is empty.");
  return body.data;
}

function TrendChart({ data }: { data: Trend[] }) {
  const maximum = Math.max(1, ...data.flatMap((item) => [item.created, item.grtt]));
  const points = (key: "created" | "grtt") => data.map((item, index) => `${34 + index * 56},${172 - item[key] / maximum * 130}`).join(" ");
  return <div className="dash-trend-chart" role="img" aria-label="Monthly new land and GRTT chart"><svg viewBox="0 0 684 205" preserveAspectRatio="none" aria-hidden="true">
    {[0, 1, 2, 3].map((row) => <line key={row} x1="34" x2="662" y1={42 + row * 43.3} y2={42 + row * 43.3} stroke="#e8edf2" strokeDasharray="4 5" />)}
    <polyline points={points("created")} fill="none" stroke="#237eae" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" />
    <polyline points={points("grtt")} fill="none" stroke="#28a67c" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" />
    {data.map((item, index) => <g key={item.month}><circle cx={34 + index * 56} cy={172 - item.created / maximum * 130} r="4" fill="#237eae"><title>{months[index]}: {item.created} new lands</title></circle><circle cx={34 + index * 56} cy={172 - item.grtt / maximum * 130} r="4" fill="#28a67c"><title>{months[index]}: {item.grtt} GRTT</title></circle><text x={34 + index * 56} y="197" textAnchor="middle" fill="#8391a4" fontSize="11">{months[index]}</text></g>)}
  </svg></div>;
}

export default function DashboardPage() {
  const profile = useUserProfile();
  const [companyID, setCompanyID] = useState<number | null>(null);
  const [year, setYear] = useState(new Date().getFullYear());
  const [summary, setSummary] = useState<Summary | null>(null);
  const [modules, setModules] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [officerLands, setOfficerLands] = useState<OfficerLand[]>([]);
  const [officerLoading, setOfficerLoading] = useState(false);
  const [officerError, setOfficerError] = useState("");
  const [officerSearch, setOfficerSearch] = useState("");

  useEffect(() => {
    const key = `gis.selected-company.${profile.username.toLowerCase()}`;
    const read = () => { const saved = window.localStorage.getItem(key); setCompanyID(saved === null ? null : Number(saved) || 0); };
    read();
    window.addEventListener("gis:company-change", read);
    window.addEventListener("storage", read);
    return () => { window.removeEventListener("gis:company-change", read); window.removeEventListener("storage", read); };
  }, [profile.username]);
  useEffect(() => {
    let active = true;
    void apiGet<string[]>("/api/auth/menu-access").then((codes) => { if (active) setModules(new Set(codes.map((code) => code.toLowerCase()))); }).catch(() => { if (active) setModules(new Set()); });
    return () => { active = false; };
  }, []);
  const load = useCallback(async (active: () => boolean) => {
    if (companyID === null) return;
    setLoading(true); setError(""); setSummary(null);
    try { const data = await apiGet<Summary>(`/api/dashboard/summary?company_id=${companyID}&year=${year}`); if (active()) setSummary(data); }
    catch (cause) { if (active()) { setSummary(null); setError(cause instanceof Error ? cause.message : "Unable to load dashboard data."); } }
    finally { if (active()) setLoading(false); }
  }, [companyID, year]);
  useEffect(() => { let active = true; const timer = window.setTimeout(() => void load(() => active), 0); return () => { active = false; window.clearTimeout(timer); }; }, [load]);

  const role = (profile.process_role || "").toLowerCase();
  const roleStage = role.includes("legal") || role.includes("gis") ? 3 : role.includes("operation") ? 4 : role.includes("payment to owner") ? 7 : stages.findIndex((stage) => role.includes(stage.name.toLowerCase()));
  const roleHref = role.includes("legal") ? "/admin/acquisition/legal-approval" : roleStage >= 0 ? stages[roleStage].href : "";
  const canProcess = modules.has("acqproses");
  const canReport = modules.has("rpt_all_progress");
  useEffect(() => {
    if (companyID === null || !canProcess || roleStage < 0) return;
    let active = true;
    const apiPath = roleHref.replace("/admin/acquisition/", "/api/");
    const timer = window.setTimeout(async () => {
      setOfficerLoading(true); setOfficerError(""); setOfficerLands([]);
      try {
        const data = await apiGet<OfficerPage>(`${apiPath}?company_id=${companyID}`);
        if (active) setOfficerLands([...new Map((data.lands ?? []).map((land) => [land.id, land])).values()]);
      } catch (cause) {
        if (active) setOfficerError(cause instanceof Error ? cause.message : "Unable to load the officer's land list.");
      } finally { if (active) setOfficerLoading(false); }
    }, 0);
    return () => { active = false; window.clearTimeout(timer); };
  }, [companyID, canProcess, roleStage, roleHref]);
  const visibleOfficerLands = officerLands.filter((land) => {
    const search = officerSearch.trim().toLowerCase();
    return !search || [land.id, land.estate_name, land.owner, land.village, land.company_name].some((value) => String(value ?? "").toLowerCase().includes(search));
  });
  const statusTotal = summary?.statuses.reduce((total, item) => total + item.count, 0) || 0;
  const segments = (summary?.statuses ?? []).filter((item) => item.count).map((item, index, items) => {
    const start = items.slice(0, index).reduce((total, prior) => total + prior.count, 0) / statusTotal * 100;
    const end = start + item.count / statusTotal * 100;
    return `${statusMeta[item.key]?.color ?? "#a2aebd"} ${start}% ${end}%`;
  });
  const donut = `conic-gradient(${segments.join(", ") || "#e9edf2 0% 100%"})`;
  const maxBreakdown = Math.max(1, ...(summary?.breakdown ?? []).map((item) => item.area));
  const currentYear = new Date().getFullYear();

  return <main className="gis-dashboard">
    <div className="dash-heading"><div><span className="dash-eyebrow">GIS · AEP PLANTATIONS</span><h1>Land Acquisition Dashboard</h1><p>An overview of {summary?.scope || "the selected company"} based on current GIS records.</p></div><div className="dash-heading-actions"><label htmlFor="dashboard-year">Trend year</label><select id="dashboard-year" value={year} onChange={(event) => setYear(Number(event.target.value))}>{Array.from({ length: 7 }, (_, index) => currentYear - index).map((item) => <option key={item} value={item}>{item}</option>)}</select></div></div>
    {error && <div className="dash-error" role="alert">{error} <button type="button" onClick={() => void load(() => true)}>Retry</button></div>}
    {loading && !summary && <div className="dash-loading">Loading dashboard summary…</div>}
    {summary && <>
      <div className="dash-scope"><span className="dash-scope-dot" />{companyID === 0 ? "All companies" : summary.scope}<span className="dash-scope-divider" />{fmt(summary.company_count)} companies · {fmt(summary.estate_count)} estates<span className="dash-scope-updated">Updated {new Date(summary.generated_at).toLocaleString("en-GB")}</span></div>
      <section className="dash-kpis" aria-label="Key indicators">
        <div className="dash-kpi"><span className="dash-kpi-icon green">▣</span><span>Landowners</span><strong>{fmt(summary.owner_count)}</strong><small>Registered owners</small></div>
        <div className="dash-kpi"><span className="dash-kpi-icon blue">▦</span><span>Total land parcels</span><strong>{fmt(summary.land_count)}</strong><small>Across all statuses</small></div>
        <div className="dash-kpi"><span className="dash-kpi-icon teal">◇</span><span>Active land area</span><strong>{fmt(summary.active_area, 2)} <em>ha</em></strong><small>Excluding canceled parcels</small></div>
        <div className="dash-kpi"><span className="dash-kpi-icon amber">✓</span><span>Completed parcels</span><strong>{fmt(summary.completed_count)}</strong><small>{fmt(summary.completed_area, 2)} ha</small></div>
        <div className="dash-kpi dash-kpi-price"><span className="dash-kpi-icon purple">Rp</span><span>Estimated land value</span><strong>Rp {fmt(summary.estimated_price)}</strong><small>Land, crops and administration · excluding canceled parcels</small></div>
      </section>
      {roleStage >= 0 && canProcess && <>
        <section className="dash-role-panel"><div><span className="dash-eyebrow">OFFICER WORKSPACE</span><h2>{profile.process_role}</h2><p>{officerLoading ? "Loading assigned parcels…" : `${fmt(officerLands.length)} land parcels available for this stage in the selected company.`}</p></div><Link href={roleHref}>Open process <span aria-hidden="true">→</span></Link></section>
        <section className="dash-panel dash-officer-panel"><div className="dash-panel-head"><div><span className="dash-eyebrow">READY TO REVIEW</span><h2>{profile.process_role} land list</h2><p>Open a parcel in the process page to review its details.</p></div><label className="dash-officer-search"><span className="visually-hidden">Search officer land list</span><input type="search" placeholder="Search land ID, owner, estate…" value={officerSearch} onChange={(event) => setOfficerSearch(event.target.value)} /></label></div>
          <div className="dash-officer-table-wrap"><table className="dash-officer-table"><thead><tr><th>Land ID</th>{companyID === 0 && <th>Company</th>}<th>Estate</th><th>Owner</th><th>Village</th><th>Area (ha)</th><th>Action</th></tr></thead><tbody>
            {officerLoading ? <tr><td colSpan={companyID === 0 ? 7 : 6} className="dash-table-empty">Loading land parcels…</td></tr> : officerError ? <tr><td colSpan={companyID === 0 ? 7 : 6} className="dash-table-empty dash-table-error">{officerError}</td></tr> : visibleOfficerLands.length === 0 ? <tr><td colSpan={companyID === 0 ? 7 : 6} className="dash-table-empty">{officerSearch ? "No parcels match your search." : "No land parcels are currently available for this stage."}</td></tr> : visibleOfficerLands.map((land) => <tr key={land.id}><td className="dash-table-id">#{land.id}</td>{companyID === 0 && <td>{land.company_name || "—"}</td>}<td>{land.estate_name || "—"}</td><td>{land.owner || "—"}</td><td>{land.village || "—"}</td><td>{fmt(Number(land.land_wide), 2)}</td><td><Link className="dash-table-action" href={`${roleHref}?land_id=${land.id}`}>Find in process →</Link></td></tr>)}
          </tbody></table></div><div className="dash-officer-foot">Showing {fmt(visibleOfficerLands.length)} of {fmt(officerLands.length)} parcels</div>
        </section>
      </>}
      <div className="dash-chart-grid">
        <section className="dash-panel dash-trend"><div className="dash-panel-head"><div><span className="dash-eyebrow">ACTIVITY</span><h2>Land activity in {year}</h2><p>New parcels and GRTT records by month</p></div><div className="dash-legend"><span><i className="blue" />New parcels</span><span><i className="green" />GRTT</span></div></div><TrendChart data={summary.trend} /></section>
        <section className="dash-panel dash-status"><div className="dash-panel-head"><div><span className="dash-eyebrow">COMPOSITION</span><h2>Land status</h2><p>All parcels in the selected company</p></div></div><div className="dash-status-body"><div className="dash-donut" style={{ background: donut }} role="img" aria-label={`Status distribution across ${statusTotal} parcels`}><div><strong>{fmt(statusTotal)}</strong><span>parcels</span></div></div><div className="dash-status-list">{summary.statuses.filter((item) => item.count > 0).map((item) => <div key={item.key}><i style={{ background: statusMeta[item.key]?.color }} /><span>{statusMeta[item.key]?.label || item.key}</span><strong>{fmt(item.count)}</strong><small>{fmt(item.count / Math.max(statusTotal, 1) * 100, 1)}%</small></div>)}</div></div></section>
      </div>
      <div className="dash-lower-grid">
        <section className="dash-panel"><div className="dash-panel-head"><div><span className="dash-eyebrow">COMPARISON</span><h2>{summary.breakdown_kind === "company" ? "Area by company" : "Area by estate"}</h2><p>Active area and parcel count</p></div></div><div className="dash-bars">{summary.breakdown.length === 0 && <p className="dash-empty">No land records yet.</p>}{summary.breakdown.slice(0, 10).map((item) => <div className="dash-bar-row" key={item.id}><div className="dash-bar-label"><strong title={item.name}>{item.name || `#${item.id}`}</strong><span>{fmt(item.land_count)} parcels</span></div><div className="dash-bar-track"><span style={{ width: `${item.area / maxBreakdown * 100}%` }} /></div><b>{fmt(item.area, 2)} ha</b></div>)}</div></section>
        <section className="dash-panel"><div className="dash-panel-head"><div><span className="dash-eyebrow">ACQUISITION WORKFLOW</span><h2>Parcels in progress</h2><p>Parcel count by current status</p></div></div><div className="dash-stage-list">{stages.map((stage, index) => <div key={stage.name} className="dash-stage-row"><span className="dash-stage-number">{String(index + 1).padStart(2, "0")}</span><span>{stage.name}</span><strong>{fmt(summary.stages[index]?.count || 0)}</strong></div>)}</div>{canProcess && <Link className="dash-panel-link" href="/admin/acquisition/request">View acquisition process <span aria-hidden="true">→</span></Link>}</section>
      </div>
      {canReport && <div className="dash-footer-link"><span>Need a company and estate breakdown?</span><Link href="/admin/report/all-estate-progress">View All Estate Progress <span aria-hidden="true">→</span></Link></div>}
    </>}
  </main>;
}
