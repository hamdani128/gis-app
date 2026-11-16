"use client";

import TableLoadingRows from "../../../../components/ui/table-loading-rows";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import { useUserProfile } from "../../../../components/auth/user-profile-context";
import HCVMap from "./hcv-map";

function MapIcon() { return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m3 6 6-3 6 3 6-3v15l-6 3-6-3-6 3V6Zm6-3v15m6-12v15" stroke="currentColor" strokeWidth="1.8" strokeLinejoin="round" /></svg>; }
function SettingsIcon() { return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M12 3v3m0 12v3M3 12h3m12 0h3M5.6 5.6l2.1 2.1m8.6 8.6 2.1 2.1M18.4 5.6l-2.1 2.1m-8.6 8.6-2.1 2.1" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" /><circle cx="12" cy="12" r="4" stroke="currentColor" strokeWidth="1.8" /></svg>; }
function EditIcon() { return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m14 5 5 5M4 20l4.5-1 10.8-10.8a2.12 2.12 0 0 0-3-3L5.5 16 4 20Z" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>; }
function TrashIcon() { return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M4 7h16M10 11v6m4-6v6M6 7l1 14h10l1-14M9 7V4h6v3" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>; }

type Row = { id: number; name: string; land_permit_id: number; land_permit: string; spatial_type_id: number | null; spatial_type: string; area_length: string; note: string };
type Option = { id: number; name: string };
type Options = { land_permits: Option[]; spatial_types: Option[] };
type Form = { name: string; land_permit_id: number; spatial_type_id: number | null; area_length: string; note: string };
type Envelope<T> = { data?: T; message?: string };
const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");
const blankForm = (): Form => ({ name: "", land_permit_id: 0, spatial_type_id: null, area_length: "", note: "" });

async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  if (init?.body && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  const response = await fetch(`${apiBase}${path}`, { ...init, headers, credentials: "include", cache: "no-store" });
  const result = (await response.json().catch(() => null)) as Envelope<T> | null;
  if (!response.ok) throw new Error(result?.message || "Permintaan gagal.");
  return result?.data as T;
}

export default function HCVLandsPage() {
  const profile = useUserProfile();
  const [companyID, setCompanyID] = useState(0);
  const [companyName, setCompanyName] = useState("");
  const isAllCompanies = companyName.trim() === "00" || /^00(?:\b|[\s-])/i.test(companyName);
  const [items, setItems] = useState<Row[]>([]);
  const [options, setOptions] = useState<Options>({ land_permits: [], spatial_types: [] });
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(25);
  const [editing, setEditing] = useState<Row | null>(null);
  const [form, setForm] = useState<Form>(blankForm());
  const [modal, setModal] = useState<"form" | "map" | null>(null);
  const [mapMode, setMapMode] = useState<"view" | "settings">("view");

  useEffect(() => {
    const storageKey = `gis.selected-company.${profile.username.toLowerCase()}`;
    const read = () => {
      const id = Number(window.localStorage.getItem(storageKey)) || 0;
      setCompanyID(id);
      if (!id) setCompanyName("");
    };
    const changed = (event: Event) => {
      const detail = (event as CustomEvent<{ id?: number; name?: string }>).detail;
      if (detail && Number.isFinite(Number(detail.id))) { const name = detail.name ?? ""; setCompanyID(/^(?:00)(?:\b|[\s-])/i.test(name.trim()) || name.trim() === "00" ? 0 : Number(detail.id)); setCompanyName(name); }
      else read();
    };
    read(); window.addEventListener("gis:company-change", changed); window.addEventListener("storage", read);
    return () => { window.removeEventListener("gis:company-change", changed); window.removeEventListener("storage", read); };
  }, [profile.username]);

  const load = useCallback(async () => {
    if (!companyID && !isAllCompanies) { setItems([]); setOptions({ land_permits: [], spatial_types: [] }); setLoading(false); return; }
    setLoading(true); setError("");
    try {
      const params = `?company_id=${companyID}`;
      const [rows, opts] = await Promise.all([apiRequest<Row[]>(`/api/hcv-lands${params}`), apiRequest<Options>(`/api/hcv-lands/options${params}`)]);
      setItems(Array.isArray(rows) ? rows : []);
      setOptions({ land_permits: opts?.land_permits ?? [], spatial_types: opts?.spatial_types ?? [] });
    } catch (e) { setItems([]); setError(e instanceof Error ? e.message : "Data HCV Lands gagal dimuat."); }
    finally { setLoading(false); }
  }, [companyID, isAllCompanies]);
  useEffect(() => { const timer = window.setTimeout(() => { void load(); }, 0); return () => window.clearTimeout(timer); }, [load]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return q ? items.filter((item) => [item.name, item.land_permit, item.spatial_type, item.area_length, item.note].some((value) => String(value ?? "").toLowerCase().includes(q))) : items;
  }, [items, query]);
  const pageCount = Math.max(1, Math.ceil(filtered.length / pageSize));
  const currentPage = Math.min(page, pageCount);
  const shown = filtered.slice((currentPage - 1) * pageSize, currentPage * pageSize);

  function openCreate() { setEditing(null); setForm(blankForm()); setError(""); setModal("form"); }
  function openEdit(item: Row) { setEditing(item); setForm({ name: item.name, land_permit_id: item.land_permit_id, spatial_type_id: item.spatial_type_id, area_length: item.area_length, note: item.note }); setError(""); setModal("form"); }
  function openMap(item: Row, mode: "view" | "settings") { setEditing(item); setMapMode(mode); setError(""); setModal("map"); }
  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setSaving(true); setError("");
    const payload = { ...form, land_permit_id: Number(form.land_permit_id), spatial_type_id: form.spatial_type_id ? Number(form.spatial_type_id) : null, area_length: form.area_length.trim() || null, note: form.note.trim() || null };
    try { await apiRequest<Row>(editing ? `/api/hcv-lands/${editing.id}?company_id=${companyID}` : `/api/hcv-lands?company_id=${companyID}`, { method: editing ? "PUT" : "POST", body: JSON.stringify(payload) }); setModal(null); setEditing(null); setForm(blankForm()); await load(); }
    catch (e) { setError(e instanceof Error ? e.message : "Data HCV Land gagal disimpan."); }
    finally { setSaving(false); }
  }
  async function remove(item: Row) {
    const confirmation = window.Swal
      ? await window.Swal.fire({ title: "Hapus HCV Land ini?", text: `${item.name} akan dihapus.`, icon: "warning", showCancelButton: true, confirmButtonText: "Ya, hapus", cancelButtonText: "Batal", reverseButtons: true, buttonsStyling: false, customClass: { popup: "rounded-4 shadow-lg", title: "fw-bold text-gray-900", htmlContainer: "text-muted", confirmButton: "btn btn-danger ms-2", cancelButton: "btn btn-light me-2" } })
      : { isConfirmed: window.confirm(`Hapus HCV Land ${item.name}?`) };
    if (!confirmation.isConfirmed) return;
    try { await apiRequest<null>(`/api/hcv-lands/${item.id}?company_id=${companyID}`, { method: "DELETE" }); await load(); if (window.Swal) await window.Swal.fire({ title: "Berhasil", text: "HCV Land berhasil dihapus.", icon: "success", buttonsStyling: false, confirmButtonText: "Oke", customClass: { popup: "rounded-4 shadow-lg", title: "fw-bold text-gray-900", confirmButton: "btn btn-dark" } }); }
    catch (e) { setError(e instanceof Error ? e.message : "Data HCV Land gagal dihapus."); }
  }

  return <div className="d-flex flex-column gap-6">
    <nav aria-label="Breadcrumb"><ol className="breadcrumb breadcrumb-separatorless fw-semibold fs-7 mb-0"><li className="breadcrumb-item"><Link href="/admin/dashboard" className="text-muted text-hover-primary">Admin</Link></li><li className="breadcrumb-item"><span className="bullet bg-gray-300 w-5px h-2px" /></li><li className="breadcrumb-item text-muted">Lands</li><li className="breadcrumb-item"><span className="bullet bg-gray-300 w-5px h-2px" /></li><li className="breadcrumb-item text-muted">HCV Lands</li></ol></nav>
    <div className="d-flex flex-wrap align-items-center justify-content-between gap-4"><div><h1 className="text-gray-900 fw-bolder fs-2 mb-1">HCV Lands</h1><p className="text-muted fw-semibold mb-0">Company aktif: <strong>{isAllCompanies ? "Semua company (00)" : companyName || "Belum dipilih"}</strong></p></div><button type="button" className="btn btn-dark" onClick={openCreate} disabled={!companyID && !isAllCompanies}><span className="me-2 fs-3 lh-1">+</span>Add HCV Land</button></div>
    {error && !modal && <div className="alert alert-danger d-flex align-items-center justify-content-between mb-0" role="alert"><span>{error}</span><button type="button" className="btn-close" aria-label="Tutup pesan error" onClick={() => setError("")} /></div>}
    <section className="card border shadow-sm"><div className="card-header flex-wrap gap-4 py-4"><span className="text-muted fw-semibold fs-7">Total data: {items.length}</span><label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7 ms-auto">Search<input className="form-control form-control-sm form-control-solid ref-table-search" type="search" placeholder="Name, permit, spatial type..." value={query} onChange={(e) => { setQuery(e.target.value); setPage(1); }} /></label><label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7">Show<select className="form-select form-select-sm form-select-solid w-auto" value={pageSize} onChange={(e) => { setPageSize(Number(e.target.value)); setPage(1); }}><option>10</option><option>25</option><option>50</option><option>100</option></select>entries</label><button type="button" className="btn btn-icon btn-sm btn-light" onClick={() => void load()} disabled={loading} aria-label="Muat ulang">{loading ? <span className="spinner-border spinner-border-sm" /> : "↻"}</button></div>
      <div className="card-body pt-0"><div className="table-responsive"><table className="table admin-data-table align-middle"><thead className="bg-dark"><tr className="text-white fw-bold text-uppercase fs-8"><th className="bg-dark text-white">Name</th><th className="bg-dark text-white">Land Permit</th><th className="bg-dark text-white">Spatial Type</th><th className="bg-dark text-white">Area/Length</th><th className="bg-dark text-white">Note</th><th className="bg-dark text-white text-end">Actions</th></tr></thead><tbody>
        {loading ? <TableLoadingRows columns={6} /> : !companyID && !isAllCompanies ? <tr><td colSpan={6} className="text-center py-12 text-muted">Pilih company aktif di header untuk melihat data.</td></tr> : shown.length === 0 ? <tr><td colSpan={6} className="text-center py-12 text-muted">{query ? "Data tidak ditemukan." : "Belum ada HCV Land."}</td></tr> : shown.map((item, index) => <tr key={item.id || `${item.name}-${index}`}><td className="text-gray-900 fw-bold">{item.name}</td><td>{item.land_permit}</td><td>{item.spatial_type}</td><td>{item.area_length}</td><td title={item.note}>{item.note}</td><td><div className="admin-table-actions"><button type="button" className="btn btn-light-info" title="Lihat peta HCV" aria-label={`Lihat peta ${item.name}`} onClick={() => openMap(item, "view")}><MapIcon /></button><button type="button" className="btn btn-light-secondary" title="Atur koordinat HCV" aria-label={`Atur koordinat ${item.name}`} onClick={() => openMap(item, "settings")}><SettingsIcon /></button><button type="button" className="btn btn-light-warning" title="Edit HCV Land" aria-label={`Edit ${item.name}`} onClick={() => openEdit(item)}><EditIcon /></button><button type="button" className="btn btn-light-danger" title="Hapus HCV Land" aria-label={`Hapus ${item.name}`} onClick={() => void remove(item)}><TrashIcon /></button></div></td></tr>)}
      </tbody></table></div></div><div className="card-footer d-flex flex-wrap align-items-center justify-content-between gap-3 py-4"><span className="text-muted fw-semibold fs-7">Showing {filtered.length ? (currentPage - 1) * pageSize + 1 : 0}–{Math.min(currentPage * pageSize, filtered.length)} of {filtered.length} entries</span><div className="d-flex align-items-center gap-2"><button type="button" className="btn btn-sm btn-light" onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={currentPage <= 1}>Previous</button><span className="text-muted fw-semibold fs-7">{currentPage} / {pageCount}</span><button type="button" className="btn btn-sm btn-light" onClick={() => setPage((p) => Math.min(pageCount, p + 1))} disabled={currentPage >= pageCount}>Next</button></div></div>
    </section>
    {modal && <div role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget && !saving) { setModal(null); setEditing(null); } }} style={{ position: "fixed", inset: 0, zIndex: 1055, background: "rgba(0,0,0,.45)", display: "grid", placeItems: "center", padding: modal === "map" ? 0 : 16 }}><div role="dialog" aria-modal="true" aria-labelledby="hcv-modal-title" className="card shadow-lg" style={{ width: modal === "map" ? "100vw" : "min(100%, 900px)", height: modal === "map" ? "100vh" : undefined, maxHeight: modal === "map" ? "100vh" : "90vh", overflow: modal === "map" ? "hidden" : "auto" }}>
      <div className={`card-header py-4 ${modal === "form" ? `form-modal-header ${editing ? "form-modal-header--edit" : "form-modal-header--add"}` : "hcv-map-modal-header"}`}><h2 id="hcv-modal-title" className="card-title fs-4 fw-bold">{modal === "map" ? "HCV Land Map" : `${editing ? "Edit" : "Add"} HCV Land`}</h2><button type="button" className="btn btn-sm btn-icon btn-light" aria-label="Tutup" disabled={saving} onClick={() => { setModal(null); setEditing(null); }}>×</button></div>
      {modal === "map" && editing ? <HCVMap key={`${editing.id}-${mapMode}`} id={editing.id} name={editing.name} permit={editing.land_permit} companyID={companyID} initialMode={mapMode} /> : <form onSubmit={(event) => void save(event)}><div className="card-body py-0">{error && <div className="alert alert-danger mt-4">{error}</div>}
        <div className="row align-items-center py-3 border-bottom"><label className="col-sm-3 col-form-label">Name</label><div className="col-sm-6"><input className="form-control form-control-solid" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required /></div></div>
        <div className="row align-items-center py-3 border-bottom"><label className="col-sm-3 col-form-label">Land Permit</label><div className="col-sm-6"><select className="form-select form-select-solid" value={form.land_permit_id || ""} onChange={(e) => setForm({ ...form, land_permit_id: Number(e.target.value) || 0 })} required><option value="">Select Land Permit</option>{options.land_permits.map((option) => <option key={option.id} value={option.id}>{option.name}</option>)}</select></div></div>
        <div className="row align-items-center py-3 border-bottom"><label className="col-sm-3 col-form-label">Spatial Type</label><div className="col-sm-6"><select className="form-select form-select-solid" value={form.spatial_type_id ?? ""} onChange={(e) => setForm({ ...form, spatial_type_id: Number(e.target.value) || null })}><option value="">Select Spatial Type</option>{options.spatial_types.map((option) => <option key={option.id} value={option.id}>{option.name}</option>)}</select></div></div>
        <div className="row align-items-center py-3 border-bottom"><label className="col-sm-3 col-form-label">Area/Length</label><div className="col-sm-6"><input className="form-control form-control-solid" value={form.area_length} onChange={(e) => setForm({ ...form, area_length: e.target.value })} /></div></div>
        <div className="row align-items-center py-3"><label className="col-sm-3 col-form-label">Note</label><div className="col-sm-6"><input className="form-control form-control-solid" value={form.note} onChange={(e) => setForm({ ...form, note: e.target.value })} /></div></div>
      </div><div className="card-footer d-flex gap-3 py-4"><button type="submit" className={`btn ${editing ? "btn-warning" : "btn-dark"}`} disabled={saving}>{saving ? "Saving..." : editing ? "Update" : "Submit"}</button><button type="button" className="btn btn-light" disabled={saving} onClick={() => { setModal(null); setEditing(null); }}>Cancel</button></div></form>}
    </div></div>}
  </div>;
}
