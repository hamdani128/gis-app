"use client";

import TableLoadingRows from "../../../../components/ui/table-loading-rows";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { FormEvent } from "react";
import { useUserProfile } from "../../../../components/auth/user-profile-context";
import CompensatedDetail from "../compensatedlands/compensated-detail";

function LandsIcon() { return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M3 6.5 9 4l6 2.5L21 4v13.5L15 20l-6-2.5L3 20V6.5Zm6-2.5v13.5m6-11V20" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>; }
function EditIcon() { return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m14 5 5 5M4 20l4.5-1 10.8-10.8a2.12 2.12 0 0 0-3-3L5.5 16 4 20Z" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>; }
function TrashIcon() { return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M4 7h16M10 11v6m4-6v6M6 7l1 14h10l1-14M9 7V4h6v3" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>; }
function SubmitIcon() { return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M21 3 3 10l8 3 3 8 7-18ZM11 13l5-5" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>; }

type Owner = { id: number; Gis_Companies_id: number; company: string; name: string; ktp_number: string; address: string; note: string; date_created: string | null };
type Land = { id: number; owner: string; ktp_number: string; village: string; status: number; land_wide: number; wet_land_wide: number; wet_land_perha_price: number; dry_land_wide: number; dry_land_perha_price: number; admin_price: number; plant_price: number };
type LandDetail = Land & { block_plan: string; [key: string]: unknown };
type Company = { id: number; name: string };
type Privileges = { p_add: boolean; p_edit: boolean; p_del: boolean; p_view: boolean };
type Form = { Gis_Companies_id: string; name: string; ktp_number: string; address: string; note: string; date_created: string };
type Envelope<T> = { data?: T; message?: string };
type OwnerColumn = "company" | "name" | "ktp_number" | "address" | "note" | "date_created";
type LandColumn = "owner" | "village" | "status" | "land_wide" | "wet_land_wide" | "wet_land_perha_price" | "dry_land_wide" | "dry_land_perha_price" | "average" | "admin_price" | "plant_price" | "total";
const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");
const emptyPrivileges = (): Privileges => ({ p_add: false, p_edit: false, p_del: false, p_view: false });
const emptyForm = (companyID = 0): Form => ({ Gis_Companies_id: companyID ? String(companyID) : "", name: "", ktp_number: "", address: "", note: "", date_created: "" });
const ownerColumns: { key: OwnerColumn; label: string }[] = [{ key: "company", label: "Company" }, { key: "name", label: "Name" }, { key: "ktp_number", label: "KTP number" }, { key: "address", label: "Address" }, { key: "note", label: "Note" }, { key: "date_created", label: "Date created" }];
const landColumns: { key: LandColumn; label: string }[] = [{ key: "owner", label: "Owner" }, { key: "village", label: "Village" }, { key: "status", label: "Status" }, { key: "land_wide", label: "Land Size (ha)" }, { key: "wet_land_wide", label: "Wet Size (ha)" }, { key: "wet_land_perha_price", label: "Wet price (@Ha)" }, { key: "dry_land_wide", label: "Dry Size (ha)" }, { key: "dry_land_perha_price", label: "Dry price (@Ha)" }, { key: "average", label: "Avg (@Ha)" }, { key: "admin_price", label: "Admin price" }, { key: "plant_price", label: "Plant price" }, { key: "total", label: "Total price" }];
const money = (n: number) => new Intl.NumberFormat("id-ID", { maximumFractionDigits: 0 }).format(n || 0);
const area = (n: number) => new Intl.NumberFormat("id-ID", { maximumFractionDigits: 2 }).format(n || 0);
const total = (land: Land) => land.wet_land_wide * land.wet_land_perha_price + land.dry_land_wide * land.dry_land_perha_price + land.admin_price + land.plant_price;
const average = (land: Land) => land.land_wide ? Math.round((land.wet_land_wide * land.wet_land_perha_price + land.dry_land_wide * land.dry_land_perha_price) / land.land_wide) : 0;
const dateDisplay = (date: string | null) => date ? new Intl.DateTimeFormat("id-ID", { day: "2-digit", month: "2-digit", year: "numeric", hour: "2-digit", minute: "2-digit" }).format(new Date(date)) : "-";
const dateInput = (date: string | null) => date ? new Date(date).toLocaleString("sv-SE", { hour12: false }).replace(" ", "T").slice(0, 16) : "";

async function api<T>(path: string, init?: RequestInit, refreshed = false): Promise<T> {
  const headers = new Headers(init?.headers);
  if (init?.body) headers.set("Content-Type", "application/json");
  const response = await fetch(`${apiBase}${path}`, { ...init, headers, credentials: "include", cache: "no-store" });
  if (response.status === 401 && !refreshed) { const renewal = await fetch(`${apiBase}/api/auth/refresh`, { method: "POST", credentials: "include" }); if (renewal.ok) return api<T>(path, init, true); }
  const result = await response.json().catch(() => null) as Envelope<T> | null;
  if (!response.ok) throw new Error(result?.message || "Permintaan gagal.");
  return result?.data as T;
}

export default function LandOwnerPage() {
  const profile = useUserProfile();
  const [companyID, setCompanyID] = useState(0);
  const [companyName, setCompanyName] = useState("");
  const allCompanies = companyName.trim() === "00" || /^00(?:\b|[\s-])/i.test(companyName);
  const [owners, setOwners] = useState<Owner[]>([]);
  const [companies, setCompanies] = useState<Company[]>([]);
  const [privileges, setPrivileges] = useState(emptyPrivileges());
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [formError, setFormError] = useState("");
  const [query, setQuery] = useState("");
  const [filters, setFilters] = useState<Partial<Record<OwnerColumn, string>>>({});
  const [sort, setSort] = useState<{ key: OwnerColumn; asc: boolean }>({ key: "date_created", asc: false });
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(25);
  const [editing, setEditing] = useState<Owner | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState<Form>(emptyForm());
  const [selected, setSelected] = useState<Owner | null>(null);
  const [lands, setLands] = useState<Land[]>([]);
  const [landsLoading, setLandsLoading] = useState(false);
  const [landQuery, setLandQuery] = useState("");
  const [landSort, setLandSort] = useState<{ key: LandColumn; asc: boolean }>({ key: "owner", asc: true });
  const [landPage, setLandPage] = useState(1);
  const [landPageSize, setLandPageSize] = useState(25);
  const [mapLand, setMapLand] = useState<Land | null>(null);
  const [detailLand, setDetailLand] = useState<LandDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const detailRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const storageKey = `gis.selected-company.${profile.username.toLowerCase()}`;
    const read = () => { const id = Number(window.localStorage.getItem(storageKey)) || 0; setCompanyID(id); if (!id) setCompanyName(""); };
    const changed = (event: Event) => { const detail = (event as CustomEvent<{ id?: number; name?: string }>).detail; if (detail && Number.isFinite(Number(detail.id))) { const name = detail.name ?? ""; setCompanyID(name.trim() === "00" || /^00(?:\b|[\s-])/i.test(name.trim()) ? 0 : Number(detail.id)); setCompanyName(name); } else read(); };
    read(); window.addEventListener("gis:company-change", changed); window.addEventListener("storage", read);
    return () => { window.removeEventListener("gis:company-change", changed); window.removeEventListener("storage", read); };
  }, [profile.username]);

  const load = useCallback(async () => {
    if (!companyID && !allCompanies) { setOwners([]); setCompanies([]); setPrivileges(emptyPrivileges()); setLoading(false); return; }
    setLoading(true); setError("");
    try {
      const params = `?company_id=${companyID}`;
      const [rows, options, rights] = await Promise.all([api<Owner[]>(`/api/land-owners${params}`), api<Company[]>(`/api/land-owners/options${params}`), api<Privileges>(`/api/land-owners/permissions${params}`)]);
      setOwners(Array.isArray(rows) ? rows : []); setCompanies(Array.isArray(options) ? options : []); setPrivileges(rights);
    } catch (cause) { setOwners([]); setCompanies([]); setPrivileges(emptyPrivileges()); setError(cause instanceof Error ? cause.message : "Land Owners gagal dimuat."); }
    finally { setLoading(false); }
  }, [companyID, allCompanies]);
  useEffect(() => { const timer = window.setTimeout(() => void load(), 0); return () => window.clearTimeout(timer); }, [load]);

  const ownerValue = (owner: Owner, key: OwnerColumn) => key === "date_created" ? dateDisplay(owner.date_created) : String(owner[key] ?? "");
  const filtered = useMemo(() => owners.filter((owner) => {
    const values = ownerColumns.map(({ key }) => ownerValue(owner, key).toLowerCase());
    return (!query.trim() || values.some((value) => value.includes(query.trim().toLowerCase()))) && ownerColumns.every(({ key }, index) => !filters[key]?.trim() || values[index].includes(filters[key]!.trim().toLowerCase()));
  }).sort((a, b) => { const av = sort.key === "date_created" ? a.date_created || "" : ownerValue(a, sort.key); const bv = sort.key === "date_created" ? b.date_created || "" : ownerValue(b, sort.key); return String(av).localeCompare(String(bv), "id", { numeric: true }) * (sort.asc ? 1 : -1); }), [owners, query, filters, sort]);
  const pageCount = Math.max(1, Math.ceil(filtered.length / pageSize));
  const currentPage = Math.min(page, pageCount);
  const shown = filtered.slice((currentPage - 1) * pageSize, currentPage * pageSize);

  function openCreate() { setEditing(null); setForm(emptyForm(companyID)); setFormError(""); setShowForm(true); }
  function openEdit(owner: Owner) { setEditing(owner); setForm({ Gis_Companies_id: String(owner.Gis_Companies_id), name: owner.name, ktp_number: owner.ktp_number, address: owner.address, note: owner.note, date_created: dateInput(owner.date_created) }); setFormError(""); setShowForm(true); }
  function closeForm() { if (!saving) { setShowForm(false); setEditing(null); setFormError(""); } }
  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setSaving(true); setFormError("");
    try {
      await api<Owner>(editing ? `/api/land-owners/${editing.id}?company_id=${companyID}` : `/api/land-owners?company_id=${companyID}`, { method: editing ? "PUT" : "POST", body: JSON.stringify({ ...form, Gis_Companies_id: Number(form.Gis_Companies_id), date_created: form.date_created ? new Date(form.date_created).toISOString() : null }) });
      setShowForm(false); setEditing(null); await load();
    } catch (cause) { setFormError(cause instanceof Error ? cause.message : "Land Owner gagal disimpan."); }
    finally { setSaving(false); }
  }
  async function remove(owner: Owner) {
    const result = window.Swal ? await window.Swal.fire({ title: "Hapus Land Owner ini?", text: `${owner.name} akan dihapus.`, icon: "warning", showCancelButton: true, confirmButtonText: "Ya, hapus", cancelButtonText: "Batal", reverseButtons: true, buttonsStyling: false, customClass: { popup: "rounded-4 shadow-lg", confirmButton: "btn btn-danger ms-2", cancelButton: "btn btn-light" } }) : { isConfirmed: window.confirm(`Hapus Land Owner ${owner.name}?`) };
    if (!result.isConfirmed) return;
    try { await api<null>(`/api/land-owners/${owner.id}?company_id=${companyID}`, { method: "DELETE" }); await load(); }
    catch (cause) { setError(cause instanceof Error ? cause.message : "Land Owner gagal dihapus."); }
  }
  async function openLands(owner: Owner) {
    setSelected(owner); setMapLand(null); setDetailLand(null); setLands([]); setLandQuery(""); setLandPage(1); setLandsLoading(true); setError("");
    try { const rows = await api<Land[]>(`/api/land-owners/${owner.id}/lands?company_id=${companyID}`); setLands(Array.isArray(rows) ? rows : []); }
    catch (cause) { setError(cause instanceof Error ? cause.message : "Daftar land gagal dimuat."); }
    finally { setLandsLoading(false); }
  }
  async function openLandDetail(land: Land) {
    if (!selected) return;
    setMapLand(land); setDetailLand(null); setDetailLoading(true); setError("");
    try { setDetailLand(await api<LandDetail>(`/api/land-owners/${selected.id}/lands/${land.id}?company_id=${companyID}`)); }
    catch (cause) { setError(cause instanceof Error ? cause.message : "Detail land gagal dimuat."); }
    finally { setDetailLoading(false); }
  }
  useEffect(() => {
    if (!detailLand) return;
    const frame = window.requestAnimationFrame(() => detailRef.current?.scrollIntoView({ behavior: "smooth", block: "start" }));
    return () => window.cancelAnimationFrame(frame);
  }, [detailLand]);
  const landValue = (land: Land, key: LandColumn): string => key === "owner" ? `${land.owner} - ${land.ktp_number}` : key === "average" ? String(average(land)) : key === "total" ? String(total(land)) : String(land[key] ?? "");
  const visibleLands = useMemo(() => lands.filter((land) => {
    const values = landColumns.map(({ key }) => landValue(land, key).toLowerCase());
    return !landQuery.trim() || values.some((value) => value.includes(landQuery.trim().toLowerCase()));
  }).sort((a, b) => { const av = landValue(a, landSort.key), bv = landValue(b, landSort.key); return av.localeCompare(bv, "id", { numeric: true }) * (landSort.asc ? 1 : -1); }), [lands, landQuery, landSort]);
  const landPageCount = Math.max(1, Math.ceil(visibleLands.length / landPageSize));
  const currentLandPage = Math.min(landPage, landPageCount);
  const shownLands = visibleLands.slice((currentLandPage - 1) * landPageSize, currentLandPage * landPageSize);

  return <div className="d-flex flex-column gap-6 landowner-page">
    <nav aria-label="Breadcrumb"><ol className="breadcrumb breadcrumb-separatorless fw-semibold fs-7 mb-0"><li className="breadcrumb-item"><Link href="/admin/dashboard" className="text-muted">Admin</Link></li><li className="breadcrumb-item text-muted ms-2">/ Lands / Land Owners</li></ol></nav>
    <div className="d-flex flex-wrap align-items-center justify-content-between gap-4"><div><h1 className="text-gray-900 fw-bolder fs-2 mb-1">Land Owners</h1><p className="text-muted fw-semibold mb-0">Company aktif: <strong>{allCompanies ? "Semua company (00)" : companyName || "Belum dipilih"}</strong></p></div>{privileges.p_add && <button type="button" className="btn btn-dark" onClick={openCreate}>+ Add Land Owner</button>}</div>
    {error && !selected && <div className="alert alert-danger" role="alert">{error}</div>}
    <section className="card border shadow-sm"><div className="card-header flex-wrap gap-4 py-4"><span className="text-muted fw-semibold fs-7">Total data: {owners.length}</span><label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7 ms-auto">Search<input className="form-control form-control-sm form-control-solid ref-table-search" type="search" value={query} onChange={(event) => { setQuery(event.target.value); setPage(1); }} /></label><label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7">Show<select className="form-select form-select-sm form-select-solid w-auto" value={pageSize} onChange={(event) => { setPageSize(Number(event.target.value)); setPage(1); }}>{[10,25,50,100].map((size) => <option key={size}>{size}</option>)}</select>entries</label><button type="button" className="btn btn-sm btn-light" onClick={() => void load()} disabled={loading}>Refresh</button></div>
      <div className="card-body pt-0"><div className="table-responsive"><table className="table admin-data-table align-middle"><thead className="bg-dark"><tr className="text-white fw-bold text-uppercase fs-8">{ownerColumns.map(({ key, label }) => <th key={key} className="bg-dark text-white text-nowrap"><button type="button" className="btn btn-link p-0 text-white fw-bold text-decoration-none" onClick={() => setSort({ key, asc: sort.key === key ? !sort.asc : true })}>{label} {sort.key === key ? (sort.asc ? "↑" : "↓") : ""}</button></th>)}<th className="bg-dark text-white">Actions</th></tr><tr>{ownerColumns.map(({ key, label }) => <th key={key}><input className="form-control form-control-sm" aria-label={`Filter ${label}`} placeholder={`Search ${label}`} value={filters[key] || ""} onChange={(event) => { setFilters((current) => ({ ...current, [key]: event.target.value })); setPage(1); }} /></th>)}<th><button type="button" className="btn btn-sm btn-light" onClick={() => { setFilters({}); setQuery(""); setPage(1); }}>Clear filtering</button></th></tr></thead><tbody>
        {loading ? <TableLoadingRows columns={7} /> : !companyID && !allCompanies ? <tr><td colSpan={7} className="text-center py-12 text-muted">Pilih company aktif di header untuk melihat data.</td></tr> : !shown.length ? <tr><td colSpan={7} className="text-center py-12 text-muted">{query || Object.values(filters).some(Boolean) ? "Data tidak ditemukan." : "Belum ada Land Owner."}</td></tr> : shown.map((owner) => <tr key={owner.id}><td>{owner.company}</td><td className="text-gray-900 fw-bold">{owner.name}</td><td>{owner.ktp_number}</td><td>{owner.address}</td><td>{owner.note}</td><td className="text-nowrap">{dateDisplay(owner.date_created)}</td><td><div className="admin-table-actions"><button type="button" className="btn btn-light-primary d-inline-flex align-items-center gap-2" title="Lands" onClick={() => void openLands(owner)} aria-label={`Lands ${owner.name}`}><LandsIcon /><span>Lands</span></button>{privileges.p_edit && <button type="button" className="btn btn-light-warning" title="Edit" onClick={() => openEdit(owner)} aria-label={`Edit ${owner.name}`}><EditIcon /></button>}{privileges.p_del && <button type="button" className="btn btn-light-danger" title="Hapus" onClick={() => void remove(owner)} aria-label={`Hapus ${owner.name}`}><TrashIcon /></button>}</div></td></tr>)}
      </tbody></table></div></div><div className="card-footer d-flex flex-wrap align-items-center justify-content-between gap-3 py-4"><span className="text-muted fw-semibold fs-7">Showing {filtered.length ? (currentPage - 1) * pageSize + 1 : 0}–{Math.min(currentPage * pageSize, filtered.length)} of {filtered.length} entries</span><div className="d-flex align-items-center gap-2"><button type="button" className="btn btn-sm btn-light" onClick={() => setPage((current) => Math.max(1, current - 1))} disabled={currentPage <= 1}>Previous</button><span>{currentPage} / {pageCount}</span><button type="button" className="btn btn-sm btn-light" onClick={() => setPage((current) => Math.min(pageCount, current + 1))} disabled={currentPage >= pageCount}>Next</button></div></div></section>
    {showForm && <div role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) closeForm(); }} style={{ position: "fixed", inset: 0, zIndex: 1055, background: "rgba(0,0,0,.45)", display: "grid", placeItems: "center", padding: 16 }}><div role="dialog" aria-modal="true" aria-labelledby="owner-form-title" className="card shadow-lg" style={{ width: "min(100%, 900px)", maxHeight: "92vh", overflow: "auto" }}><div className={`card-header py-4 form-modal-header ${editing ? "form-modal-header--edit" : "form-modal-header--add"}`}><h2 id="owner-form-title" className="card-title fs-4 fw-bold">{editing ? "Edit" : "Add"} Land Owner</h2><button type="button" className="btn btn-sm btn-icon btn-light" onClick={closeForm} disabled={saving} aria-label="Tutup">×</button></div><form onSubmit={(event) => void save(event)}><div className="card-body py-0">{formError && <div className="alert alert-danger mt-4" role="alert">{formError}</div>}
      <div className="row align-items-center py-3 border-bottom"><label htmlFor="owner-company" className="col-sm-3 col-form-label">Company *</label><div className="col-sm-7"><select id="owner-company" className="form-select form-select-solid" value={form.Gis_Companies_id} required onChange={(event) => setForm((current) => ({ ...current, Gis_Companies_id: event.target.value }))}><option value="">Select Company</option>{companies.map((company) => <option value={company.id} key={company.id}>{company.name}</option>)}</select></div></div>
      {([{ key: "name", label: "Name", required: true }, { key: "ktp_number", label: "KTP number" }, { key: "address", label: "Address" }, { key: "note", label: "Note" }] as const).map(({ key, label, ...rest }) => <div className="row align-items-center py-3 border-bottom" key={key}><label htmlFor={`owner-${key}`} className="col-sm-3 col-form-label">{label}{"required" in rest ? " *" : ""}</label><div className="col-sm-7"><input id={`owner-${key}`} type="text" className="form-control form-control-solid" required={"required" in rest} value={form[key]} onChange={(event) => setForm((current) => ({ ...current, [key]: event.target.value }))} /></div></div>)}
      <div className="row align-items-center py-3 border-bottom"><label htmlFor="owner-date" className="col-sm-3 col-form-label">Date created</label><div className="col-sm-7"><div className="d-flex gap-2"><input id="owner-date" type="datetime-local" className="form-control form-control-solid" value={form.date_created} onChange={(event) => setForm((current) => ({ ...current, date_created: event.target.value }))} /><button type="button" className="btn btn-light" onClick={() => setForm((current) => ({ ...current, date_created: "" }))}>Clear</button></div></div></div>
    </div><div className="card-footer d-flex gap-3 py-4"><button type="submit" className={`btn d-inline-flex align-items-center gap-2 ${editing ? "btn-warning" : "btn-dark"}`} disabled={saving}>{editing ? <EditIcon /> : <SubmitIcon />}{saving ? "Menyimpan..." : editing ? "Update" : "Submit"}</button><button type="button" className="btn btn-light" onClick={closeForm} disabled={saving}>Cancel</button></div></form></div></div>}
    {selected && (
      <div role="presentation" className="compensated-modal-backdrop" data-fullscreen="true" onMouseDown={(event) => {
        if (event.target === event.currentTarget) { setSelected(null); setMapLand(null); setDetailLand(null); }
      }}>
        <div role="dialog" aria-modal="true" aria-labelledby="owner-lands-title" className="card shadow-lg compensated-modal landowner-lands-modal" data-fullscreen="true">
          <div className="card-header py-4 hcv-map-modal-header">
            <h2 id="owner-lands-title" className="card-title fs-4 fw-bold">Lands — {selected.name}</h2>
            <button type="button" className="btn btn-sm btn-light ms-auto" onClick={() => { setSelected(null); setMapLand(null); setDetailLand(null); }}>Close</button>
          </div>
          <div className="card-body landowner-lands-list">
            {error && <div className="alert alert-danger" role="alert">{error}</div>}
            <div className="d-flex flex-wrap gap-3 align-items-center mb-4">
              <label className="d-flex align-items-center gap-2">Show
                <select className="form-select form-select-sm w-auto" value={landPageSize} onChange={(event) => { setLandPageSize(Number(event.target.value)); setLandPage(1); }}>
                  {[10, 25, 50, 100].map((size) => <option key={size}>{size}</option>)}
                </select>entries
              </label>
              <label className="d-flex align-items-center gap-2 ms-auto">Search
                <input type="search" className="form-control form-control-sm" value={landQuery} onChange={(event) => { setLandQuery(event.target.value); setLandPage(1); }} />
              </label>
            </div>
            <div className="table-responsive">
              <table className="table admin-data-table align-middle">
                <thead className="bg-dark"><tr>
                  {landColumns.map(({ key, label }) => <th key={key} className="bg-dark text-white text-nowrap"><button type="button" className="btn btn-link p-0 text-white fw-bold text-decoration-none" onClick={() => setLandSort({ key, asc: landSort.key === key ? !landSort.asc : true })}>{label} {landSort.key === key ? (landSort.asc ? "↑" : "↓") : ""}</button></th>)}
                  <th className="bg-dark text-white">Actions</th>
                </tr></thead>
                <tbody>
                  {landsLoading ? <TableLoadingRows columns={13} />
                    : !shownLands.length ? <tr><td colSpan={13} className="text-center py-8">Tidak ada data land.</td></tr>
                    : shownLands.map((land) => <tr key={land.id}>
                      <td className="text-nowrap">{land.owner} - {land.ktp_number}</td>
                      <td>{land.village}</td><td>{land.status}</td><td>{area(land.land_wide)}</td>
                      <td>{area(land.wet_land_wide)}</td><td>{money(land.wet_land_perha_price)}</td>
                      <td>{area(land.dry_land_wide)}</td><td>{money(land.dry_land_perha_price)}</td>
                      <td>{money(average(land))}</td><td>{money(land.admin_price)}</td>
                      <td>{money(land.plant_price)}</td><td>{money(total(land))}</td>
                      <td><div className="admin-table-actions">
                        <button type="button" className="btn btn-light-primary" onClick={() => void openLandDetail(land)}>Map</button>
                        <button type="button" className="btn btn-light-secondary" onClick={() => void downloadKML(selected.id, land.id, companyID).catch((cause) => setError(cause instanceof Error ? cause.message : "KML gagal diunduh."))}>KML</button>
                      </div></td>
                    </tr>)}
                </tbody>
              </table>
            </div>
            <div className="d-flex justify-content-between align-items-center gap-3 py-4">
              <span>Showing {visibleLands.length ? (currentLandPage - 1) * landPageSize + 1 : 0}–{Math.min(currentLandPage * landPageSize, visibleLands.length)} of {visibleLands.length} entries</span>
              <div className="d-flex gap-2 align-items-center">
                <button type="button" className="btn btn-sm btn-light" onClick={() => setLandPage((current) => Math.max(1, current - 1))} disabled={currentLandPage <= 1}>Previous</button>
                {currentLandPage} / {landPageCount}
                <button type="button" className="btn btn-sm btn-light" onClick={() => setLandPage((current) => Math.min(landPageCount, current + 1))} disabled={currentLandPage >= landPageCount}>Next</button>
              </div>
            </div>
            {mapLand && <section ref={detailRef} className="card border shadow-sm landowner-lands-detail mt-4" aria-label="Detail land">
              <div className="card-header py-4"><h3 className="card-title fs-4 fw-bold">Detail {mapLand.status === 8 ? "New Compensated Land" : mapLand.status === 10 ? "Compensated Land" : "Land"} #{mapLand.id}</h3></div>
              {detailLoading ? <div className="card-body p-6">Loading detail land...</div>
                : detailLand ? <CompensatedDetail key={detailLand.id} row={detailLand} companyID={companyID} canEdit={false} apiPath={"/api/land-owners/" + selected.id + "/lands"} showTimeline onMap={() => detailRef.current?.querySelector(".compensated-overview-map")?.scrollIntoView({ behavior: "smooth", block: "center" })} onEdit={() => {}} />
                : <div className="card-body p-6">Detail land tidak tersedia.</div>}
            </section>}
          </div>
        </div>
      </div>
    )}
  </div>;
}

