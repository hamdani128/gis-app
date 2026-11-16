"use client";

import TableLoadingRows from "../../../components/ui/table-loading-rows";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";

type CompanyInfo = {
  id: number | null;
  name: string;
  province_id: number | null;
  province: string;
  region_id: number | null;
  region: string;
  city: string | null;
  address: string | null;
};
type Option = { id: number; name: string };
type Options = { provinces: Option[]; regions: Option[] };
type CompanyInput = Pick<CompanyInfo, "name" | "province_id" | "region_id" | "city" | "address">;
type APIEnvelope<T> = { message?: string; data?: T };
type SortColumn = "name" | "province" | "region" | "city" | "address";

const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");

async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  if (init?.body && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  const response = await fetch(`${apiBase}${path}`, { ...init, credentials: "include", headers });
  const result = (await response.json().catch(() => null)) as APIEnvelope<T> | null;
  if (!response.ok) throw new Error(result?.message || "Permintaan gagal. Silakan coba lagi.");
  return result?.data as T;
}

function EditIcon() {
  return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m14 5 5 5M4 20l4.5-1 10.8-10.8a2.12 2.12 0 0 0-3-3L5.5 16 4 20Z" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>;
}
function DeleteIcon() {
  return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M4 7h16M10 11v6m4-6v6M6 7l1 14h10l1-14M9 7V4h6v3" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>;
}
function SaveIcon() {
  return <svg className="me-2" width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M5 4h12l3 3v13H4V4h1Zm3 0v6h8V4M8 20v-6h8v6" stroke="currentColor" strokeWidth="1.8" strokeLinejoin="round" /></svg>;
}

function uniqueOptions(options: Option[]) {
  const seen = new Set<number>();
  return options.filter((option) => option && Number.isFinite(option.id) && !seen.has(option.id) && Boolean(seen.add(option.id)));
}

