"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import Image from "next/image";
import { useUserProfile } from "../../../../components/auth/user-profile-context";
import "./report-period.css";

type Sequence = { id: number; name: string };
type Land = {
  id: number;
  owner: string;
  status: number;
  date_created: string;
  grtt_date: string;
  stage_dates: Record<string, string>;
  area: number;
  area_source: string;
  total_price: number;
  price_per_hectare: number;
};
type Division = {
  id: number;
  name: string;
  company: string;
  permit_number: string;
  area: number;
  has_spatial_area: boolean;
  lands: Land[];
  total_area: number;
  total_price: number;
  price_per_hectare: number;
  progress_percent: number;
};
type PeriodReport = {
  start_date: string;
  end_date: string;
  sequences: Sequence[];
  divisions: Division[];
  land_count: number;
  truncated: boolean;
};
type Envelope<T> = { data?: T; message?: string };

const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");
const dateValue = (value: Date) => `${value.getFullYear()}-${String(value.getMonth() + 1).padStart(2, "0")}-${String(value.getDate()).padStart(2, "0")}`;
const currentMonth = () => {
  const now = new Date();
  return {
    start: dateValue(new Date(now.getFullYear(), now.getMonth(), 1)),
    end: dateValue(new Date(now.getFullYear(), now.getMonth() + 1, 0)),
  };
};
const number = (value: number, digits = 0) => new Intl.NumberFormat("id-ID", {
  minimumFractionDigits: digits,
  maximumFractionDigits: digits,
}).format(Number(value) || 0);
const date = (value: string, short = false) => value
  ? new Date(`${value}T00:00:00`).toLocaleDateString("id-ID", {
      day: "2-digit",
      month: "short",
      year: short ? "2-digit" : "numeric",
    })
  : "—";

async function getReport(path: string): Promise<PeriodReport> {
  let response = await fetch(`${apiBase}${path}`, { credentials: "include", cache: "no-store" });
  if (response.status === 401) {
    const renewal = await fetch(`${apiBase}/api/auth/refresh`, { method: "POST", credentials: "include" });
    if (renewal.ok) response = await fetch(`${apiBase}${path}`, { credentials: "include", cache: "no-store" });
  }
  const body = await response.json().catch(() => null) as Envelope<PeriodReport> | null;
  if (!response.ok) throw new Error(body?.message || "Report gagal dimuat.");
  if (!body?.data) throw new Error("Data report kosong.");
  return body.data;
}

