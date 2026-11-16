"use client";

import Link from "next/link";
import AcquisitionLandTabs from "../acquisition-land-tabs";
import { useCallback, useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import { useUserProfile } from "../../../../components/auth/user-profile-context";
import TableLoadingRows from "../../../../components/ui/table-loading-rows";

type Memo = {
  id: number;
  company_id: number;
  company_name: string;
  memo_number: string;
  memo_date: string;
  memo_status: number;
  status_name: string;
  memo_doc: string;
  memo_note: string;
};
type Land = {
  id: number;
  division: string;
  owner: string;
  village: string;
  land_wide: number;
  wet_land_wide: number;
  wet_land_perha_price: number;
  dry_land_wide: number;
  dry_land_perha_price: number;
  admin_price: number;
  plant_price: number;
  status: number;
  status_name: string;
};
type Company = { id: number; name: string; is_primary: boolean };
type Permissions = { p_add: boolean; p_edit: boolean; p_del: boolean; p_view: boolean };
type MemoForm = { company_id: string; memo_number: string; memo_date: string; memo_note: string; clear_doc: boolean };
type Envelope<T> = { data?: T; message?: string };
type MemoSort = "company_name" | "memo_number" | "memo_date" | "status_name";
type LandSort = "division" | "owner" | "village" | "land_wide" | "total" | "status_name";

const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");
const emptyPermissions: Permissions = { p_add: false, p_edit: false, p_del: false, p_view: false };
const emptyForm = (): MemoForm => ({ company_id: "", memo_number: "", memo_date: "", memo_note: "", clear_doc: false });
const money = (value: number) => new Intl.NumberFormat("id-ID", { maximumFractionDigits: 0 }).format(value || 0);
const area = (value: number) => new Intl.NumberFormat("id-ID", { maximumFractionDigits: 2 }).format(value || 0);
const totalPrice = (land: Land) => land.wet_land_wide * land.wet_land_perha_price + land.dry_land_wide * land.dry_land_perha_price + land.admin_price + land.plant_price;
const averagePrice = (land: Land) => land.land_wide > 0 ? Math.round((land.wet_land_wide * land.wet_land_perha_price + land.dry_land_wide * land.dry_land_perha_price) / land.land_wide) : 0;
const noteText = (value: string) => value.replace(/<[^>]+>/g, " ").replace(/&nbsp;/gi, " ").replace(/\s+/g, " ").trim();
const formatDate = (value: string) => value ? new Date(value).toLocaleString("id-ID", { dateStyle: "medium", timeStyle: "short" }) : "—";

function EditIcon() { return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m14 5 5 5M4 20l4.5-1 10.8-10.8a2.12 2.12 0 0 0-3-3L5.5 16 4 20Z" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>; }
function TrashIcon() { return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M4 7h16M10 11v6m4-6v6M6 7l1 14h10l1-14M9 7V4h6v3" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>; }
function LandIcon() { return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M3 17 8 6l5 12 4-8 4 7M3 20h18" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>; }
function PlusIcon() { return <svg width="17" height="17" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M12 5v14M5 12h14" stroke="currentColor" strokeWidth="2" strokeLinecap="round" /></svg>; }
function SaveIcon() { return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M5 3h12l3 3v15H4V3h1Zm2 0v6h9V3M7 21v-8h10v8" stroke="currentColor" strokeWidth="1.8" strokeLinejoin="round" /></svg>; }

async function apiRequest<T>(path: string, init?: RequestInit, refreshed = false): Promise<T> {
  const response = await fetch(`${apiBase}${path}`, { ...init, credentials: "include", cache: "no-store" });
  if (response.status === 401 && !refreshed) {
    const renewal = await fetch(`${apiBase}/api/auth/refresh`, { method: "POST", credentials: "include", cache: "no-store" });
    if (renewal.ok) return apiRequest<T>(path, init, true);
  }
  const result = (await response.json().catch(() => null)) as Envelope<T> | null;
  if (!response.ok) throw new Error(result?.message || "Permintaan gagal.");
  return result?.data as T;
}

export default function AcquisitionRequestPage() {
  const profile = useUserProfile();
  const [companyID, setCompanyID] = useState(0);
  const [companyName, setCompanyName] = useState("");
  const isAllCompanies = companyName.trim() === "00" || /^00(?:\b|[\s-])/i.test(companyName);
  const [companies, setCompanies] = useState<Company[]>([]);
  const [items, setItems] = useState<Memo[]>([]);
  const [permissions, setPermissions] = useState<Permissions>(emptyPermissions);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [filters, setFilters] = useState({ company: "", number: "", date: "", status: "", note: "" });
  const [sortColumn, setSortColumn] = useState<MemoSort>("memo_date");
  const [sortDesc, setSortDesc] = useState(true);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(25);
  const [editing, setEditing] = useState<Memo | null>(null);
  const [form, setForm] = useState<MemoForm>(emptyForm());
  const [file, setFile] = useState<File | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [landMemo, setLandMemo] = useState<Memo | null>(null);
  const [lands, setLands] = useState<Land[]>([]);
  const [landsLoading, setLandsLoading] = useState(false);
  const [landsError, setLandsError] = useState("");
  const [landQuery, setLandQuery] = useState("");
  const [landFilters, setLandFilters] = useState({ division: "", owner: "", village: "", status: "" });
  const [landSort, setLandSort] = useState<LandSort>("division");
  const [landSortDesc, setLandSortDesc] = useState(false);
  const [landPage, setLandPage] = useState(1);
  const [landPageSize, setLandPageSize] = useState(25);
  const [selectedLand, setSelectedLand] = useState<Land | null>(null);

  useEffect(() => {
    const storageKey = `gis.selected-company.${profile.username.toLowerCase()}`;
    const read = () => {
      const id = Number(window.localStorage.getItem(storageKey)) || 0;
      setCompanyID(id);
      if (!id) setCompanyName("");
    };
    const changed = (event: Event) => {
      const detail = (event as CustomEvent<{ id?: number; name?: string }>).detail;
      if (!detail || !Number.isFinite(Number(detail.id))) { read(); return; }
      const name = detail.name ?? "";
      setCompanyID(name.trim() === "00" || /^00(?:\b|[\s-])/i.test(name) ? 0 : Number(detail.id));
      setCompanyName(name);
    };
    read();
    window.addEventListener("gis:company-change", changed);
    window.addEventListener("storage", read);
    return () => { window.removeEventListener("gis:company-change", changed); window.removeEventListener("storage", read); };
  }, [profile.username]);

  useEffect(() => {
    let active = true;
    void apiRequest<Company[]>("/api/auth/companies")
      .then((result) => { if (active) setCompanies(Array.isArray(result) ? result : []); })
      .catch(() => { if (active) setCompanies([]); });
    return () => { active = false; };
  }, []);

  const load = useCallback(async () => {
    if (!companyID && !isAllCompanies) {
      setItems([]); setPermissions(emptyPermissions); setLoading(false);
      return;
    }
    setLoading(true); setError("");
    try {
      const params = `?company_id=${companyID}`;
      const [rows, priv] = await Promise.all([
        apiRequest<Memo[]>(`/api/acquisition-memos${params}`),
        apiRequest<Permissions>(`/api/acquisition-memos/permissions${params}`),
      ]);
      setItems(Array.isArray(rows) ? rows : []);
      setPermissions(priv ?? emptyPermissions);
    } catch (cause) {
      setItems([]); setPermissions(emptyPermissions);
      setError(cause instanceof Error ? cause.message : "Data memo gagal dimuat.");
    } finally { setLoading(false); }
  }, [companyID, isAllCompanies]);
  useEffect(() => { const timer = window.setTimeout(() => { void load(); }, 0); return () => window.clearTimeout(timer); }, [load]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    const matches = (value: string, filter: string) => value.toLowerCase().includes(filter.trim().toLowerCase());
    return items.filter((item) =>
      (!q || [item.company_name, item.memo_number, item.memo_date, item.status_name, noteText(item.memo_note)].some((value) => value.toLowerCase().includes(q)))
      && matches(item.company_name, filters.company)
      && matches(item.memo_number, filters.number)
      && matches(item.memo_date, filters.date)
      && matches(item.status_name, filters.status)
      && matches(noteText(item.memo_note), filters.note));
  }, [items, query, filters]);
  const sorted = useMemo(() => [...filtered].sort((a, b) => {
    const comparison = String(a[sortColumn] ?? "").localeCompare(String(b[sortColumn] ?? ""), undefined, { numeric: true, sensitivity: "base" });
    return sortDesc ? -comparison : comparison;
  }), [filtered, sortColumn, sortDesc]);
  const pageCount = Math.max(1, Math.ceil(sorted.length / pageSize));
  const currentPage = Math.min(page, pageCount);
  const shown = sorted.slice((currentPage - 1) * pageSize, currentPage * pageSize);

  const filteredLands = useMemo(() => {
    const q = landQuery.trim().toLowerCase();
    const matches = (value: string, filter: string) => value.toLowerCase().includes(filter.trim().toLowerCase());
    return lands.filter((item) =>
      (!q || [item.division, item.owner, item.village, item.status_name, String(item.id)].some((value) => value.toLowerCase().includes(q)))
      && matches(item.division, landFilters.division)
      && matches(item.owner, landFilters.owner)
      && matches(item.village, landFilters.village)
      && matches(item.status_name, landFilters.status));
  }, [lands, landQuery, landFilters]);
  const sortedLands = useMemo(() => [...filteredLands].sort((a, b) => {
    const left = landSort === "total" ? totalPrice(a) : a[landSort];
    const right = landSort === "total" ? totalPrice(b) : b[landSort];
    const comparison = typeof left === "number" && typeof right === "number" ? left - right : String(left).localeCompare(String(right), undefined, { numeric: true, sensitivity: "base" });
    return landSortDesc ? -comparison : comparison;
  }), [filteredLands, landSort, landSortDesc]);
  const landPageCount = Math.max(1, Math.ceil(sortedLands.length / landPageSize));
  const currentLandPage = Math.min(landPage, landPageCount);
  const shownLands = sortedLands.slice((currentLandPage - 1) * landPageSize, currentLandPage * landPageSize);

  function toggleSort(column: MemoSort) { if (sortColumn === column) setSortDesc((value) => !value); else { setSortColumn(column); setSortDesc(false); } }
  function toggleLandSort(column: LandSort) { if (landSort === column) setLandSortDesc((value) => !value); else { setLandSort(column); setLandSortDesc(false); } }
  function setFilter(key: keyof typeof filters, value: string) { setFilters((current) => ({ ...current, [key]: value })); setPage(1); }
  function setLandFilter(key: keyof typeof landFilters, value: string) { setLandFilters((current) => ({ ...current, [key]: value })); setLandPage(1); }
  function openCreate() { setEditing(null); setForm({ ...emptyForm(), company_id: companyID ? String(companyID) : "" }); setFile(null); setError(""); setShowForm(true); }
  function openEdit(item: Memo) { setEditing(item); setForm({ company_id: String(item.company_id), memo_number: item.memo_number, memo_date: item.memo_date.slice(0, 16), memo_note: item.memo_note, clear_doc: false }); setFile(null); setError(""); setShowForm(true); }
  function closeForm() { if (!saving) { setShowForm(false); setEditing(null); setFile(null); setError(""); } }
  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setSaving(true); setError("");
    const body = new FormData();
    body.set("data", JSON.stringify({ ...form, company_id: Number(form.company_id) }));
    if (file) body.set("memo_doc", file);
    try {
      await apiRequest<Memo>(editing ? `/api/acquisition-memos/${editing.id}?company_id=${companyID}` : `/api/acquisition-memos?company_id=${companyID}`, { method: editing ? "PUT" : "POST", body });
      setShowForm(false); setEditing(null); setFile(null); await load();
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Memo gagal disimpan."); }
    finally { setSaving(false); }
  }
  async function remove(item: Memo) {
    const result = window.Swal ? await window.Swal.fire({ title: "Hapus memo ini?", text: `${item.memo_number} akan dihapus.`, icon: "warning", showCancelButton: true, confirmButtonText: "Ya, hapus", cancelButtonText: "Batal", reverseButtons: true, buttonsStyling: false, customClass: { popup: "rounded-4 shadow-lg", confirmButton: "btn btn-danger ms-2", cancelButton: "btn btn-light" } }) : { isConfirmed: window.confirm(`Hapus memo ${item.memo_number}?`) };
    if (!result.isConfirmed) return;
    try { await apiRequest<null>(`/api/acquisition-memos/${item.id}?company_id=${companyID}`, { method: "DELETE" }); await load(); }
    catch (cause) { setError(cause instanceof Error ? cause.message : "Memo gagal dihapus."); }
  }
  async function openLands(item: Memo) {
    setLandMemo(item); setLands([]); setSelectedLand(null); setLandQuery(""); setLandFilters({ division: "", owner: "", village: "", status: "" }); setLandPage(1); setLandsError(""); setLandsLoading(true);
    try { const rows = await apiRequest<Land[]>(`/api/acquisition-memos/${item.id}/lands?company_id=${companyID}`); setLands(Array.isArray(rows) ? rows : []); }
    catch (cause) { setLandsError(cause instanceof Error ? cause.message : "Propose Land gagal dimuat."); }
    finally { setLandsLoading(false); }
  }

  return <div className="d-flex flex-column gap-6">
    <nav aria-label="Breadcrumb"><ol className="breadcrumb breadcrumb-separatorless fw-semibold fs-7 mb-0"><li className="breadcrumb-item"><Link href="/admin/dashboard" className="text-muted">Admin</Link></li><li className="breadcrumb-item text-muted ms-2">/ Acquisition Process / Acquisition Request</li></ol></nav>
    <div className="d-flex flex-wrap align-items-center justify-content-between gap-4"><div><h1 className="text-gray-900 fw-bolder fs-2 mb-1">Acquisition Request</h1><p className="text-muted fw-semibold mb-0">Company aktif: <strong>{isAllCompanies ? "Semua company (00)" : companyName || companies.find((item) => item.id === companyID)?.name || "Belum dipilih"}</strong></p></div>{permissions.p_add && <button type="button" className="btn btn-dark d-inline-flex align-items-center gap-2" onClick={openCreate}><PlusIcon /> Add Acquisition Memo</button>}</div>
    {error && !showForm && <div className="alert alert-danger" role="alert">{error}</div>}
    <section className="card border shadow-sm">
      <div className="card-header flex-wrap gap-4 py-4"><span className="text-muted fw-semibold fs-7">Total data: {items.length}</span><label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7 ms-auto">Search<input className="form-control form-control-sm form-control-solid ref-table-search" type="search" aria-label="Cari memo" placeholder="Company, memo, status..." value={query} onChange={(event) => { setQuery(event.target.value); setPage(1); }} /></label><label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7">Show<select className="form-select form-select-sm form-select-solid w-auto" value={pageSize} onChange={(event) => { setPageSize(Number(event.target.value)); setPage(1); }}><option>10</option><option>25</option><option>50</option><option>100</option></select>entries</label><button type="button" className="btn btn-sm btn-light" disabled={loading} onClick={() => void load()}>Refresh</button></div>
      <div className="card-body pt-0"><div className="table-responsive"><table className="table admin-data-table align-middle"><thead className="bg-dark"><tr>{([["company_name", "Company"], ["memo_number", "Memo number"], ["memo_date", "Memo date"], ["status_name", "Memo status"]] as [MemoSort, string][]).map(([column, label]) => <th key={column} className="bg-dark text-white text-nowrap"><button type="button" className="btn btn-link p-0 text-white fw-bold text-uppercase fs-8 text-decoration-none" onClick={() => toggleSort(column)}>{label} {sortColumn === column ? (sortDesc ? "↓" : "↑") : "↕"}</button></th>)}<th className="bg-dark text-white text-uppercase fs-8">Memo doc</th><th className="bg-dark text-white text-uppercase fs-8">Memo note</th><th className="bg-dark text-white text-uppercase fs-8">Actions</th></tr><tr>{(["company", "number", "date", "status"] as const).map((key) => <th key={key}><input type="search" className="form-control form-control-sm form-control-solid" aria-label={`Filter ${key}`} placeholder={`Filter ${key}`} value={filters[key]} onChange={(event) => setFilter(key, event.target.value)} /></th>)}<th /><th><input type="search" className="form-control form-control-sm form-control-solid" aria-label="Filter memo note" placeholder="Filter note" value={filters.note} onChange={(event) => setFilter("note", event.target.value)} /></th><th /></tr></thead><tbody>
        {loading ? <TableLoadingRows columns={7} /> : !companyID && !isAllCompanies ? <tr><td colSpan={7} className="text-center py-12 text-muted">Pilih company aktif di header untuk melihat data.</td></tr> : shown.length === 0 ? <tr><td colSpan={7} className="text-center py-12 text-muted">{query || Object.values(filters).some(Boolean) ? "Data tidak ditemukan." : "Belum ada memo."}</td></tr> : shown.map((item) => <tr key={item.id}><td className="text-gray-900 fw-bold text-nowrap">{item.company_name || "—"}</td><td className="text-nowrap">{item.memo_number}</td><td className="text-nowrap">{formatDate(item.memo_date)}</td><td><span className="badge badge-light-primary">{item.status_name || `Status ${item.memo_status}`}</span></td><td>{item.memo_doc ? <a href={`${apiBase}/api/acquisition-memos/${item.id}/document?company_id=${companyID}`} target="_blank" rel="noopener noreferrer">View Doc</a> : "—"}</td><td className="text-truncate" style={{ maxWidth: 260 }} title={noteText(item.memo_note)}>{noteText(item.memo_note) || "—"}</td><td><div className="d-flex align-items-center gap-2 text-nowrap"><button type="button" className="btn btn-sm btn-light-primary d-inline-flex align-items-center gap-2" onClick={() => void openLands(item)}><LandIcon /> Propose Land</button><div className="admin-table-actions">{permissions.p_edit && <button type="button" className="btn btn-light-warning" title="Edit" aria-label={`Edit memo ${item.memo_number}`} onClick={() => openEdit(item)}><EditIcon /></button>}{permissions.p_del && <button type="button" className="btn btn-light-danger" title="Hapus" aria-label={`Hapus memo ${item.memo_number}`} onClick={() => void remove(item)}><TrashIcon /></button>}</div></div></td></tr>)}
      </tbody></table></div></div>
      <div className="card-footer d-flex flex-wrap align-items-center justify-content-between gap-3 py-4"><span className="text-muted fw-semibold fs-7">Showing {sorted.length ? (currentPage - 1) * pageSize + 1 : 0}–{Math.min(currentPage * pageSize, sorted.length)} of {sorted.length} entries</span><div className="d-flex align-items-center gap-2"><button type="button" className="btn btn-sm btn-light" onClick={() => setPage((value) => Math.max(1, value - 1))} disabled={currentPage <= 1}>Previous</button><span className="text-muted fw-semibold fs-7">{currentPage} / {pageCount}</span><button type="button" className="btn btn-sm btn-light" onClick={() => setPage((value) => Math.min(pageCount, value + 1))} disabled={currentPage >= pageCount}>Next</button></div></div>
    </section>

    {showForm && <div role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) closeForm(); }} style={{ position: "fixed", inset: 0, zIndex: 1055, background: "rgba(0,0,0,.45)", display: "grid", placeItems: "center", padding: 16 }}><div role="dialog" aria-modal="true" aria-labelledby="memo-modal-title" className="card shadow-lg" style={{ width: "min(100%, 940px)", maxHeight: "92vh", overflow: "auto" }}><div className={`card-header py-4 form-modal-header ${editing ? "form-modal-header--edit" : "form-modal-header--add"}`}><h2 id="memo-modal-title" className="card-title fs-4 fw-bold">{editing ? `Edit Acquisition Memo #${editing.id}` : "Add Acquisition Memo"}</h2><button type="button" className="btn btn-sm btn-icon btn-light" aria-label="Tutup" onClick={closeForm} disabled={saving}>×</button></div><form onSubmit={(event) => void save(event)}><div className="card-body"><p className="text-muted fs-7">Isi informasi memo untuk company yang dipilih.</p>{error && <div className="alert alert-danger" role="alert">{error}</div>}
      <div className="row align-items-center py-3 border-bottom"><label htmlFor="memo-company" className="col-sm-3 col-form-label">Company <span className="text-danger">*</span></label><div className="col-sm-8"><select id="memo-company" className="form-select form-select-solid" value={form.company_id} onChange={(event) => setForm((current) => ({ ...current, company_id: event.target.value }))} disabled={!isAllCompanies} required><option value="">Select Company</option>{companies.filter((company) => isAllCompanies || company.id === companyID || company.id === editing?.company_id).map((company) => <option key={company.id} value={company.id}>{company.name}</option>)}</select></div></div>
      <div className="row align-items-center py-3 border-bottom"><label htmlFor="memo-number" className="col-sm-3 col-form-label">Memo number <span className="text-danger">*</span></label><div className="col-sm-8"><input id="memo-number" className="form-control form-control-solid" value={form.memo_number} onChange={(event) => setForm((current) => ({ ...current, memo_number: event.target.value }))} required maxLength={255} /></div></div>
      <div className="row align-items-center py-3 border-bottom"><label htmlFor="memo-date" className="col-sm-3 col-form-label">Memo date <span className="text-danger">*</span></label><div className="col-sm-8"><input id="memo-date" type="datetime-local" className="form-control form-control-solid" value={form.memo_date} onChange={(event) => setForm((current) => ({ ...current, memo_date: event.target.value }))} required /></div></div>
      <div className="row align-items-center py-3 border-bottom"><label htmlFor="memo-document" className="col-sm-3 col-form-label">Memo doc</label><div className="col-sm-8"><input id="memo-document" type="file" className="form-control form-control-solid" accept=".pdf,.png,.jpg,.jpeg,.doc,.docx" onChange={(event) => setFile(event.target.files?.[0] ?? null)} /><small className="text-muted">PDF, gambar, DOC, atau DOCX. Maksimal 15 MB.</small>{editing?.memo_doc && <div className="d-flex flex-wrap align-items-center gap-3 mt-2"><a href={`${apiBase}/api/acquisition-memos/${editing.id}/document?company_id=${companyID}`} target="_blank" rel="noopener noreferrer">Lihat dokumen saat ini</a><label className="d-flex align-items-center gap-2"><input type="checkbox" checked={form.clear_doc} onChange={(event) => setForm((current) => ({ ...current, clear_doc: event.target.checked }))} /> Hapus dokumen saat ini</label></div>}</div></div>
      <div className="row align-items-start py-3"><label htmlFor="memo-note" className="col-sm-3 col-form-label">Memo note</label><div className="col-sm-8"><textarea id="memo-note" className="form-control form-control-solid" rows={9} value={form.memo_note} onChange={(event) => setForm((current) => ({ ...current, memo_note: event.target.value }))} /></div></div>
    </div><div className="card-footer d-flex gap-3 py-4"><button type="submit" className={`btn d-inline-flex align-items-center gap-2 ${editing ? "btn-warning" : "btn-dark"}`} disabled={saving}><SaveIcon /> {saving ? "Saving..." : editing ? "Update" : "Submit"}</button><button type="button" className="btn btn-light" disabled={saving} onClick={closeForm}>Cancel</button></div></form></div></div>}

    {landMemo && <div role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) setLandMemo(null); }} style={{ position: "fixed", inset: 0, zIndex: 1055, background: "rgba(0,0,0,.55)", padding: 16, overflow: "auto" }}><section role="dialog" aria-modal="true" aria-labelledby="memo-lands-title" className="card shadow-lg mx-auto my-4" style={{ width: "min(100%, 1500px)", minHeight: "min(75vh, 700px)" }}><div className="card-header py-4 form-modal-header form-modal-header--add"><div><h2 id="memo-lands-title" className="card-title fs-4 fw-bold mb-1">Propose Land · {landMemo.memo_number}</h2><span className="text-muted fs-7">Memo date: {formatDate(landMemo.memo_date)}</span></div><button type="button" className="btn btn-sm btn-icon btn-light" aria-label="Tutup" onClick={() => setLandMemo(null)}>×</button></div><div className="card-body"><div className="d-flex flex-wrap align-items-center gap-3 mb-4"><span className="text-muted fw-semibold fs-7">Total data: {lands.length}</span><label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7 ms-auto">Search<input type="search" className="form-control form-control-sm form-control-solid ref-table-search" placeholder="Division, owner, village..." value={landQuery} onChange={(event) => { setLandQuery(event.target.value); setLandPage(1); }} /></label><label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7">Show<select className="form-select form-select-sm form-select-solid w-auto" value={landPageSize} onChange={(event) => { setLandPageSize(Number(event.target.value)); setLandPage(1); }}><option>10</option><option>25</option><option>50</option><option>100</option></select>entries</label></div>{landsError && <div className="alert alert-danger" role="alert">{landsError}</div>}<div className="table-responsive"><table className="table admin-data-table align-middle"><thead className="bg-dark"><tr>{([["division", "Division"], ["owner", "Owner"], ["village", "Village"], ["land_wide", "Land Size (ha)"]] as [LandSort, string][]).map(([column, label]) => <th key={column} className="bg-dark text-white text-nowrap"><button type="button" className="btn btn-link p-0 text-white fw-bold fs-8 text-decoration-none" onClick={() => toggleLandSort(column)}>{label} {landSort === column ? (landSortDesc ? "↓" : "↑") : "↕"}</button></th>)}{["Wet Size (ha)", "Wet price (@Ha)", "Dry Size (ha)", "Dry price (@Ha)", "avg (@Ha)", "Admin price", "Plant price"].map((label) => <th key={label} className="bg-dark text-white text-nowrap fs-8">{label}</th>)}<th className="bg-dark text-white text-nowrap"><button type="button" className="btn btn-link p-0 text-white fw-bold fs-8 text-decoration-none" onClick={() => toggleLandSort("total")}>Total price {landSort === "total" ? (landSortDesc ? "↓" : "↑") : "↕"}</button></th><th className="bg-dark text-white text-nowrap"><button type="button" className="btn btn-link p-0 text-white fw-bold fs-8 text-decoration-none" onClick={() => toggleLandSort("status_name")}>Status {landSort === "status_name" ? (landSortDesc ? "↓" : "↑") : "↕"}</button></th><th className="bg-dark text-white fs-8">Actions</th></tr><tr>{(["division", "owner", "village"] as const).map((key) => <th key={key}><input type="search" className="form-control form-control-sm form-control-solid" aria-label={`Filter ${key}`} placeholder={`Filter ${key}`} value={landFilters[key]} onChange={(event) => setLandFilter(key, event.target.value)} /></th>)}{Array.from({ length: 9 }, (_, index) => <th key={index} />)}<th><input type="search" className="form-control form-control-sm form-control-solid" aria-label="Filter status" placeholder="Filter status" value={landFilters.status} onChange={(event) => setLandFilter("status", event.target.value)} /></th><th /></tr></thead><tbody>
      {landsLoading ? <TableLoadingRows columns={14} /> : shownLands.length === 0 ? <tr><td colSpan={14} className="text-center py-12 text-muted">{landQuery || Object.values(landFilters).some(Boolean) ? "Data tidak ditemukan." : "Belum ada Propose Land pada memo ini."}</td></tr> : shownLands.map((land) => <tr key={land.id}><td className="text-gray-900 fw-bold">{land.division || "—"}</td><td>{land.owner || "—"}</td><td>{land.village || "—"}</td><td>{area(land.land_wide)}</td><td>{area(land.wet_land_wide)}</td><td>{money(land.wet_land_perha_price)}</td><td>{area(land.dry_land_wide)}</td><td>{money(land.dry_land_perha_price)}</td><td>{money(averagePrice(land))}</td><td>{money(land.admin_price)}</td><td>{money(land.plant_price)}</td><td>{money(totalPrice(land))}</td><td>{land.status_name}</td><td><button type="button" className="btn btn-sm btn-light-primary" onClick={() => setSelectedLand(land)}>Detail</button></td></tr>)}
    </tbody></table></div><div className="d-flex flex-wrap align-items-center justify-content-between gap-3 mt-4"><span className="text-muted fs-7">Showing {sortedLands.length ? (currentLandPage - 1) * landPageSize + 1 : 0}–{Math.min(currentLandPage * landPageSize, sortedLands.length)} of {sortedLands.length} entries</span><div className="d-flex align-items-center gap-2"><button type="button" className="btn btn-sm btn-light" disabled={currentLandPage <= 1} onClick={() => setLandPage((value) => Math.max(1, value - 1))}>Previous</button><span className="text-muted fs-7">{currentLandPage} / {landPageCount}</span><button type="button" className="btn btn-sm btn-light" disabled={currentLandPage >= landPageCount} onClick={() => setLandPage((value) => Math.min(landPageCount, value + 1))}>Next</button></div></div>
      {selectedLand && landMemo && <section className="card border shadow-sm mt-6" aria-label={`Detail Propose Land ${selectedLand.id}`}><div className="card-header py-4"><h3 className="card-title fs-4 fw-bold mb-0">Detail Propose Land #{selectedLand.id}</h3><button type="button" className="btn btn-sm btn-light" onClick={() => setSelectedLand(null)}>Close detail</button></div><AcquisitionLandTabs key={selectedLand.id} land={{ ...selectedLand, company_id: landMemo.company_id, company_name: landMemo.company_name, block_plan: selectedLand.division }} apiPath={"/api/acquisition-memos/" + landMemo.id + "/lands"} memoNumber={landMemo.memo_number} /></section>}
    </div><div className="card-footer py-4"><button type="button" className="btn btn-light" onClick={() => setLandMemo(null)}>Close</button></div></section></div>}
  </div>;
}