export default function CompanyInfoPage() {
  const [items, setItems] = useState<CompanyInfo[]>([]);
  const [provinces, setProvinces] = useState<Option[]>([]);
  const [regions, setRegions] = useState<Option[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [deletingID, setDeletingID] = useState<number | null>(null);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [pageSize, setPageSize] = useState(25);
  const [currentPage, setCurrentPage] = useState(1);
  const [sortColumn, setSortColumn] = useState<SortColumn>("name");
  const [sortAscending, setSortAscending] = useState(true);
  const [formOpen, setFormOpen] = useState(false);
  const [editingID, setEditingID] = useState<number | null>(null);
  const [name, setName] = useState("");
  const [provinceID, setProvinceID] = useState("");
  const [regionID, setRegionID] = useState("");
  const [city, setCity] = useState("");
  const [address, setAddress] = useState("");

  const loadItems = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [companyData, optionData] = await Promise.all([
        apiRequest<CompanyInfo[]>("/api/company-info"),
        apiRequest<Options>("/api/company-info/options"),
      ]);
      setItems(Array.isArray(companyData) ? companyData : []);
      setProvinces(Array.isArray(optionData?.provinces) ? uniqueOptions(optionData.provinces) : []);
      setRegions(Array.isArray(optionData?.regions) ? uniqueOptions(optionData.regions) : []);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : "Data gagal dimuat.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void loadItems(); }, [loadItems]);
  useEffect(() => {
    if (!formOpen) return;
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === "Escape" && !saving) closeForm();
    }
    document.addEventListener("keydown", closeOnEscape);
    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener("keydown", closeOnEscape);
    };
  }, [formOpen, saving]);

  const filteredItems = useMemo(() => {
    const search = query.trim().toLowerCase();
    if (!search) return items;
    return items.filter((item) => [item.name, item.province, item.region, item.city ?? "", item.address ?? ""]
      .some((value) => value.toLowerCase().includes(search)));
  }, [items, query]);
  const sortedItems = useMemo(() => [...filteredItems].sort((left, right) => {
    const comparison = String(left[sortColumn] ?? "").localeCompare(String(right[sortColumn] ?? ""), "id", { numeric: true, sensitivity: "base" });
    return sortAscending ? comparison : -comparison;
  }), [filteredItems, sortAscending, sortColumn]);
  const pageCount = Math.max(1, Math.ceil(sortedItems.length / pageSize));
  const visiblePage = Math.min(currentPage, pageCount);
  const firstRow = sortedItems.length === 0 ? 0 : (visiblePage - 1) * pageSize + 1;
  const lastRow = Math.min(visiblePage * pageSize, sortedItems.length);
  const visibleItems = sortedItems.slice(firstRow - 1, lastRow);

  function changeSort(column: SortColumn) {
    setCurrentPage(1);
    if (sortColumn === column) setSortAscending((ascending) => !ascending);
    else { setSortColumn(column); setSortAscending(true); }
  }
  function sortIndicator(column: SortColumn) {
    if (sortColumn !== column) return "↕";
    return sortAscending ? "↑" : "↓";
  }
  function clearForm() {
    setEditingID(null); setName(""); setProvinceID(""); setRegionID(""); setCity(""); setAddress("");
  }
  function closeForm() {
    setFormOpen(false); setError(""); clearForm();
  }
  function startCreate() {
    clearForm(); setError(""); setFormOpen(true);
  }
  function startEdit(item: CompanyInfo) {
    if (item.id === null) return;
    setEditingID(item.id);
    setName(item.name);
    setProvinceID(item.province_id === null ? "" : String(item.province_id));
    setRegionID(item.region_id === null ? "" : String(item.region_id));
    setCity(item.city ?? "");
    setAddress(item.address ?? "");
    setError(""); setFormOpen(true);
  }
  async function saveItem(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setSaving(true); setError("");
    const wasEditing = editingID !== null;
    const payload: CompanyInput = {
      name: name.trim(),
      province_id: provinceID === "" ? null : Number(provinceID),
      region_id: regionID === "" ? null : Number(regionID),
      city: city.trim() || null,
      address: address.trim() || null,
    };
    try {
      await apiRequest<CompanyInfo>(wasEditing ? `/api/company-info/${editingID}` : "/api/company-info", { method: wasEditing ? "PUT" : "POST", body: JSON.stringify(payload) });
      closeForm(); setCurrentPage(1); await loadItems();
      const message = wasEditing ? "Company berhasil diperbarui." : "Company berhasil ditambahkan.";
      if (window.Swal) await window.Swal.fire({ title: "Berhasil", text: message, icon: "success", buttonsStyling: false, confirmButtonText: "Oke", customClass: { popup: "rounded-4 shadow-lg", title: "fw-bold text-gray-900", confirmButton: "btn btn-dark" } });
      else window.alert(message);
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : "Data gagal disimpan.");
    } finally { setSaving(false); }
  }
  async function deleteItem(item: CompanyInfo) {
    if (item.id === null) return;
    const confirmation = window.Swal
      ? await window.Swal.fire({ title: "Hapus company ini?", text: `${item.name} akan dihapus.`, icon: "warning", showCancelButton: true, confirmButtonText: "Ya, hapus", cancelButtonText: "Batal", reverseButtons: true, buttonsStyling: false, customClass: { popup: "rounded-4 shadow-lg", title: "fw-bold text-gray-900", htmlContainer: "text-muted", confirmButton: "btn btn-danger ms-2", cancelButton: "btn btn-light me-2" } })
      : { isConfirmed: window.confirm(`Hapus company ${item.name}?`) };
    if (!confirmation.isConfirmed) return;
    setDeletingID(item.id); setError("");
    try {
      await apiRequest<null>(`/api/company-info/${item.id}`, { method: "DELETE" }); await loadItems();
      if (window.Swal) await window.Swal.fire({ title: "Berhasil", text: "Company berhasil dihapus.", icon: "success", buttonsStyling: false, confirmButtonText: "Oke", customClass: { popup: "rounded-4 shadow-lg", title: "fw-bold text-gray-900", confirmButton: "btn btn-dark" } });
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : "Data gagal dihapus.");
    } finally { setDeletingID(null); }
  }

  const headers: [SortColumn, string][] = [["name", "Name"], ["province", "Province"], ["region", "Region"], ["city", "City"], ["address", "Address"]];

  return (
    <div className="d-flex flex-column gap-6">
      <nav aria-label="Breadcrumb"><ol className="breadcrumb breadcrumb-separatorless fw-semibold fs-7 mb-0"><li className="breadcrumb-item"><Link href="/admin/dashboard" className="text-muted text-hover-primary">Admin</Link></li><li className="breadcrumb-item"><span className="bullet bg-gray-300 w-5px h-2px" /></li><li className="breadcrumb-item text-muted">Companies</li><li className="breadcrumb-item"><span className="bullet bg-gray-300 w-5px h-2px" /></li><li className="breadcrumb-item text-muted" aria-current="page">Companies Info</li></ol></nav>
      <div className="d-flex flex-wrap align-items-center justify-content-between gap-4"><div><h1 className="text-gray-900 fw-bolder fs-2 mb-1">Companies Info</h1><p className="text-muted fw-semibold mb-0">Kelola nama perusahaan, lokasi, dan alamat kantor.</p></div><button type="button" className="btn btn-dark" onClick={startCreate}><span className="me-2 fs-3 lh-1">+</span>Add Company</button></div>
      {!formOpen && error && <div className="alert alert-danger d-flex align-items-center justify-content-between mb-0" role="alert"><span>{error}</span><button type="button" className="btn-close" aria-label="Tutup pesan error" onClick={() => setError("")} /></div>}

      <section className="card border shadow-sm">
        <div className="card-header flex-wrap gap-4 py-4"><span className="text-muted fw-semibold fs-7">Total data: {items.length}</span>
          <label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7 me-auto">Search<input className="form-control form-control-sm form-control-solid ref-table-search" type="search" aria-label="Cari company" placeholder="Nama, province, region, city, address..." value={query} onChange={(event) => { setQuery(event.target.value); setCurrentPage(1); }} /></label>
          <label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7">Show<select className="form-select form-select-sm form-select-solid w-auto" value={pageSize} onChange={(event) => { setPageSize(Number(event.target.value)); setCurrentPage(1); }} aria-label="Jumlah baris per halaman"><option value={10}>10</option><option value={25}>25</option><option value={50}>50</option><option value={100}>100</option></select>entries</label>
          <button type="button" className="btn btn-icon btn-sm btn-light ms-auto" onClick={() => void loadItems()} disabled={loading} aria-label="Muat ulang daftar"><span className={loading ? "spinner-border spinner-border-sm" : "svg-icon svg-icon-2"} aria-hidden="true">{!loading && <svg viewBox="0 0 24 24" fill="none"><path d="M20 11a8 8 0 0 0-14.9-3M4 4v4h4m-4 5a8 8 0 0 0 14.9 3M20 20v-4h-4" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>}</span></button>
        </div>
        <div className="card-body pt-0"><div className="table-responsive"><table className="table admin-data-table align-middle">
          <thead className="bg-dark"><tr className="bg-dark text-white fw-bold text-uppercase fs-8">{headers.map(([column, label]) => <th key={column} className="bg-dark text-white" aria-sort={sortColumn === column ? (sortAscending ? "ascending" : "descending") : "none"}><button type="button" className="btn btn-link p-0 text-white fw-bold text-uppercase fs-8 text-decoration-none" onClick={() => changeSort(column)}>{label} <span aria-hidden="true">{sortIndicator(column)}</span></button></th>)}<th className="bg-dark text-white text-end">Actions</th></tr></thead>
          <tbody>{loading ? <TableLoadingRows columns={6} /> : visibleItems.length === 0 ? <tr><td colSpan={6} className="text-center py-12 text-muted">{query ? "Data tidak ditemukan." : "Belum ada company."}</td></tr> : visibleItems.map((item, index) => <tr key={item.id ?? `null-${index}`}>
            <td className="text-gray-900 fw-bold">{item.name || "—"}</td><td>{item.province || "—"}</td><td>{item.region || "—"}</td><td>{item.city || "—"}</td><td title={item.address ?? ""}>{item.address || "—"}</td>
            <td><div className="admin-table-actions"><button type="button" className="btn btn-light-primary" onClick={() => startEdit(item)} disabled={item.id === null} title="Edit" aria-label={`Edit ${item.name}`}><EditIcon /></button><button type="button" className="btn btn-light-danger" onClick={() => void deleteItem(item)} disabled={item.id === null || deletingID === item.id} title="Delete" aria-label={`Hapus ${item.name}`}><DeleteIcon /></button></div></td>
          </tr>)}</tbody>
        </table></div></div>
        <div className="card-footer d-flex flex-wrap align-items-center justify-content-between gap-3 py-4"><span className="text-muted fw-semibold fs-7">Showing {firstRow}–{lastRow} of {sortedItems.length} entries</span><div className="d-flex align-items-center gap-2"><button type="button" className="btn btn-sm btn-light" onClick={() => setCurrentPage((page) => Math.max(1, page - 1))} disabled={visiblePage <= 1}>Previous</button><span className="text-muted fw-semibold fs-7">{visiblePage} / {pageCount}</span><button type="button" className="btn btn-sm btn-light" onClick={() => setCurrentPage((page) => Math.min(pageCount, page + 1))} disabled={visiblePage >= pageCount}>Next</button></div></div>
      </section>

      {formOpen && <div className="ref-modal-backdrop" onMouseDown={(event) => { if (event.target === event.currentTarget && !saving) closeForm(); }}>
        <section className="card ref-modal" role="dialog" aria-modal="true" aria-labelledby="company-info-form-title">
          <div className={`card-header min-h-70px form-modal-header ${editingID === null ? "form-modal-header--add" : "form-modal-header--edit"}`}><div className="card-title"><h2 id="company-info-form-title" className="fs-4 fw-bold mb-0">{editingID === null ? "Add Company" : `Edit Company #${editingID}`}</h2></div><button type="button" className="btn btn-light form-modal-close" onClick={closeForm} disabled={saving} aria-label="Tutup modal"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m6 6 12 12M18 6 6 18" stroke="currentColor" strokeWidth="2" strokeLinecap="round" /></svg></button></div>
          <form onSubmit={saveItem}><div className="card-body p-0">
            {error && <div className="alert alert-danger m-4 mb-0" role="alert">{error}</div>}
            <div className="doc-type-form-row"><label className="form-label mb-0" htmlFor="company-info-name">Name:</label><input id="company-info-name" className="form-control form-control-solid doc-type-input" value={name} onChange={(event) => setName(event.target.value)} required autoFocus /></div>
            <div className="doc-type-form-row"><label className="form-label mb-0" htmlFor="company-info-province">Province:</label><select id="company-info-province" className="form-select form-select-solid doc-type-select" value={provinceID} onChange={(event) => setProvinceID(event.target.value)}><option value="">Select Province</option>{provinces.map((province) => <option key={province.id} value={String(province.id)}>{province.name}</option>)}</select></div>
            <div className="doc-type-form-row"><label className="form-label mb-0" htmlFor="company-info-region">Region:</label><select id="company-info-region" className="form-select form-select-solid doc-type-select" value={regionID} onChange={(event) => setRegionID(event.target.value)}><option value="">Select Region</option>{regions.map((region) => <option key={region.id} value={String(region.id)}>{region.name}</option>)}</select></div>
            <div className="doc-type-form-row"><label className="form-label mb-0" htmlFor="company-info-city">City:</label><input id="company-info-city" className="form-control form-control-solid doc-type-input" value={city} onChange={(event) => setCity(event.target.value)} /></div>
            <div className="doc-type-form-row"><label className="form-label mb-0" htmlFor="company-info-address">Address:</label><input id="company-info-address" className="form-control form-control-solid doc-type-input" value={address} onChange={(event) => setAddress(event.target.value)} /></div>
          </div><div className="card-footer d-flex flex-wrap justify-content-end gap-3 py-4"><button type="button" className="btn btn-light" onClick={closeForm} disabled={saving}>Cancel</button><button type="submit" className={`btn ${editingID === null ? "btn-dark" : "btn-warning"}`} disabled={saving}>{saving ? <span className="spinner-border spinner-border-sm me-2" aria-hidden="true" /> : <SaveIcon />}{saving ? "Submitting..." : "Submit"}</button></div></form>
        </section>
      </div>}
    </div>
  );
}