function DivisionTable({ division, sequences, active }: {
  division: Division;
  sequences: Sequence[];
  active: boolean;
}) {
  return (
    <section
      className={`card border-0 shadow-sm report-period-division ${active ? "is-active" : ""}`}
      id={`report-division-panel-${division.id}`}
      role="tabpanel"
      aria-labelledby={`report-division-tab-${division.id}`}
      aria-hidden={!active}
    >
      <div className="card-header d-flex flex-wrap align-items-center justify-content-between gap-2">
        <div>
          <h2 className="card-title fw-bold fs-4 mb-0">{division.name || `Division #${division.id}`}</h2>
          <div className="text-muted fs-7">{division.company} · {division.permit_number || "Tanpa nomor permit"}</div>
        </div>
        <div className="text-muted fs-7 text-end">
          <div>{number(division.lands.length)} lahan</div>
          <div>Luas divisi: {division.has_spatial_area ? `${number(division.area, 2)} ha` : "Koordinat tidak tersedia"}</div>
        </div>
      </div>
      <div className="report-period-table-wrap">
        <table className="table table-row-bordered align-middle mb-0 report-period-table" style={{ minWidth: Math.max(1400, 470 + sequences.length * 110) }}>
          <thead>
            <tr>
              <th className="report-period-owner">Owner / Land ID</th>
              {sequences.map((sequence) => <th key={sequence.id}>{sequence.name}</th>)}
              <th className="text-end report-period-area">Area (ha)</th>
              <th className="text-end report-period-money">Harga (IDR)</th>
              <th className="text-end report-period-money">Harga/ha (IDR)</th>
            </tr>
          </thead>
          <tbody>
            {division.lands.map((land) => (
              <tr key={land.id}>
                <td className="report-period-owner">
                  <strong>{land.owner || "—"}</strong>
                  <div className="text-muted fs-8">#{land.id}</div>
                </td>
                {sequences.map((sequence) => {
                  const stageDate = land.stage_dates[String(sequence.id)] || "";
                  return <td key={sequence.id}>
                    <span className="report-period-screen-date">{date(stageDate)}</span>
                    <span className="report-period-print-date">{date(stageDate, true)}</span>
                  </td>;
                })}
                <td className="text-end">{number(land.area, 2)}{land.area_source !== "spatial" && <span title="Menggunakan luas tercatat karena koordinat tidak dapat dihitung"> *</span>}</td>
                <td className="text-end">{number(land.total_price)}</td>
                <td className="text-end">{number(land.price_per_hectare)}</td>
              </tr>
            ))}
            <tr className="fw-bold report-period-total">
              <td colSpan={sequences.length + 1} className="text-end">TOTAL</td>
              <td className="text-end">{number(division.total_area, 2)}</td>
              <td className="text-end">{number(division.total_price)}</td>
              <td className="text-end">{number(division.price_per_hectare)}</td>
            </tr>
            <tr>
              <td colSpan={sequences.length + 1} className="text-end fw-bold">% area divisi</td>
              <td className="text-end">{division.has_spatial_area ? `${number(division.progress_percent, 2)}%` : "—"}</td>
              <td colSpan={2} />
            </tr>
          </tbody>
        </table>
      </div>
      {division.lands.length === 0 && <div className="card-footer text-muted">Tidak ada lahan pada periode ini.</div>}
    </section>
  );
}

