"use client";

import { useCallback, useEffect, useState } from "react";
import type { FormEvent } from "react";
import Image from "next/image";
import { useUserProfile } from "../../../../components/auth/user-profile-context";
import "./all-estate-progress.css";

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
type Summary = { generated_at: string; start_date: string; end_date: string; years: number[]; companies: Company[] };
type Envelope<T> = { data?: T; message?: string };

const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");
const number = (value: number, digits = 0) => new Intl.NumberFormat("id-ID", {
  minimumFractionDigits: digits,
  maximumFractionDigits: digits,
}).format(Number(value) || 0);

async function getSummary(companyID: number, start: string, end: string): Promise<Summary> {
  const params = new URLSearchParams({ company_id: String(companyID) });
  if (start && end) { params.set("start_date", start); params.set("end_date", end); }
  const path = `/api/reports/rpt_all_progress/summary?${params}`;
  let response = await fetch(`${apiBase}${path}`, { credentials: "include", cache: "no-store" });
  if (response.status === 401) {
    const renewal = await fetch(`${apiBase}/api/auth/refresh`, { method: "POST", credentials: "include" });
    if (renewal.ok) response = await fetch(`${apiBase}${path}`, { credentials: "include", cache: "no-store" });
  }
  const body = await response.json().catch(() => null) as Envelope<Summary> | null;
  if (!response.ok) throw new Error(body?.message || "Report gagal dimuat.");
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

function CompanySection({ company, years, active, tabbed }: { company: Company; years: number[]; active: boolean; tabbed: boolean }) {
  const total = company.estates.reduce((value, estate) => ({
    potential: value.potential + estate.potential_area,
    compensated: value.compensated + estate.compensated.area,
    payment: value.payment + estate.payment_approval.area,
    progress: value.progress + estate.on_progress.area,
  }), { potential: 0, compensated: 0, payment: 0, progress: 0 });

  return <section
    id={`all-estate-panel-${company.id}`}
    className={`card border-0 shadow-sm all-estate-company ${active ? "is-active" : ""}`}
    role={tabbed ? "tabpanel" : undefined}
    aria-labelledby={tabbed ? `all-estate-tab-${company.id}` : undefined}
    aria-hidden={tabbed ? !active : undefined}
  >
    <div className="card-header d-flex flex-wrap align-items-center justify-content-between gap-3">
      <div><h2 className="card-title fw-bold fs-4 mb-0">{company.name || `Company #${company.id}`}</h2><div className="text-muted fs-7">{number(company.estates.length)} estate</div></div>
      <div className="d-flex flex-wrap gap-5 text-end">
        <div><div className="text-muted fs-8">Permit Land Area</div><strong>{number(company.permit_area, 2)} ha</strong></div>
        <div><div className="text-muted fs-8">HCV Land Area</div><strong>{number(company.hcv_area, 2)} ha</strong></div>
      </div>
    </div>
    <div className="all-estate-table-wrap">
      <table className="table table-row-bordered align-middle mb-0 all-estate-table">
        <thead>
          <tr><th rowSpan={2} scope="col">Estate</th><th rowSpan={2} scope="col" className="text-end">Potential Area (ha)</th><th colSpan={3} scope="colgroup">Compensated land</th><th colSpan={3} scope="colgroup">Payment Approval land</th><th colSpan={3} scope="colgroup">On Progress land</th></tr>
          <tr>{["compensated", "payment", "progress"].map((group) => <FragmentHeaders key={group} />)}</tr>
        </thead>
        <tbody>
          {company.estates.map((estate) => <tr key={estate.id}>
            <th scope="row">{estate.name || `Estate #${estate.id}`}</th>
            <td className="text-end">{estate.has_spatial_area ? number(estate.potential_area, 2) : "—"}</td>
            <MetricCells metric={estate.compensated} />
            <MetricCells metric={estate.payment_approval} />
            <MetricCells metric={estate.on_progress} />
          </tr>)}
          {company.estates.length === 0 && <tr><td colSpan={11} className="text-center text-muted py-6">Belum ada estate.</td></tr>}
        </tbody>
      </table>
    </div>
    <div className="all-estate-overview px-7 pb-5 text-muted fs-7">Total potential area {number(total.potential, 2)} ha · Compensated {number(total.compensated, 2)} ha · Payment Approval {number(total.payment, 2)} ha · On Progress {number(total.progress, 2)} ha</div>
    {years.length > 0 && <div className="px-7 pb-7">
      <h3 className="fw-bold fs-5 mb-3">Rata-rata harga lahan per ha menurut tahun GRTT</h3>
      <div className="all-estate-year-wrap">
        <table className="table table-row-bordered align-middle mb-0 all-estate-year-table">
          <thead><tr><th scope="col">Estate</th>{years.map((year) => <th scope="col" className="text-end" key={year}>{year}</th>)}</tr></thead>
          <tbody>{company.estates.map((estate) => <tr key={estate.id}><th scope="row">{estate.name || `Estate #${estate.id}`}</th>{years.map((year) => <td className="text-end" key={year}>{estate.average_price_by_year[String(year)] ? number(estate.average_price_by_year[String(year)]) : "—"}</td>)}</tr>)}</tbody>
        </table>
      </div>
    </div>}
    {company.estates.some((estate) => estate.missing_spatial_count > 0) && <div className="card-footer text-muted fs-8">Lahan tanpa polygon valid tidak masuk hitungan luas dan nilai status. Harga rata-rata tahunan tetap memakai luas wet/dry yang tercatat.</div>}
  </section>;
}

function FragmentHeaders() {
  return <><th scope="col" className="text-end">Area (ha)</th><th scope="col" className="text-end">%</th><th scope="col" className="text-end">Price (IDR)</th></>;
}

export default function AllEstateProgressPage() {
  const profile = useUserProfile();
  const [companyID, setCompanyID] = useState(0);
  const [companyReady, setCompanyReady] = useState(false);
  const [report, setReport] = useState<Summary | null>(null);
  const [selectedCompanyID, setSelectedCompanyID] = useState<number | null>(null);
  const [start, setStart] = useState("");
  const [end, setEnd] = useState("");
  const [appliedStart, setAppliedStart] = useState("");
  const [appliedEnd, setAppliedEnd] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const key = `gis.selected-company.${profile.username.toLowerCase()}`;
    const read = () => {
      const saved = window.localStorage.getItem(key);
      setCompanyID(Number(saved) || 0);
      setCompanyReady(true);
      setSelectedCompanyID(null);
    };
    read();
    window.addEventListener("gis:company-change", read);
    window.addEventListener("storage", read);
    return () => {
      window.removeEventListener("gis:company-change", read);
      window.removeEventListener("storage", read);
    };
  }, [profile.username]);

  const load = useCallback(async (active: () => boolean) => {
    if (!companyReady) { setReport(null); return; }
    setLoading(true);
    setError("");
    try {
      const data = await getSummary(companyID, appliedStart, appliedEnd);
      if (active()) setReport(data);
    } catch (cause) {
      if (active()) { setReport(null); setError(cause instanceof Error ? cause.message : "Report gagal dimuat."); }
    } finally {
      if (active()) setLoading(false);
    }
  }, [companyID, companyReady, appliedStart, appliedEnd]);

  useEffect(() => {
    let active = true;
    const timer = window.setTimeout(() => void load(() => active), 0);
    return () => { active = false; window.clearTimeout(timer); };
  }, [load]);

  const companies = report?.companies ?? [];
  const tabbed = companyID === 0 && companies.length > 1;
  const activeCompanyID = tabbed
    ? (companies.some((company) => company.id === selectedCompanyID)
      ? selectedCompanyID
      : (companies.find((company) => company.estates.length > 0)?.id ?? companies[0]?.id ?? null))
    : companies[0]?.id ?? null;
  const estateCount = companies.reduce((sum, company) => sum + company.estates.length, 0);

  const applyPeriod = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if ((start || end) && (!start || !end || end < start)) {
      setError("Isi tanggal awal dan akhir periode GRTT dengan benar.");
      return;
    }
    setError("");
    setAppliedStart(start);
    setAppliedEnd(end);
  };

  const clearPeriod = () => {
    setError("");
    setStart("");
    setEnd("");
    setAppliedStart("");
    setAppliedEnd("");
  };

  return <div className="all-estate-page d-flex flex-column gap-5 py-5">
    <div className="all-estate-heading d-flex flex-wrap justify-content-between align-items-center gap-3">
      <div><span className="text-muted fw-semibold text-uppercase fs-7">Report</span><h1 className="fw-bold fs-2 mb-0">Land Acquisition Progress All Company</h1></div>
      <div className="d-flex align-items-center gap-4">
        {report && <Image className="all-estate-logo" src="/metronic/upload/aeplogonew.png" alt="AEP Plantations" width={80} height={80} loading="eager" />}
        <button type="button" className="btn btn-light-primary all-estate-print" disabled={!report || loading} onClick={() => window.print()}>Cetak / Simpan PDF</button>
      </div>
    </div>
    {report && <div className="all-estate-print-period">Periode GRTT: {report.start_date && report.end_date ? `${report.start_date} s.d. ${report.end_date}` : "Semua periode"} · Dicetak {new Date(report.generated_at).toLocaleString("id-ID")}</div>}
    <form className="card border-0 shadow-sm all-estate-filter" onSubmit={applyPeriod}>
      <div className="card-body d-flex flex-wrap align-items-end gap-4">
        <div><label htmlFor="all-estate-start" className="form-label fw-semibold">Periode GRTT dari</label><input id="all-estate-start" type="date" className="form-control" value={start} max={end || undefined} onChange={(event) => setStart(event.target.value)} /></div>
        <div><label htmlFor="all-estate-end" className="form-label fw-semibold">Sampai</label><input id="all-estate-end" type="date" className="form-control" value={end} min={start || undefined} onChange={(event) => setEnd(event.target.value)} /></div>
        <button type="submit" className="btn btn-primary" disabled={loading}>Terapkan periode</button>
        <button type="button" className="btn btn-light" disabled={loading} onClick={clearPeriod}>Semua periode</button>
      </div>
    </form>
    {error && <div className="alert alert-danger" role="alert">{error}</div>}
    {loading && <div className="text-muted">Memuat report...</div>}
    {report && <>
      <div className="card border-0 shadow-sm all-estate-summary"><div className="card-body d-flex flex-wrap justify-content-between gap-4">
        <div><div className="text-muted fs-7">Cakupan</div><strong>{companyID === 0 ? "Semua company" : companies[0]?.name || "Company aktif"}</strong></div>
        <div><div className="text-muted fs-7">Total company / estate</div><strong>{number(companies.length)} / {number(estateCount)}</strong></div>
        <div><div className="text-muted fs-7">Periode GRTT</div><strong>{report.start_date && report.end_date ? `${report.start_date} s.d. ${report.end_date}` : "Semua periode"}</strong></div>
        <div><div className="text-muted fs-7">Tanggal report</div><strong>{new Date(report.generated_at).toLocaleString("id-ID")}</strong></div>
      </div></div>
      {tabbed && <nav className="all-estate-tabs d-flex gap-2 pb-2" role="tablist" aria-label="Pilih company">
        {companies.map((company) => <button key={company.id} id={`all-estate-tab-${company.id}`} type="button" role="tab" aria-selected={company.id === activeCompanyID} aria-controls={`all-estate-panel-${company.id}`} className="all-estate-tab" onClick={() => setSelectedCompanyID(company.id)}><span className="d-block fw-bold text-truncate" title={company.name}>{company.name || `Company #${company.id}`}</span><span className="d-block fs-8">{number(company.estates.length)} estate</span></button>)}
      </nav>}
      {companies.map((company) => <CompanySection key={company.id} company={company} years={report.years} active={company.id === activeCompanyID} tabbed={tabbed} />)}
      {companies.length === 0 && <div className="alert alert-info">Belum ada company atau estate untuk report ini.</div>}
    </>}
  </div>;
}
