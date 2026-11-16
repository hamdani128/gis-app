"use client";

import { useCallback, useEffect, useState } from "react";
import Image from "next/image";
import { useUserProfile } from "../../../../components/auth/user-profile-context";
import "./estate-progress-report.css";

type Metric = { area: number; percent: number; total_price: number; land_count: number };
type Estate = {
  id: number;
  name: string;
  potential_area: number;
  has_spatial_area: boolean;
  compensated: Metric;
  payment_approval: Metric;
  on_progress: Metric;
  average_price_by_year: Record<string, number>;
  missing_spatial_count: number;
};
type Company = { id: number; name: string; permit_area: number; hcv_area: number; estates: Estate[] };
type Month = { estate_id: number; year: number; month: number; area: number; price: number; average: number };
type Summary = { generated_at: string; years: number[]; companies: Company[]; monthly: Month[] };
type Envelope<T> = { data?: T; message?: string };

const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");
const months = ["Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"];
const number = (value: number, digits = 0) => new Intl.NumberFormat("id-ID", {
  minimumFractionDigits: digits,
  maximumFractionDigits: digits,
}).format(Number(value) || 0);

async function getSummary(companyID: number): Promise<Summary> {
  const url = `${apiBase}/api/reports/aestategrttreport/summary?company_id=${companyID}`;
  let response = await fetch(url, { credentials: "include", cache: "no-store" });
  if (response.status === 401) {
    const renewal = await fetch(`${apiBase}/api/auth/refresh`, { method: "POST", credentials: "include" });
    if (renewal.ok) response = await fetch(url, { credentials: "include", cache: "no-store" });
  }
  const body = await response.json().catch(() => null) as Envelope<Summary> | null;
  if (!response.ok) throw new Error(body?.message || "Estate Progress Report gagal dimuat.");
  if (!body?.data) throw new Error("Data report kosong.");
  return body.data;
}

function MetricCells({ metric }: { metric: Metric }) {
  return <>
    <td className="text-end">{number(metric.area, 2)}</td>
    <td className="text-end">{number(metric.percent, 2)}%</td>
    <td className="text-end">{number(metric.total_price)}</td>
  </>;
}

