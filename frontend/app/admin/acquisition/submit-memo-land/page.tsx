"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import { useUserProfile } from "../../../../components/auth/user-profile-context";
import { dedupeHistory } from "../process-history";
import AcquisitionProcessOverview, { acquisitionStages } from "../acquisition-process-overview";
import TableLoadingRows from "../../../../components/ui/table-loading-rows";
import CompensatedMap from "../../lands/compensatedlands/compensated-map";
import AcquisitionLandTabs from "../acquisition-land-tabs";
import AcquisitionStage, { ProcessApprovalBadge } from "../acquisition-stage";

type Land = { id: number; company_id: number; company_name: string; estate_name: string; owner: string; village: string; block_plan: string; permit_number: string; land_wide: number; wet_land_wide: number; wet_land_perha_price: number; dry_land_wide: number; dry_land_perha_price: number; admin_price: number; plant_price: number; status: number; grit_date?: string; measurement_date?: string; measured_by?: string; vegetation_note?: string; submit_count: number };
type Officer = { employee_id: number; name: string; company_id: number };
type Memo = { id: number; memo_number: string; memo_date: string; memo_status: number; status_name: string };
type Process = { id: number; sequence_id: number; name: string; request_number: string | null; request_date: string | null; approval: number | null; note: string | null; revision: number | null; date_created: string | null; username: string; employee_name: string };
type Detail = { land: Land; memos: Memo[]; history: Process[] };
type Envelope<T> = { data?: T; message?: string };
type Column = "id" | "estate_name" | "owner" | "village" | "land_wide" | "wet_land_wide" | "wet_land_perha_price" | "dry_land_wide" | "dry_land_perha_price" | "average" | "admin_price" | "plant_price" | "total" | "status" | "grit_date";
const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");
const columns: [Column, string][] = [["id", "Land ID"], ["estate_name", "Estate"], ["owner", "Owner"], ["village", "Village"], ["land_wide", "Land Size (ha)"], ["wet_land_wide", "Wet (ha)"], ["wet_land_perha_price", "Wet price (@Ha)"], ["dry_land_wide", "Dry (ha)"], ["dry_land_perha_price", "Dry price (@Ha)"], ["average", "Avg (@Ha)"], ["admin_price", "Admin price"], ["plant_price", "Plant price"], ["total", "Total price"], ["status", "Status"], ["grit_date", "Grit date"]];
const steps = acquisitionStages.map(([, name]) => name);
const money = (value: unknown) => new Intl.NumberFormat("id-ID", { maximumFractionDigits: 0 }).format(Number(value) || 0);
const area = (value: unknown) => new Intl.NumberFormat("id-ID", { maximumFractionDigits: 2 }).format(Number(value) || 0);
const total = (land: Land) => Number(land.wet_land_wide || 0) * Number(land.wet_land_perha_price || 0) + Number(land.dry_land_wide || 0) * Number(land.dry_land_perha_price || 0) + Number(land.admin_price || 0) + Number(land.plant_price || 0);
const average = (land: Land) => Number(land.land_wide) > 0 ? Math.round(total(land) / Number(land.land_wide)) : 0;
const display = (value: unknown) => value == null || String(value).trim() === "" ? "—" : String(value);
const date = (value: string | null | undefined) => value ? new Date(value).toLocaleDateString("id-ID") : "—";

async function api<T>(path: string, init?: RequestInit, renewed = false): Promise<T> {
  const headers = new Headers(init?.headers);
  if (init?.body) headers.set("Content-Type", "application/json");
  const response = await fetch(`${apiBase}${path}`, { ...init, headers, credentials: "include", cache: "no-store" });
  if (response.status === 401 && !renewed) {
    const refresh = await fetch(`${apiBase}/api/auth/refresh`, { method: "POST", credentials: "include" });
    if (refresh.ok) return api<T>(path, init, true);
  }
  const result = await response.json().catch(() => null) as Envelope<T> | null;
  if (!response.ok) throw new Error(result?.message || "Permintaan gagal.");
  return result?.data as T;
}

