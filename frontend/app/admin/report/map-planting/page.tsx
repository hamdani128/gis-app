"use client";

import { useCallback, useEffect, useState } from "react";
import type { FormEvent } from "react";
import { useUserProfile } from "../../../../components/auth/user-profile-context";
import PlantingMapDialog from "./planting-map-dialog";
import "./map-planting.css";

export type Estate = {
  id: number; potential_name: string; estate_name: string; estate_address: string;
  estate_manager: string; permit_number: string; note: string; company_name: string;
};
type EstatePage = { rows: Estate[]; total: number; page: number; page_size: number };
type Envelope<T> = { data?: T; message?: string };
const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");

async function getEstates(companyID: number, search: string, page: number, pageSize: number): Promise<EstatePage> {
  const params = new URLSearchParams({ company_id: String(companyID), page: String(page), page_size: String(pageSize) });
  if (search) params.set("search", search);
  const url = `${apiBase}/api/reports/plantmaprep?${params}`;
  let response = await fetch(url, { credentials: "include", cache: "no-store" });
  if (response.status === 401) {
    const refresh = await fetch(`${apiBase}/api/auth/refresh`, { method: "POST", credentials: "include" });
    if (refresh.ok) response = await fetch(url, { credentials: "include", cache: "no-store" });
  }
  const result = await response.json().catch(() => null) as Envelope<EstatePage> | null;
  if (!response.ok) throw new Error(result?.message || "Map Planting gagal dimuat.");
  if (!result?.data) throw new Error("Data Map Planting kosong.");
  return result.data;
}

export default function MapPlantingPage() {
  const profile = useUserProfile();
  const [companyID, setCompanyID] = useState(0);
  const [companyReady, setCompanyReady] = useState(false);
  const [searchInput, setSearchInput] = useState("");
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(25);
  const [data, setData] = useState<EstatePage | null>(null);
  const [selected, setSelected] = useState<Estate | null>(null);
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
    try { const result = await getEstates(companyID, search, page, pageSize); if (active()) setData(result); }
    catch (cause) { if (active()) { setData(null); setError(cause instanceof Error ? cause.message : "Map Planting gagal dimuat."); } }
    finally { if (active()) setLoading(false); }
  }, [companyID, companyReady, search, page, pageSize]);
  useEffect(() => { let active = true; const timer = window.setTimeout(() => void load(() => active), 0); return () => { active = false; window.clearTimeout(timer); }; }, [load]);
  function submitSearch(event: FormEvent<HTMLFormElement>) { event.preventDefault(); setPage(1); setSearch(searchInput.trim()); }

  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  return <div className="planting-report-page d-flex flex-column gap-5 py-5">
    <div className="d-flex flex-wrap align-items-center justify-content-between gap-3"><div><span className="text-muted fw-semibold text-uppercase fs-7">Report</span><h1 className="text-gray-900 fw-bold fs-2 mb-1">Map Planting</h1><p className="text-muted mb-0">Peta penanaman berdasarkan estate pada company aktif.</p></div><button type="button" className="btn btn-light-primary planting-print" disabled={!data || loading} onClick={() => window.print()}>Cetak daftar</button></div>
    {error && <div className="alert alert-danger" role="alert">{error}</div>}
    <section className="card border-0 shadow-sm">
      <div className="card-header d-flex flex-wrap align-items-center gap-3 py-4"><h2 className="card-title fw-bold fs-4 mb-0">Daftar estate</h2><span className="text-muted ms-auto">{new Intl.NumberFormat("id-ID").format(total)} data</span></div>
      <div className="card-body pb-0"><div className="d-flex flex-wrap gap-3 align-items-end mb-4 planting-controls">
        <form onSubmit={submitSearch} className="d-flex gap-2 flex-grow-1"><input className="form-control" type="search" aria-label="Cari estate" placeholder="Cari potential, estate, alamat, manager, permit..." value={searchInput} onChange={(event) => setSearchInput(event.target.value)} /><button type="submit" className="btn btn-primary">Cari</button></form>
        <label className="d-flex align-items-center gap-2 text-muted">Tampilkan <select className="form-select" aria-label="Jumlah baris" value={pageSize} onChange={(event) => { setPageSize(Number(event.target.value)); setPage(1); }}><option value={10}>10</option><option value={25}>25</option><option value={50}>50</option><option value={100}>100</option></select></label>
      </div></div>
      <div className="table-responsive planting-table-wrap"><table className="table table-row-bordered table-hover align-middle mb-0 planting-table">
        <thead><tr><th>Potential name</th><th>Estate name</th><th>Estate address</th><th>Est. Manager</th><th>Land Permit</th><th>Note</th><th>Actions</th></tr></thead>
        <tbody>{loading && !data ? <tr><td colSpan={7} className="text-center text-muted py-8">Memuat estate...</td></tr> : (data?.rows ?? []).length === 0 ? <tr><td colSpan={7} className="text-center text-muted py-8">Tidak ada estate yang sesuai.</td></tr> : data?.rows.map((estate) => <tr key={estate.id}><td className="fw-bold">{estate.potential_name || "—"}</td><td>{estate.estate_name || "—"}</td><td>{estate.estate_address || "—"}</td><td>{estate.estate_manager || "—"}</td><td>{estate.permit_number || "—"}</td><td>{estate.note || "—"}</td><td><button type="button" className="btn btn-sm btn-light-primary text-nowrap" onClick={() => setSelected(estate)}>View Map Report</button></td></tr>)}</tbody>
      </table></div>
      <div className="card-footer d-flex flex-wrap align-items-center justify-content-between gap-3"><span className="text-muted">Menampilkan {total ? (page - 1) * pageSize + 1 : 0}–{Math.min(page * pageSize, total)} dari {total}</span><div className="d-flex align-items-center gap-3"><button type="button" className="btn btn-sm btn-light" disabled={page <= 1 || loading} onClick={() => setPage((value) => value - 1)}>Sebelumnya</button><span>{page} / {totalPages}</span><button type="button" className="btn btn-sm btn-light" disabled={page >= totalPages || loading} onClick={() => setPage((value) => value + 1)}>Berikutnya</button></div></div>
    </section>
    {selected && <PlantingMapDialog key={selected.id} estate={selected} companyID={companyID} onClose={() => setSelected(null)} />}
  </div>;
}
