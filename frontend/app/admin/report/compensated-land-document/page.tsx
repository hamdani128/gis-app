"use client";

import { useCallback, useEffect, useState } from "react";
import type { FormEvent } from "react";
import { useUserProfile } from "../../../../components/auth/user-profile-context";
import LandHistoryDocuments from "../../acquisition/land-history-documents";
import "./compensated-land-document.css";

type Land = {
  id: number; division: string; estate: string; owner: string; status: number; date_created: string; village: string;
  land_wide: number; wet_land_wide: number; wet_land_perha_price: number;
  dry_land_wide: number; dry_land_perha_price: number; cost_per_ha: number;
  admin_price: number; plant_price: number; total_price: number; document_count: number;
};
type Report = { rows: Land[]; total: number; page: number; page_size: number; total_area: number; total_price: number };
type Envelope<T> = { data?: T; message?: string };
const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");
const monthNames = ["Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"];
const number = (value: number, digits = 0) => new Intl.NumberFormat("id-ID", { minimumFractionDigits: digits, maximumFractionDigits: digits }).format(Number(value) || 0);

async function getReport(companyID: number, count: string, year: string, month: string, search: string, page: number, pageSize: number): Promise<Report> {
  const params = new URLSearchParams({ company_id: String(companyID), page: String(page), page_size: String(pageSize) });
  if (count !== "") params.set("count_doc", count);
  if (year !== "") params.set("year", year);
  if (month !== "") params.set("month", month);
  if (search) params.set("search", search);
  const url = `${apiBase}/api/reports/reportdocland?${params}`;
  let response = await fetch(url, { credentials: "include", cache: "no-store" });
  if (response.status === 401) {
    const refresh = await fetch(`${apiBase}/api/auth/refresh`, { method: "POST", credentials: "include" });
    if (refresh.ok) response = await fetch(url, { credentials: "include", cache: "no-store" });
  }
  const body = await response.json().catch(() => null) as Envelope<Report> | null;
  if (!response.ok) throw new Error(body?.message || "Report dokumen gagal dimuat.");
  if (!body?.data) throw new Error("Data report kosong.");
  return body.data;
}