function value(land: Land, column: Column): string | number {
  if (column === "average") return average(land);
  if (column === "total") return total(land);
  if (column === "status") return land.status === 0 ? "New Land" : land.submit_count ? "Submitted" : `Status ${land.status}`;
  if (column === "grit_date") return date(land.grit_date);
  return land[column] ?? "";
}

export default function SubmitMemoLandPage() {
  const profile = useUserProfile();
  const [companyID, setCompanyID] = useState(0);
  const [companySelected, setCompanySelected] = useState(false);
  const [companyReady, setCompanyReady] = useState(false);
  const allCompanies = companySelected && companyID === 0;
  const [lands, setLands] = useState<Land[]>([]);
  const [officer, setOfficer] = useState<Officer | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [query, setQuery] = useState("");
  useEffect(() => {
    const landID = new URLSearchParams(window.location.search).get("land_id");
    if (!landID || !/^\d+$/.test(landID)) return;
    const timer = window.setTimeout(() => setQuery(landID), 0);
    return () => window.clearTimeout(timer);
  }, []);
  const [sort, setSort] = useState<Column>("id");
  const [descending, setDescending] = useState(true);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [selected, setSelected] = useState<Land | null>(null);
  const [detail, setDetail] = useState<Detail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [modalError, setModalError] = useState("");
  const [memoID, setMemoID] = useState("");
  const [note, setNote] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    const key = `gis.selected-company.${profile.username.toLowerCase()}`;
    const read = () => { const saved = window.localStorage.getItem(key); setCompanySelected(saved !== null); setCompanyID(Number(saved) || 0); setCompanyReady(true); };
    const change = (event: Event) => { const item = (event as CustomEvent<{ id?: number; name?: string }>).detail; if (item) { setCompanySelected(true); setCompanyID(Number(item.id) || 0); } else read(); };
    read(); window.addEventListener("gis:company-change", change); window.addEventListener("storage", read);
    return () => { window.removeEventListener("gis:company-change", change); window.removeEventListener("storage", read); };
  }, [profile.username]);

  const load = useCallback(async () => {
    if (!companyReady || (!companyID && !allCompanies)) { setLands([]); setOfficer(null); return; }
    setLoading(true); setError("");
    try {
      const data = await api<{ officer: Officer | null; lands: Land[] }>(`/api/submit-memo-land?company_id=${companyID}`);
      setOfficer(data.officer);
      setLands([...new Map((data.lands ?? []).map((land) => [land.id, land])).values()]);
    } catch (cause) { setLands([]); setOfficer(null); setError(cause instanceof Error ? cause.message : "Data gagal dimuat."); }
    finally { setLoading(false); }
  }, [companyID, allCompanies, companyReady]);
  useEffect(() => { const timer = window.setTimeout(() => void load(), 0); return () => window.clearTimeout(timer); }, [load]);

  const shown = useMemo(() => {
    const text = query.trim().toLowerCase();
    const rows = lands.filter((land) => !text || columns.some(([key]) => String(value(land, key)).toLowerCase().includes(text)));
    rows.sort((a, b) => { const left = value(a, sort); const right = value(b, sort); const result = typeof left === "number" && typeof right === "number" ? left - right : String(left).localeCompare(String(right), "id", { numeric: true }); return descending ? -result : result; });
    return rows;
  }, [lands, query, sort, descending]);
  const pages = Math.max(1, Math.ceil(shown.length / pageSize));
  const currentPage = Math.min(page, pages);
  const pageRows = shown.slice((currentPage - 1) * pageSize, currentPage * pageSize);

  async function open(land: Land) {
    setSelected(land); setDetail(null); setDetailLoading(true); setModalError(""); setSuccess(""); setMemoID(""); setNote("");
    try { const loaded = await api<Detail>(`/api/submit-memo-land/${land.id}?company_id=${land.company_id}`); setDetail({ ...loaded, history: dedupeHistory(loaded.history) }); }
    catch (cause) { setModalError(cause instanceof Error ? cause.message : "Detail gagal dimuat."); }
    finally { setDetailLoading(false); }
  }
  function close() { if (!saving) { setSelected(null); setDetail(null); setModalError(""); } }
  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!selected || !memoID) { setModalError("Pilih memo request terlebih dahulu."); return; }
    setSaving(true); setModalError("");
    try {
      await api(`/api/submit-memo-land/${selected.id}?company_id=${selected.company_id}`, { method: "POST", body: JSON.stringify({ memo_id: Number(memoID), note }) });
      setSelected(null); setDetail(null); setMemoID(""); setNote("");
      setSuccess(`Submit Memo Land #${selected.id} berhasil disimpan.`);
      await load();
    } catch (cause) { setModalError(cause instanceof Error ? cause.message : "Submit gagal."); }
    finally { setSaving(false); }
  }
  const activeLand = detail?.land ?? selected;
  const canSubmit = !!officer && !!activeLand && (officer.company_id === activeLand.company_id || (allCompanies && officer.company_id === 0)) && activeLand.status === 0 && Number(activeLand.submit_count) === 0;

  return <div className="container-fluid py-5">
    <div className="mb-5"><h1 className="fs-2 fw-bold mb-1">Submit Memo Land</h1><p className="text-muted mb-0">{officer ? `Petugas: ${officer.name} · Company ${officer.company_id}` : "Lahan berdasarkan company aktif"}</p></div>
    {error && <div role="alert" className="alert alert-danger">{error}</div>}
    {success && <div role="status" className="alert alert-success">{success}</div>}
    <section className="card border shadow-sm"><div className="card-header flex-wrap gap-3 py-4"><span className="text-muted fw-semibold fs-7">Total data: {lands.length}</span><label className="d-flex align-items-center gap-2 ms-auto">Search<input type="search" className="form-control form-control-sm form-control-solid ref-table-search" value={query} onChange={(event) => { setQuery(event.target.value); setPage(1); }} /></label><label className="d-flex align-items-center gap-2">Show<select className="form-select form-select-sm w-auto" value={pageSize} onChange={(event) => { setPageSize(Number(event.target.value)); setPage(1); }}>{[10,25,50,100].map((size) => <option key={size}>{size}</option>)}</select>entries</label><button type="button" className="btn btn-sm btn-light" onClick={() => void load()} disabled={loading}>Refresh</button></div>
      <div className="card-body pt-0"><div className="table-responsive"><table className="table admin-data-table align-middle"><thead className="bg-dark"><tr>{columns.map(([key, title]) => <th key={key} className="bg-dark text-white text-nowrap"><button type="button" className="btn btn-link p-0 text-white fw-bold text-uppercase fs-8 text-decoration-none text-nowrap" onClick={() => { setSort(key); setDescending(sort === key ? !descending : false); }}>{title} {sort === key ? descending ? "↓" : "↑" : "↕"}</button></th>)}<th className="bg-dark text-white">Actions</th></tr></thead><tbody>
        {loading ? <TableLoadingRows columns={16} /> : !companyID && !allCompanies ? <tr><td colSpan={16} className="text-center py-12 text-muted">Pilih company aktif di header.</td></tr> : pageRows.length === 0 ? <tr><td colSpan={16} className="text-center py-12 text-muted">Belum ada lahan pada company ini.</td></tr> : pageRows.map((land) => <tr key={land.id}>{columns.map(([key]) => <td key={key} className="text-nowrap">{key === "status" ? <span className={`badge ${land.submit_count ? "badge-light-success" : "badge-light-primary"}`}>{value(land,key)}</span> : ["land_wide","wet_land_wide","dry_land_wide"].includes(key) ? area(value(land,key)) : ["wet_land_perha_price","dry_land_perha_price","average","admin_price","plant_price","total"].includes(key) ? money(value(land,key)) : display(value(land,key))}</td>)}<td><button type="button" className="btn btn-sm btn-light-primary text-nowrap" onClick={() => void open(land)}>{land.submit_count ? "History" : (officer?.company_id === land.company_id || (allCompanies && officer?.company_id === 0)) ? "Process" : "Detail"}</button></td></tr>)}
      </tbody></table></div></div><div className="card-footer d-flex align-items-center justify-content-between gap-3"><span className="text-muted">Showing {shown.length ? (currentPage - 1) * pageSize + 1 : 0}–{Math.min(currentPage * pageSize, shown.length)} of {shown.length}</span><div className="d-flex align-items-center gap-2"><button className="btn btn-sm btn-light" disabled={currentPage <= 1} onClick={() => setPage(currentPage - 1)}>Previous</button><span>{currentPage} / {pages}</span><button className="btn btn-sm btn-light" disabled={currentPage >= pages} onClick={() => setPage(currentPage + 1)}>Next</button></div></div></section>
    {selected && <div className="compensated-modal-backdrop" data-fullscreen="true" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) close(); }}><div className="card shadow-lg compensated-modal" data-fullscreen="true" role="dialog" aria-modal="true" aria-labelledby="submit-memo-title"><div className="card-header py-4 hcv-map-modal-header"><h2 id="submit-memo-title" className="card-title fs-4 fw-bold">Submit Memo Land · Land #{selected.id}</h2><button type="button" className="btn btn-sm btn-icon btn-light" onClick={close} aria-label="Tutup" disabled={saving}>×</button></div>
      {modalError && <div role="alert" className="alert alert-danger rounded-0 mb-0">{modalError}</div>}
      {detailLoading ? <div className="p-8 text-muted">Memuat proses dan detail lahan...</div> : activeLand && detail && <div className="submit-memo-workspace"><aside className="submit-memo-process"><div className="d-flex justify-content-end mb-2"><button type="button" className="btn btn-sm btn-light" onClick={close}>Back</button></div><AcquisitionProcessOverview history={detail.history} /><div className="alert alert-light-primary py-3"><strong>Status: {activeLand.status === 0 ? "New Propose Land" : `Status ${activeLand.status}`}</strong></div>
        <div className="submit-memo-timeline">{steps.map((step, index) => { const records = detail.history.filter((item) => item.sequence_id === acquisitionStages[index][0]); return <AcquisitionStage key={step} title={step} number={index + 1} count={records.length} active={index === 0 && canSubmit}>{index === 0 && canSubmit ? <form onSubmit={(event) => void submit(event)}><strong className="d-block mb-2">Request Land Payment</strong><label className="form-label" htmlFor="submit-memo-select">Select Memo Request</label><select id="submit-memo-select" className="form-select form-select-sm mb-3" value={memoID} onChange={(event) => setMemoID(event.target.value)} required><option value="">-- Select --</option>{detail.memos.map((memo) => <option key={memo.id} value={memo.id}>{memo.memo_number} · {date(memo.memo_date)} · {memo.status_name || `Status ${memo.memo_status}`}</option>)}</select><label className="form-label" htmlFor="submit-memo-note">Note</label><textarea id="submit-memo-note" className="form-control mb-3" rows={4} maxLength={10000} value={note} onChange={(event) => setNote(event.target.value)} /><button type="submit" className="btn btn-sm btn-primary" disabled={saving || !memoID}>{saving ? "Menyimpan..." : "Submit"}</button><div className="text-muted fs-7 mt-3">{officer?.name || profile.name}</div></form> : records.length ? <div className="d-flex flex-column gap-2">{records.map((record, recordIndex) => <div className="submit-memo-history" key={`${record.id}-${recordIndex}`}><ProcessApprovalBadge approval={record.approval} /><div className="fs-7 mt-2">{record.request_number && <div>Memo: {record.request_number}</div>}<div>{record.employee_name || record.username} · {date(record.date_created)}</div>{record.note && <div>Note: {record.note}</div>}</div></div>)}</div> : <p className="text-muted fst-italic mb-1">{index === 0 ? "Hanya petugas Submit Memo Land untuk company ini yang dapat memproses lahan." : "Waiting previous process"}</p>}</AcquisitionStage>; })}</div>
      </aside><div className="submit-memo-main"><div className="submit-memo-map"><CompensatedMap key={activeLand.id} id={activeLand.id} companyID={activeLand.company_id} title="Geographic Overview" editable={false} initialMode="view" apiPath="/api/submit-memo-land" /></div><AcquisitionLandTabs key={activeLand.id} land={activeLand} apiPath="/api/submit-memo-land" memoNumber={detail.history.find((item) => item.sequence_id === 1)?.request_number ?? undefined} /></div></div>}
    </div></div>}
  </div>;
}
