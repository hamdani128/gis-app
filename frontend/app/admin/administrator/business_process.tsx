"use client";

import Link from "next/link";
import TableLoadingRows from "../../../components/ui/table-loading-rows";
import { useCallback, useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";

type BusinessProcessRole = {
  id: number | null;
  role_name: string;
  valid_from: string | null;
  valid_until: string | null;
  note: string | null;
  created_by: string | null;
  date_created: string | null;
  modified_by: string | null;
  date_modified: string | null;
};

type BusinessProcessRoleInput = Pick<BusinessProcessRole, "role_name" | "valid_from" | "valid_until" | "note">;
type APIEnvelope<T> = { message?: string; data?: T };
type SortColumn = "id" | "role_name" | "valid_from" | "valid_until" | "note" | "date_created";

const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");

async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  if (init?.body && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  const response = await fetch(`${apiBase}${path}`, { ...init, credentials: "include", headers });
  const result = (await response.json().catch(() => null)) as APIEnvelope<T> | null;
  if (!response.ok) throw new Error(result?.message || "Permintaan gagal. Silakan coba lagi.");
  return result?.data as T;
}

function formatDateTime(value: string | null) {
  if (!value) return "—";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString("id-ID");
}

function toDateTimeInput(value: string | null) {
  return value ? value.replace(" ", "T").slice(0, 16) : "";
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

export default function BusinessProcessPage() {
  const [items, setItems] = useState<BusinessProcessRole[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [deletingID, setDeletingID] = useState<number | null>(null);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [pageSize, setPageSize] = useState(25);
  const [currentPage, setCurrentPage] = useState(1);
  const [sortColumn, setSortColumn] = useState<SortColumn>("role_name");
  const [sortAscending, setSortAscending] = useState(true);
  const [formOpen, setFormOpen] = useState(false);
  const [editingID, setEditingID] = useState<number | null>(null);
  const [roleName, setRoleName] = useState("");
  const [validFrom, setValidFrom] = useState("");
  const [validUntil, setValidUntil] = useState("");
  const [note, setNote] = useState("");

  const loadItems = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const data = await apiRequest<BusinessProcessRole[]>("/api/business-process-roles");
      setItems(Array.isArray(data) ? data : []);
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
    return items.filter((item) => [String(item.id ?? ""), item.role_name, item.valid_from ?? "", item.valid_until ?? "", item.note ?? ""]
      .some((value) => value.toLowerCase().includes(search)));
  }, [items, query]);

  const sortedItems = useMemo(() => [...filteredItems].sort((left, right) => {
    const leftValue = left[sortColumn];
    const rightValue = right[sortColumn];
    const comparison = sortColumn === "id"
      ? Number(leftValue ?? 0) - Number(rightValue ?? 0)
      : sortColumn === "valid_from" || sortColumn === "date_created"
        ? new Date(leftValue ?? 0).getTime() - new Date(rightValue ?? 0).getTime()
        : String(leftValue ?? "").localeCompare(String(rightValue ?? ""), "id", { numeric: true, sensitivity: "base" });
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
    else {
      setSortColumn(column);
      setSortAscending(true);
    }
  }

  function sortIndicator(column: SortColumn) {
    if (sortColumn !== column) return "↕";
    return sortAscending ? "↑" : "↓";
  }

  function clearForm() {
    setEditingID(null);
    setRoleName("");
    setValidFrom("");
    setValidUntil("");
    setNote("");
  }

  function closeForm() {
    setFormOpen(false);
    setError("");
    clearForm();
  }

  function startCreate() {
    clearForm();
    setError("");
    setFormOpen(true);
  }

  function startEdit(item: BusinessProcessRole) {
    if (item.id === null) return;
    setEditingID(item.id);
    setRoleName(item.role_name);
    setValidFrom(toDateTimeInput(item.valid_from));
    setValidUntil(toDateTimeInput(item.valid_until));
    setNote(item.note ?? "");
    setError("");
    setFormOpen(true);
  }

  async function saveItem(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    setError("");
    const wasEditing = editingID !== null;
    const payload: BusinessProcessRoleInput = {
      role_name: roleName.trim(),
      valid_from: validFrom || null,
      valid_until: validUntil.trim() || null,
      note: note.trim() || null,
    };
    try {
      await apiRequest<BusinessProcessRole>(wasEditing ? `/api/business-process-roles/${editingID}` : "/api/business-process-roles", {
        method: wasEditing ? "PUT" : "POST",
        body: JSON.stringify(payload),
      });
      closeForm();
      setCurrentPage(1);
      await loadItems();
      const message = wasEditing ? "Business process role berhasil diperbarui." : "Business process role berhasil ditambahkan.";
      if (window.Swal) {
        await window.Swal.fire({ title: "Berhasil", text: message, icon: "success", buttonsStyling: false, confirmButtonText: "Oke", customClass: { popup: "rounded-4 shadow-lg", title: "fw-bold text-gray-900", confirmButton: "btn btn-dark" } });
      } else window.alert(message);
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : "Data gagal disimpan.");
    } finally {
      setSaving(false);
    }
  }

  async function deleteItem(item: BusinessProcessRole) {
    if (item.id === null) return;
    const confirmation = window.Swal
      ? await window.Swal.fire({ title: "Hapus role ini?", text: `${item.role_name} akan dihapus.`, icon: "warning", showCancelButton: true, confirmButtonText: "Ya, hapus", cancelButtonText: "Batal", reverseButtons: true, buttonsStyling: false, customClass: { popup: "rounded-4 shadow-lg", title: "fw-bold text-gray-900", htmlContainer: "text-muted", confirmButton: "btn btn-danger ms-2", cancelButton: "btn btn-light me-2" } })
      : { isConfirmed: window.confirm(`Hapus ${item.role_name}?`) };
    if (!confirmation.isConfirmed) return;
    setDeletingID(item.id);
    setError("");
    try {
      await apiRequest<null>(`/api/business-process-roles/${item.id}`, { method: "DELETE" });
      await loadItems();
      if (window.Swal) {
        await window.Swal.fire({ title: "Berhasil", text: "Business process role berhasil dihapus.", icon: "success", buttonsStyling: false, confirmButtonText: "Oke", customClass: { popup: "rounded-4 shadow-lg", title: "fw-bold text-gray-900", confirmButton: "btn btn-dark" } });
      }
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : "Data gagal dihapus.");
    } finally {
      setDeletingID(null);
    }
  }

  const headers: [SortColumn, string][] = [["id", "ID"], ["role_name", "Role name"], ["valid_from", "Valid from"], ["valid_until", "Valid until"], ["note", "Note"], ["date_created", "Created"]];

  return (
    <div className="d-flex flex-column gap-6">
      <nav aria-label="Breadcrumb">
        <ol className="breadcrumb breadcrumb-separatorless fw-semibold fs-7 mb-0">
          <li className="breadcrumb-item"><Link href="/admin/dashboard" className="text-muted text-hover-primary">Admin</Link></li>
          <li className="breadcrumb-item"><span className="bullet bg-gray-300 w-5px h-2px" /></li>
          <li className="breadcrumb-item text-muted" aria-current="page">Business Process</li>
        </ol>
      </nav>

      <div className="d-flex flex-wrap align-items-center justify-content-between gap-4">
        <div><h1 className="text-gray-900 fw-bolder fs-2 mb-1">Business Process</h1><p className="text-muted fw-semibold mb-0">Kelola role business process.</p></div>
        <button type="button" className="btn btn-dark" onClick={startCreate}><span className="me-2 fs-3 lh-1">+</span>Add Role</button>
      </div>

      {!formOpen && error && <div className="alert alert-danger d-flex align-items-center justify-content-between mb-0" role="alert"><span>{error}</span><button type="button" className="btn-close" aria-label="Tutup pesan error" onClick={() => setError("")} /></div>}

      <section className="card border shadow-sm">
        <div className="card-header flex-wrap gap-4 py-4">
          <span className="text-muted fw-semibold fs-7">Total data: {items.length}</span>
          <label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7 me-auto">Search<input className="form-control form-control-sm form-control-solid ref-table-search" type="search" aria-label="Cari business process roles" placeholder="Role name, valid date, note..." value={query} onChange={(event) => { setQuery(event.target.value); setCurrentPage(1); }} /></label>
          <label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7">Show<select className="form-select form-select-sm form-select-solid w-auto" value={pageSize} onChange={(event) => { setPageSize(Number(event.target.value)); setCurrentPage(1); }} aria-label="Jumlah baris per halaman"><option value={10}>10</option><option value={25}>25</option><option value={50}>50</option><option value={100}>100</option></select>entries</label>
          <button type="button" className="btn btn-icon btn-sm btn-light ms-auto" onClick={() => void loadItems()} disabled={loading} aria-label="Muat ulang daftar"><span className={loading ? "spinner-border spinner-border-sm" : "svg-icon svg-icon-2"} aria-hidden="true">{!loading && <svg viewBox="0 0 24 24" fill="none"><path d="M20 11a8 8 0 0 0-14.9-3M4 4v4h4m-4 5a8 8 0 0 0 14.9 3M20 20v-4h-4" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>}</span></button>
        </div>

        <div className="card-body pt-0">
          <div className="table-responsive">
            <table className="table admin-data-table align-middle">
              <thead className="bg-dark"><tr className="bg-dark text-white fw-bold text-uppercase fs-8">
                {headers.map(([column, label]) => <th key={column} className="bg-dark text-white" aria-sort={sortColumn === column ? (sortAscending ? "ascending" : "descending") : "none"}><button type="button" className="btn btn-link p-0 text-white fw-bold text-uppercase fs-8 text-decoration-none" onClick={() => changeSort(column)}>{label} <span aria-hidden="true">{sortIndicator(column)}</span></button></th>)}
                <th className="bg-dark text-white text-end">Actions</th>
              </tr></thead>
              <tbody>
                {loading ? <TableLoadingRows columns={7} /> : visibleItems.length === 0 ? <tr><td colSpan={7} className="text-center py-12 text-muted">{query ? "Data tidak ditemukan." : "Belum ada business process role."}</td></tr> : visibleItems.map((item, index) => <tr key={item.id ?? `null-${index}`}>
                  <td className="text-muted fw-semibold">{item.id ?? "—"}</td><td className="text-gray-900 fw-bold">{item.role_name}</td><td>{formatDateTime(item.valid_from)}</td><td>{item.valid_until || "—"}</td><td>{item.note || "—"}</td><td className="text-muted fs-7">{formatDateTime(item.date_created)}</td>
                  <td><div className="admin-table-actions"><button type="button" className="btn btn-light-primary" onClick={() => startEdit(item)} disabled={item.id === null} title="Edit" aria-label={`Edit ${item.role_name}`}><EditIcon /></button><button type="button" className="btn btn-light-danger" onClick={() => void deleteItem(item)} disabled={item.id === null || deletingID === item.id} title="Delete" aria-label={`Delete ${item.role_name}`}><DeleteIcon /></button></div></td>
                </tr>)}
              </tbody>
            </table>
          </div>
        </div>

        <div className="card-footer d-flex flex-wrap align-items-center justify-content-between gap-3 py-4">
          <span className="text-muted fw-semibold fs-7">Showing {firstRow}–{lastRow} of {sortedItems.length} entries</span>
          <div className="d-flex align-items-center gap-2"><button type="button" className="btn btn-sm btn-light" onClick={() => setCurrentPage((page) => Math.max(1, page - 1))} disabled={visiblePage <= 1}>Previous</button><span className="text-muted fw-semibold fs-7">{visiblePage} / {pageCount}</span><button type="button" className="btn btn-sm btn-light" onClick={() => setCurrentPage((page) => Math.min(pageCount, page + 1))} disabled={visiblePage >= pageCount}>Next</button></div>
        </div>
      </section>

      {formOpen && <div className="ref-modal-backdrop" onMouseDown={(event) => { if (event.target === event.currentTarget && !saving) closeForm(); }}>
        <section className="card ref-modal" role="dialog" aria-modal="true" aria-labelledby="process-role-form-title">
          <div className={`card-header min-h-70px form-modal-header ${editingID === null ? "form-modal-header--add" : "form-modal-header--edit"}`}>
            <div className="card-title"><h2 id="process-role-form-title" className="fs-4 fw-bold mb-0">{editingID === null ? "Add Business Process Role" : `Edit Business Process Role #${editingID}`}</h2></div>
            <button type="button" className="btn btn-light form-modal-close" onClick={closeForm} disabled={saving} aria-label="Tutup modal"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m6 6 12 12M18 6 6 18" stroke="currentColor" strokeWidth="2" strokeLinecap="round" /></svg></button>
          </div>
          <form onSubmit={saveItem}>
            <div className="card-body p-0">
              {error && <div className="alert alert-danger m-4 mb-0" role="alert">{error}</div>}
              <div className="doc-type-form-row"><label className="form-label mb-0" htmlFor="process-role-name">Role name:</label><input id="process-role-name" className="form-control form-control-solid doc-type-input" value={roleName} onChange={(event) => setRoleName(event.target.value)} required autoFocus /></div>
              <div className="doc-type-form-row"><label className="form-label mb-0" htmlFor="process-role-valid-from">Valid from:</label><input id="process-role-valid-from" className="form-control form-control-solid doc-type-input" type="datetime-local" value={validFrom} onChange={(event) => setValidFrom(event.target.value)} /></div>
              <div className="doc-type-form-row"><label className="form-label mb-0" htmlFor="process-role-valid-until">Valid until:</label><input id="process-role-valid-until" className="form-control form-control-solid doc-type-input" type="datetime-local" value={validUntil} onChange={(event) => setValidUntil(event.target.value)} /></div>
              <div className="doc-type-form-row"><label className="form-label mb-0" htmlFor="process-role-note">Note:</label><textarea id="process-role-note" className="form-control form-control-solid doc-type-input" rows={3} value={note} onChange={(event) => setNote(event.target.value)} /></div>
            </div>
            <div className="card-footer d-flex flex-wrap justify-content-end gap-3 py-4">
              <button type="button" className="btn btn-light" onClick={closeForm} disabled={saving}>Cancel</button>
              <button type="submit" className={`btn ${editingID === null ? "btn-dark" : "btn-warning"}`} disabled={saving}>{saving ? <span className="spinner-border spinner-border-sm me-2" aria-hidden="true" /> : <SaveIcon />}{saving ? "Submitting..." : "Submit"}</button>
            </div>
          </form>
        </section>
      </div>}
    </div>
  );
}
