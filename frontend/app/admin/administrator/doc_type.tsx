"use client";

import Link from "next/link";
import TableLoadingRows from "../../../components/ui/table-loading-rows";
import { useCallback, useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";

type DocType = {
  id: number;
  type_name: string;
  mandatory: number;
  file_type: string;
  seq_related: number | null;
  type_doc_group: string | null;
  created_by: string | null;
  modified_by: string | null;
  date_modified: string | null;
  date_created: string | null;
};

type DocTypeInput = Pick<DocType, "type_name" | "mandatory" | "file_type" | "seq_related" | "type_doc_group">;
type DocTypeSortColumn = "id" | "type_name" | "mandatory" | "file_type" | "seq_related" | "type_doc_group" | "date_created";
type APIEnvelope<T> = { message?: string; data?: T };

const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");

async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  if (init?.body && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  const response = await fetch(`${apiBase}${path}`, { ...init, credentials: "include", headers });
  const result = (await response.json().catch(() => null)) as APIEnvelope<T> | null;
  if (!response.ok) throw new Error(result?.message || "Permintaan gagal. Silakan coba lagi.");
  return result?.data as T;
}

function formatDate(value: string | null) {
  if (!value) return "—";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleDateString("id-ID");
}

function EditIcon() {
  return <svg className="me-1" width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m14 5 5 5M4 20l4.5-1 10.8-10.8a2.12 2.12 0 0 0-3-3L5.5 16 4 20Z" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>;
}

function DeleteIcon() {
  return <svg className="me-1" width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M4 7h16M10 11v6m4-6v6M6 7l1 14h10l1-14M9 7V4h6v3" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>;
}

export default function DocTypePage() {
  const [items, setItems] = useState<DocType[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [deletingID, setDeletingID] = useState<number | null>(null);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [pageSize, setPageSize] = useState(25);
  const [currentPage, setCurrentPage] = useState(1);
  const [sortColumn, setSortColumn] = useState<DocTypeSortColumn>("type_name");
  const [sortAscending, setSortAscending] = useState(true);
  const [formOpen, setFormOpen] = useState(false);
  const [editingID, setEditingID] = useState<number | null>(null);
  const [typeName, setTypeName] = useState("");
  const [mandatory, setMandatory] = useState(1);
  const [fileType, setFileType] = useState("");
  const [seqRelated, setSeqRelated] = useState("");
  const [typeDocGroup, setTypeDocGroup] = useState("");

  const loadItems = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const data = await apiRequest<DocType[]>("/api/doc-types");
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
      if (event.key === "Escape" && !saving) {
        setFormOpen(false);
        setTypeName("");
        setMandatory(1);
        setFileType("");
        setSeqRelated("");
        setTypeDocGroup("");
        setEditingID(null);
        setError("");
      }
    }
    document.addEventListener("keydown", closeOnEscape);
    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener("keydown", closeOnEscape);
    };
  }, [formOpen, saving]);

  const filteredItems = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase();
    if (!normalizedQuery) return items;
    return items.filter((item) => [
      item.type_name,
      item.file_type,
      item.type_doc_group ?? "",
      String(item.id),
      item.mandatory ? "active" : "inactive",
    ].some((value) => value.toLowerCase().includes(normalizedQuery)));
  }, [items, query]);

  const sortedItems = useMemo(() => [...filteredItems].sort((left, right) => {
    const leftValue = left[sortColumn];
    const rightValue = right[sortColumn];
    const comparison = sortColumn === "id" || sortColumn === "mandatory" || sortColumn === "seq_related"
      ? Number(leftValue ?? 0) - Number(rightValue ?? 0)
      : sortColumn === "date_created"
        ? new Date(leftValue ?? 0).getTime() - new Date(rightValue ?? 0).getTime()
        : String(leftValue ?? "").localeCompare(String(rightValue ?? ""), "id", { numeric: true, sensitivity: "base" });
    return sortAscending ? comparison : -comparison;
  }), [filteredItems, sortAscending, sortColumn]);

  const pageCount = Math.max(1, Math.ceil(sortedItems.length / pageSize));
  const visiblePage = Math.min(currentPage, pageCount);
  const firstRow = sortedItems.length === 0 ? 0 : (visiblePage - 1) * pageSize + 1;
  const lastRow = Math.min(visiblePage * pageSize, sortedItems.length);
  const visibleItems = sortedItems.slice(firstRow - 1, lastRow);

  function changeSort(column: DocTypeSortColumn) {
    setCurrentPage(1);
    if (sortColumn === column) setSortAscending((ascending) => !ascending);
    else {
      setSortColumn(column);
      setSortAscending(true);
    }
  }

  function sortIndicator(column: DocTypeSortColumn) {
    if (sortColumn !== column) return "↕";
    return sortAscending ? "↑" : "↓";
  }

  const fileTypeOptions = useMemo(() => Array.from(new Set(items.map((item) => item.file_type).filter(Boolean))).sort(), [items]);
  const groupOptions = useMemo(() => Array.from(new Set(items.map((item) => item.type_doc_group).filter((value): value is string => Boolean(value)))).sort(), [items]);

  function clearForm() {
    setTypeName("");
    setMandatory(1);
    setFileType(fileTypeOptions[0] ?? "");
    setSeqRelated("");
    setTypeDocGroup("");
    setEditingID(null);
  }

  function startCreate() {
    setError("");
    clearForm();
    setFormOpen(true);
  }

  function startEdit(item: DocType) {
    setError("");
    setEditingID(item.id);
    setTypeName(item.type_name);
    setMandatory(item.mandatory ? 1 : 0);
    setFileType(item.file_type);
    setSeqRelated(item.seq_related === null ? "" : String(item.seq_related));
    setTypeDocGroup(item.type_doc_group ?? "");
    setFormOpen(true);
  }

  async function saveItem(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const wasEditing = editingID !== null;
    setSaving(true);
    setError("");
    const payload: DocTypeInput = {
      type_name: typeName.trim(),
      mandatory,
      file_type: fileType,
      seq_related: seqRelated.trim() ? Number(seqRelated) : null,
      type_doc_group: typeDocGroup || null,
    };

    try {
      await apiRequest<DocType>(
        editingID === null ? "/api/doc-types" : `/api/doc-types/${editingID}`,
        { method: editingID === null ? "POST" : "PUT", body: JSON.stringify(payload) },
      );
      setFormOpen(false);
      clearForm();
      await loadItems();
      if (window.Swal) {
        await window.Swal.fire({
          title: "Berhasil",
          text: wasEditing ? "Doc type berhasil diperbarui." : "Doc type berhasil ditambahkan.",
          icon: "success",
          buttonsStyling: false,
          confirmButtonText: "Oke",
          customClass: { popup: "rounded-4 shadow-lg", title: "fw-bold text-gray-900", confirmButton: "btn btn-dark" },
        });
      } else {
        window.alert(wasEditing ? "Doc type berhasil diperbarui." : "Doc type berhasil ditambahkan.");
      }
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : "Data gagal disimpan.");
    } finally {
      setSaving(false);
    }
  }

  async function deleteItem(item: DocType) {
    const confirmation = window.Swal
      ? await window.Swal.fire({
          title: "Hapus doc type ini?",
          text: `${item.type_name} akan dihapus.`,
          icon: "warning",
          showCancelButton: true,
          confirmButtonText: "Ya, hapus",
          cancelButtonText: "Batal",
          reverseButtons: true,
          buttonsStyling: false,
          customClass: { popup: "rounded-4 shadow-lg", title: "fw-bold text-gray-900", htmlContainer: "text-muted", confirmButton: "btn btn-danger ms-2", cancelButton: "btn btn-light me-2" },
        })
      : { isConfirmed: window.confirm(`Hapus ${item.type_name}?`) };
    if (!confirmation.isConfirmed) return;

    setDeletingID(item.id);
    setError("");
    try {
      await apiRequest<null>(`/api/doc-types/${item.id}`, { method: "DELETE" });
      await loadItems();
      if (window.Swal) {
        await window.Swal.fire({ title: "Berhasil", text: "Doc type berhasil dihapus.", icon: "success", buttonsStyling: false, confirmButtonText: "Oke", customClass: { popup: "rounded-4 shadow-lg", title: "fw-bold text-gray-900", confirmButton: "btn btn-dark" } });
      } else {
        window.alert("Doc type berhasil dihapus.");
      }
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : "Data gagal dihapus.");
    } finally {
      setDeletingID(null);
    }
  }

  return (
    <div className="d-flex flex-column gap-6">
      <nav aria-label="Breadcrumb">
        <ol className="breadcrumb breadcrumb-separatorless fw-semibold fs-7 mb-0">
          <li className="breadcrumb-item"><Link href="/admin/dashboard" className="text-muted text-hover-primary">Admin</Link></li>
          <li className="breadcrumb-item"><span className="bullet bg-gray-300 w-5px h-2px" /></li>
          <li className="breadcrumb-item text-muted" aria-current="page">Doc Type Setting</li>
        </ol>
      </nav>

      <div className="d-flex flex-wrap align-items-center justify-content-between gap-4">
        <div>
          <h1 className="text-gray-900 fw-bolder fs-2 mb-1">Doc Type Setting</h1>
          <p className="text-muted fw-semibold mb-0">Kelola tipe dokumen yang didukung aplikasi.</p>
        </div>
        {!formOpen && <button type="button" className="btn btn-dark" onClick={startCreate}><span className="me-2 fs-3 lh-1">+</span>Add Doc Type</button>}
      </div>

      {!formOpen && error && <div className="alert alert-danger d-flex align-items-center justify-content-between mb-0" role="alert"><span>{error}</span><button type="button" className="btn-close" aria-label="Tutup pesan error" onClick={() => setError("")} /></div>}

      {formOpen && (
        <div className="ref-modal-backdrop" onMouseDown={(event) => {
          if (event.target === event.currentTarget && !saving) {
            setFormOpen(false);
            clearForm();
            setError("");
          }
        }}>
          <section className="card ref-modal" role="dialog" aria-modal="true" aria-labelledby="doc-type-form-title">
            <div className={`card-header min-h-70px form-modal-header ${editingID === null ? "form-modal-header--add" : "form-modal-header--edit"}`}>
              <div className="card-title"><h2 id="doc-type-form-title" className="fs-4 fw-bold mb-0">{editingID === null ? "Add Doc Type" : `Edit Doc Type #${editingID}`}</h2></div>
              <button type="button" className="btn btn-light form-modal-close" onClick={() => { setFormOpen(false); clearForm(); setError(""); }} disabled={saving} aria-label="Tutup modal"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m6 6 12 12M18 6 6 18" stroke="currentColor" strokeWidth="2" strokeLinecap="round" /></svg></button>
            </div>
          <form onSubmit={saveItem}>
            <div className="card-body p-0">
              {error && <div className="alert alert-danger m-4 mb-0" role="alert">{error}</div>}
              <div className="doc-type-form-row">
                <label className="form-label mb-0" htmlFor="doc-type-name">Type name:</label>
                <input id="doc-type-name" className="form-control form-control-solid doc-type-input" value={typeName} onChange={(event) => setTypeName(event.target.value)} required autoFocus />
              </div>
              <fieldset className="doc-type-form-row">
                <legend className="form-label mb-0">Mandatory:</legend>
                <div className="d-flex align-items-center gap-4">
                  <label className="d-flex align-items-center gap-2 mb-0"><input className="form-check-input" type="radio" name="doc-type-mandatory" value="1" checked={mandatory === 1} onChange={() => setMandatory(1)} />Active</label>
                  <label className="d-flex align-items-center gap-2 mb-0"><input className="form-check-input" type="radio" name="doc-type-mandatory" value="0" checked={mandatory === 0} onChange={() => setMandatory(0)} />Inactive</label>
                </div>
              </fieldset>
              <div className="doc-type-form-row">
                <label className="form-label mb-0" htmlFor="doc-type-file">File type:</label>
                <select id="doc-type-file" className="form-select form-select-solid doc-type-select" value={fileType} onChange={(event) => setFileType(event.target.value)} required>
                  <option value="" disabled>Select File type</option>
                  {fileTypeOptions.map((option) => <option key={option} value={option}>{option}</option>)}
                </select>
              </div>
              <div className="doc-type-form-row">
                <label className="form-label mb-0" htmlFor="doc-type-seq">Seq related:</label>
                <input id="doc-type-seq" className="form-control form-control-solid doc-type-input" type="number" min="0" step="1" value={seqRelated} onChange={(event) => setSeqRelated(event.target.value)} />
              </div>
              <div className="doc-type-form-row">
                <label className="form-label mb-0" htmlFor="doc-type-group">Type doc group:</label>
                <select id="doc-type-group" className="form-select form-select-solid doc-type-select" value={typeDocGroup} onChange={(event) => setTypeDocGroup(event.target.value)}>
                  <option value="">Select Type doc group</option>
                  {groupOptions.map((option) => <option key={option} value={option}>{option}</option>)}
                </select>
              </div>
            </div>
            <div className="card-footer d-flex flex-wrap justify-content-end gap-3 py-4">
              <button type="button" className="btn btn-light" onClick={() => { setFormOpen(false); clearForm(); setError(""); }} disabled={saving}>Cancel</button>
              <button type="submit" className={`btn ${editingID === null ? "btn-dark" : "btn-warning"}`} disabled={saving}>
                {saving ? <span className="spinner-border spinner-border-sm me-2" aria-hidden="true" /> : <svg className="me-2" width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M5 4h12l3 3v13H4V4h1Zm3 0v6h8V4M8 20v-6h8v6" stroke="currentColor" strokeWidth="1.8" strokeLinejoin="round" /></svg>}
                {saving ? "Submitting..." : "Submit"}
              </button>
            </div>
          </form>
          </section>
        </div>
      )}

      <section className="card border shadow-sm">
            <div className="card-header flex-wrap gap-5 py-5">
            <span className="text-muted fw-semibold fs-7">Total data: {items.length}</span>
            <label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7 me-auto">Cari<input className="form-control form-control-sm form-control-solid ref-table-search" type="search" aria-label="Cari tipe dokumen" placeholder="Type name, file type, group..." value={query} onChange={(event) => { setQuery(event.target.value); setCurrentPage(1); }} /></label>
            <label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7">Tampilkan<select className="form-select form-select-sm form-select-solid w-auto" value={pageSize} onChange={(event) => { setPageSize(Number(event.target.value)); setCurrentPage(1); }} aria-label="Jumlah baris per halaman"><option value={10}>10</option><option value={25}>25</option><option value={50}>50</option><option value={100}>100</option></select>baris</label>
            <button type="button" className="btn btn-icon btn-sm btn-light ms-auto" onClick={() => void loadItems()} disabled={loading} aria-label="Muat ulang daftar"><span className={loading ? "spinner-border spinner-border-sm" : "svg-icon svg-icon-2"} aria-hidden="true">{!loading && <svg viewBox="0 0 24 24" fill="none"><path d="M20 11a8 8 0 0 0-14.9-3M4 4v4h4m-4 5a8 8 0 0 0 14.9 3M20 20v-4h-4" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>}</span></button>
          </div>
          <div className="card-body pt-0">
            <div className="table-responsive">
              <table className="table admin-data-table align-middle">
                <thead className="bg-dark"><tr className="bg-dark text-white fw-bold text-uppercase fs-8">
                  {([
                    ["id", "ID"], ["type_name", "Type name"], ["mandatory", "Mandatory"], ["file_type", "File type"],
                    ["seq_related", "Seq related"], ["type_doc_group", "Type doc group"], ["date_created", "Created"],
                  ] as [DocTypeSortColumn, string][]).map(([column, label]) => (
                    <th key={column} className="bg-dark text-white" aria-sort={sortColumn === column ? (sortAscending ? "ascending" : "descending") : "none"}>
                      <button type="button" className="btn btn-link p-0 text-white fw-bold text-uppercase fs-8 text-decoration-none" onClick={() => changeSort(column)}>
                        {label} <span aria-hidden="true">{sortIndicator(column)}</span>
                      </button>
                    </th>
                  ))}
                  <th className="bg-dark text-white text-end">Actions</th>
                </tr></thead>
                <tbody>
                  {loading ? <TableLoadingRows columns={8} />
                    : filteredItems.length === 0 ? <tr><td colSpan={8} className="text-center py-12 text-muted">{query ? "Data tidak ditemukan." : "Belum ada doc type."}</td></tr>
                      : visibleItems.map((item) => <tr key={item.id}>
                        <td className="text-muted fw-semibold">{item.id}</td>
                        <td className="text-gray-900 fw-bold">{item.type_name}</td>
                        <td><span className={`badge ${item.mandatory ? "badge-light-success" : "badge-light-secondary"}`}>{item.mandatory ? "Active" : "Inactive"}</span></td>
                        <td>{item.file_type}</td><td>{item.seq_related ?? "—"}</td><td>{item.type_doc_group || <span className="text-muted">—</span>}</td><td className="text-muted fs-7">{formatDate(item.date_created)}</td>
                        <td><div className="admin-table-actions"><button type="button" className="btn btn-light-primary" onClick={() => startEdit(item)} title="Edit" aria-label={`Edit ${item.type_name}`}><EditIcon /></button><button type="button" className="btn btn-light-danger" onClick={() => void deleteItem(item)} disabled={deletingID === item.id} title={deletingID === item.id ? "Deleting..." : "Delete"} aria-label={`Delete ${item.type_name}`}><DeleteIcon /></button></div></td>
                      </tr>)}
                </tbody>
              </table>
            </div>
          </div>
          <div className="card-footer d-flex flex-wrap align-items-center justify-content-between gap-3 py-4">
            <span className="text-muted fw-semibold fs-7">Menampilkan {firstRow}–{lastRow} dari {sortedItems.length} data</span>
            <div className="d-flex align-items-center gap-2">
              <button type="button" className="btn btn-sm btn-light" onClick={() => setCurrentPage((page) => Math.max(1, page - 1))} disabled={visiblePage <= 1}>Sebelumnya</button>
              <span className="text-muted fw-semibold fs-7">Halaman {visiblePage} / {pageCount}</span>
              <button type="button" className="btn btn-sm btn-light" onClick={() => setCurrentPage((page) => Math.min(pageCount, page + 1))} disabled={visiblePage >= pageCount}>Berikutnya</button>
            </div>
          </div>
      </section>
    </div>
  );
}
