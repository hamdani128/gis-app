"use client";

import Link from "next/link";
import TableLoadingRows from "../../../components/ui/table-loading-rows";
import { useCallback, useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";

type Officer = {
  id: number;
  name: string;
  username: string;
  company_id: number | null;
  company: string;
  potential_land_id: number | null;
  estate: string;
  dept_id: number | null;
  department: string;
  email: string;
  note: string;
  date_created: string | null;
};

type EmployeeCompany = {
  id: number;
  employee_id: number | null;
  employee_name: string;
  username: string;
  company_id: number | null;
  company_name: string;
  function: string;
};

type OfficerOptions = {
  companies: { id: number; name: string }[];
  lands: { id: number; potential_name: string; estate_name: string }[];
  departments: { id: number; meaning: string }[];
};

type OfficerPrivilege = {
  module_id: number;
  module_name: string;
  module_code: string;
  module_file: string;
  p_add: number;
  p_del: number;
  p_view: number;
  p_edit: number;
};

type OfficerLog = {
  id: number | null;
  type: string;
  object: string;
  log1: string;
  log2: string;
  log3: string;
  date_created: string | null;
  created_by: string;
  username: string;
  log_code: number | null;
  ip_remote: string;
};

type APIEnvelope<T> = { message?: string; data?: T };
type Tab = "employees" | "companies";
type FormKind = "employee" | "company" | "password" | "privileges";
type OfficerSortColumn = "name" | "username" | "company" | "estate" | "department" | "email" | "note" | "employee_name" | "company_name" | "function";

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

function formatDateTime(value: string | null) {
  if (!value) return "—";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString("id-ID");
}

function EditIcon() {
  return <svg className="me-1" width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m14 5 5 5M4 20l4.5-1 10.8-10.8a2.12 2.12 0 0 0-3-3L5.5 16 4 20Z" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>;
}

function DeleteIcon() {
  return <svg className="me-1" width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M4 7h16M10 11v6m4-6v6M6 7l1 14h10l1-14M9 7V4h6v3" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>;
}

function PasswordIcon() {
  return <svg className="me-1" width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><rect x="4" y="10" width="16" height="11" rx="2" stroke="currentColor" strokeWidth="1.8" /><path d="M8 10V7a4 4 0 1 1 8 0v3m-4 5v2" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" /></svg>;
}

function PrivilegeIcon() {
  return <svg className="me-1" width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M12 3 19 6v5c0 4.7-3 8-7 10-4-2-7-5.3-7-10V6l7-3Z" stroke="currentColor" strokeWidth="1.8" strokeLinejoin="round" /><path d="m9 12 2 2 4-4" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>;
}

function CompanyIcon() {
  return <svg className="me-1" width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M4 20V5a1 1 0 0 1 1-1h9v16M14 9h5a1 1 0 0 1 1 1v10M8 8h2m-2 4h2m-2 4h2m7-3h1m-1 3h1M3 20h18" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>;
}

function LogIcon() {
  return <svg className="me-1" width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M5 4h14a1 1 0 0 1 1 1v14H4V5a1 1 0 0 1 1-1Zm3 4h8m-8 4h8m-8 4h5" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>;
}

export default function OfficerPage() {
  const [tab, setTab] = useState<Tab>("employees");
  const [employees, setEmployees] = useState<Officer[]>([]);
  const [companyRows, setCompanyRows] = useState<EmployeeCompany[]>([]);
  const [options, setOptions] = useState<OfficerOptions>({ companies: [], lands: [], departments: [] });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [deletingID, setDeletingID] = useState<number | null>(null);
  const [query, setQuery] = useState("");
  const [pageSize, setPageSize] = useState(25);
  const [currentPage, setCurrentPage] = useState(1);
  const [sortColumn, setSortColumn] = useState<OfficerSortColumn>("name");
  const [sortAscending, setSortAscending] = useState(true);
  const [error, setError] = useState("");
  const [formOpen, setFormOpen] = useState(false);
  const [formKind, setFormKind] = useState<FormKind>("employee");
  const [editingID, setEditingID] = useState<number | null>(null);
  const [name, setName] = useState("");
  const [username, setUsername] = useState("");
  const [companyID, setCompanyID] = useState("");
  const [landID, setLandID] = useState("");
  const [deptID, setDeptID] = useState("");
  const [email, setEmail] = useState("");
  const [note, setNote] = useState("");
  const [relationEmployeeID, setRelationEmployeeID] = useState("");
  const [relationCompanyID, setRelationCompanyID] = useState("");
  const [relationFunction, setRelationFunction] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [privileges, setPrivileges] = useState<OfficerPrivilege[]>([]);
  const [logOpen, setLogOpen] = useState(false);
  const [logs, setLogs] = useState<OfficerLog[]>([]);
  const [logOfficer, setLogOfficer] = useState<Officer | null>(null);
  const [logsLoading, setLogsLoading] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [employeeData, relationData, formOptions] = await Promise.all([
        apiRequest<Officer[]>("/api/officers"),
        apiRequest<EmployeeCompany[]>("/api/officer-company-relations"),
        apiRequest<OfficerOptions>("/api/officers/options"),
      ]);
      setEmployees(Array.isArray(employeeData) ? employeeData : []);
      setCompanyRows(Array.isArray(relationData) ? relationData : []);
      setOptions(formOptions ?? { companies: [], lands: [], departments: [] });
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : "Data gagal dimuat.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void loadData(); }, [loadData]);

  useEffect(() => {
    if (!formOpen) return;
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === "Escape" && !saving) {
        setFormOpen(false);
        setError("");
      }
    }
    document.addEventListener("keydown", closeOnEscape);
    return () => {
      document.body.style.overflow = previousOverflow;
      document.removeEventListener("keydown", closeOnEscape);
    };
  }, [formOpen, saving]);

  const filteredEmployees = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return employees;
    return employees.filter((item) => [item.name, item.username, item.company, item.estate, item.department, item.email, item.note]
      .some((value) => value.toLowerCase().includes(q)));
  }, [employees, query]);

  const filteredCompanies = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return companyRows;
    return companyRows.filter((item) => [item.employee_name, item.username, item.company_name, item.function]
      .some((value) => value.toLowerCase().includes(q)));
  }, [companyRows, query]);

  const sortedEmployees = useMemo(() => [...filteredEmployees].sort((left, right) => {
    const fields: Record<OfficerSortColumn, keyof Officer> = {
      name: "name", username: "username", company: "company", estate: "estate",
      department: "department", email: "email", note: "note", employee_name: "name",
      company_name: "company", function: "note",
    };
    const comparison = String(left[fields[sortColumn]] ?? "").localeCompare(String(right[fields[sortColumn]] ?? ""), "id", { numeric: true, sensitivity: "base" });
    return sortAscending ? comparison : -comparison;
  }), [filteredEmployees, sortAscending, sortColumn]);

  const sortedCompanies = useMemo(() => [...filteredCompanies].sort((left, right) => {
    const fields: Record<OfficerSortColumn, keyof EmployeeCompany> = {
      name: "employee_name", username: "username", company: "company_name", estate: "function",
      department: "function", email: "username", note: "function", employee_name: "employee_name",
      company_name: "company_name", function: "function",
    };
    const comparison = String(left[fields[sortColumn]] ?? "").localeCompare(String(right[fields[sortColumn]] ?? ""), "id", { numeric: true, sensitivity: "base" });
    return sortAscending ? comparison : -comparison;
  }), [filteredCompanies, sortAscending, sortColumn]);

  const activeCount = tab === "employees" ? sortedEmployees.length : sortedCompanies.length;
  const pageCount = Math.max(1, Math.ceil(activeCount / pageSize));
  const visiblePage = Math.min(currentPage, pageCount);
  const firstRow = activeCount === 0 ? 0 : (visiblePage - 1) * pageSize + 1;
  const lastRow = Math.min(visiblePage * pageSize, activeCount);
  const visibleEmployees = sortedEmployees.slice(firstRow - 1, lastRow);
  const visibleCompanies = sortedCompanies.slice(firstRow - 1, lastRow);

  function changeTab(nextTab: Tab) {
    setTab(nextTab);
    setQuery("");
    setCurrentPage(1);
    setSortColumn(nextTab === "employees" ? "name" : "employee_name");
    setSortAscending(true);
    setError("");
  }

  function changeSort(column: OfficerSortColumn) {
    setCurrentPage(1);
    if (sortColumn === column) setSortAscending((ascending) => !ascending);
    else {
      setSortColumn(column);
      setSortAscending(true);
    }
  }

  function sortIndicator(column: OfficerSortColumn) {
    if (sortColumn !== column) return "↕";
    return sortAscending ? "↑" : "↓";
  }

  function clearForm() {
    setEditingID(null);
    setName("");
    setUsername("");
    setCompanyID("");
    setLandID("");
    setDeptID("");
    setEmail("");
    setNote("");
    setRelationEmployeeID("");
    setRelationCompanyID("");
    setRelationFunction("");
    setNewPassword("");
    setPrivileges([]);
  }

  function closeForm() {
    setFormOpen(false);
    setError("");
    clearForm();
  }

  function startAddEmployee() {
    clearForm();
    setFormKind("employee");
    setError("");
    setFormOpen(true);
  }

  function startEditEmployee(item: Officer) {
    clearForm();
    setFormKind("employee");
    setEditingID(item.id);
    setName(item.name);
    setUsername(item.username);
    setCompanyID(item.company_id === null ? "" : String(item.company_id));
    setLandID(item.potential_land_id === null ? "" : String(item.potential_land_id));
    setDeptID(item.dept_id === null ? "" : String(item.dept_id));
    setEmail(item.email);
    setNote(item.note);
    setError("");
    setFormOpen(true);
  }

  function startAddRelation() {
    clearForm();
    setFormKind("company");
    setError("");
    setFormOpen(true);
  }

  function startEditRelation(item: EmployeeCompany) {
    clearForm();
    setFormKind("company");
    setEditingID(item.id);
    setRelationEmployeeID(item.employee_id === null ? "" : String(item.employee_id));
    setRelationCompanyID(item.company_id === null ? "" : String(item.company_id));
    setRelationFunction(item.function);
    setError("");
    setFormOpen(true);
  }

  function startSetPassword(item: Officer) {
    clearForm();
    setFormKind("password");
    setEditingID(item.id);
    setUsername(item.username);
    setError("");
    setFormOpen(true);
  }

  async function startEditPrivileges(item: Officer) {
    clearForm();
    setFormKind("privileges");
    setEditingID(item.id);
    setUsername(item.username);
    setError("");
    setSaving(true);
    try {
      const items = await apiRequest<OfficerPrivilege[]>(`/api/officers/${item.id}/privileges`);
      setPrivileges(Array.isArray(items) ? items : []);
      setFormOpen(true);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : "Privilege gagal dimuat.");
    } finally {
      setSaving(false);
    }
  }

  async function openLogs(item: Officer) {
    setError("");
    setLogOfficer(item);
    setLogs([]);
    setLogsLoading(true);
    setLogOpen(true);
    try {
      const items = await apiRequest<OfficerLog[]>(`/api/officers/${item.id}/logs`);
      setLogs(Array.isArray(items) ? items : []);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : "Log user gagal dimuat.");
      setLogOpen(false);
    } finally {
      setLogsLoading(false);
    }
  }

  function showCompanyAssignments(item: Officer) {
    setTab("companies");
    setQuery(item.username || item.name);
    setCurrentPage(1);
    setSortColumn("employee_name");
    setSortAscending(true);
    setError("");
  }

  async function saveItem(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    setError("");
    const wasEditing = editingID !== null;
    const currentKind = formKind;
    try {
      if (currentKind === "password") {
        await apiRequest<null>(`/api/officers/${editingID}/password`, {
          method: "PUT",
          body: JSON.stringify({ password: newPassword }),
        });
      } else if (currentKind === "privileges") {
        await apiRequest<OfficerPrivilege[]>(`/api/officers/${editingID}/privileges`, {
          method: "PUT",
          body: JSON.stringify({ permissions: privileges.map(({ module_id, p_add, p_del, p_view, p_edit }) => ({ module_id, p_add, p_del, p_view, p_edit })) }),
        });
      } else if (currentKind === "employee") {
        const payload = {
          name: name.trim(),
          username: username.trim(),
          company_id: companyID ? Number(companyID) : null,
          potential_land_id: landID ? Number(landID) : null,
          dept_id: deptID ? Number(deptID) : null,
          email: email.trim(),
          note: note.trim(),
        };
        await apiRequest<Officer>(wasEditing ? `/api/officers/${editingID}` : "/api/officers", {
          method: wasEditing ? "PUT" : "POST",
          body: JSON.stringify(payload),
        });
      } else {
        const payload = {
          employee_id: relationEmployeeID ? Number(relationEmployeeID) : null,
          company_id: relationCompanyID ? Number(relationCompanyID) : null,
          function: relationFunction.trim(),
        };
        await apiRequest<EmployeeCompany>(wasEditing ? `/api/officer-company-relations/${editingID}` : "/api/officer-company-relations", {
          method: wasEditing ? "PUT" : "POST",
          body: JSON.stringify(payload),
        });
      }
      closeForm();
      await loadData();
      const message = currentKind === "password"
        ? "Password berhasil diperbarui."
        : currentKind === "privileges"
          ? "Privilege user berhasil diperbarui."
        : currentKind === "company"
          ? wasEditing ? "Relasi company employee berhasil diperbarui." : "Relasi company employee berhasil ditambahkan."
          : wasEditing ? "Employee berhasil diperbarui." : "Employee berhasil ditambahkan.";
      if (window.Swal) {
        await window.Swal.fire({ title: "Berhasil", text: message, icon: "success", buttonsStyling: false, confirmButtonText: "Oke", customClass: { popup: "rounded-4 shadow-lg", title: "fw-bold text-gray-900", confirmButton: "btn btn-dark" } });
      } else {
        window.alert(message);
      }
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : "Data gagal disimpan.");
    } finally {
      setSaving(false);
    }
  }

  async function deleteItem(item: Officer | EmployeeCompany) {
    const isEmployee = "type_name" in item || "name" in item;
    const label = isEmployee ? (item as Officer).name : (item as EmployeeCompany).employee_name;
    const path = isEmployee ? `/api/officers/${item.id}` : `/api/officer-company-relations/${item.id}`;
    const confirmation = window.Swal
      ? await window.Swal.fire({ title: "Hapus data ini?", text: `${label} akan dihapus.`, icon: "warning", showCancelButton: true, confirmButtonText: "Ya, hapus", cancelButtonText: "Batal", reverseButtons: true, buttonsStyling: false, customClass: { popup: "rounded-4 shadow-lg", title: "fw-bold text-gray-900", htmlContainer: "text-muted", confirmButton: "btn btn-danger ms-2", cancelButton: "btn btn-light me-2" } })
      : { isConfirmed: window.confirm(`Hapus ${label}?`) };
    if (!confirmation.isConfirmed) return;
    setDeletingID(item.id);
    setError("");
    try {
      await apiRequest<null>(path, { method: "DELETE" });
      await loadData();
      if (window.Swal) {
        await window.Swal.fire({ title: "Berhasil", text: isEmployee ? "Employee berhasil dihapus." : "Relasi berhasil dihapus.", icon: "success", buttonsStyling: false, confirmButtonText: "Oke", customClass: { popup: "rounded-4 shadow-lg", title: "fw-bold text-gray-900", confirmButton: "btn btn-dark" } });
      }
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : "Data gagal dihapus.");
    } finally {
      setDeletingID(null);
    }
  }

  const isAddMode = editingID === null || formKind === "password";
  const modalTitle = formKind === "password"
    ? `Setting Password — ${username}`
    : formKind === "privileges"
      ? `Privilege — ${username}`
    : formKind === "company"
      ? editingID === null ? "Add Company Employee" : `Edit Company Employee #${editingID}`
      : editingID === null ? "Add Employee" : `Edit Employee #${editingID}`;

  return (
    <div className="d-flex flex-column gap-6">
      <nav aria-label="Breadcrumb">
        <ol className="breadcrumb breadcrumb-separatorless fw-semibold fs-7 mb-0">
          <li className="breadcrumb-item"><Link href="/admin/dashboard" className="text-muted text-hover-primary">Admin</Link></li>
          <li className="breadcrumb-item"><span className="bullet bg-gray-300 w-5px h-2px" /></li>
          <li className="breadcrumb-item text-muted" aria-current="page">Officer</li>
        </ol>
      </nav>

      <div className="d-flex flex-wrap align-items-center justify-content-between gap-4">
        <div><h1 className="text-gray-900 fw-bolder fs-2 mb-1">Officer</h1><p className="text-muted fw-semibold mb-0">Kelola employee dan relasi company employee.</p></div>
        <button type="button" className="btn btn-dark" onClick={tab === "employees" ? startAddEmployee : startAddRelation}><span className="me-2 fs-3 lh-1">+</span>{tab === "employees" ? "Add Employee" : "Add Company Employee"}</button>
      </div>

      {error && !formOpen && <div className="alert alert-danger d-flex align-items-center justify-content-between mb-0" role="alert"><span>{error}</span><button type="button" className="btn-close" aria-label="Tutup pesan error" onClick={() => setError("")} /></div>}

      <section className="card border shadow-sm">
        <div className="card-header flex-wrap gap-4 py-4">
          <ul className="nav nav-tabs nav-line-tabs nav-line-tabs-2x border-0 w-100" role="tablist">
            <li className="nav-item" role="presentation"><button type="button" className={`nav-link ${tab === "employees" ? "active" : ""}`} role="tab" aria-selected={tab === "employees"} onClick={() => changeTab("employees")}>Employees</button></li>
            <li className="nav-item" role="presentation"><button type="button" className={`nav-link ${tab === "companies" ? "active" : ""}`} role="tab" aria-selected={tab === "companies"} onClick={() => changeTab("companies")}>Companies Employees</button></li>
          </ul>
          <label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7 me-auto">Search<input className="form-control form-control-sm form-control-solid ref-table-search" type="search" aria-label={`Cari ${tab === "employees" ? "employees" : "companies employees"}`} placeholder={tab === "employees" ? "Name, username, company..." : "Employee, company, function..."} value={query} onChange={(event) => { setQuery(event.target.value); setCurrentPage(1); }} /></label>
          <label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7">Show<select className="form-select form-select-sm form-select-solid w-auto" value={pageSize} onChange={(event) => { setPageSize(Number(event.target.value)); setCurrentPage(1); }} aria-label="Jumlah baris per halaman"><option value={10}>10</option><option value={25}>25</option><option value={50}>50</option></select>entries</label>
          <button type="button" className="btn btn-icon btn-sm btn-light ms-auto" onClick={() => void loadData()} disabled={loading} aria-label="Muat ulang daftar"><span className={loading ? "spinner-border spinner-border-sm" : "svg-icon svg-icon-2"} aria-hidden="true">{!loading && <svg viewBox="0 0 24 24" fill="none"><path d="M20 11a8 8 0 0 0-14.9-3M4 4v4h4m-4 5a8 8 0 0 0 14.9 3M20 20v-4h-4" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>}</span></button>
        </div>
        <div className="card-body pt-0">
          <div className="text-muted fw-semibold fs-7 mb-3">Total data: {tab === "employees" ? employees.length : companyRows.length}</div>
          <div className="table-responsive">
            {tab === "employees" ? (
              <table className="table admin-data-table align-middle">
                <thead className="bg-dark"><tr className="bg-dark text-white fw-bold text-uppercase fs-8">
                  {([ ["name", "Name"], ["username", "Username"], ["company", "Company"], ["estate", "Estate"], ["department", "Dept"], ["email", "Email"], ["note", "Note"] ] as [OfficerSortColumn, string][]).map(([column, label]) => (
                    <th key={column} className="bg-dark text-white" aria-sort={sortColumn === column ? (sortAscending ? "ascending" : "descending") : "none"}>
                      <button type="button" className="btn btn-link p-0 text-white fw-bold text-uppercase fs-8 text-decoration-none" onClick={() => changeSort(column)}>{label} <span aria-hidden="true">{sortIndicator(column)}</span></button>
                    </th>
                  ))}
                  <th className="bg-dark text-white text-end">Actions</th>
                </tr></thead>
                <tbody>
                  {loading ? <TableLoadingRows columns={8} /> : filteredEmployees.length === 0 ? <tr><td colSpan={8} className="text-center py-12 text-muted">{query ? "Data tidak ditemukan." : "Belum ada employee."}</td></tr> : visibleEmployees.map((item) => <tr key={item.id}>
                    <td className="text-gray-900 fw-bold">{item.name}</td><td>{item.username}</td><td>{item.company}</td><td>{item.estate}</td><td>{item.department}</td><td>{item.email}</td><td>{item.note}</td>
                    <td><div className="admin-table-actions"><button type="button" className="btn btn-light-primary" onClick={() => startSetPassword(item)} disabled={!item.username} title="Setting password" aria-label={`Setting password ${item.name}`}><PasswordIcon /></button><button type="button" className="btn btn-light-info" onClick={() => void startEditPrivileges(item)} disabled={!item.username || saving} title="Privileges" aria-label={`Atur privilege ${item.name}`}><PrivilegeIcon /></button><button type="button" className="btn btn-light-primary" onClick={() => showCompanyAssignments(item)} title="Companies" aria-label={`Perusahaan ${item.name}`}><CompanyIcon /></button><button type="button" className="btn btn-light-secondary" onClick={() => void openLogs(item)} title="User log" aria-label={`Lihat log ${item.name}`}><LogIcon /></button><button type="button" className="btn btn-light-primary" onClick={() => startEditEmployee(item)} title="Edit" aria-label={`Edit ${item.name}`}><EditIcon /></button><button type="button" className="btn btn-light-danger" onClick={() => void deleteItem(item)} disabled={deletingID === item.id} title="Delete" aria-label={`Delete ${item.name}`}><DeleteIcon /></button></div></td>
                  </tr>)}
                </tbody>
              </table>
            ) : (
              <table className="table admin-data-table align-middle">
                <thead className="bg-dark"><tr className="bg-dark text-white fw-bold text-uppercase fs-8">
                  {([ ["employee_name", "Employee"], ["username", "Username"], ["company_name", "Company"], ["function", "Function"] ] as [OfficerSortColumn, string][]).map(([column, label]) => (
                    <th key={column} className="bg-dark text-white" aria-sort={sortColumn === column ? (sortAscending ? "ascending" : "descending") : "none"}>
                      <button type="button" className="btn btn-link p-0 text-white fw-bold text-uppercase fs-8 text-decoration-none" onClick={() => changeSort(column)}>{label} <span aria-hidden="true">{sortIndicator(column)}</span></button>
                    </th>
                  ))}
                  <th className="bg-dark text-white text-end">Actions</th>
                </tr></thead>
                <tbody>
                  {loading ? <TableLoadingRows columns={5} /> : filteredCompanies.length === 0 ? <tr><td colSpan={5} className="text-center py-12 text-muted">{query ? "Data tidak ditemukan." : "Belum ada relasi company employee."}</td></tr> : visibleCompanies.map((item) => <tr key={item.id}>
                    <td className="text-gray-900 fw-bold">{item.employee_name}</td><td>{item.username}</td><td>{item.company_name}</td><td>{item.function}</td><td><div className="admin-table-actions"><button type="button" className="btn btn-light-primary" onClick={() => startEditRelation(item)} title="Edit" aria-label={`Edit relasi ${item.employee_name}`}><EditIcon /></button><button type="button" className="btn btn-light-danger" onClick={() => void deleteItem(item)} disabled={deletingID === item.id} title="Delete" aria-label={`Hapus relasi ${item.employee_name}`}><DeleteIcon /></button></div></td>
                  </tr>)}
                </tbody>
              </table>
            )}
          </div>
          </div>
          <div className="card-footer d-flex flex-wrap align-items-center justify-content-between gap-3 py-4">
            <span className="text-muted fw-semibold fs-7">Showing {firstRow}–{lastRow} of {activeCount} entries</span>
            <div className="d-flex align-items-center gap-2">
              <button type="button" className="btn btn-sm btn-light" onClick={() => setCurrentPage((page) => Math.max(1, page - 1))} disabled={visiblePage <= 1}>Previous</button>
              <span className="text-muted fw-semibold fs-7">{visiblePage} / {pageCount}</span>
              <button type="button" className="btn btn-sm btn-light" onClick={() => setCurrentPage((page) => Math.min(pageCount, page + 1))} disabled={visiblePage >= pageCount}>Next</button>
            </div>
          </div>
        </section>

      {formOpen && (
        <div className="ref-modal-backdrop" onMouseDown={(event) => { if (event.target === event.currentTarget && !saving) closeForm(); }}>
          <section className="card ref-modal" role="dialog" aria-modal="true" aria-labelledby="officer-form-title">
            <div className={`card-header min-h-70px form-modal-header ${isAddMode ? "form-modal-header--add" : "form-modal-header--edit"}`}>
              <div className="card-title"><h2 id="officer-form-title" className="fs-4 fw-bold mb-0">{modalTitle}</h2></div>
              <button type="button" className="btn btn-light form-modal-close" onClick={closeForm} disabled={saving} aria-label="Tutup modal"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m6 6 12 12M18 6 6 18" stroke="currentColor" strokeWidth="2" strokeLinecap="round" /></svg></button>
            </div>
            <form onSubmit={saveItem}>
              <div className="card-body p-0">
                {error && <div className="alert alert-danger m-4 mb-0" role="alert">{error}</div>}
                {formKind === "employee" && <>
                  <div className="doc-type-form-row"><label className="form-label mb-0" htmlFor="officer-name">Name:</label><input id="officer-name" className="form-control form-control-solid doc-type-input" value={name} onChange={(event) => setName(event.target.value)} required autoFocus /></div>
                  <div className="doc-type-form-row"><label className="form-label mb-0" htmlFor="officer-username">Username:</label><input id="officer-username" className="form-control form-control-solid doc-type-input" value={username} onChange={(event) => setUsername(event.target.value)} required /></div>
                  <div className="doc-type-form-row"><label className="form-label mb-0" htmlFor="officer-company">Company:</label><select id="officer-company" className="form-select form-select-solid doc-type-select" value={companyID} onChange={(event) => setCompanyID(event.target.value)}><option value="">Select Company</option>{options.companies.map((option) => <option key={option.id} value={option.id}>{option.name}</option>)}</select></div>
                  <div className="doc-type-form-row"><label className="form-label mb-0" htmlFor="officer-land">Estate:</label><select id="officer-land" className="form-select form-select-solid doc-type-select" value={landID} onChange={(event) => setLandID(event.target.value)}><option value="">Select Estate</option>{options.lands.map((option) => <option key={option.id} value={option.id}>{option.estate_name || option.potential_name}</option>)}</select></div>
                  <div className="doc-type-form-row"><label className="form-label mb-0" htmlFor="officer-dept">Dept:</label><select id="officer-dept" className="form-select form-select-solid doc-type-select" value={deptID} onChange={(event) => setDeptID(event.target.value)}><option value="">Select Dept</option>{options.departments.map((option) => <option key={option.id} value={option.id}>{option.meaning || option.id}</option>)}</select></div>
                  <div className="doc-type-form-row"><label className="form-label mb-0" htmlFor="officer-email">Email:</label><input id="officer-email" className="form-control form-control-solid doc-type-input" type="email" value={email} onChange={(event) => setEmail(event.target.value)} /></div>
                  <div className="doc-type-form-row"><label className="form-label mb-0" htmlFor="officer-note">Note:</label><input id="officer-note" className="form-control form-control-solid doc-type-input" value={note} onChange={(event) => setNote(event.target.value)} /></div>
                </>}
                {formKind === "company" && (
                  <div>
                    <div className="doc-type-form-row">
                      <label className="form-label mb-0" htmlFor="relation-employee">Employee:</label>
                      <select
                        id="relation-employee"
                        className="form-select form-select-solid doc-type-select"
                        value={relationEmployeeID}
                        onChange={(event) => setRelationEmployeeID(event.target.value)}
                        required
                      >
                        <option value="">Select Employee</option>
                        {employees.map((person) => (
                          <option key={person.id} value={person.id}>
                            {person.name} ({person.username})
                          </option>
                        ))}
                      </select>
                    </div>
                    <div className="doc-type-form-row">
                      <label className="form-label mb-0" htmlFor="relation-company">Company:</label>
                      <select
                        id="relation-company"
                        className="form-select form-select-solid doc-type-select"
                        value={relationCompanyID}
                        onChange={(event) => setRelationCompanyID(event.target.value)}
                        required
                      >
                        <option value="">Select Company</option>
                        {options.companies.map((option) => (
                          <option key={option.id} value={option.id}>{option.name}</option>
                        ))}
                      </select>
                    </div>
                    <div className="doc-type-form-row">
                      <label className="form-label mb-0" htmlFor="relation-function">Function:</label>
                      <input
                        id="relation-function"
                        className="form-control form-control-solid doc-type-input"
                        value={relationFunction}
                        onChange={(event) => setRelationFunction(event.target.value)}
                      />
                    </div>
                  </div>
                )}
                {formKind === "password" && <>
                  <div className="doc-type-form-row"><label className="form-label mb-0" htmlFor="officer-pass-username">Username:</label><input id="officer-pass-username" className="form-control form-control-solid doc-type-input" value={username} readOnly /></div>
                  <div className="doc-type-form-row"><label className="form-label mb-0" htmlFor="officer-new-password">New password:</label><input id="officer-new-password" className="form-control form-control-solid doc-type-input" type="password" minLength={6} value={newPassword} onChange={(event) => setNewPassword(event.target.value)} required autoFocus /></div>
                </>}
                {formKind === "privileges" && <div className="table-responsive p-4">
                  <table className="table admin-data-table align-middle">
                    <thead className="bg-dark"><tr className="text-white"><th className="bg-dark text-white">Module</th><th className="bg-dark text-white text-center">View</th><th className="bg-dark text-white text-center">Add</th><th className="bg-dark text-white text-center">Edit</th><th className="bg-dark text-white text-center">Delete</th></tr></thead>
                    <tbody>{privileges.map((item, index) => <tr key={item.module_id}>
                      <td><div className="fw-bold">{item.module_name || item.module_code || item.module_file}</div><small className="text-muted">{item.module_file}</small></td>
                      {(["p_view", "p_add", "p_edit", "p_del"] as const).map((field) => <td className="text-center" key={field}><input className="form-check-input" type="checkbox" aria-label={`${field} ${item.module_name}`} checked={item[field] === 1} onChange={(event) => setPrivileges((current) => current.map((permission, row) => row === index ? { ...permission, [field]: event.target.checked ? 1 : 0 } : permission))} /></td>)}
                    </tr>)}</tbody>
                  </table>
                  {privileges.length === 0 && <div className="text-center text-muted py-5">Modul belum tersedia.</div>}
                </div>}
              </div>
              <div className="card-footer d-flex justify-content-end gap-3 py-4">
                <button type="button" className="btn btn-light" onClick={closeForm} disabled={saving}>Cancel</button>
                <button type="submit" className={`btn ${isAddMode ? "btn-dark" : "btn-warning"}`} disabled={saving}>{saving ? <span className="spinner-border spinner-border-sm me-2" aria-hidden="true" /> : <svg className="me-2" width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M5 4h12l3 3v13H4V4h1Zm3 0v6h8V4M8 20v-6h8v6" stroke="currentColor" strokeWidth="1.8" strokeLinejoin="round" /></svg>}{saving ? "Submitting..." : "Submit"}</button>
              </div>
            </form>
          </section>
        </div>
      )}

      {logOpen && (
        <div className="ref-modal-backdrop" onMouseDown={(event) => { if (event.target === event.currentTarget) setLogOpen(false); }}>
          <section className="card ref-modal ref-modal--wide" role="dialog" aria-modal="true" aria-labelledby="officer-log-title">
            <div className="card-header min-h-70px form-modal-header form-modal-header--add">
              <div className="card-title"><h2 id="officer-log-title" className="fs-4 fw-bold mb-0">User Log — {logOfficer?.username}</h2></div>
              <button type="button" className="btn btn-light form-modal-close" onClick={() => setLogOpen(false)} aria-label="Tutup log"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m6 6 12 12M18 6 6 18" stroke="currentColor" strokeWidth="2" strokeLinecap="round" /></svg></button>
            </div>
            <div className="card-body"><div className="table-responsive"><table className="table admin-data-table align-middle"><thead className="bg-dark"><tr className="text-white"><th className="bg-dark text-white">Date</th><th className="bg-dark text-white">Type</th><th className="bg-dark text-white">Object</th><th className="bg-dark text-white">Details</th><th className="bg-dark text-white">Created by</th><th className="bg-dark text-white">IP</th></tr></thead><tbody>
              {logsLoading ? <TableLoadingRows columns={6} /> : logs.length === 0 ? <tr><td colSpan={6} className="text-center py-8 text-muted">Tidak ada log untuk user ini.</td></tr> : logs.map((item, index) => <tr key={item.id ?? index}><td>{formatDateTime(item.date_created)}</td><td>{item.type}</td><td>{item.object}</td><td>{[item.log1, item.log2, item.log3].filter(Boolean).join(" · ")}</td><td>{item.created_by || item.username}</td><td>{item.ip_remote}</td></tr>)}
            </tbody></table></div></div>
            <div className="card-footer d-flex justify-content-end py-4"><button type="button" className="btn btn-light" onClick={() => setLogOpen(false)}>Close</button></div>
          </section>
        </div>
      )}
    </div>
  );
}