export default function ListAcquisitionProcessPage() {
  const profile = useUserProfile();
  const initial = useMemo(() => currentMonth(), []);
  const [start, setStart] = useState(initial.start);
  const [end, setEnd] = useState(initial.end);
  const [appliedStart, setAppliedStart] = useState(initial.start);
  const [appliedEnd, setAppliedEnd] = useState(initial.end);
  const [companyID, setCompanyID] = useState(0);
  const [companySelected, setCompanySelected] = useState(false);
  const [companyReady, setCompanyReady] = useState(false);
  const [report, setReport] = useState<PeriodReport | null>(null);
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
    const params = new URLSearchParams({ company_id: String(companyID), start_date: appliedStart, end_date: appliedEnd });
    try {
      const data = await getReport(`/api/reports/rlist_acq_process/period?${params}`);
      if (active()) setReport(data);
    } catch (cause) {
      if (active()) {
        setReport(null);
        setError(cause instanceof Error ? cause.message : "Report gagal dimuat.");
      }
    } finally {
      if (active()) setLoading(false);
    }
  }, [appliedEnd, appliedStart, companyID, companyReady, companySelected]);

  useEffect(() => {
    let active = true;
    const timer = window.setTimeout(() => void load(() => active), 0);
    return () => { active = false; window.clearTimeout(timer); };
  }, [load]);

  const divisions = report?.divisions ?? [];
  const activeDivisionID = divisions.some((division) => division.id === selectedDivisionID)
    ? selectedDivisionID
    : (divisions.find((division) => division.lands.length > 0)?.id ?? divisions[0]?.id ?? null);
  const companies = [...new Set(divisions.map((division) => division.company).filter(Boolean))];
  const permits = [...new Set(divisions.map((division) => division.permit_number).filter(Boolean))];
  const totals = divisions.reduce((value, division) => ({
    area: value.area + division.total_area,
    price: value.price + division.total_price,
  }), { area: 0, price: 0 });

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!start || !end || end < start) {
      setError("Tanggal awal harus sama atau sebelum tanggal akhir.");
      return;
    }
    setSelectedDivisionID(null);
    setAppliedStart(start);
    setAppliedEnd(end);
  }

  return <div className="report-period-page d-flex flex-column gap-5 py-5">
    <div className="report-period-heading d-flex flex-wrap align-items-center justify-content-between gap-3">
      <div>
        <span className="text-muted fw-semibold text-uppercase fs-7">Report</span>
        <h1 className="fw-bold fs-2 mb-0">List Acquisition Progress</h1>
      </div>
      <div className="d-flex align-items-center gap-4">
        {report && <Image className="report-period-logo" src="/metronic/upload/aeplogonew.png" alt="AEP Plantations" width={80} height={80} loading="eager" style={{ objectFit: "contain" }} />}
        <button type="button" className="btn btn-light-primary report-period-print" disabled={!report || loading} onClick={() => window.print()}>
          Cetak / Simpan PDF
        </button>
      </div>
    </div>

    <form className="card border-0 shadow-sm report-period-toolbar" onSubmit={submit}>
      <div className="card-body d-flex flex-wrap gap-3 align-items-end">
        <div>
          <label className="form-label fw-semibold" htmlFor="report-start">Awal periode GRTT</label>
          <input className="form-control" id="report-start" type="date" value={start} onChange={(event) => setStart(event.target.value)} required />
        </div>
        <div>
          <label className="form-label fw-semibold" htmlFor="report-end">Akhir periode GRTT</label>
          <input className="form-control" id="report-end" type="date" value={end} onChange={(event) => setEnd(event.target.value)} required />
        </div>
        <button className="btn btn-primary" type="submit" disabled={loading}>Tampilkan</button>
      </div>
    </form>

    {!companySelected && companyReady && <div className="alert alert-info">Pilih company aktif di header untuk melihat report.</div>}
    {error && <div className="alert alert-danger" role="alert">{error}</div>}
    {loading && <div className="text-muted">Memuat report...</div>}

    {report && <>
      {report.truncated && <div className="alert alert-warning">Hasil dibatasi 10.000 lahan. Persempit periode GRTT agar seluruh data masuk.</div>}
      <div className="card border-0 shadow-sm report-period-summary">
        <div className="card-body d-flex flex-wrap justify-content-between gap-4">
          <div>
            <div className="text-muted fs-7">Company</div>
            <strong>{companies.join(", ") || (companyID === 0 ? "Semua company" : `Company #${companyID}`)}</strong>
            <div className="text-muted fs-7 mt-3">Land permit</div>
            <strong>{permits.join(", ") || "—"}</strong>
          </div>
          <div>
            <div className="text-muted fs-7">Periode GRTT</div>
            <strong>{date(report.start_date)} – {date(report.end_date)}</strong>
            <div className="text-muted fs-7 mt-3">Tanggal report</div>
            <strong>{new Date().toLocaleString("id-ID")}</strong>
          </div>
          <div>
            <div className="text-muted fs-7">Total lahan</div>
            <strong>{number(report.land_count)}</strong>
            <div className="text-muted fs-7 mt-3">Luas / harga</div>
            <strong>{number(totals.area, 2)} ha / Rp {number(totals.price)}</strong>
          </div>
        </div>
      </div>

      {divisions.length > 0 && <nav className="report-period-tabs d-flex gap-2 pb-2" role="tablist" aria-label="Pilih divisi">
        {divisions.map((division) => <button
          key={division.id}
          id={`report-division-tab-${division.id}`}
          type="button"
          role="tab"
          aria-selected={division.id === activeDivisionID}
          aria-controls={`report-division-panel-${division.id}`}
          className="report-period-tab"
          onClick={() => setSelectedDivisionID(division.id)}
        >
          <span className="d-block fw-bold text-truncate" title={division.name}>{division.name || `Division #${division.id}`}</span>
          <span className="d-block fs-8">{number(division.lands.length)} lahan · {number(division.total_area, 2)} ha</span>
        </button>)}
      </nav>}

      {divisions.map((division) => <DivisionTable
        key={division.id}
        division={division}
        sequences={report.sequences}
        active={division.id === activeDivisionID}
      />)}
      {divisions.length === 0 && <div className="alert alert-info">Belum ada divisi untuk company ini.</div>}
      <p className="text-muted fs-7">* Luas memakai land_wide saat mapdata spasial tidak dapat dihitung. Periode mencakup tanggal awal dan akhir.</p>
    </>}
  </div>;
}
