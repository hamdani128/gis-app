"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import type { FormEvent } from "react";
import { useUserProfile } from "../../../components/auth/user-profile-context";
import CompensatedMap from "../lands/compensatedlands/compensated-map";
import LandHistoryDocuments from "../acquisition/land-history-documents";

type Land = {
  id: number; company_id: number; company_name: string; estate_name: string;
  owner: string; village: string; block_plan: string; permit_number: string;
  status: number; land_wide: number; wet_land_wide: number; dry_land_wide: number;
  admin_price: number; plant_price: number; total_price: number;
  plant_division: string; plant_block: string; document_count: number;
  process_count: number; last_process: string;
};
type Group = { estate: string; status: number; lands: number; area: number; total_price: number };
type ReportData = { rows: Land[]; total: number; page: number; page_size: number; groups: Group[]; total_area: number; total_price: number };
type Process = { id: number; name: string; approval: number | null; request_date: string | null; date_created: string | null; username: string };
type Envelope<T> = { data?: T; message?: string };

const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");
const money = (value: number) => new Intl.NumberFormat("id-ID", { maximumFractionDigits: 0 }).format(Number(value) || 0);
const area = (value: number) => new Intl.NumberFormat("id-ID", { maximumFractionDigits: 2 }).format(Number(value) || 0);
const display = (value: string | number | null | undefined) => value == null || String(value).trim() === "" ? "—" : String(value);
const statusLabels: Record<number, string> = { 0: "New Land", 6: "Payment", 7: "Payment to Owner", 8: "New Compensated Land", 10: "Compensated Land", 99: "Canceled Land" };
const statusName = (status: number) => statusLabels[status] ?? `Status ${status}`;
const date = (value: string | null) => value ? new Date(value).toLocaleDateString("id-ID") : "—";
const isMapReport = (code: string) => code === "mapreport" || code === "plantmaprep";
const isEstateReport = (code: string) => code === "rpt_all_progress" || code === "aestategrttreport";

async function getData<T>(path: string): Promise<T> {
  let response = await fetch(`${apiBase}${path}`, { credentials: "include", cache: "no-store" });
  if (response.status === 401) {
    const renewal = await fetch(`${apiBase}/api/auth/refresh`, { method: "POST", credentials: "include" });
    if (renewal.ok) response = await fetch(`${apiBase}${path}`, { credentials: "include", cache: "no-store" });
  }
  const result = await response.json().catch(() => null) as Envelope<T> | null;
  if (!response.ok) throw new Error(result?.message || "Data report gagal dimuat.");
  return result?.data as T;
}

