"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useUserProfile } from "../../../../components/auth/user-profile-context";
import CompensatedMap from "../../lands/compensatedlands/compensated-map";
import AcquisitionLandTabs from "../acquisition-land-tabs";
import AcquisitionStage, { ProcessApprovalBadge } from "../acquisition-stage";
import TableLoadingRows from "../../../../components/ui/table-loading-rows";
import { dedupeHistory } from "../process-history";
import AcquisitionProcessOverview, { acquisitionStages } from "../acquisition-process-overview";
import PaymentDocumentUpload, { type PaymentDocumentType, type PendingPaymentDocument } from "./payment-document-upload";

type Land = { id: number; company_id: number; company_name: string; estate_name: string; owner: string; village: string; block_plan: string; permit_number: string; land_wide: number; wet_land_wide: number; wet_land_perha_price: number; dry_land_wide: number; dry_land_perha_price: number; admin_price: number; plant_price: number; status: number; measurement_date?: string; measured_by?: string; vegetation_note?: string };
type Officer = { employee_id: number; name: string; company_id: number };
type Process = { id: number; sequence_id: number; name: string; request_number: string | null; approval: number | null; note: string | null; date_created: string | null; username: string; employee_name: string };
type Detail = { land: Land; history: Process[] };
type Envelope<T> = { data?: T; message?: string };
type Column = "id" | "estate_name" | "owner" | "village" | "land_wide" | "wet_land_wide" | "dry_land_wide" | "admin_price" | "plant_price" | "total" | "status";

const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");
const columns: [Column, string][] = [["id", "Land ID"], ["estate_name", "Estate"], ["owner", "Owner"], ["village", "Village"], ["land_wide", "Land Size (ha)"], ["wet_land_wide", "Wet (ha)"], ["dry_land_wide", "Dry (ha)"], ["admin_price", "Admin price"], ["plant_price", "Plant price"], ["total", "Total price"], ["status", "Status"]];
const steps = acquisitionStages;
const money = (value: unknown) => new Intl.NumberFormat("id-ID", { maximumFractionDigits: 0 }).format(Number(value) || 0);
const area = (value: unknown) => new Intl.NumberFormat("id-ID", { maximumFractionDigits: 2 }).format(Number(value) || 0);
const total = (land: Land) => Number(land.wet_land_wide || 0) * Number(land.wet_land_perha_price || 0) + Number(land.dry_land_wide || 0) * Number(land.dry_land_perha_price || 0) + Number(land.admin_price || 0) + Number(land.plant_price || 0);
const display = (value: unknown) => value == null || String(value).trim() === "" ? "—" : String(value);
const date = (value: string | null | undefined) => value ? new Date(value).toLocaleDateString("id-ID") : "—";

async function api<T>(path: string, init?: RequestInit, renewed = false): Promise<T> {
  const response = await fetch(`${apiBase}${path}`, { ...init, credentials: "include", cache: "no-store" });
  if (response.status === 401 && !renewed) {
    const refresh = await fetch(`${apiBase}/api/auth/refresh`, { method: "POST", credentials: "include" });
    if (refresh.ok) return api<T>(path, init, true);
  }
  const result = await response.json().catch(() => null) as Envelope<T> | null;
  if (!response.ok) throw new Error(result?.message || "Permintaan gagal.");
  return result?.data as T;
}

function value(land: Land, column: Column, ownerStage: boolean): string | number {
  if (column === "total") return total(land);
  if (column === "status") return ownerStage ? "Menunggu Payment to Owner" : "Menunggu Payment";
  return land[column] ?? "";
}