type Geometry = { type: string; coordinates: unknown };
function xml(value: string) { return value.replace(/[&<>"']/g, (char) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&apos;" })[char] || char); }
function kmlGeometry(geometry: Geometry): string {
  const coords = (points: unknown) => Array.isArray(points) ? points.map((point) => Array.isArray(point) ? point.join(",") : "").join(" ") : "";
  if (geometry.type === "Point") return `<Point><coordinates>${coords([geometry.coordinates])}</coordinates></Point>`;
  if (geometry.type === "LineString") return `<LineString><coordinates>${coords(geometry.coordinates)}</coordinates></LineString>`;
  if (geometry.type === "Polygon" && Array.isArray(geometry.coordinates)) return `<Polygon>${geometry.coordinates.map((ring, index) => `<${index ? "inner" : "outer"}BoundaryIs><LinearRing><coordinates>${coords(ring)}</coordinates></LinearRing></${index ? "inner" : "outer"}BoundaryIs>`).join("")}</Polygon>`;
  if (geometry.type === "MultiPolygon" && Array.isArray(geometry.coordinates)) return `<MultiGeometry>${geometry.coordinates.map((polygon) => kmlGeometry({ type: "Polygon", coordinates: polygon })).join("")}</MultiGeometry>`;
  if (geometry.type === "MultiLineString" && Array.isArray(geometry.coordinates)) return `<MultiGeometry>${geometry.coordinates.map((line) => kmlGeometry({ type: "LineString", coordinates: line })).join("")}</MultiGeometry>`;
  if (geometry.type === "MultiPoint" && Array.isArray(geometry.coordinates)) return `<MultiGeometry>${geometry.coordinates.map((point) => kmlGeometry({ type: "Point", coordinates: point })).join("")}</MultiGeometry>`;
  return "";
}
async function downloadKML(ownerID: number, landID: number, companyID: number) {
  const result = await api<{ map_data?: string }>(`/api/land-owners/${ownerID}/lands/${landID}/map?company_id=${companyID}`);
  const collection = JSON.parse(result.map_data || "{}") as { features?: { geometry?: Geometry; properties?: { name?: string } }[] };
  const placemarks = (collection.features || []).map((feature, index) => feature.geometry ? `<Placemark><name>${xml(feature.properties?.name || `Land ${landID} #${index + 1}`)}</name>${kmlGeometry(feature.geometry)}</Placemark>` : "").join("");
  if (!placemarks) throw new Error("Koordinat land belum tersedia untuk diunduh.");
  const blob = new Blob([`<?xml version="1.0" encoding="UTF-8"?><kml xmlns="http://www.opengis.net/kml/2.2"><Document><name>Land ${landID}</name>${placemarks}</Document></kml>`], { type: "application/vnd.google-earth.kml+xml" });
  const url = URL.createObjectURL(blob); const anchor = document.createElement("a"); anchor.href = url; anchor.download = `land-${landID}.kml`; anchor.click(); window.setTimeout(() => URL.revokeObjectURL(url), 1000);
}