export default function ReportContent({ title, moduleCode }: { title: string; moduleCode: string }) {
  const profile = useUserProfile();
  const code = moduleCode.toLowerCase();
  const apiPath = `/api/reports/${encodeURIComponent(code)}`;
  const [companyID, setCompanyID] = useState(0);
  const [companySelected, setCompanySelected] = useState(false);
  const [companyReady, setCompanyReady] = useState(false);
  const [data, setData] = useState<ReportData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("");
  const [estate, setEstate] = useState("");
  const [estateOptions, setEstateOptions] = useState<string[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(25);
  const [selected, setSelected] = useState<Land | null>(null);
  const [mapLand, setMapLand] = useState<Land | null>(null);
  const [history, setHistory] = useState<Process[]>([]);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [historyError, setHistoryError] = useState("");

  useEffect(() => {
    const key = `gis.selected-company.${profile.username.toLowerCase()}`;
    const read = () => {
      const saved = window.localStorage.getItem(key);
      setCompanySelected(saved !== null);
      setCompanyID(Number(saved) || 0);
      setCompanyReady(true);
      setPage(1);
      setSelected(null);
      setMapLand(null);
    };
    read();
    window.addEventListener("gis:company-change", read);
    window.addEventListener("storage", read);
    return () => { window.removeEventListener("gis:company-change", read); window.removeEventListener("storage", read); };
  }, [profile.username]);

  const load = useCallback(async (active: () => boolean) => {
    if (!companyReady || !companySelected) { setData(null); return; }
    setLoading(true); setError("");
    const params = new URLSearchParams({ company_id: String(companyID), page: String(page), page_size: String(pageSize) });
    if (search) params.set("search", search);
    if (status) params.set("status", status);
    if (estate) params.set("estate", estate);
    try {
      const result = await getData<ReportData>(`${apiPath}?${params}`);
      if (!active()) return;
      setData(result);
      if (!estate) setEstateOptions([...new Set((result.groups ?? []).map((group) => group.estate).filter(Boolean))].sort());
      if (isMapReport(code)) setMapLand((current) => result.rows.find((row) => row.id === current?.id) ?? result.rows[0] ?? null);
    } catch (cause) {
      if (active()) { setData(null); setError(cause instanceof Error ? cause.message : "Report gagal dimuat."); }
    } finally { if (active()) setLoading(false); }
  }, [apiPath, code, companyID, companyReady, companySelected, estate, page, pageSize, search, status]);

  useEffect(() => {
    let active = true;
    const timer = window.setTimeout(() => void load(() => active), 0);
    return () => { active = false; window.clearTimeout(timer); };
  }, [load]);

  useEffect(() => {
    if (!selected) return;
    let active = true;
    const timer = window.setTimeout(() => {
      setHistory([]); setHistoryLoading(true); setHistoryError("");
      void getData<Process[]>(`${apiPath}/${selected.id}/history?company_id=${companyID}`)
        .then((items) => { if (active) setHistory(items ?? []); })
        .catch((cause) => { if (active) setHistoryError(cause instanceof Error ? cause.message : "Riwayat gagal dimuat."); })
        .finally(() => { if (active) setHistoryLoading(false); });
    }, 0);
    return () => { active = false; window.clearTimeout(timer); };
  }, [apiPath, companyID, selected]);

  const estateGroups = useMemo(() => {
    const groups = new Map<string, { estate: string; lands: number; area: number; total_price: number; statuses: Group[] }>();
    for (const item of data?.groups ?? []) {
      const key = item.estate || "Tanpa estate";
      const group = groups.get(key) ?? { estate: key, lands: 0, area: 0, total_price: 0, statuses: [] };
      group.lands += item.lands; group.area += item.area; group.total_price += item.total_price; group.statuses.push(item);
      groups.set(key, group);
    }
    return [...groups.values()].sort((a, b) => a.estate.localeCompare(b.estate));
  }, [data]);
  const statusGroups = useMemo(() => {
    const groups = new Map<number, { status: number; lands: number; area: number }>();
    for (const item of data?.groups ?? []) {
      const group = groups.get(item.status) ?? { status: item.status, lands: 0, area: 0 };
      group.lands += item.lands; group.area += item.area; groups.set(item.status, group);
    }
    return [...groups.values()].sort((a, b) => a.status - b.status);
  }, [data]);

  const firstRow = (page - 1) * pageSize + 1;
  const lastRow = Math.min(page * pageSize, data?.total ?? 0);
  const totalPages = Math.max(1, Math.ceil((data?.total ?? 0) / pageSize));
  const columns = code === "plantmaprep"
    ? ["Land ID", "Estate", "Owner", "Plant division", "Plant block", "Area (ha)", "Status", "Actions"]
    : code === "reportdocland"
      ? ["Land ID", "Estate", "Owner", "Village", "Dokumen", "Status", "Actions"]
      : ["Land ID", "Company", "Estate", "Owner", "Village", "Area (ha)", "Total harga", "Status", "Proses terakhir", "Actions"];

  function submitSearch(event: FormEvent<HTMLFormElement>) { event.preventDefault(); setPage(1); setSearch(searchInput.trim()); }

  return <div className="d-flex flex-column gap-5 py-5">
    <div><span className="text-muted fw-semibold text-uppercase fs-7">Report</span><h1 className="text-gray-900 fw-bold fs-2 mb-1">{title}</h1><p className="text-muted mb-0">Data lahan berdasarkan company aktif.</p></div>
    {!companySelected && companyReady ? <div className="alert alert-info">Pilih company aktif di header untuk melihat report.</div> : <>
      {error && <div className="alert alert-danger" role="alert">{error}</div>}
      <div className="row g-4">
        {[["Total lahan", money(data?.total ?? 0)], ["Total luas (ha)", area(data?.total_area ?? 0)], ["Total harga", `Rp ${money(data?.total_price ?? 0)}`], ["Estate", money(estateGroups.length)]].map(([label, value]) => <div className="col-sm-6 col-xl-3" key={label}><div className="card border-0 shadow-sm h-100"><div className="card-body p-5"><div className="text-muted fw-semibold fs-7 mb-2">{label}</div><div className="fw-bold fs-3 text-gray-900">{loading && !data ? "…" : value}</div></div></div></div>)}
      </div>
      {(code === "acqprogress" || isEstateReport(code)) && <div className="card border-0 shadow-sm"><div className="card-header align-items-center"><h2 className="card-title fw-bold fs-4 mb-0">{isEstateReport(code) ? "Progress per estate" : "Progress per status"}</h2></div><div className="card-body py-4"><div className="table-responsive"><table className="table table-row-bordered align-middle mb-0"><thead><tr><th>{isEstateReport(code) ? "Estate" : "Status"}</th><th>Jumlah lahan</th><th>Luas (ha)</th>{isEstateReport(code) && <th>Total harga</th>}{isEstateReport(code) && <th>Rincian status</th>}</tr></thead><tbody>{isEstateReport(code) ? estateGroups.map((group) => <tr key={group.estate}><td className="fw-semibold">{group.estate}</td><td>{money(group.lands)}</td><td>{area(group.area)}</td><td>Rp {money(group.total_price)}</td><td>{group.statuses.map((item) => `${statusName(item.status)}: ${item.lands}`).join(" · ")}</td></tr>) : statusGroups.map((group) => <tr key={group.status}><td><span className="badge badge-light-primary">{statusName(group.status)}</span></td><td>{money(group.lands)}</td><td>{area(group.area)}</td></tr>)}{!data?.groups?.length && <tr><td colSpan={5} className="text-center text-muted py-6">Belum ada data.</td></tr>}</tbody></table></div></div></div>}
      {isMapReport(code) && mapLand && <div className="card border-0 shadow-sm overflow-hidden"><div className="card-body p-0"><CompensatedMap key={`${code}-${mapLand.id}`} id={mapLand.id} companyID={companyID} title={`Peta lahan #${mapLand.id} · ${mapLand.estate_name || mapLand.owner}`} editable={false} initialMode="view" apiPath={apiPath} /></div></div>}
      <div className="card border-0 shadow-sm"><div className="card-header flex-wrap align-items-center gap-3"><h2 className="card-title fw-bold fs-4 mb-0">{isMapReport(code) ? "Pilih lahan pada peta" : "Data lahan"}</h2><span className="text-muted">{money(data?.total ?? 0)} data</span></div><div className="card-body pb-0"><div className="d-flex flex-wrap gap-3 align-items-end mb-4"><form onSubmit={submitSearch} className="d-flex gap-2 flex-grow-1" style={{ minWidth: 230 }}><input className="form-control" aria-label="Cari lahan" placeholder="Cari Land ID, owner, desa, estate" value={searchInput} onChange={(event) => setSearchInput(event.target.value)} /><button className="btn btn-primary" type="submit">Cari</button></form><select className="form-select" aria-label="Filter estate" style={{ width: 190 }} value={estate} onChange={(event) => { setEstate(event.target.value); setPage(1); }}><option value="">Semua estate</option>{estateOptions.map((value) => <option key={value} value={value}>{value}</option>)}</select><select className="form-select" aria-label="Filter status" style={{ width: 185 }} value={status} onChange={(event) => { setStatus(event.target.value); setPage(1); }}><option value="">Semua status</option>{[...new Set([...Object.keys(statusLabels).map(Number), ...statusGroups.map((item) => item.status)])].sort((a,b) => a-b).map((value) => <option key={value} value={value}>{statusName(value)}</option>)}</select><select className="form-select" aria-label="Jumlah baris" style={{ width: 95 }} value={pageSize} onChange={(event) => { setPageSize(Number(event.target.value)); setPage(1); }}><option value={10}>10</option><option value={25}>25</option><option value={50}>50</option><option value={100}>100</option></select></div></div>
        <div className={`table-responsive${code === "detaillandacq" ? " detail-land-report-table" : ""}`}><table className="table table-row-bordered table-hover align-middle mb-0"><thead><tr>{columns.map((column) => <th className="text-nowrap" key={column}>{column}</th>)}</tr></thead><tbody>{loading && !data ? <tr><td colSpan={columns.length} className="text-center text-muted py-10">Memuat report...</td></tr> : (data?.rows ?? []).length === 0 ? <tr><td colSpan={columns.length} className="text-center text-muted py-10">Tidak ada data yang sesuai.</td></tr> : data?.rows.map((land) => <tr key={land.id} className={mapLand?.id === land.id && isMapReport(code) ? "table-active" : ""}>
          {code === "plantmaprep" ? <><td className="fw-bold">{land.id}</td><td>{display(land.estate_name)}</td><td>{display(land.owner)}</td><td>{display(land.plant_division)}</td><td>{display(land.plant_block)}</td><td>{area(land.land_wide)}</td><td><span className="badge badge-light-primary">{statusName(land.status)}</span></td></> : code === "reportdocland" ? <><td className="fw-bold">{land.id}</td><td>{display(land.estate_name)}</td><td>{display(land.owner)}</td><td>{display(land.village)}</td><td>{money(land.document_count)}</td><td><span className="badge badge-light-primary">{statusName(land.status)}</span></td></> : <><td className="fw-bold">{land.id}</td><td>{display(land.company_name)}</td><td>{display(land.estate_name)}</td><td>{display(land.owner)}</td><td>{display(land.village)}</td><td>{area(land.land_wide)}</td><td>Rp {money(land.total_price)}</td><td><span className="badge badge-light-primary">{statusName(land.status)}</span></td><td>{display(land.last_process)}</td></>}
          <td><div className="d-flex gap-2">{isMapReport(code) && <button type="button" className="btn btn-sm btn-light-info" onClick={() => setMapLand(land)}>Peta</button>}<button type="button" className="btn btn-sm btn-light-primary" onClick={() => setSelected(land)}>Detail</button></div></td>
        </tr>)}</tbody></table></div><div className="card-footer d-flex flex-wrap justify-content-between align-items-center gap-3"><span className="text-muted">{data?.total ? `${firstRow}–${lastRow} dari ${money(data.total)}` : "0 data"}</span><div className="d-flex align-items-center gap-3"><button type="button" className="btn btn-sm btn-light" disabled={page <= 1 || loading} onClick={() => setPage((value) => value - 1)}>Sebelumnya</button><span>{page} / {totalPages}</span><button type="button" className="btn btn-sm btn-light" disabled={page >= totalPages || loading} onClick={() => setPage((value) => value + 1)}>Berikutnya</button></div></div></div>
    </>}
    {selected && <div className="modal d-block show" style={{ zIndex: 1100, background: "rgba(0,0,0,.65)" }} role="dialog" aria-modal="true" aria-label={`Detail lahan ${selected.id}`}><div className="modal-dialog modal-xl modal-dialog-centered" style={{ maxWidth: "min(1200px, 97vw)" }}><div className="modal-content" style={{ maxHeight: "94vh" }}><div className="modal-header"><h2 className="modal-title fs-4">{title} · Land #{selected.id}</h2><button type="button" className="btn-close" aria-label="Tutup detail" onClick={() => setSelected(null)} /></div><div className="modal-body overflow-auto"><div className="row g-4"><div className="col-lg-6"><h3 className="fs-5 fw-bold">Informasi lahan</h3><div className="row g-2">{[["Company", selected.company_name], ["Estate", selected.estate_name], ["Owner", selected.owner], ["Village", selected.village], ["Block plan", selected.block_plan], ["Permit number", selected.permit_number], ["Status", statusName(selected.status)], ["Proses terakhir", selected.last_process], ["Plant division", selected.plant_division], ["Plant block", selected.plant_block], ["Luas (ha)", area(selected.land_wide)], ["Wet (ha)", area(selected.wet_land_wide)], ["Dry (ha)", area(selected.dry_land_wide)], ["Admin price", `Rp ${money(selected.admin_price)}`], ["Plant price", `Rp ${money(selected.plant_price)}`], ["Total price", `Rp ${money(selected.total_price)}`]].map(([label, value]) => <div className="col-sm-6" key={label}><div className="text-muted fs-7">{label}</div><strong>{display(value)}</strong></div>)}</div><h3 className="fs-5 fw-bold mt-6">Riwayat proses</h3>{historyLoading ? <p className="text-muted">Memuat riwayat...</p> : historyError ? <div className="alert alert-danger">{historyError}</div> : history.length ? <div className="table-responsive"><table className="table table-row-bordered align-middle"><thead><tr><th>Tahap</th><th>Persetujuan</th><th>Tanggal</th><th>Petugas</th></tr></thead><tbody>{history.map((item) => <tr key={item.id}><td>{display(item.name)}</td><td>{item.approval === 1 ? "Approved" : item.approval === 0 ? "Pending" : "—"}</td><td>{date(item.date_created || item.request_date)}</td><td>{display(item.username)}</td></tr>)}</tbody></table></div> : <p className="text-muted">Belum ada catatan proses.</p>}</div><div className="col-lg-6"><CompensatedMap key={`${code}-detail-${selected.id}`} id={selected.id} companyID={companyID} title="Geographic Overview" editable={false} initialMode="view" apiPath={apiPath} /></div></div><div className="mt-5"><LandHistoryDocuments landID={selected.id} companyID={companyID} apiPath={apiPath} /></div></div><div className="modal-footer"><button type="button" className="btn btn-light" onClick={() => setSelected(null)}>Tutup</button></div></div></div></div>}
  </div>;
}