export function AcquisitionPaymentPage({ ownerStage = false }: { ownerStage?: boolean }) {
  const apiPath = ownerStage ? "/api/payment-to-owner" : "/api/payment";
  const stageName = ownerStage ? "Payment to Owner" : "Payment";
  const stageStatus = ownerStage ? 7 : 6;
  const stageSequence = ownerStage ? 7 : 6;
  const profile = useUserProfile();
  const [companyID, setCompanyID] = useState(0);
  const [companySelected, setCompanySelected] = useState(false);
  const [companyReady, setCompanyReady] = useState(false);
  const [lands, setLands] = useState<Land[]>([]);
  const [officer, setOfficer] = useState<Officer | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  useEffect(() => {
    const landID = new URLSearchParams(window.location.search).get("land_id");
    if (!landID || !/^\d+$/.test(landID)) return;
    const timer = window.setTimeout(() => setQuery(landID), 0);
    return () => window.clearTimeout(timer);
  }, []);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [selected, setSelected] = useState<Land | null>(null);
  const [pdfPreview, setPdfPreview] = useState(false);
  const [detail, setDetail] = useState<Detail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [modalError, setModalError] = useState("");
  const [saving, setSaving] = useState(false);
  const [success, setSuccess] = useState("");
  const allCompanies = companySelected && companyID === 0;

  useEffect(() => {
    const key = `gis.selected-company.${profile.username.toLowerCase()}`;
    const read = () => { const saved = window.localStorage.getItem(key); setCompanySelected(saved !== null); setCompanyID(Number(saved) || 0); setCompanyReady(true); };
    const change = (event: Event) => { const item = (event as CustomEvent<{ id?: number }>).detail; if (item) { setCompanySelected(true); setCompanyID(Number(item.id) || 0); } else read(); };
    read(); window.addEventListener("gis:company-change", change); window.addEventListener("storage", read);
    return () => { window.removeEventListener("gis:company-change", change); window.removeEventListener("storage", read); };
  }, [profile.username]);

  const load = useCallback(async () => {
    if (!companyReady || (!companyID && !allCompanies)) { setLands([]); setOfficer(null); return; }
    setLoading(true); setError("");
    try {
      const data = await api<{ officer: Officer | null; lands: Land[] }>(`${apiPath}?company_id=${companyID}`);
      setOfficer(data.officer); setLands(data.lands ?? []);
    } catch (cause) { setLands([]); setOfficer(null); setError(cause instanceof Error ? cause.message : `Data ${stageName} gagal dimuat.`); }
    finally { setLoading(false); }
  }, [companyID, allCompanies, companyReady, apiPath, stageName]);
  useEffect(() => { const timer = window.setTimeout(() => void load(), 0); return () => window.clearTimeout(timer); }, [load]);

  const shown = useMemo(() => {
    const text = query.trim().toLowerCase();
    return lands.filter((land) => !text || columns.some(([key]) => String(value(land, key, ownerStage)).toLowerCase().includes(text)));
  }, [lands, query, ownerStage]);
  const pages = Math.max(1, Math.ceil(shown.length / pageSize));
  const currentPage = Math.min(page, pages);
  const rows = shown.slice((currentPage - 1) * pageSize, currentPage * pageSize);
  const activeLand = detail?.land ?? selected;
  const pdfURL = selected ? `${apiBase}/api/payment/${selected.id}/signature.pdf?company_id=${selected.company_id}` : "";
  const canSubmit = !!officer && !!activeLand && activeLand.status === stageStatus && (officer.company_id === activeLand.company_id || (allCompanies && officer.company_id === 0)) && !detail?.history.some((item) => item.sequence_id === stageSequence);

  async function open(land: Land) {
    setSelected(land); setPdfPreview(false); setDetail(null); setDetailLoading(true); setModalError("");
    try { const loaded = await api<Detail>(`${apiPath}/${land.id}?company_id=${land.company_id}`); setDetail({ ...loaded, history: dedupeHistory(loaded.history) }); }
    catch (cause) { setModalError(cause instanceof Error ? cause.message : `Detail ${stageName} gagal dimuat.`); }
    finally { setDetailLoading(false); }
  }

  async function submit(transferDate: string, note: string, documents: PendingPaymentDocument[], choice: "pay_to_owner" | "cancel") {
    if (!selected || !canSubmit) return;
    setSaving(true); setModalError("");
    try {
      const form = new FormData();
      form.set("transfer_date", transferDate); form.set("note", note);
      if (ownerStage) form.set("payment_choice", choice);
      for (const document of documents) {
        form.append("document_type_ids", String(document.typeID));
        form.append("document_titles", document.title);
        form.append("document_notes", document.note);
        form.append("documents", document.file);
      }
      await api(`${apiPath}/${selected.id}?company_id=${selected.company_id}`, { method: "POST", body: form });
      setSelected(null); setDetail(null); setSuccess(`${stageName} #${selected.id} berhasil disimpan.`); await load();
    } catch (cause) { setModalError(cause instanceof Error ? cause.message : `${stageName} gagal disimpan.`); }
    finally { setSaving(false); }
  }

  return <div className="container-fluid py-5">
    <div className="mb-5"><h1 className="fs-2 fw-bold mb-1">{stageName}</h1><p className="text-muted mb-0">{officer ? `Petugas: ${officer.name} · Company ${officer.company_id}` : "Lahan berdasarkan company aktif"}</p></div>
    {error && <div role="alert" className="alert alert-danger">{error}</div>}{success && <div role="status" className="alert alert-success">{success}</div>}
    <section className="card border shadow-sm"><div className="card-header flex-wrap gap-3 py-4"><span className="text-muted fw-semibold fs-7">Total data: {lands.length}</span><label className="d-flex align-items-center gap-2 ms-auto">Search<input type="search" className="form-control form-control-sm form-control-solid ref-table-search" value={query} onChange={(event) => { setQuery(event.target.value); setPage(1); }} /></label><label className="d-flex align-items-center gap-2">Show<select className="form-select form-select-sm w-auto" value={pageSize} onChange={(event) => { setPageSize(Number(event.target.value)); setPage(1); }}>{[10, 25, 50, 100].map((size) => <option key={size}>{size}</option>)}</select>entries</label><button type="button" className="btn btn-sm btn-light" onClick={() => void load()} disabled={loading}>Refresh</button></div>
      <div className="card-body pt-0"><div className="table-responsive"><table className="table admin-data-table align-middle"><thead className="bg-dark"><tr>{columns.map(([key, title]) => <th key={key} className="bg-dark text-white text-nowrap">{title}</th>)}<th className="bg-dark text-white">Actions</th></tr></thead><tbody>{loading ? <TableLoadingRows columns={12} /> : !companyID && !allCompanies ? <tr><td colSpan={12} className="text-center py-12 text-muted">Pilih company aktif di header.</td></tr> : rows.length === 0 ? <tr><td colSpan={12} className="text-center py-12 text-muted">{ownerStage ? "Belum ada lahan status 7 yang sudah diproses Payment." : "Belum ada lahan status 6 yang sudah disetujui PD."}</td></tr> : rows.map((land) => <tr key={land.id}>{columns.map(([key]) => <td key={key} className="text-nowrap">{key === "status" ? <span className="badge badge-light-primary">{value(land, key, ownerStage)}</span> : ["land_wide", "wet_land_wide", "dry_land_wide"].includes(key) ? area(value(land, key, ownerStage)) : ["admin_price", "plant_price", "total"].includes(key) ? money(value(land, key, ownerStage)) : display(value(land, key, ownerStage))}</td>)}<td><button type="button" className="btn btn-sm btn-light-primary text-nowrap" onClick={() => void open(land)}>{officer?.company_id === land.company_id || (allCompanies && officer?.company_id === 0) ? "Process" : "Detail"}</button></td></tr>)}</tbody></table></div></div>
      <div className="card-footer d-flex align-items-center justify-content-between gap-3"><span>Showing {shown.length ? (currentPage - 1) * pageSize + 1 : 0}–{Math.min(currentPage * pageSize, shown.length)} of {shown.length}</span><div className="d-flex align-items-center gap-2"><button className="btn btn-sm btn-light" disabled={currentPage <= 1} onClick={() => setPage(currentPage - 1)}>Previous</button><span>{currentPage} / {pages}</span><button className="btn btn-sm btn-light" disabled={currentPage >= pages} onClick={() => setPage(currentPage + 1)}>Next</button></div></div>
    </section>
    {selected && <div className="modal d-block show admin-modal-backdrop" role="dialog" aria-modal="true" aria-label={`${stageName} · Land #${selected.id}`}><div className="modal-dialog modal-fullscreen"><div className="modal-content"><div className="modal-header"><h2 className="modal-title fs-4">{stageName} · Land #{selected.id}</h2><button type="button" className="btn-close" aria-label="Close" onClick={() => { if (!saving) { setSelected(null); setDetail(null); } }} /></div><div className="modal-body p-0">{modalError && <div role="alert" className="alert alert-danger m-4">{modalError}</div>}
      {detailLoading ? <div className="p-8 text-muted">Memuat proses dan detail lahan...</div> : activeLand && detail && <div className="submit-memo-workspace"><aside className="submit-memo-process"><div className="d-flex justify-content-end mb-2"><button type="button" className="btn btn-sm btn-light" onClick={() => setSelected(null)}>Back</button></div><AcquisitionProcessOverview history={detail.history} /><div className="alert alert-light-primary py-3"><strong>Status: Menunggu {stageName}</strong></div>
        <div className="submit-memo-timeline">{steps.map(([sequenceID, name], index) => { const records = detail.history.filter((item) => item.sequence_id === sequenceID); return <AcquisitionStage key={sequenceID} title={sequenceID === stageSequence ? stageName : name} number={index + 1} count={records.length} active={sequenceID === stageSequence && canSubmit}>{sequenceID === stageSequence && canSubmit ? <PaymentDocumentUpload key={activeLand.id} landID={activeLand.id} stageName={stageName} officerName={officer?.name || profile.name} saving={saving} loadTypes={() => api<PaymentDocumentType[]>(`${apiPath}/document-types?company_id=${activeLand.company_id}`)} onSubmit={submit} /> : records.length ? <div className="d-flex flex-column gap-2">{records.map((record) => <div className="submit-memo-history" key={record.id}><ProcessApprovalBadge approval={record.approval} />{sequenceID === 5 && record.approval === 1 && <div className="mt-2"><button type="button" className="btn btn-sm btn-light-primary" onClick={() => setPdfPreview(true)}>Preview PDF persetujuan PD</button></div>}<div className="fs-7 mt-2">{record.request_number && <div>Memo: {record.request_number}</div>}<div>{record.employee_name || record.username} · {date(record.date_created)}</div>{record.note && <div>Note: {record.note}</div>}</div></div>)}</div> : <p className="text-muted fst-italic mb-1">{sequenceID === stageSequence ? `Menunggu proses ${stageName}.` : "Waiting previous process"}</p>}</AcquisitionStage>; })}</div>
      </aside><div className="submit-memo-main"><div className="submit-memo-map"><CompensatedMap key={activeLand.id} id={activeLand.id} companyID={activeLand.company_id} title="Geographic Overview" editable={false} initialMode="view" apiPath={apiPath} /></div><AcquisitionLandTabs key={activeLand.id} land={activeLand} apiPath={apiPath} memoNumber={detail.history.find((item) => item.sequence_id === 1)?.request_number ?? undefined} /></div></div>}
    </div></div></div></div>}
    {pdfPreview && selected && <div className="modal d-block show" style={{ zIndex: 1100, backgroundColor: "rgba(0, 0, 0, 0.65)" }} role="dialog" aria-modal="true" aria-label={`Preview PDF persetujuan PD · Land #${selected.id}`}>
      <div className="modal-dialog modal-xl modal-dialog-centered" style={{ maxWidth: "min(1100px, 96vw)" }}>
        <div className="modal-content" style={{ height: "min(900px, 94vh)" }}>
          <div className="modal-header"><h2 className="modal-title fs-4">PDF Persetujuan PD · Land #{selected.id}</h2><button type="button" className="btn-close" aria-label="Tutup preview" onClick={() => setPdfPreview(false)} /></div>
          <div className="modal-body p-0 overflow-hidden"><iframe src={`${pdfURL}&preview=1`} title={`PDF persetujuan PD Land #${selected.id}`} style={{ display: "block", width: "100%", height: "100%", border: 0 }} /></div>
          <div className="modal-footer"><button type="button" className="btn btn-light" onClick={() => setPdfPreview(false)}>Tutup</button><a className="btn btn-primary" href={pdfURL}>Download PDF</a></div>
        </div>
      </div>
    </div>}
  </div>;
}

export default function PaymentPage() {
  return <AcquisitionPaymentPage />;
}
