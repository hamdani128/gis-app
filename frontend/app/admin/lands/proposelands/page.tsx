"use client";

import Link from "next/link";
import TableLoadingRows from "../../../../components/ui/table-loading-rows";
import { useCallback, useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import { useUserProfile } from "../../../../components/auth/user-profile-context";

function EditIcon() { return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m14 5 5 5M4 20l4.5-1 10.8-10.8a2.12 2.12 0 0 0-3-3L5.5 16 4 20Z" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>; }
function TrashIcon() { return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M4 7h16M10 11v6m4-6v6M6 7l1 14h10l1-14M9 7V4h6v3" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>; }

type Option = { id: number; name: string };
type CodeOption = { value: string; name: string };
type Options = { block_plans: Option[]; owners: Option[]; original_statuses: CodeOption[]; land_surfaces: CodeOption[]; soil_types: CodeOption[] };
type Permissions = { p_add: boolean; p_del: boolean; p_view: boolean; p_edit: boolean };
type Row = {
  id: number; Gis_Aquistion_Divisions_id: number; Gis_Land_Owners_id: number; block_plan: string; owner: string;
  plant_division: string; plant_block: string; alas_hak_jenis: string; alas_hak_no: string; land_surface: string;
  slope_persentage: string; soil_type: string; northern_boundary: string; eastern_boundary: string;
  western_boundary: string; southern_boundary: string; village: string; dry_land_wide: number;
  wet_land_wide: number; land_wide: number; dry_land_perha_price: number; wet_land_perha_price: number;
  admin_price: number; plant_price: number; vegetation_note: string; measurement_date: string; measured_by: string;
};
type Form = Omit<Row, "id" | "block_plan" | "owner" | "Gis_Aquistion_Divisions_id" | "Gis_Land_Owners_id" | "dry_land_wide" | "wet_land_wide" | "land_wide" | "dry_land_perha_price" | "wet_land_perha_price" | "admin_price" | "plant_price"> & {
  Gis_Aquistion_Divisions_id: string; Gis_Land_Owners_id: string; dry_land_wide: string; wet_land_wide: string;
  land_wide: string; dry_land_perha_price: string; wet_land_perha_price: string; admin_price: string; plant_price: string;
};
type Envelope<T> = { data?: T; message?: string };

const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");
const blankOptions = (): Options => ({ block_plans: [], owners: [], original_statuses: [], land_surfaces: [], soil_types: [] });
const blankForm = (): Form => ({ Gis_Aquistion_Divisions_id: "", Gis_Land_Owners_id: "", plant_division: "", plant_block: "", alas_hak_jenis: "", alas_hak_no: "", land_surface: "", slope_persentage: "", soil_type: "", northern_boundary: "", eastern_boundary: "", western_boundary: "", southern_boundary: "", village: "", dry_land_wide: "", wet_land_wide: "", land_wide: "", dry_land_perha_price: "", wet_land_perha_price: "", admin_price: "", plant_price: "", vegetation_note: "", measurement_date: "", measured_by: "" });
const currency = (value: number) => new Intl.NumberFormat("id-ID", { maximumFractionDigits: 0 }).format(value);
const area = (value: number) => new Intl.NumberFormat("id-ID", { maximumFractionDigits: 2 }).format(value);
const totalPrice = (row: Row) => row.wet_land_wide * row.wet_land_perha_price + row.dry_land_wide * row.dry_land_perha_price + row.admin_price + row.plant_price;
const averagePrice = (row: Row) => row.land_wide > 0 ? Math.round((row.wet_land_wide * row.wet_land_perha_price + row.dry_land_wide * row.dry_land_perha_price) / row.land_wide) : 0;

async function apiRequest<T>(path: string, init?: RequestInit, refreshed = false): Promise<T> {
  const headers = new Headers(init?.headers);
  if (init?.body && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  const response = await fetch(`${apiBase}${path}`, { ...init, headers, credentials: "include", cache: "no-store" });
  if (response.status === 401 && !refreshed) {
    const renewal = await fetch(`${apiBase}/api/auth/refresh`, { method: "POST", credentials: "include", cache: "no-store" });
    if (renewal.ok) return apiRequest<T>(path, init, true);
  }
  const result = (await response.json().catch(() => null)) as Envelope<T> | null;
  if (!response.ok) throw new Error(result?.message || "Permintaan gagal.");
  return result?.data as T;
}

const textFields: { key: keyof Form; label: string }[] = [
  { key: "plant_division", label: "Planting Division Location" }, { key: "plant_block", label: "Planting Block Locations" },
  { key: "alas_hak_no", label: "Original Land Status Number" }, { key: "slope_persentage", label: "Slope percentage" },
  { key: "northern_boundary", label: "Northern boundary" }, { key: "eastern_boundary", label: "Eastern boundary" },
  { key: "western_boundary", label: "Western boundary" }, { key: "southern_boundary", label: "Southern boundary" },
  { key: "village", label: "Village" }, { key: "measured_by", label: "Measured by" },
];
const numberFields: { key: keyof Form; label: string; step: string }[] = [
  { key: "dry_land_wide", label: "Dry Size (ha)", step: "1" }, { key: "wet_land_wide", label: "Wet Size (ha)", step: "1" },
  { key: "land_wide", label: "Land Size (ha)", step: "any" }, { key: "dry_land_perha_price", label: "dry price (@Ha)", step: "1" },
  { key: "wet_land_perha_price", label: "wet price (@Ha)", step: "1" }, { key: "admin_price", label: "Admin price", step: "1" },
  { key: "plant_price", label: "Plant price", step: "1" },
];

export default function ProposeLandsPage() {
  const profile = useUserProfile();
  const [companyID, setCompanyID] = useState(0);
  const [companyName, setCompanyName] = useState("");
  const isAllCompanies = companyName.trim() === "00" || /^00(?:\b|[\s-])/i.test(companyName);
  const [items, setItems] = useState<Row[]>([]);
  const [options, setOptions] = useState<Options>(blankOptions());
  const [permissions, setPermissions] = useState<Permissions>({ p_add: false, p_del: false, p_view: false, p_edit: false });
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(25);
  const [editing, setEditing] = useState<Row | null>(null);
  const [form, setForm] = useState<Form>(blankForm());
  const [showForm, setShowForm] = useState(false);

  useEffect(() => {
    const storageKey = `gis.selected-company.${profile.username.toLowerCase()}`;
    const read = () => { const id = Number(window.localStorage.getItem(storageKey)) || 0; setCompanyID(id); if (!id) setCompanyName(""); };
    const changed = (event: Event) => { const detail = (event as CustomEvent<{ id?: number; name?: string }>).detail; if (detail && Number.isFinite(Number(detail.id))) { const name = detail.name ?? ""; setCompanyID(/^(?:00)(?:\b|[\s-])/i.test(name.trim()) || name.trim() === "00" ? 0 : Number(detail.id)); setCompanyName(name); } else read(); };
    read(); window.addEventListener("gis:company-change", changed); window.addEventListener("storage", read);
    return () => { window.removeEventListener("gis:company-change", changed); window.removeEventListener("storage", read); };
  }, [profile.username]);

  const load = useCallback(async () => {
    if (!companyID && !isAllCompanies) { setItems([]); setOptions(blankOptions()); setPermissions({ p_add: false, p_del: false, p_view: false, p_edit: false }); setLoading(false); return; }
    setLoading(true); setError("");
    try {
      const params = `?company_id=${companyID}`;
      const [rows, opts, priv] = await Promise.all([apiRequest<Row[]>(`/api/propose-lands${params}`), apiRequest<Options>(`/api/propose-lands/options${params}`), apiRequest<Permissions>(`/api/propose-lands/permissions${params}`)]);
      setItems(Array.isArray(rows) ? [...new Map(rows.map((row) => [row.id, row])).values()] : []);
      setPermissions(priv);
      setOptions({ block_plans: opts?.block_plans ?? [], owners: opts?.owners ?? [], original_statuses: opts?.original_statuses ?? [], land_surfaces: opts?.land_surfaces ?? [], soil_types: opts?.soil_types ?? [] });
    } catch (e) { setItems([]); setPermissions({ p_add: false, p_del: false, p_view: false, p_edit: false }); setError(e instanceof Error ? e.message : "Data Propose Lands gagal dimuat."); }
    finally { setLoading(false); }
  }, [companyID, isAllCompanies]);
  useEffect(() => { const timer = window.setTimeout(() => { void load(); }, 0); return () => window.clearTimeout(timer); }, [load]);

  const filtered = useMemo(() => { const q = query.trim().toLowerCase(); return q ? items.filter((item) => [item.owner, item.block_plan, item.village, item.plant_division, item.plant_block].some((value) => value.toLowerCase().includes(q))) : items; }, [items, query]);
  const pageCount = Math.max(1, Math.ceil(filtered.length / pageSize));
  const currentPage = Math.min(page, pageCount);
  const shown = filtered.slice((currentPage - 1) * pageSize, currentPage * pageSize);

  function setField<K extends keyof Form>(key: K, value: Form[K]) { setForm((current) => ({ ...current, [key]: value })); }
  function openCreate() { setEditing(null); setForm(blankForm()); setError(""); setShowForm(true); }
  function openEdit(item: Row) {
    setEditing(item); const next = blankForm();
    for (const key of Object.keys(next) as (keyof Form)[]) { const value = item[key as keyof Row]; next[key] = value == null ? "" : String(value); }
    setForm(next); setError(""); setShowForm(true);
  }
  function closeForm() { if (!saving) { setShowForm(false); setEditing(null); setError(""); } }
  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setSaving(true); setError("");
    const numeric = ["dry_land_wide", "wet_land_wide", "land_wide", "dry_land_perha_price", "wet_land_perha_price", "admin_price", "plant_price"] as const;
    const payload: Record<string, string | number> = { ...form, Gis_Aquistion_Divisions_id: Number(form.Gis_Aquistion_Divisions_id), Gis_Land_Owners_id: Number(form.Gis_Land_Owners_id) };
    for (const key of numeric) payload[key] = Number(form[key] || 0);
    try { await apiRequest<Row>(editing ? `/api/propose-lands/${editing.id}?company_id=${companyID}` : `/api/propose-lands?company_id=${companyID}`, { method: editing ? "PUT" : "POST", body: JSON.stringify(payload) }); setShowForm(false); setEditing(null); await load(); }
    catch (e) { setError(e instanceof Error ? e.message : "Propose Land gagal disimpan."); }
    finally { setSaving(false); }
  }
  async function remove(item: Row) {
    const result = window.Swal ? await window.Swal.fire({ title: "Hapus Propose Land ini?", text: `${item.owner || item.village} akan dihapus.`, icon: "warning", showCancelButton: true, confirmButtonText: "Ya, hapus", cancelButtonText: "Batal", reverseButtons: true, buttonsStyling: false, customClass: { popup: "rounded-4 shadow-lg", confirmButton: "btn btn-danger ms-2", cancelButton: "btn btn-light" } }) : { isConfirmed: window.confirm(`Hapus Propose Land ${item.owner || item.village}?`) };
    if (!result.isConfirmed) return;
    try { await apiRequest<null>(`/api/propose-lands/${item.id}?company_id=${companyID}`, { method: "DELETE" }); await load(); }
    catch (e) { setError(e instanceof Error ? e.message : "Propose Land gagal dihapus."); }
  }
  function selectRow(label: string, key: keyof Form, choices: { value: string; name: string }[], required = false) { return <div className="row align-items-center py-3 border-bottom" key={key}><label htmlFor={`propose-${key}`} className="col-sm-3 col-form-label">{label}{required && <span className="text-danger ms-1">*</span>}</label><div className="col-sm-6"><select id={`propose-${key}`} className="form-select form-select-solid" value={form[key]} required={required} onChange={(e) => setField(key, e.target.value)}><option value="">Select {label}</option>{choices.map((choice) => <option key={choice.value} value={choice.value}>{choice.name}</option>)}</select></div></div>; }
  function inputRow(label: string, key: keyof Form, type = "text", step?: string) { return <div className="row align-items-center py-3 border-bottom" key={key}><label htmlFor={`propose-${key}`} className="col-sm-3 col-form-label">{label}</label><div className="col-sm-6"><input id={`propose-${key}`} className="form-control form-control-solid" type={type} min={type === "number" ? 0 : undefined} step={step} value={form[key]} onChange={(e) => setField(key, e.target.value)} /></div></div>; }

  return <div className="d-flex flex-column gap-6">
    <nav aria-label="Breadcrumb"><ol className="breadcrumb breadcrumb-separatorless fw-semibold fs-7 mb-0"><li className="breadcrumb-item"><Link href="/admin/dashboard" className="text-muted">Admin</Link></li><li className="breadcrumb-item text-muted ms-2">/ Lands / Propose Lands</li></ol></nav>
    <div className="d-flex flex-wrap align-items-center justify-content-between gap-4"><div><h1 className="text-gray-900 fw-bolder fs-2 mb-1">Propose Lands</h1><p className="text-muted fw-semibold mb-0">Company aktif: <strong>{isAllCompanies ? "Semua company (00)" : companyName || "Belum dipilih"}</strong></p></div>{permissions.p_add && <button type="button" className="btn btn-dark" onClick={openCreate} disabled={!companyID && !isAllCompanies}>+ Add Propose Land</button>}</div>
    {error && !showForm && <div className="alert alert-danger" role="alert">{error}</div>}
    <section className="card border shadow-sm"><div className="card-header flex-wrap gap-4 py-4"><span className="text-muted fw-semibold fs-7">Total data: {items.length}</span><label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7 ms-auto">Search<input className="form-control form-control-sm form-control-solid ref-table-search" type="search" placeholder="Owner, village, block plan..." value={query} onChange={(e) => { setQuery(e.target.value); setPage(1); }} /></label><label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7">Show<select className="form-select form-select-sm form-select-solid w-auto" value={pageSize} onChange={(e) => { setPageSize(Number(e.target.value)); setPage(1); }}><option>10</option><option>25</option><option>50</option><option>100</option></select>entries</label><button type="button" className="btn btn-sm btn-light" onClick={() => void load()} disabled={loading}>Refresh</button></div>
      <div className="card-body pt-0"><div className="table-responsive"><table className="table admin-data-table align-middle"><thead className="bg-dark"><tr className="text-white fw-bold text-uppercase fs-8">{["Owner", "Village", "Land Size (ha)", "Wet Size (ha)", "wet price (@Ha)", "Dry Size (ha)", "dry price (@Ha)", "avg (@Ha)", "Admin price", "Plant price", "Total price", "Actions"].map((label) => <th key={label} className="bg-dark text-white text-nowrap">{label}</th>)}</tr></thead><tbody>
        {loading ? <TableLoadingRows columns={12} /> : !companyID && !isAllCompanies ? <tr><td colSpan={12} className="text-center py-12 text-muted">Pilih company aktif di header untuk melihat data.</td></tr> : shown.length === 0 ? <tr><td colSpan={12} className="text-center py-12 text-muted">{query ? "Data tidak ditemukan." : "Belum ada Propose Land."}</td></tr> : shown.map((item) => <tr key={item.id}><td className="text-gray-900 fw-bold text-nowrap">{item.owner}</td><td>{item.village}</td><td>{area(item.land_wide)}</td><td>{area(item.wet_land_wide)}</td><td>{currency(item.wet_land_perha_price)}</td><td>{area(item.dry_land_wide)}</td><td>{currency(item.dry_land_perha_price)}</td><td>{currency(averagePrice(item))}</td><td>{currency(item.admin_price)}</td><td>{currency(item.plant_price)}</td><td>{currency(totalPrice(item))}</td><td><div className="admin-table-actions">{permissions.p_edit && <button type="button" className="btn btn-light-warning" title="Edit" aria-label={`Edit ${item.owner}`} onClick={() => openEdit(item)}><EditIcon /></button>}{permissions.p_del && <button type="button" className="btn btn-light-danger" title="Hapus" aria-label={`Hapus ${item.owner}`} onClick={() => void remove(item)}><TrashIcon /></button>}</div></td></tr>)}
      </tbody></table></div></div><div className="card-footer d-flex flex-wrap align-items-center justify-content-between gap-3 py-4"><span className="text-muted fw-semibold fs-7">Showing {filtered.length ? (currentPage - 1) * pageSize + 1 : 0}–{Math.min(currentPage * pageSize, filtered.length)} of {filtered.length} entries</span><div className="d-flex align-items-center gap-2"><button type="button" className="btn btn-sm btn-light" onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={currentPage <= 1}>Previous</button><span className="text-muted fw-semibold fs-7">{currentPage} / {pageCount}</span><button type="button" className="btn btn-sm btn-light" onClick={() => setPage((p) => Math.min(pageCount, p + 1))} disabled={currentPage >= pageCount}>Next</button></div></div>
    </section>
    {showForm && <div role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) closeForm(); }} style={{ position: "fixed", inset: 0, zIndex: 1055, background: "rgba(0,0,0,.45)", display: "grid", placeItems: "center", padding: 16 }}><div role="dialog" aria-modal="true" aria-labelledby="propose-modal-title" className="card shadow-lg" style={{ width: "min(100%, 1100px)", maxHeight: "92vh", overflow: "auto" }}><div className={`card-header py-4 form-modal-header ${editing ? "form-modal-header--edit" : "form-modal-header--add"}`}><h2 id="propose-modal-title" className="card-title fs-4 fw-bold">{editing ? "Edit" : "Add"} Propose Land</h2><button type="button" className="btn btn-sm btn-icon btn-light" aria-label="Tutup" onClick={closeForm} disabled={saving}>×</button></div><form onSubmit={(event) => void save(event)}><div className="card-body py-0">{error && <div className="alert alert-danger mt-4" role="alert">{error}</div>}
      {selectRow("Block Plan", "Gis_Aquistion_Divisions_id", options.block_plans.map((option) => ({ value: String(option.id), name: option.name })), true)}
      {selectRow("Owner", "Gis_Land_Owners_id", options.owners.map((option) => ({ value: String(option.id), name: option.name })), true)}
      {textFields.slice(0, 2).map((field) => inputRow(field.label, field.key))}
      {selectRow("Original Land Status", "alas_hak_jenis", options.original_statuses)}
      {inputRow("Original Land Status Number", "alas_hak_no")}
      {selectRow("Land surface", "land_surface", options.land_surfaces)}
      {inputRow("Slope percentage", "slope_persentage")}
      {selectRow("Soil type", "soil_type", options.soil_types)}
      {textFields.slice(4, 9).map((field) => inputRow(field.label, field.key))}
      {numberFields.map((field) => inputRow(field.label, field.key, "number", field.step))}
      <div className="row align-items-start py-3 border-bottom"><label htmlFor="propose-vegetation_note" className="col-sm-3 col-form-label">Vegetation note</label><div className="col-sm-8"><textarea id="propose-vegetation_note" className="form-control form-control-solid" rows={7} value={form.vegetation_note} onChange={(e) => setField("vegetation_note", e.target.value)} /></div></div>
      {inputRow("Measurement date (dd/mm/yyyy hh:mm:ss)", "measurement_date")}
      {inputRow("Measured by", "measured_by")}
    </div><div className="card-footer d-flex gap-3 py-4"><button type="submit" className={`btn ${editing ? "btn-warning" : "btn-dark"}`} disabled={saving}>{saving ? "Saving..." : editing ? "Update" : "Save"}</button><button type="button" className="btn btn-light" disabled={saving} onClick={closeForm}>Cancel</button></div></form></div></div>}
  </div>;
}
