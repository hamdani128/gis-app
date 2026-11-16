"use client";

import Link from "next/link";
import TableLoadingRows from "../../../components/ui/table-loading-rows";
import { useCallback, useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";

type BusinessProcessOfficer = {
  id: number | null;
  employee_id: number | null;
  employee_name: string;
  process_sequence_id: number | null;
  process_sequence_name: string;
};
type EmployeeOption = { id: number | null; name: string | null; username: string | null };
type SequenceOption = { id: number | null; name: string; sequence: number | null; process_role: string };
type Input = Pick<BusinessProcessOfficer, "employee_id" | "process_sequence_id">;
type APIEnvelope<T> = { message?: string; data?: T };
type SortColumn = "employee_name" | "process_sequence_name";

function uniqueOptionsByID<T extends { id: number | null }>(options: T[]): T[] {
  const seen = new Set<number>();
  return options.filter((option) => {
    if (option === null || option.id === null || !Number.isFinite(option.id) || seen.has(option.id)) return false;
    seen.add(option.id);
    return true;
  });
}

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

export default function BusinessProcessOfficerPage() {
  const [items, setItems] = useState<BusinessProcessOfficer[]>([]);
  const [employees, setEmployees] = useState<EmployeeOption[]>([]);
  const [sequences, setSequences] = useState<SequenceOption[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [deletingID, setDeletingID] = useState<number | null>(null);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [pageSize, setPageSize] = useState(25);
  const [currentPage, setCurrentPage] = useState(1);
  const [sortColumn, setSortColumn] = useState<SortColumn>("employee_name");
  const [sortAscending, setSortAscending] = useState(true);
  const [formOpen, setFormOpen] = useState(false);
  const [editingID, setEditingID] = useState<number | null>(null);
  const [employeeID, setEmployeeID] = useState("");
  const [sequenceID, setSequenceID] = useState("");

  const loadItems = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [officerData, sequenceData, assignmentData] = await Promise.all([
        apiRequest<EmployeeOption[]>("/api/officers"),
        apiRequest<SequenceOption[]>("/api/business-process-sequences"),
        apiRequest<BusinessProcessOfficer[]>("/api/business-process-officers"),
      ]);
      setEmployees(Array.isArray(officerData) ? uniqueOptionsByID(officerData) : []);
      setSequences(Array.isArray(sequenceData) ? uniqueOptionsByID(sequenceData) : []);
      setItems(Array.isArray(assignmentData) ? assignmentData : []);
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
    return items.filter((item) => `${item.employee_name} ${item.process_sequence_name}`.toLowerCase().includes(search));
  }, [items, query]);
  const sortedItems = useMemo(() => [...filteredItems].sort((a, b) => {
    const comparison = a[sortColumn].localeCompare(b[sortColumn], "id", { numeric: true, sensitivity: "base" });
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
    setEditingID(null);
    setEmployeeID("");
    setSequenceID("");
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
  function startEdit(item: BusinessProcessOfficer) {
    if (item.id === null) return;
    setEditingID(item.id);
    setEmployeeID(item.employee_id === null ? "" : String(item.employee_id));
    setSequenceID(item.process_sequence_id === null ? "" : String(item.process_sequence_id));
    setError("");
    setFormOpen(true);
  }
  async function saveItem(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    setError("");
    const wasEditing = editingID !== null;
    const payload: Input = {
      employee_id: employeeID ? Number(employeeID) : null,
      process_sequence_id: sequenceID ? Number(sequenceID) : null,
    };
    try {
      await apiRequest<BusinessProcessOfficer>(wasEditing ? `/api/business-process-officers/${editingID}` : "/api/business-process-officers", {
        method: wasEditing ? "PUT" : "POST",
        body: JSON.stringify(payload),
      });
      closeForm();
      setCurrentPage(1);
      await loadItems();
      const message = wasEditing ? "Business process officer berhasil diperbarui." : "Business process officer berhasil ditambahkan.";
      if (window.Swal) await window.Swal.fire({ title: "Berhasil", text: message, icon: "success", buttonsStyling: false, confirmButtonText: "Oke", customClass: { popup: "rounded-4 shadow-lg", title: "fw-bold text-gray-900", confirmButton: "btn btn-dark" } });
      else window.alert(message);
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : "Data gagal disimpan.");
    } finally {
      setSaving(false);
    }
  }
  async function deleteItem(item: BusinessProcessOfficer) {
    if (item.id === null) return;
    const confirmation = window.Swal
      ? await window.Swal.fire({ title: "Hapus penugasan ini?", text: `${item.employee_name} — ${item.process_sequence_name}`, icon: "warning", showCancelButton: true, confirmButtonText: "Ya, hapus", cancelButtonText: "Batal", reverseButtons: true, buttonsStyling: false, customClass: { popup: "rounded-4 shadow-lg", title: "fw-bold text-gray-900", htmlContainer: "text-muted", confirmButton: "btn btn-danger ms-2", cancelButton: "btn btn-light me-2" } })
      : { isConfirmed: window.confirm(`Hapus penugasan ${item.employee_name} — ${item.process_sequence_name}?`) };
    if (!confirmation.isConfirmed) return;
    setDeletingID(item.id);
    setError("");
    try {
      await apiRequest<null>(`/api/business-process-officers/${item.id}`, { method: "DELETE" });
      await loadItems();
      if (window.Swal) await window.Swal.fire({ title: "Berhasil", text: "Business process officer berhasil dihapus.", icon: "success", buttonsStyling: false, confirmButtonText: "Oke", customClass: { popup: "rounded-4 shadow-lg", title: "fw-bold text-gray-900", confirmButton: "btn btn-dark" } });
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : "Data gagal dihapus.");
    } finally {
      setDeletingID(null);
    }
  }

  const headers: [SortColumn, string][] = [["employee_name", "Employee Name"], ["process_sequence_name", "Process Sequence Name"]];

  return (
    <div className="d-flex flex-column gap-6">
      <nav aria-label="Breadcrumb"><ol className="breadcrumb breadcrumb-separatorless fw-semibold fs-7 mb-0"><li className="breadcrumb-item"><Link href="/admin/dashboard" className="text-muted text-hover-primary">Admin</Link></li><li className="breadcrumb-item"><span className="bullet bg-gray-300 w-5px h-2px" /></li><li className="breadcrumb-item text-muted" aria-current="page">Business Process Officer</li></ol></nav>

      <div className="d-flex flex-wrap align-items-center justify-content-between gap-4">
        <div><h1 className="text-gray-900 fw-bolder fs-2 mb-1">Business Process Officer</h1><p className="text-muted fw-semibold mb-0">Kelola penugasan employee pada process sequence.</p></div>
        <button type="button" className="btn btn-dark" onClick={startCreate}><span className="me-2 fs-3 lh-1">+</span>Add Process Sequence</button>
      </div>

      {!formOpen && error && <div className="alert alert-danger d-flex align-items-center justify-content-between mb-0" role="alert"><span>{error}</span><button type="button" className="btn-close" aria-label="Tutup pesan error" onClick={() => setError("")} /></div>}

      <section className="card border shadow-sm">
        <div className="card-header flex-wrap gap-4 py-4">
          <span className="text-muted fw-semibold fs-7">Total data: {items.length}</span>
          <label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7 me-auto">Search<input className="form-control form-control-sm form-control-solid ref-table-search" type="search" aria-label="Cari employee atau process sequence" placeholder="Employee atau process sequence..." value={query} onChange={(event) => { setQuery(event.target.value); setCurrentPage(1); }} /></label>
          <label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7">Show<select className="form-select form-select-sm form-select-solid w-auto" value={pageSize} onChange={(event) => { setPageSize(Number(event.target.value)); setCurrentPage(1); }} aria-label="Jumlah baris per halaman"><option value={10}>10</option><option value={25}>25</option><option value={50}>50</option><option value={100}>100</option></select>entries</label>
          <button type="button" className="btn btn-icon btn-sm btn-light ms-auto" onClick={() => void loadItems()} disabled={loading} aria-label="Muat ulang daftar"><span className={loading ? "spinner-border spinner-border-sm" : "svg-icon svg-icon-2"} aria-hidden="true">{!loading && <svg viewBox="0 0 24 24" fill="none"><path d="M20 11a8 8 0 0 0-14.9-3M4 4v4h4m-4 5a8 8 0 0 0 14.9 3M20 20v-4h-4" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>}</span></button>
        </div>

        <div className="card-body pt-0">
          <div className="table-responsive"><table className="table admin-data-table align-middle">
            <thead className="bg-dark"><tr className="bg-dark text-white fw-bold text-uppercase fs-8">
              {headers.map(([column, label]) => <th key={column} className="bg-dark text-white" aria-sort={sortColumn === column ? (sortAscending ? "ascending" : "descending") : "none"}><button type="button" className="btn btn-link p-0 text-white fw-bold text-uppercase fs-8 text-decoration-none" onClick={() => changeSort(column)}>{label} <span aria-hidden="true">{sortIndicator(column)}</span></button></th>)}
              <th className="bg-dark text-white text-end">Actions</th>
            </tr></thead>
            <tbody>
              {loading ? <TableLoadingRows columns={3} /> : visibleItems.length === 0 ? <tr><td colSpan={3} className="text-center py-12 text-muted">{query ? "Data tidak ditemukan." : "Belum ada business process officer."}</td></tr> : visibleItems.map((item, index) => <tr key={item.id ?? `null-${index}`}>
                <td className="text-gray-900 fw-bold">{item.employee_name || "—"}</td><td>{item.process_sequence_name || "—"}</td>
                <td><div className="admin-table-actions"><button type="button" className="btn btn-light-primary" onClick={() => startEdit(item)} disabled={item.id === null} title="Edit" aria-label={`Edit penugasan ${item.employee_name}`}><EditIcon /></button><button type="button" className="btn btn-light-danger" onClick={() => void deleteItem(item)} disabled={item.id === null || deletingID === item.id} title="Delete" aria-label={`Hapus penugasan ${item.employee_name}`}><DeleteIcon /></button></div></td>
              </tr>)}
            </tbody>
          </table></div>
        </div>

        <div className="card-footer d-flex flex-wrap align-items-center justify-content-between gap-3 py-4"><span className="text-muted fw-semibold fs-7">Showing {firstRow}–{lastRow} of {sortedItems.length} entries</span><div className="d-flex align-items-center gap-2"><button type="button" className="btn btn-sm btn-light" onClick={() => setCurrentPage((page) => Math.max(1, page - 1))} disabled={visiblePage <= 1}>Previous</button><span className="text-muted fw-semibold fs-7">{visiblePage} / {pageCount}</span><button type="button" className="btn btn-sm btn-light" onClick={() => setCurrentPage((page) => Math.min(pageCount, page + 1))} disabled={visiblePage >= pageCount}>Next</button></div></div>
      </section>

      {formOpen && <div className="ref-modal-backdrop" onMouseDown={(event) => { if (event.target === event.currentTarget && !saving) closeForm(); }}>
        <section className="card ref-modal" role="dialog" aria-modal="true" aria-labelledby="process-officer-form-title">
          <div className={`card-header min-h-70px form-modal-header ${editingID === null ? "form-modal-header--add" : "form-modal-header--edit"}`}><div className="card-title"><h2 id="process-officer-form-title" className="fs-4 fw-bold mb-0">{editingID === null ? "Add Process Sequence" : `Edit Process Sequence #${editingID}`}</h2></div><button type="button" className="btn btn-light form-modal-close" onClick={closeForm} disabled={saving} aria-label="Tutup modal"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m6 6 12 12M18 6 6 18" stroke="currentColor" strokeWidth="2" strokeLinecap="round" /></svg></button></div>
          <form onSubmit={saveItem}>
            <div className="card-body p-0">
              {error && <div className="alert alert-danger m-4 mb-0" role="alert">{error}</div>}
              <div className="doc-type-form-row">
                <label className="form-label mb-0" htmlFor="process-officer-employee">Employee Name:</label>
                <select id="process-officer-employee" className="form-select form-select-solid doc-type-select" value={employeeID} onChange={(event) => setEmployeeID(event.target.value)} required>
                  <option value="">Select Employee Name</option>
                  {employees.map((employee) => <option key={employee.id} value={String(employee.id)}>{employee.name || employee.username || `Employee ${employee.id}`}{employee.username ? ` (${employee.username})` : ""}</option>)}
                </select>
              </div>
              <div className="doc-type-form-row"><label className="form-label mb-0" htmlFor="process-officer-sequence">Process Sequence Name:</label><select id="process-officer-sequence" className="form-select form-select-solid doc-type-select" value={sequenceID} onChange={(event) => setSequenceID(event.target.value)} required><option value="">Select Process Sequence Name</option>{sequences.map((sequence) => <option key={sequence.id} value={String(sequence.id ?? "")}>{sequence.name}{sequence.process_role ? ` — ${sequence.process_role}` : ""}</option>)}</select></div>
            </div>
            <div className="card-footer d-flex flex-wrap justify-content-end gap-3 py-4"><button type="button" className="btn btn-light" onClick={closeForm} disabled={saving}>Cancel</button><button type="submit" className={`btn ${editingID === null ? "btn-dark" : "btn-warning"}`} disabled={saving}>{saving ? <span className="spinner-border spinner-border-sm me-2" aria-hidden="true" /> : <SaveIcon />}{saving ? "Submitting..." : "Submit"}</button></div>
          </form>
        </section>
      </div>}
    </div>
  );
}
