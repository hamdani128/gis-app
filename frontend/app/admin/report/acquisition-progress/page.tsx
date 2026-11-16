"use client";

import { useCallback, useEffect, useState } from "react";
import Image from "next/image";
import Link from "next/link";
import { useUserProfile } from "../../../../components/auth/user-profile-context";
import "./acquisition-progress.css";

type Metric = { area: number; percent: number; total_price: number; land_count: number };
type Stage = Metric & { id: number; sequence: number; name: string };
type Division = {
  id: number;
  name: string;
  company: string;
  permit_number: string;
  area: number;
  has_spatial_area: boolean;
  stages: Stage[];
  compensated: Metric;
  total: Metric;
  missing_spatial_count: number;
};
type Summary = {
  company_name: string;
  permits: string[];
  generated_at: string;
  divisions: Division[];
};
type Envelope<T> = { data?: T; message?: string };

const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");
const number = (value: number, digits = 0) => new Intl.NumberFormat("id-ID", {
  minimumFractionDigits: digits,
  maximumFractionDigits: digits,
}).format(Number(value) || 0);

async function getSummary(companyID: number): Promise<Summary> {
  const path = `/api/reports/acqprogress/summary?company_id=${companyID}`;
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

function MetricRow({ label, metric, emphasis = false }: { label: string; metric: Metric; emphasis?: boolean }) {
  return <tr className={emphasis ? "acquisition-progress-emphasis" : ""}>
    <th scope="row">{label}</th>
    <td className="text-end">{number(metric.land_count)}</td>
    <td className="text-end">{number(metric.area, 2)}</td>
    <td className="text-end">{number(metric.percent, 2)}%</td>
    <td className="text-end">{number(metric.total_price)}</td>
  </tr>;
}

function DivisionSection({ division, active }: { division: Division; active: boolean }) {
  return <section
    id={`acquisition-progress-panel-${division.id}`}
    className={`card border-0 shadow-sm acquisition-progress-division ${active ? "is-active" : ""}`}
    role="tabpanel"
    aria-labelledby={`acquisition-progress-tab-${division.id}`}
    aria-hidden={!active}
  >
    <div className="card-header d-flex flex-wrap align-items-center justify-content-between gap-2">
      <div>
        <h2 className="card-title fw-bold fs-4 mb-0">{division.name || `Division #${division.id}`}</h2>
        <div className="text-muted fs-7">{division.company} · {division.permit_number || "Tanpa nomor permit"}</div>
      </div>
      <div className="text-end">
        <div className="text-muted fs-8">Luas divisi</div>
        <strong>{division.has_spatial_area ? `${number(division.area, 2)} ha` : "Koordinat tidak tersedia"}</strong>
      </div>
    </div>
    <div className="acquisition-progress-table-wrap">
      <table className="table table-row-bordered align-middle mb-0 acquisition-progress-table">
        <thead><tr><th scope="col">Tahap akuisisi</th><th scope="col" className="text-end">Jumlah lahan</th><th scope="col" className="text-end">Luas (ha)</th><th scope="col" className="text-end">% divisi</th><th scope="col" className="text-end">Nilai (IDR)</th></tr></thead>
        <tbody>
          {division.stages.map((stage) => <MetricRow key={stage.id} label={stage.name || `Status ${stage.sequence}`} metric={stage} />)}
          <MetricRow label="Compensated land" metric={division.compensated} emphasis />
          <MetricRow label="TOTAL" metric={division.total} emphasis />
        </tbody>
      </table>
    </div>
    {division.missing_spatial_count > 0 && <div className="card-footer text-muted fs-8">
      {number(division.missing_spatial_count)} lahan belum memiliki koordinat polygon valid dan tidak masuk perhitungan.
    </div>}
  </section>;
}

export default function AcquisitionProgressPage() {
  const profile = useUserProfile();
  const [companyID, setCompanyID] = useState(0);
  const [companyReady, setCompanyReady] = useState(false);
  const [companySelected, setCompanySelected] = useState(false);
  const [report, setReport] = useState<Summary | null>(null);
  const [selectedDivisionID, setSelectedDivisionID] = useState<number | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const key = `gis.selected-company.${profile.username.toLowerCase()}`;
    const read = () => {
      const saved = window.localStorage.getItem(key);
      setCompanySelected(saved !== null);
      setCompanyID(Number(saved) || 0);
      setCompanyReady(true);
      setSelectedDivisionID(null);
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
    if (!companyReady || !companySelected) { setReport(null); return; }
    setLoading(true);
    setError("");
    try {
      const data = await getSummary(companyID);
      if (active()) setReport(data);
    } catch (cause) {
      if (active()) {
        setReport(null);
        setError(cause instanceof Error ? cause.message : "Report gagal dimuat.");
      }
    } finally {
      if (active()) setLoading(false);
    }
  }, [companyID, companyReady, companySelected]);

  useEffect(() => {
    let active = true;
    const timer = window.setTimeout(() => void load(() => active), 0);
    return () => { active = false; window.clearTimeout(timer); };
  }, [load]);

  const divisions = report?.divisions ?? [];
  const activeDivisionID = divisions.some((division) => division.id === selectedDivisionID)
    ? selectedDivisionID
    : (divisions.find((division) => division.total.land_count > 0)?.id ?? divisions[0]?.id ?? null);
  const totals = divisions.reduce((value, division) => ({
    area: value.area + division.total.area,
    price: value.price + division.total.total_price,
    lands: value.lands + division.total.land_count,
  }), { area: 0, price: 0, lands: 0 });

  return <div className="acquisition-progress-page d-flex flex-column gap-5 py-5">
    <div className="acquisition-progress-heading d-flex flex-wrap justify-content-between align-items-center gap-3">
      <div><span className="text-muted fw-semibold text-uppercase fs-7">Report</span><h1 className="fw-bold fs-2 mb-0">Land Acquisition Progress</h1></div>
      <div className="d-flex align-items-center gap-4">
        {report && <Image className="acquisition-progress-logo" src="/metronic/upload/aeplogonew.png" alt="AEP Plantations" width={80} height={80} loading="eager" />}
        <button type="button" className="btn btn-light-primary acquisition-progress-print" disabled={!report || loading} onClick={() => window.print()}>Cetak / Simpan PDF</button>
      </div>
    </div>

    {!companySelected && companyReady && <div className="alert alert-info">Pilih company aktif di header untuk melihat report.</div>}
    {error && <div className="alert alert-danger" role="alert">{error}</div>}
    {loading && <div className="text-muted">Memuat report...</div>}

    {report && <>
      <div className="card border-0 shadow-sm acquisition-progress-summary">
        <div className="card-body d-flex flex-wrap justify-content-between gap-4">
          <div><div className="text-muted fs-7">Company</div><strong>{report.company_name || (companyID === 0 ? "Semua company" : `Company #${companyID}`)}</strong><div className="text-muted fs-7 mt-3">Land permit</div><strong>{report.permits.join(", ") || "—"}</strong></div>
          <div><div className="text-muted fs-7">Tanggal report</div><strong>{new Date(report.generated_at).toLocaleString("id-ID")}</strong><div className="text-muted fs-7 mt-3">Divisi</div><strong>{number(divisions.length)}</strong></div>
          <div><div className="text-muted fs-7">Total lahan / luas</div><strong>{number(totals.lands)} / {number(totals.area, 2)} ha</strong><div className="text-muted fs-7 mt-3">Total nilai</div><strong>Rp {number(totals.price)}</strong></div>
        </div>
      </div>

      {divisions.length > 0 && <nav className="acquisition-progress-tabs d-flex gap-2 pb-2" role="tablist" aria-label="Pilih divisi">
        {divisions.map((division) => <button
          key={division.id}
          id={`acquisition-progress-tab-${division.id}`}
          type="button"
          role="tab"
          aria-selected={division.id === activeDivisionID}
          aria-controls={`acquisition-progress-panel-${division.id}`}
          className="acquisition-progress-tab"
          onClick={() => setSelectedDivisionID(division.id)}
        >
          <span className="d-block fw-bold text-truncate" title={division.name}>{division.name || `Division #${division.id}`}</span>
          <span className="d-block fs-8">{number(division.total.land_count)} lahan · {number(division.total.area, 2)} ha</span>
        </button>)}
      </nav>}

      {divisions.map((division) => <DivisionSection key={division.id} division={division} active={division.id === activeDivisionID} />)}
      {divisions.length === 0 && <div className="alert alert-info">Belum ada divisi untuk company ini.</div>}
      <div className="d-flex justify-content-end acquisition-progress-more"><Link href="/admin/report/list-acquisition-process" className="btn btn-light-primary">Lihat detail proses akuisisi</Link></div>
    </>}
  </div>;
}