function CompanyReport({ company, years, monthly, active, tabbed }: {
  company: Company;
  years: number[];
  monthly: Month[];
  active: boolean;
  tabbed: boolean;
}) {
  const [selectedYear, setSelectedYear] = useState<number | null>(null);
  const estateIDs = new Set(company.estates.map((estate) => estate.id));
  const companyMonths = monthly.filter((item) => estateIDs.has(item.estate_id));
  const monthlyYears = [...new Set(companyMonths.map((item) => item.year))].sort((a, b) => b - a);
  const activeYear = selectedYear !== null && monthlyYears.includes(selectedYear) ? selectedYear : monthlyYears[0];
  const byKey = new Map(companyMonths.map((item) => [`${item.estate_id}-${item.year}-${item.month}`, item]));
  const missingSpatial = company.estates.reduce((sum, estate) => sum + estate.missing_spatial_count, 0);

  return <section className={`card border-0 shadow-sm estate-progress-company ${active ? "is-active" : ""}`}
    id={`estate-progress-panel-${company.id}`} role={tabbed ? "tabpanel" : undefined}
    aria-labelledby={tabbed ? `estate-progress-tab-${company.id}` : undefined} aria-hidden={tabbed ? !active : undefined}>
    <div className="card-header d-flex flex-wrap align-items-center justify-content-between gap-3">
      <div><h2 className="card-title fw-bold fs-4 mb-0">{company.name || `Company #${company.id}`}</h2><div className="text-muted fs-7">{number(company.estates.length)} estate</div></div>
      <div className="d-flex flex-wrap gap-5 text-end">
        <div><div className="text-muted fs-8">Permit Land Area</div><strong>{number(company.permit_area, 2)} ha</strong></div>
        <div><div className="text-muted fs-8">HCV Land Area</div><strong>{number(company.hcv_area, 2)} ha</strong></div>
      </div>
    </div>
    <div className="card-body">
      <h3 className="fw-bold fs-5 mb-3">Progress akuisisi per estate</h3>
      <div className="estate-progress-table-wrap"><table className="table table-row-bordered align-middle mb-0 estate-progress-table">
        <thead><tr><th rowSpan={2}>Estate</th><th rowSpan={2} className="text-end">Potential Area (ha)</th><th colSpan={3}>Compensated land</th><th colSpan={3}>Payment Approval land</th><th colSpan={3}>On Progress land</th></tr>
          <tr>{["compensated", "payment", "progress"].map((name) => <MetricHeaders key={name} />)}</tr></thead>
        <tbody>{company.estates.map((estate) => <tr key={estate.id}><th scope="row">{estate.name || `Estate #${estate.id}`}</th><td className="text-end">{estate.has_spatial_area ? number(estate.potential_area, 2) : "—"}</td><MetricCells metric={estate.compensated} /><MetricCells metric={estate.payment_approval} /><MetricCells metric={estate.on_progress} /></tr>)}
          {!company.estates.length && <tr><td colSpan={11} className="text-center text-muted py-6">Belum ada estate.</td></tr>}</tbody>
      </table></div>

      <h3 className="fw-bold fs-5 mt-7 mb-3">Rata-rata harga lahan per ha menurut tahun GRTT</h3>
      <div className="estate-progress-table-wrap"><table className="table table-row-bordered align-middle mb-0 estate-progress-year-table">
        <thead><tr><th>Estate</th>{years.map((year) => <th key={year} className="text-end">{year}</th>)}</tr></thead>
        <tbody>{company.estates.map((estate) => <tr key={estate.id}><th scope="row">{estate.name || `Estate #${estate.id}`}</th>{years.map((year) => <td key={year} className="text-end">{estate.average_price_by_year[String(year)] ? number(estate.average_price_by_year[String(year)]) : "—"}</td>)}</tr>)}
          {!company.estates.length && <tr><td colSpan={years.length + 1} className="text-center text-muted py-6">Belum ada estate.</td></tr>}</tbody>
      </table></div>

      <div className="d-flex flex-wrap align-items-center justify-content-between gap-3 mt-7 mb-3"><h3 className="fw-bold fs-5 mb-0">Rata-rata harga bulanan menurut tahun GRTT</h3>
        {monthlyYears.length > 1 && <div className="estate-progress-year-tabs" role="tablist" aria-label="Tahun GRTT">{monthlyYears.map((year) => <button key={year} type="button" role="tab" aria-selected={year === activeYear} className="btn btn-sm btn-light-primary" onClick={() => setSelectedYear(year)}>{year}</button>)}</div>}</div>
      {monthlyYears.map((year) => <div key={year} className={`estate-progress-month-section ${year === activeYear ? "is-active" : ""}`}>
        <h4 className="fw-bold fs-6 mb-3">Tahun {year}</h4>
        <div className="estate-progress-table-wrap"><table className="table table-row-bordered align-middle mb-0 estate-progress-month-table"><thead><tr><th>Estate</th><th>Ukuran</th>{months.map((month) => <th key={month} className="text-end">{month}</th>)}</tr></thead>
          <tbody>{company.estates.map((estate) => (["Rata-rata/ha", "Luas (ha)", "Harga (Rp)"] as const).map((label, index) => <tr key={`${estate.id}-${label}`}>{index === 0 && <th rowSpan={3} scope="rowgroup">{estate.name || `Estate #${estate.id}`}</th>}<th scope="row">{label}</th>{months.map((_, monthIndex) => { const item = byKey.get(`${estate.id}-${year}-${monthIndex + 1}`); const value = index === 0 ? item?.average : index === 1 ? item?.area : item?.price; return <td key={monthIndex} className="text-end">{item ? number(value ?? 0, index === 1 ? 2 : 0) : "—"}</td>; })}</tr>))}</tbody></table></div>
      </div>)}
      {!monthlyYears.length && <p className="text-muted mb-0">Belum ada tanggal GRTT untuk perhitungan bulanan.</p>}
      {missingSpatial > 0 && <p className="text-muted fs-8 mt-5 mb-0">{number(missingSpatial)} lahan tanpa polygon valid tidak dihitung dalam luas dan harga per status. Harga rata-rata tahunan dan bulanan tetap memakai luas wet/dry tercatat.</p>}
    </div>
  </section>;
}

