"use client";

import Link from "next/link";
import TableLoadingRows from "../../../../components/ui/table-loading-rows";
import { useCallback, useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import { useUserProfile } from "../../../../components/auth/user-profile-context";
import CompensatedMap from "./compensated-map";
import CompensatedDetail from "./compensated-detail";
import { preloadLeaflet } from "../../companies/land-info";

type Row = {
  id: number; status: number; Gis_Aquistion_Divisions_id: number; Gis_Land_Owners_id: number;
  block_plan: string; owner: string; owner_ktp: string; company_name: string; company_id: number;
  potential_land: string; estate_name: string; permit_number: string;
  alas_hak_jenis: string; alas_hak_no: string; land_surface: string; slope_persentage: string; soil_type: string;
  northern_boundary: string; eastern_boundary: string; western_boundary: string; southern_boundary: string;
  village: string; dry_land_wide: number; wet_land_wide: number; land_wide: number;
  dry_land_perha_price: number; wet_land_perha_price: number; admin_price: number; plant_price: number;
  vegetation_note: string; measurement_date: string; grit_date?: string; measured_by: string;
  [key: string]: unknown;
};
type Option = { id: number; name: string };
type CodeOption = { value: string; name: string };
type Options = { block_plans: Option[]; owners: Option[]; original_statuses: CodeOption[]; land_surfaces: CodeOption[]; soil_types: CodeOption[] };
type Permissions = { p_view: boolean; p_edit: boolean };
type Envelope<T> = { data?: T; message?: string };
type Mode = "edit" | "map" | "detail" | null;
type Form = Record<string, string>;
const tableHeaders = [["block_plan", "Div"], ["id", "ID"], ["owner", "Owner"], ["status", "Status"], ["village", "Village"], ["land_wide", "Land Size (ha)"], ["wet_land_wide", "Wet Size (ha)"], ["wet_land_perha_price", "Wet price (@Ha)"], ["dry_land_wide", "Dry Size (ha)"], ["dry_land_perha_price", "Dry price (@Ha)"], ["average", "Avg (@Ha)"], ["admin_price", "Admin price"], ["plant_price", "Plant price"], ["total", "Total price"]] as const;
type SortColumn = (typeof tableHeaders)[number][0];
const numericSortColumns = new Set<SortColumn>(["id", "status", "land_wide", "wet_land_wide", "wet_land_perha_price", "dry_land_wide", "dry_land_perha_price", "average", "admin_price", "plant_price", "total"]);
const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");
const numericFields = ["dry_land_wide", "wet_land_wide", "land_wide", "dry_land_perha_price", "wet_land_perha_price", "admin_price", "plant_price"] as const;
const formFields = ["Gis_Aquistion_Divisions_id", "Gis_Land_Owners_id", "alas_hak_jenis", "alas_hak_no", "land_surface", "slope_persentage", "soil_type", "northern_boundary", "eastern_boundary", "western_boundary", "southern_boundary", "village", ...numericFields, "vegetation_note", "measurement_date", "grit_date", "measured_by"] as const;
const money = (value: unknown) => new Intl.NumberFormat("id-ID", { maximumFractionDigits: 0 }).format(Number(value) || 0);
const area = (value: unknown) => new Intl.NumberFormat("id-ID", { maximumFractionDigits: 2 }).format(Number(value) || 0);
const label = (value: unknown) => value == null || String(value).trim() === "" ? "-" : String(value);
const total = (row: Row) => (Number(row.wet_land_wide) || 0) * (Number(row.wet_land_perha_price) || 0) + (Number(row.dry_land_wide) || 0) * (Number(row.dry_land_perha_price) || 0) + (Number(row.admin_price) || 0) + (Number(row.plant_price) || 0);
const average = (row: Row) => Number(row.land_wide) > 0 ? Math.round(total(row) / Number(row.land_wide)) : 0;
const sortValue = (row: Row, column: SortColumn) => column === "average" ? average(row) : column === "total" ? total(row) : row[column];

async function apiRequest<T>(path: string, init?: RequestInit, renewed = false): Promise<T> {
  const headers = new Headers(init?.headers);
  if (init?.body) headers.set("Content-Type", "application/json");
  const response = await fetch(`${apiBase}${path}`, { ...init, headers, credentials: "include", cache: "no-store" });
  if (response.status === 401 && !renewed) { const refresh = await fetch(`${apiBase}/api/auth/refresh`, { method: "POST", credentials: "include" }); if (refresh.ok) return apiRequest<T>(path, init, true); }
  const body = await response.json().catch(() => null) as Envelope<T> | null;
  if (!response.ok) throw new Error(body?.message || "Permintaan gagal.");
  return body?.data as T;
}

function EditIcon() { return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m14 5 5 5M4 20l4.5-1 10.8-10.8a2.12 2.12 0 0 0-3-3L5.5 16 4 20Z" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>; }
function MapIcon() { return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m3 6 6-3 6 3 6-3v15l-6 3-6-3-6 3V6Zm6-3v15m6-12v15" stroke="currentColor" strokeWidth="1.8" strokeLinejoin="round" /></svg>; }
function DetailIcon() { return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M3 12s3.5-6 9-6 9 6 9 6-3.5 6-9 6-9-6-9-6Z" stroke="currentColor" strokeWidth="1.8" strokeLinejoin="round" /><circle cx="12" cy="12" r="2.5" stroke="currentColor" strokeWidth="1.8" /></svg>; }
function SettingsIcon() { return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M12 3v3m0 12v3M3 12h3m12 0h3M5.6 5.6l2.1 2.1m8.6 8.6 2.1 2.1M18.4 5.6l-2.1 2.1m-8.6 8.6-2.1 2.1" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" /><circle cx="12" cy="12" r="4" stroke="currentColor" strokeWidth="1.8" /></svg>; }

export default function CompensatedLandsView({ title, apiPath, readOnly, detailName }: { title: string; apiPath: string; readOnly: boolean; detailName?: string }) {
  const profile = useUserProfile();
  useEffect(() => { void preloadLeaflet().catch(() => {}); }, []);
  const [companyID, setCompanyID] = useState(0);
  const [companyName, setCompanyName] = useState("");
  const allCompanies = companyName.trim() === "00" || /^00(?:\b|[\s-])/i.test(companyName);
  const [items, setItems] = useState<Row[]>([]);
  const [options, setOptions] = useState<Options>({ block_plans: [], owners: [], original_statuses: [], land_surfaces: [], soil_types: [] });
  const [permissions, setPermissions] = useState<Permissions>({ p_view: false, p_edit: false });
  const canEdit = !readOnly && permissions.p_edit;
  const recordName = detailName ?? (readOnly ? "New Compensated Land" : "Compensated Land");
  const [blockPlan, setBlockPlan] = useState(0);
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(25);
  const [sortColumn, setSortColumn] = useState<SortColumn | null>(null);
  const [sortAscending, setSortAscending] = useState(true);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [selected, setSelected] = useState<Row | null>(null);
  const [mode, setMode] = useState<Mode>(null);
  const [mapMode, setMapMode] = useState<"view" | "settings">("view");
  const [form, setForm] = useState<Form>({});

  useEffect(() => {
    const key = `gis.selected-company.${profile.username.toLowerCase()}`;
    const read = () => { const id = Number(window.localStorage.getItem(key)) || 0; setCompanyID(id); if (!id) setCompanyName(""); };
    const change = (event: Event) => { const detail = (event as CustomEvent<{ id?: number; name?: string }>).detail; if (detail) { const name = detail.name ?? ""; setCompanyName(name); setCompanyID(/^00(?:\b|[\s-])/i.test(name.trim()) || name.trim() === "00" ? 0 : Number(detail.id) || 0); } else read(); };
    read(); window.addEventListener("gis:company-change", change); window.addEventListener("storage", read);
    return () => { window.removeEventListener("gis:company-change", change); window.removeEventListener("storage", read); };
  }, [profile.username]);

  const load = useCallback(async () => {
    if (!companyID && !allCompanies) { setItems([]); setLoading(false); return; }
    setLoading(true); setError("");
    try {
      const params = `?company_id=${companyID}`;
      const [rows, opts, priv] = await Promise.all([apiRequest<Row[]>(`${apiPath}${params}`), apiRequest<Options>(`${apiPath}/options${params}`), apiRequest<Permissions>(`${apiPath}/permissions${params}`)]);
      setItems(Array.isArray(rows) ? [...new Map(rows.map((row) => [row.id, row])).values()] : []); setOptions(opts); setPermissions(priv);
    } catch (cause) { setItems([]); setError(cause instanceof Error ? cause.message : `${title} gagal dimuat.`); }
    finally { setLoading(false); }
  }, [companyID, allCompanies, apiPath, title]);
  useEffect(() => { const timer = window.setTimeout(() => { void load(); }, 0); return () => window.clearTimeout(timer); }, [load]);

  const filtered = useMemo(() => items.filter((item) => (!blockPlan || Number(item.Gis_Aquistion_Divisions_id) === blockPlan) && (!query.trim() || [item.id, item.owner, item.block_plan, item.village, item.status].some((value) => String(value ?? "").toLowerCase().includes(query.trim().toLowerCase())))), [items, blockPlan, query]);
  const sorted = useMemo(() => {
    if (!sortColumn) return filtered;
    return [...filtered].sort((left, right) => {
      const a = sortValue(left, sortColumn);
      const b = sortValue(right, sortColumn);
      const comparison = numericSortColumns.has(sortColumn)
        ? (Number(a) || 0) - (Number(b) || 0)
        : String(a ?? "").localeCompare(String(b ?? ""), "id", { numeric: true, sensitivity: "base" });
      return sortAscending ? comparison : -comparison;
    });
  }, [filtered, sortColumn, sortAscending]);
  const pageCount = Math.max(1, Math.ceil(filtered.length / pageSize));
  const currentPage = Math.min(page, pageCount);
  const shown = sorted.slice((currentPage - 1) * pageSize, currentPage * pageSize);

  function changeSort(column: SortColumn) {
    setPage(1);
    if (sortColumn === column) setSortAscending((ascending) => !ascending);
    else { setSortColumn(column); setSortAscending(true); }
  }

  function open(item: Row, next: Mode, nextMapMode: "view" | "settings" = "view") {
    setSelected(item); setMode(next); setMapMode(nextMapMode); setError("");
    if (next === "edit") setForm(Object.fromEntries(formFields.map((key) => [key, item[key] == null ? "" : String(item[key])] )));
  }
  function close() { if (!saving) { setMode(null); setSelected(null); setError(""); } }
  function change(key: string, value: string) { setForm((current) => ({ ...current, [key]: value })); }

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); if (!selected || readOnly) return;
    const payload: Record<string, string | number> = {};
    for (const key of formFields) payload[key] = key === "Gis_Aquistion_Divisions_id" || key === "Gis_Land_Owners_id" || (numericFields as readonly string[]).includes(key) ? Number(form[key] || 0) : (form[key] || "").trim();
    setSaving(true); setError("");
    try { await apiRequest<Row>(`${apiPath}/${selected.id}?company_id=${companyID}`, { method: "PUT", body: JSON.stringify(payload) }); setMode(null); setSelected(null); await load(); }
    catch (cause) { setError(cause instanceof Error ? cause.message : "Compensated Land gagal diperbarui."); }
    finally { setSaving(false); }
  }

  function field(name: string, key: string, type = "text") { return <div className="row align-items-center py-3 border-bottom" key={key}><label className="col-sm-3 col-form-label" htmlFor={`comp-${key}`}>{name}</label><div className="col-sm-6"><input id={`comp-${key}`} className="form-control form-control-solid" type={type} min={type === "number" ? 0 : undefined} step={type === "number" ? "any" : undefined} value={form[key] || ""} onChange={(event) => change(key, event.target.value)} /></div></div>; }
  function select(name: string, key: string, choices: CodeOption[], required = false) { return <div className="row align-items-center py-3 border-bottom" key={key}><label className="col-sm-3 col-form-label" htmlFor={`comp-${key}`}>{name}</label><div className="col-sm-6"><select id={`comp-${key}`} className="form-select form-select-solid" value={form[key] || ""} onChange={(event) => change(key, event.target.value)} required={required}><option value="">Select {name}</option>{form[key] && !choices.some((item) => item.value === form[key]) && <option value={form[key]}>{form[key]}</option>}{choices.map((item) => <option key={item.value} value={item.value}>{item.name}</option>)}</select></div></div>; }

  return <div className="d-flex flex-column gap-6">
    <nav aria-label="Breadcrumb"><ol className="breadcrumb breadcrumb-separatorless fw-semibold fs-7 mb-0"><li className="breadcrumb-item"><Link href="/admin/dashboard" className="text-muted">Admin</Link></li><li className="breadcrumb-item text-muted ms-2">/ Lands / {title}</li></ol></nav>
    <div><h1 className="text-gray-900 fw-bolder fs-2 mb-1">{title}</h1><p className="text-muted fw-semibold mb-0">Company aktif: <strong>{allCompanies ? "Semua company (00)" : companyName || "Belum dipilih"}</strong></p></div>
    {error && !mode && <div role="alert" className="alert alert-danger">{error}</div>}
    <div className="d-flex flex-column gap-3"><div className="d-flex justify-content-end"><label className="d-flex flex-wrap align-items-center gap-3 fw-semibold compensated-plan-filter">Acquisition Block Plan<select className="form-select compensated-plan-select" value={blockPlan} onChange={(event) => { setBlockPlan(Number(event.target.value)); setPage(1); }}><option value={0}>Semua Block Plan</option>{options.block_plans.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label></div>
    <section className="card border shadow-sm"><div className="card-header flex-wrap gap-4 py-4"><span className="text-muted fw-semibold fs-7">Total data: {items.length}</span><label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7">Search<input className="form-control form-control-sm form-control-solid ref-table-search" type="search" placeholder="ID, owner, village..." value={query} onChange={(event) => { setQuery(event.target.value); setPage(1); }} /></label><label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7">Show<select className="form-select form-select-sm form-select-solid w-auto" value={pageSize} onChange={(event) => { setPageSize(Number(event.target.value)); setPage(1); }}><option>10</option><option>25</option><option>50</option><option>100</option></select>entries</label><button type="button" className="btn btn-sm btn-light" onClick={() => void load()} disabled={loading}>Refresh</button></div>
      <div className="card-body pt-0"><div className="table-responsive"><table className="table admin-data-table align-middle"><thead className="bg-dark"><tr className="text-white fw-bold text-uppercase fs-8">
        {tableHeaders.map(([column, name]) => <th key={column} className="bg-dark text-white text-nowrap" aria-sort={sortColumn === column ? (sortAscending ? "ascending" : "descending") : "none"}><button type="button" className="btn btn-link p-0 text-white fw-bold text-uppercase fs-8 text-decoration-none text-nowrap" onClick={() => changeSort(column)}>{name} <span aria-hidden="true">{sortColumn === column ? (sortAscending ? "\u2191" : "\u2193") : "\u2195"}</span></button></th>)}
        <th className="bg-dark text-white text-nowrap">Actions</th>
      </tr></thead><tbody>
      {loading ? <TableLoadingRows columns={15} /> : !companyID && !allCompanies ? <tr><td colSpan={15} className="text-center py-12 text-muted">Pilih company aktif di header untuk melihat data.</td></tr> : shown.length === 0 ? <tr><td colSpan={15} className="text-center py-12 text-muted">{query || blockPlan ? "Data tidak ditemukan." : `Belum ada ${title}.`}</td></tr> : shown.map((item) => <tr key={item.id}><td className="text-nowrap">{label(item.block_plan)}</td><td>{item.id}</td><td className="text-gray-900 fw-bold text-nowrap">{label(item.owner)}</td><td>{item.status}</td><td>{label(item.village)}</td><td>{area(item.land_wide)}</td><td>{area(item.wet_land_wide)}</td><td>{money(item.wet_land_perha_price)}</td><td>{area(item.dry_land_wide)}</td><td>{money(item.dry_land_perha_price)}</td><td>{money(average(item))}</td><td>{money(item.admin_price)}</td><td>{money(item.plant_price)}</td><td className="fw-bold">{money(total(item))}</td><td><div className="admin-table-actions"><button type="button" className="btn btn-light-primary" title="Detail informasi" aria-label={`Detail informasi ${item.id}`} onClick={() => open(item, "detail")}><DetailIcon /></button>{!readOnly && <button type="button" className="btn btn-light-info" title="Lihat peta" aria-label={`Lihat peta ${item.id}`} onClick={() => open(item, "map")}><MapIcon /></button>}{canEdit && <button type="button" className="btn btn-light-secondary" title="Atur koordinat" aria-label={`Atur koordinat ${item.id}`} onClick={() => open(item, "map", "settings")}><SettingsIcon /></button>}{canEdit && <button type="button" className="btn btn-light-warning" title="Edit" aria-label={`Edit ${item.id}`} onClick={() => open(item, "edit")}><EditIcon /></button>}</div></td></tr>)}
      </tbody></table></div></div><div className="card-footer d-flex flex-wrap align-items-center justify-content-between gap-3 py-4"><span className="text-muted fw-semibold fs-7">Showing {filtered.length ? (currentPage - 1) * pageSize + 1 : 0}-{Math.min(currentPage * pageSize, filtered.length)} of {filtered.length} entries</span><div className="d-flex align-items-center gap-2"><button type="button" className="btn btn-sm btn-light" disabled={currentPage <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>Previous</button><span className="text-muted fw-semibold fs-7">{currentPage} / {pageCount}</span><button type="button" className="btn btn-sm btn-light" disabled={currentPage >= pageCount} onClick={() => setPage((p) => Math.min(pageCount, p + 1))}>Next</button></div></div>
    </section></div>
    {mode && selected && <div role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) close(); }} className="compensated-modal-backdrop" data-fullscreen={mode !== "edit"}><div role="dialog" aria-modal="true" aria-labelledby="compensated-modal-title" className="card shadow-lg compensated-modal" data-fullscreen={mode !== "edit"}><div className={`card-header py-4 ${mode === "edit" ? "form-modal-header form-modal-header--edit" : "hcv-map-modal-header"}`}><h2 id="compensated-modal-title" className="card-title fs-4 fw-bold">{mode === "edit" ? `Edit ${recordName} #${selected.id}` : mode === "detail" ? `Detail ${recordName} #${selected.id}` : `${recordName} Map #${selected.id}`}</h2><button type="button" className="btn btn-sm btn-icon btn-light" aria-label="Tutup" onClick={close} disabled={saving}>x</button></div>
      {mode === "edit" ? <form onSubmit={(event) => void save(event)} className="compensated-edit-form"><div className="card-body py-0">{error && <div role="alert" className="alert alert-danger mt-4">{error}</div>}{select("Div", "Gis_Aquistion_Divisions_id", options.block_plans.map((item) => ({ value: String(item.id), name: item.name })), true)}{select("Owner", "Gis_Land_Owners_id", options.owners.map((item) => ({ value: String(item.id), name: item.name })), true)}{select("Alas hak jenis", "alas_hak_jenis", options.original_statuses)}{field("Alas hak no", "alas_hak_no")}{select("Land surface", "land_surface", options.land_surfaces)}{field("Slope percentage", "slope_persentage")}{select("Soil type", "soil_type", options.soil_types)}{[["Northern boundary", "northern_boundary"], ["Eastern boundary", "eastern_boundary"], ["Western boundary", "western_boundary"], ["Southern boundary", "southern_boundary"], ["Village", "village"], ["Dry Size (ha)", "dry_land_wide"], ["Wet Size (ha)", "wet_land_wide"], ["Land Size (ha)", "land_wide"], ["Dry price (@Ha)", "dry_land_perha_price"], ["Wet price (@Ha)", "wet_land_perha_price"], ["Admin price", "admin_price"], ["Plant price", "plant_price"]].map(([name, key]) => field(name, key, (numericFields as readonly string[]).includes(key) ? "number" : "text"))}<div className="row align-items-start py-3 border-bottom"><label htmlFor="comp-vegetation_note" className="col-sm-3 col-form-label">Vegetation note</label><div className="col-sm-8"><textarea id="comp-vegetation_note" className="form-control form-control-solid" rows={5} value={form.vegetation_note || ""} onChange={(event) => change("vegetation_note", event.target.value)} /></div></div>{field("Measurement date", "measurement_date")}{selected.grit_date !== undefined && field("Grit date", "grit_date")}{field("Measured by", "measured_by")}</div><div className="card-footer d-flex gap-3 py-4"><button type="submit" className="btn btn-warning" disabled={saving}>{saving ? "Menyimpan..." : "Update changes"}</button><button type="button" className="btn btn-light" onClick={close} disabled={saving}>Cancel</button></div></form> : mode === "map" ? <CompensatedMap key={`${selected.id}-${mapMode}`} id={selected.id} companyID={companyID} title={`${selected.owner}  -  ${selected.block_plan}`} editable={canEdit} initialMode={mapMode} apiPath={apiPath} /> : <CompensatedDetail key={selected.id} row={selected} companyID={companyID} canEdit={canEdit} apiPath={apiPath} showTimeline={readOnly} onMap={() => setMode("map")} onEdit={() => open(selected, "edit")} />}
    </div></div>}
  </div>;
}