export default function CompensatedLandDocumentPage() {
  const profile = useUserProfile();
  const [companyID, setCompanyID] = useState(0);
  const [companyReady, setCompanyReady] = useState(false);
  const [countInput, setCountInput] = useState("");
  const [count, setCount] = useState("");
  const [yearInput, setYearInput] = useState("");
  const [monthInput, setMonthInput] = useState("");
  const [year, setYear] = useState("");
  const [month, setMonth] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(25);
  const [report, setReport] = useState<Report | null>(null);
  const [selected, setSelected] = useState<Land | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const key = `gis.selected-company.${profile.username.toLowerCase()}`;
    const read = () => { setCompanyID(Number(window.localStorage.getItem(key)) || 0); setCompanyReady(true); setPage(1); setSelected(null); };
    read();
    window.addEventListener("gis:company-change", read);
    window.addEventListener("storage", read);
    return () => { window.removeEventListener("gis:company-change", read); window.removeEventListener("storage", read); };
  }, [profile.username]);

  const load = useCallback(async (active: () => boolean) => {
    if (!companyReady) return;
    setLoading(true); setError("");
    try { const data = await getReport(companyID, count, year, month, search, page, pageSize); if (active()) setReport(data); }
    catch (cause) { if (active()) { setReport(null); setError(cause instanceof Error ? cause.message : "Report dokumen gagal dimuat."); } }
    finally { if (active()) setLoading(false); }
  }, [companyID, companyReady, count, year, month, search, page, pageSize]);
  useEffect(() => { let active = true; const timer = window.setTimeout(() => void load(() => active), 0); return () => { active = false; window.clearTimeout(timer); }; }, [load]);

  function applyCount(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (countInput !== "" && (!/^\d+$/.test(countInput) || Number(countInput) > 1000000)) { setError("Jumlah dokumen harus angka nol atau lebih."); return; }
    if (yearInput !== "" && (!/^\d{4}$/.test(yearInput) || Number(yearInput) < 1900 || Number(yearInput) > 2100)) { setError("Tahun harus antara 1900 dan 2100."); return; }
    setPage(1); setCount(countInput); setYear(yearInput); setMonth(monthInput); setError("");
  }
  function applySearch(event: FormEvent<HTMLFormElement>) { event.preventDefault(); setPage(1); setSearch(searchInput.trim()); }
  const total = report?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const rows = report?.rows ?? [];
  const period = `${month ? monthNames[Number(month) - 1] : "Semua bulan"} · ${year || "Semua tahun"}`;

  return <div className="document-report-page d-flex flex-column gap-5 py-5">
    <div className="d-flex flex-wrap align-items-center justify-content-between gap-3"><div><span className="text-muted fw-semibold text-uppercase fs-7">Report</span><h1 className="text-gray-900 fw-bold fs-2 mb-1">Compensated Land Document</h1><p className="text-muted mb-0">Lahan compensated (status 10) berdasarkan company aktif. Tanggal dibuat: {period}.</p></div><button type="button" className="btn btn-light-primary document-report-print" disabled={!report || loading} onClick={() => window.print()}>Cetak halaman ini</button></div>
    <form className="card border-0 shadow-sm document-report-filter" onSubmit={applyCount}><div className="card-body d-flex flex-wrap align-items-end gap-3"><div><label htmlFor="document-count" className="form-label fw-semibold">Filter Doc Count</label><input id="document-count" type="number" min="0" step="1" className="form-control" value={countInput} onChange={(event) => setCountInput(event.target.value)} placeholder="Semua jumlah" /></div><div><label htmlFor="document-year" className="form-label fw-semibold">Tahun lahan dibuat</label><input id="document-year" type="number" min="1900" max="2100" step="1" className="form-control" value={yearInput} onChange={(event) => setYearInput(event.target.value)} placeholder="Semua tahun" /></div><div><label htmlFor="document-month" className="form-label fw-semibold">Bulan lahan dibuat</label><select id="document-month" className="form-select" value={monthInput} onChange={(event) => setMonthInput(event.target.value)}><option value="">Semua bulan</option>{monthNames.map((name, index) => <option key={name} value={index + 1}>{name}</option>)}</select></div><button type="submit" className="btn btn-primary" disabled={loading}>Terapkan</button><button type="button" className="btn btn-light" disabled={loading} onClick={() => { setCountInput(""); setCount(""); setYearInput(""); setYear(""); setMonthInput(""); setMonth(""); setPage(1); }}>Semua</button><span className="text-muted fs-7 pb-2">Isi 0 untuk lahan tanpa dokumen.</span></div></form>
    {error && <div className="alert alert-danger" role="alert">{error}</div>}
    <div className="row g-4">{[["Total lahan", number(total)], ["Total luas (ha)", number(report?.total_area ?? 0, 2)], ["Total harga", `Rp ${number(report?.total_price ?? 0)}`], ["Filter dokumen", count === "" ? "Semua" : count]].map(([label, value]) => <div className="col-sm-6 col-xl-3" key={label}><div className="card border-0 shadow-sm h-100"><div className="card-body p-5"><div className="text-muted fs-7 mb-2">{label}</div><strong className="fs-3">{value}</strong></div></div></div>)}</div>
    <section className="card border-0 shadow-sm"><div className="card-header d-flex flex-wrap align-items-center gap-3"><h2 className="card-title fw-bold fs-4 mb-0">Data lahan</h2><span className="text-muted ms-auto">{number(total)} data</span></div>
      <div className="card-body pb-0 document-report-controls"><div className="d-flex flex-wrap gap-3 align-items-end mb-4"><form className="d-flex gap-2 flex-grow-1" onSubmit={applySearch}><input type="search" className="form-control" aria-label="Cari lahan" placeholder="Cari Land ID, owner, divisi, estate, desa..." value={searchInput} onChange={(event) => setSearchInput(event.target.value)} /><button type="submit" className="btn btn-primary">Cari</button></form><label className="d-flex align-items-center gap-2 text-muted">Tampilkan <select className="form-select" aria-label="Jumlah baris" value={pageSize} onChange={(event) => { setPageSize(Number(event.target.value)); setPage(1); }}><option value={10}>10</option><option value={25}>25</option><option value={50}>50</option><option value={100}>100</option></select></label></div></div>
      <div className="table-responsive document-report-table-wrap"><table className="table table-row-bordered table-hover align-middle mb-0 document-report-table"><thead><tr><th>Div</th><th>Land ID</th><th>Owner</th><th>Status</th><th>Tanggal dibuat</th><th>Village</th><th className="text-end">Land Size (ha)</th><th className="text-end">Wet Size (ha)</th><th className="text-end">Wet Price/ha</th><th className="text-end">Dry Size (ha)</th><th className="text-end">Dry Price/ha</th><th className="text-end">Avg/ha</th><th className="text-end">Admin Price</th><th className="text-end">Plant Price</th><th className="text-end">Total Price</th><th className="text-end">Doc</th><th>Actions</th></tr></thead>
        <tbody>{loading && !report ? <tr><td colSpan={17} className="text-center text-muted py-8">Memuat lahan...</td></tr> : rows.length === 0 ? <tr><td colSpan={17} className="text-center text-muted py-8">Tidak ada lahan untuk filter ini.</td></tr> : rows.map((land) => <tr key={land.id}><td>{land.division || "—"}</td><td className="fw-bold">{land.id}</td><td>{land.owner || "—"}</td><td><span className="badge badge-light-success">Compensated Land</span></td><td>{land.date_created || "—"}</td><td>{land.village || "—"}</td><td className="text-end">{number(land.land_wide, 2)}</td><td className="text-end">{number(land.wet_land_wide, 2)}</td><td className="text-end">{number(land.wet_land_perha_price)}</td><td className="text-end">{number(land.dry_land_wide, 2)}</td><td className="text-end">{number(land.dry_land_perha_price)}</td><td className="text-end">{number(land.cost_per_ha)}</td><td className="text-end">{number(land.admin_price)}</td><td className="text-end">{number(land.plant_price)}</td><td className="text-end fw-semibold">{number(land.total_price)}</td><td className="text-end"><span className={`badge ${land.document_count ? "badge-light-primary" : "badge-light"}`}>{number(land.document_count)}</span></td><td><button type="button" className="btn btn-sm btn-light-primary text-nowrap" onClick={() => setSelected(land)}>Lihat dokumen</button></td></tr>)}</tbody></table></div>
      <div className="card-footer d-flex flex-wrap align-items-center justify-content-between gap-3"><span className="text-muted">Menampilkan {total ? (page - 1) * pageSize + 1 : 0}–{Math.min(page * pageSize, total)} dari {total}</span><div className="d-flex align-items-center gap-3"><button type="button" className="btn btn-sm btn-light" disabled={page <= 1 || loading} onClick={() => setPage((value) => value - 1)}>Sebelumnya</button><span>{page} / {totalPages}</span><button type="button" className="btn btn-sm btn-light" disabled={page >= totalPages || loading} onClick={() => setPage((value) => value + 1)}>Berikutnya</button></div></div>
    </section>
    {selected && <div className="modal d-block show document-report-modal" role="dialog" aria-modal="true" aria-label={`Dokumen lahan ${selected.id}`}><div className="modal-dialog modal-xl modal-dialog-centered"><div className="modal-content"><div className="modal-header"><div><h2 className="modal-title fs-4">Dokumen · Land #{selected.id}</h2><div className="text-muted fs-7">{selected.owner} · {selected.division}</div></div><button type="button" className="btn-close" aria-label="Tutup dokumen" onClick={() => setSelected(null)} /></div><div className="modal-body overflow-auto"><LandHistoryDocuments landID={selected.id} companyID={companyID} apiPath="/api/reports/reportdocland" /></div><div className="modal-footer"><button type="button" className="btn btn-light" onClick={() => setSelected(null)}>Tutup</button></div></div></div></div>}
  </div>;
}