function MetricHeaders() {
  return <><th className="text-end">Area (ha)</th><th className="text-end">%</th><th className="text-end">Price (IDR)</th></>;
}

export default function EstateProgressReportPage() {
  const profile = useUserProfile();
  const [companyID, setCompanyID] = useState(0);
  const [companyReady, setCompanyReady] = useState(false);
  const [selectedCompanyID, setSelectedCompanyID] = useState<number | null>(null);
  const [report, setReport] = useState<Summary | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const key = `gis.selected-company.${profile.username.toLowerCase()}`;
    const read = () => { setCompanyID(Number(window.localStorage.getItem(key)) || 0); setCompanyReady(true); setSelectedCompanyID(null); };
    read();
    window.addEventListener("gis:company-change", read);
    window.addEventListener("storage", read);
    return () => { window.removeEventListener("gis:company-change", read); window.removeEventListener("storage", read); };
  }, [profile.username]);

  const load = useCallback(async (active: () => boolean) => {
    if (!companyReady) return;
    setLoading(true); setError("");
    try { const data = await getSummary(companyID); if (active()) setReport(data); }
    catch (cause) { if (active()) { setReport(null); setError(cause instanceof Error ? cause.message : "Report gagal dimuat."); } }
    finally { if (active()) setLoading(false); }
  }, [companyID, companyReady]);
  useEffect(() => { let active = true; const timer = window.setTimeout(() => void load(() => active), 0); return () => { active = false; window.clearTimeout(timer); }; }, [load]);

  const companies = report?.companies ?? [];
  const tabbed = companyID === 0 && companies.length > 1;
  const activeCompanyID = tabbed
    ? (companies.some((company) => company.id === selectedCompanyID) ? selectedCompanyID : (companies.find((company) => company.estates.length)?.id ?? companies[0]?.id ?? null))
    : companies[0]?.id ?? null;
  const estateCount = companies.reduce((sum, company) => sum + company.estates.length, 0);

  return <div className="estate-progress-page d-flex flex-column gap-5 py-5">
    <div className="estate-progress-heading d-flex flex-wrap align-items-center justify-content-between gap-3"><div><span className="text-muted fw-semibold text-uppercase fs-7">Report</span><h1 className="fw-bold fs-2 mb-0">Estate Progress Report</h1><p className="text-muted mb-0">Land Acquisition Progress menurut estate dan tahun GRTT.</p></div><div className="d-flex align-items-center gap-4"><Image className="estate-progress-logo" src="/metronic/upload/aeplogonew.png" alt="AEP Plantations" width={80} height={80} /><button type="button" className="btn btn-light-primary estate-progress-print" disabled={!report || loading} onClick={() => window.print()}>Cetak / Simpan PDF</button></div></div>
    {error && <div className="alert alert-danger" role="alert">{error}</div>}
    {loading && <p className="text-muted">Memuat report...</p>}
    {report && <><div className="card border-0 shadow-sm estate-progress-summary"><div className="card-body d-flex flex-wrap justify-content-between gap-4"><div><div className="text-muted fs-7">Cakupan</div><strong>{companyID === 0 ? "Semua company" : companies[0]?.name || "Company aktif"}</strong></div><div><div className="text-muted fs-7">Total company / estate</div><strong>{number(companies.length)} / {number(estateCount)}</strong></div><div><div className="text-muted fs-7">Tanggal report</div><strong>{new Date(report.generated_at).toLocaleString("id-ID")}</strong></div></div></div>
      {tabbed && <nav className="estate-progress-tabs" role="tablist" aria-label="Pilih company">{companies.map((company) => <button key={company.id} id={`estate-progress-tab-${company.id}`} type="button" role="tab" aria-selected={company.id === activeCompanyID} aria-controls={`estate-progress-panel-${company.id}`} onClick={() => setSelectedCompanyID(company.id)}><strong>{company.name || `Company #${company.id}`}</strong><span>{number(company.estates.length)} estate</span></button>)}</nav>}
      {companies.map((company) => <CompanyReport key={company.id} company={company} years={report.years} monthly={report.monthly} active={company.id === activeCompanyID} tabbed={tabbed} />)}
      {!companies.length && <div className="alert alert-info">Belum ada company atau estate untuk report ini.</div>}
    </>}
  </div>;
}
