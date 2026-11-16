"use client";

import TableLoadingRows from "../../../components/ui/table-loading-rows";

import { useCallback, useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import Link from "next/link";

type RefValueSetting = {
  id: number;
  domain: string;
  low_value: string;
  meaning: string | null;
  date_created: string | null;
  created_by: string | null;
  date_modified: string | null;
  modified_by: string | null;
};

type RefValueInput = {
  domain: string;
  low_value: string;
  meaning: string | null;
};

type APIEnvelope<T> = {
  message?: string;
  data?: T;
};

type SortColumn = "id" | "domain" | "low_value" | "meaning" | "date_created";

const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");

async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  if (init?.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(`${apiBase}${path}`, {
    ...init,
    credentials: "include",
    headers,
  });
  const result = (await response.json().catch(() => null)) as APIEnvelope<T> | null;

  if (!response.ok) {
    throw new Error(result?.message || "Permintaan gagal. Silakan coba lagi.");
  }

  return result?.data as T;
}

function formatDate(value: string | null) {
  if (!value) return "—";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleDateString("id-ID");
}

export default function RefValueSettingPage() {
  const [items, setItems] = useState<RefValueSetting[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [deletingID, setDeletingID] = useState<number | null>(null);
  const [query, setQuery] = useState("");
  const [pageSize, setPageSize] = useState(10);
  const [currentPage, setCurrentPage] = useState(1);
  const [sortColumn, setSortColumn] = useState<SortColumn>("id");
  const [sortAscending, setSortAscending] = useState(false);
  const [error, setError] = useState("");
  const [formOpen, setFormOpen] = useState(false);
  const [editingID, setEditingID] = useState<number | null>(null);
  const [domain, setDomain] = useState("");
  const [lowValue, setLowValue] = useState("");
  const [meaning, setMeaning] = useState("");

  const loadItems = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const data = await apiRequest<RefValueSetting[]>("/api/ref-value-settings");
      setItems(Array.isArray(data) ? data : []);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : "Data gagal dimuat.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadItems();
  }, [loadItems]);

  const filteredItems = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase();
    if (!normalizedQuery) return items;
    return items.filter((item) =>
      [item.domain, item.low_value, item.meaning ?? "", String(item.id)]
        .some((value) => value.toLowerCase().includes(normalizedQuery)),
    );
  }, [items, query]);

  const sortedItems = useMemo(() => {
    return [...filteredItems].sort((left, right) => {
      const leftValue = left[sortColumn] ?? "";
      const rightValue = right[sortColumn] ?? "";
      const comparison = sortColumn === "id"
        ? Number(leftValue) - Number(rightValue)
        : String(leftValue).localeCompare(String(rightValue), "id", { numeric: true, sensitivity: "base" });
      return sortAscending ? comparison : -comparison;
    });
  }, [filteredItems, sortAscending, sortColumn]);

  const pageCount = Math.max(1, Math.ceil(sortedItems.length / pageSize));
  const visiblePage = Math.min(currentPage, pageCount);
  const firstRow = sortedItems.length === 0 ? 0 : (visiblePage - 1) * pageSize + 1;
  const lastRow = Math.min(visiblePage * pageSize, sortedItems.length);
  const visibleItems = sortedItems.slice(firstRow - 1, lastRow);

  function changeSort(column: SortColumn) {
    setCurrentPage(1);
    if (sortColumn === column) {
      setSortAscending((ascending) => !ascending);
    } else {
      setSortColumn(column);
      setSortAscending(true);
    }
  }

  function sortIndicator(column: SortColumn) {
    if (sortColumn !== column) return "↕";
    return sortAscending ? "↑" : "↓";
  }

  const resetForm = useCallback(() => {
    setDomain("");
    setLowValue("");
    setMeaning("");
    setEditingID(null);
    setFormOpen(false);
  }, []);

  useEffect(() => {
    if (!formOpen) return;
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === "Escape" && !saving) resetForm();
    }
    document.addEventListener("keydown", closeOnEscape);
    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener("keydown", closeOnEscape);
    };
  }, [formOpen, resetForm, saving]);

  function startCreate() {
    setError("");
    resetForm();
    setFormOpen(true);
  }

  function startEdit(item: RefValueSetting) {
    setError("");
    setEditingID(item.id);
    setDomain(item.domain);
    setLowValue(item.low_value);
    setMeaning(item.meaning ?? "");
    setFormOpen(true);
  }

  async function saveItem(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const wasEditing = editingID !== null;
    setSaving(true);
    setError("");

    const payload: RefValueInput = {
      domain: domain.trim(),
      low_value: lowValue.trim(),
      meaning: meaning.trim() || null,
    };

    try {
      await apiRequest<RefValueSetting>(
        editingID === null
          ? "/api/ref-value-settings"
          : `/api/ref-value-settings/${editingID}`,
        {
          method: editingID === null ? "POST" : "PUT",
          body: JSON.stringify(payload),
        },
      );
      resetForm();
      await loadItems();
      const successMessage = wasEditing ? "Data berhasil diperbarui." : "Data berhasil ditambahkan.";
      if (window.Swal) {
        await window.Swal.fire({
          title: "Berhasil",
          text: successMessage,
          icon: "success",
          buttonsStyling: false,
          confirmButtonText: "Oke",
          customClass: {
            popup: "rounded-4 shadow-lg",
            title: "fw-bold text-gray-900",
            confirmButton: "btn btn-dark",
          },
        });
      } else {
        window.alert(successMessage);
      }
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : "Data gagal disimpan.");
    } finally {
      setSaving(false);
    }
  }

  async function deleteItem(item: RefValueSetting) {
    const confirmation = window.Swal
      ? await window.Swal.fire({
          title: "Hapus data ini?",
          text: `Domain ${item.domain} dengan low value ${item.low_value} akan dihapus.`,
          icon: "warning",
          showCancelButton: true,
          confirmButtonText: "Ya, hapus",
          cancelButtonText: "Batal",
          reverseButtons: true,
          buttonsStyling: false,
          customClass: {
            popup: "rounded-4 shadow-lg",
            title: "fw-bold text-gray-900",
            htmlContainer: "text-muted",
            confirmButton: "btn btn-danger ms-2",
            cancelButton: "btn btn-light me-2",
          },
        })
      : { isConfirmed: window.confirm(`Hapus ${item.domain} / ${item.low_value}?`) };

    if (!confirmation.isConfirmed) return;

    setDeletingID(item.id);
    setError("");
    try {
      await apiRequest<null>(`/api/ref-value-settings/${item.id}`, { method: "DELETE" });
      await loadItems();
      if (window.Swal) {
        await window.Swal.fire({
          title: "Berhasil",
          text: "Data berhasil dihapus.",
          icon: "success",
          buttonsStyling: false,
          confirmButtonText: "Oke",
          customClass: {
            popup: "rounded-4 shadow-lg",
            title: "fw-bold text-gray-900",
            confirmButton: "btn btn-dark",
          },
        });
      } else {
        window.alert("Data berhasil dihapus.");
      }
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : "Data gagal dihapus.");
    } finally {
      setDeletingID(null);
    }
  }

  return (
    <div className="d-flex flex-column gap-6">
      <nav aria-label="Breadcrumb" className="mb-n4">
        <ol className="breadcrumb breadcrumb-separatorless fw-semibold fs-7 mb-0">
          <li className="breadcrumb-item">
            <Link href="/admin/dashboard" className="text-muted text-hover-primary">Admin</Link>
          </li>
          <li className="breadcrumb-item">
            <span className="bullet bg-gray-300 w-5px h-2px" />
          </li>
          <li className="breadcrumb-item text-muted" aria-current="page">Ref Value Setting</li>
        </ol>
      </nav>
      <div className="d-flex flex-wrap align-items-center justify-content-between gap-4">
        <div>
          <h1 className="text-gray-900 fw-bolder fs-2 mb-1">Ref Value Setting</h1>
          <p className="text-muted fw-semibold mb-0">Kelola nilai referensi yang digunakan aplikasi.</p>
        </div>
        <button type="button" className="btn btn-dark" onClick={startCreate}>
          <span className="me-2 fs-3 lh-1">+</span>Tambah data
        </button>
      </div>

      {error && (
        <div className="alert alert-danger d-flex align-items-center justify-content-between mb-0" role="alert">
          <span>{error}</span>
          <button type="button" className="btn-close" aria-label="Tutup pesan error" onClick={() => setError("")} />
        </div>
      )}

      {formOpen && (
        <div
          className="ref-modal-backdrop"
          onMouseDown={(event) => {
            if (event.target === event.currentTarget && !saving) resetForm();
          }}
        >
          <section className="card ref-modal" role="dialog" aria-modal="true" aria-labelledby="ref-form-title">
            <div className={`card-header min-h-70px form-modal-header ${editingID === null ? "form-modal-header--add" : "form-modal-header--edit"}`}>
              <div className="card-title d-flex flex-column align-items-start">
                <h2 id="ref-form-title" className="fs-4 fw-bold mb-1">
                  {editingID === null ? "Tambah Ref Value" : `Ubah Ref Value #${editingID}`}
                </h2>
                <span className="text-muted fw-semibold fs-7">Isi informasi domain dan nilai referensi.</span>
              </div>
              <button type="button" className="btn btn-light form-modal-close" onClick={resetForm} disabled={saving} aria-label="Tutup modal">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                  <path d="m6 6 12 12M18 6 6 18" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
                </svg>
              </button>
            </div>
            <form onSubmit={saveItem}>
              <div className="card-body p-6 p-md-8">
                {error && (
                  <div className="alert alert-danger mb-5" role="alert">{error}</div>
                )}
                <div className="row g-5">
                  <div className="col-12 col-md-6">
                    <label className="form-label fw-semibold" htmlFor="ref-domain">Domain</label>
                    <input
                      id="ref-domain"
                      className="form-control form-control-solid"
                      value={domain}
                      onChange={(event) => setDomain(event.target.value)}
                      required
                      autoFocus
                    />
                  </div>
                  <div className="col-12 col-md-6">
                    <label className="form-label fw-semibold" htmlFor="ref-low-value">Low value</label>
                    <input
                      id="ref-low-value"
                      className="form-control form-control-solid"
                      value={lowValue}
                      onChange={(event) => setLowValue(event.target.value)}
                      required
                    />
                  </div>
                  <div className="col-12">
                    <label className="form-label fw-semibold" htmlFor="ref-meaning">Meaning</label>
                    <textarea
                      id="ref-meaning"
                      className="form-control form-control-solid"
                      rows={4}
                      value={meaning}
                      onChange={(event) => setMeaning(event.target.value)}
                      placeholder="Tambahkan keterangan (opsional)"
                    />
                  </div>
                </div>
              </div>
              <div className="card-footer d-flex justify-content-end gap-3 py-4 px-6 px-md-8">
                <button type="button" className="btn btn-light" onClick={resetForm} disabled={saving}>
                  Batal
                </button>
                <button type="submit" className={`btn ${editingID === null ? "btn-dark" : "btn-warning"}`} disabled={saving}>
                  {saving ? "Menyimpan..." : editingID === null ? "Simpan data" : "Simpan perubahan"}
                </button>
              </div>
            </form>
          </section>
        </div>
      )}

      <section className="card border shadow-sm">
        <div className="card-header flex-wrap gap-5 py-5">
          <span className="text-muted fw-semibold fs-7">Total data: {items.length}</span>
          <label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7 me-auto">
            Cari
            <input
              className="form-control form-control-sm form-control-solid ref-table-search"
              type="search"
              aria-label="Cari nilai referensi"
              placeholder="Domain, value, meaning..."
              value={query}
              onChange={(event) => {
                setQuery(event.target.value);
                setCurrentPage(1);
              }}
            />
          </label>
          <div className="card-toolbar d-flex flex-wrap align-items-center gap-4 ms-auto">
            <label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7">
              Tampilkan
              <select
                className="form-select form-select-sm form-select-solid w-auto"
                value={pageSize}
                onChange={(event) => {
                  setPageSize(Number(event.target.value));
                  setCurrentPage(1);
                }}
                aria-label="Jumlah baris per halaman"
              >
                <option value={10}>10</option>
                <option value={25}>25</option>
                <option value={50}>50</option>
              </select>
              baris
            </label>
            <button type="button" className="btn btn-icon btn-sm btn-light" onClick={() => void loadItems()} disabled={loading} aria-label="Muat ulang daftar">
              <span className={loading ? "spinner-border spinner-border-sm" : "svg-icon svg-icon-2"} aria-hidden="true">
                {!loading && (
                  <svg viewBox="0 0 24 24" fill="none">
                    <path d="M20 11a8 8 0 0 0-14.9-3M4 4v4h4m-4 5a8 8 0 0 0 14.9 3M20 20v-4h-4" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
                  </svg>
                )}
              </span>
            </button>
          </div>
        </div>

        <div className="card-body pt-0">
          <div className="table-responsive">
            <table className="table admin-data-table align-middle">
              <thead className="bg-dark">
                <tr className="bg-dark text-white fw-bold text-uppercase fs-8">
                  <th className="min-w-60px bg-dark text-white" aria-sort={sortColumn === "id" ? (sortAscending ? "ascending" : "descending") : "none"}>
                    <button type="button" className="btn btn-link p-0 text-white fw-bold text-uppercase fs-8 text-decoration-none" onClick={() => changeSort("id")}>ID <span aria-hidden="true">{sortIndicator("id")}</span></button>
                  </th>
                  <th className="min-w-180px bg-dark text-white" aria-sort={sortColumn === "domain" ? (sortAscending ? "ascending" : "descending") : "none"}>
                    <button type="button" className="btn btn-link p-0 text-white fw-bold text-uppercase fs-8 text-decoration-none" onClick={() => changeSort("domain")}>Domain <span aria-hidden="true">{sortIndicator("domain")}</span></button>
                  </th>
                  <th className="min-w-150px bg-dark text-white" aria-sort={sortColumn === "low_value" ? (sortAscending ? "ascending" : "descending") : "none"}>
                    <button type="button" className="btn btn-link p-0 text-white fw-bold text-uppercase fs-8 text-decoration-none" onClick={() => changeSort("low_value")}>Low value <span aria-hidden="true">{sortIndicator("low_value")}</span></button>
                  </th>
                  <th className="min-w-200px bg-dark text-white" aria-sort={sortColumn === "meaning" ? (sortAscending ? "ascending" : "descending") : "none"}>
                    <button type="button" className="btn btn-link p-0 text-white fw-bold text-uppercase fs-8 text-decoration-none" onClick={() => changeSort("meaning")}>Meaning <span aria-hidden="true">{sortIndicator("meaning")}</span></button>
                  </th>
                  <th className="min-w-120px bg-dark text-white" aria-sort={sortColumn === "date_created" ? (sortAscending ? "ascending" : "descending") : "none"}>
                    <button type="button" className="btn btn-link p-0 text-white fw-bold text-uppercase fs-8 text-decoration-none" onClick={() => changeSort("date_created")}>Dibuat <span aria-hidden="true">{sortIndicator("date_created")}</span></button>
                  </th>
                  <th className="text-end min-w-140px bg-dark text-white">Aksi</th>
                </tr>
              </thead>
              <tbody>
                {loading ? (
                  <TableLoadingRows columns={6} />
                ) : visibleItems.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="text-center py-12">
                      <div className="text-gray-700 fw-bold mb-1">{query ? "Data tidak ditemukan" : "Belum ada data"}</div>
                      <div className="text-muted fs-7">{query ? "Coba kata kunci lain." : "Tambahkan nilai referensi pertama Anda."}</div>
                    </td>
                  </tr>
                ) : visibleItems.map((item) => (
                  <tr key={item.id}>
                    <td className="text-muted fw-semibold">{item.id}</td>
                    <td className="text-gray-900 fw-bold">{item.domain}</td>
                    <td><span className="badge badge-light-dark fw-bold">{item.low_value}</span></td>
                    <td className="text-gray-700">{item.meaning || <span className="text-muted">—</span>}</td>
                    <td className="text-muted fs-7">{formatDate(item.date_created)}</td>
                    <td>
                      <div className="admin-table-actions">
                        <button type="button" className="btn btn-light-primary" onClick={() => startEdit(item)} title="Ubah" aria-label={`Ubah ${item.domain} / ${item.low_value}`}>
                          <svg className="me-1" width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                            <path d="m14 5 5 5M4 20l4.5-1 10.8-10.8a2.12 2.12 0 0 0-3-3L5.5 16 4 20Z" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
                          </svg>
                        </button>
                        <button type="button" className="btn btn-light-danger" onClick={() => void deleteItem(item)} disabled={deletingID === item.id} title={deletingID === item.id ? "Menghapus..." : "Hapus"} aria-label={`Hapus ${item.domain} / ${item.low_value}`}>
                          <svg className="me-1" width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                            <path d="M4 7h16M10 11v6m4-6v6M6 7l1 14h10l1-14M9 7V4h6v3" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
                          </svg>
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
        <div className="card-footer d-flex flex-wrap align-items-center justify-content-between gap-4 py-4">
          <span className="text-muted fw-semibold fs-7">
            Menampilkan {firstRow}–{lastRow} dari {sortedItems.length} data
            {query && ` (hasil pencarian dari ${items.length} data)`}
          </span>
          <nav aria-label="Navigasi halaman tabel">
            <ul className="pagination pagination-sm mb-0">
              <li className={`page-item ${visiblePage <= 1 ? "disabled" : ""}`}>
                <button type="button" className="page-link" onClick={() => setCurrentPage(1)} disabled={visiblePage <= 1} aria-label="Halaman pertama">«</button>
              </li>
              <li className={`page-item ${visiblePage <= 1 ? "disabled" : ""}`}>
                <button type="button" className="page-link" onClick={() => setCurrentPage((page) => Math.max(1, page - 1))} disabled={visiblePage <= 1} aria-label="Halaman sebelumnya">‹</button>
              </li>
              {Array.from({ length: pageCount }, (_, index) => index + 1).map((page) => (
                <li className={`page-item ${page === visiblePage ? "active" : ""}`} key={page}>
                  <button type="button" className="page-link" onClick={() => setCurrentPage(page)} aria-current={page === visiblePage ? "page" : undefined}>
                    {page}
                  </button>
                </li>
              ))}
              <li className={`page-item ${visiblePage >= pageCount ? "disabled" : ""}`}>
                <button type="button" className="page-link" onClick={() => setCurrentPage((page) => Math.min(pageCount, page + 1))} disabled={visiblePage >= pageCount} aria-label="Halaman berikutnya">›</button>
              </li>
              <li className={`page-item ${visiblePage >= pageCount ? "disabled" : ""}`}>
                <button type="button" className="page-link" onClick={() => setCurrentPage(pageCount)} disabled={visiblePage >= pageCount} aria-label="Halaman terakhir">»</button>
              </li>
            </ul>
          </nav>
        </div>
      </section>
    </div>
  );
}
