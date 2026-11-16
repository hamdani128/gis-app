"use client";

import { useEffect, useId, useState } from "react";
import CompensatedMap from "./compensated-map";
import { LandHistoryPanel } from "../../acquisition/land-history-documents";
import styles from "./compensated-detail.module.css";
import { acquisitionStages } from "../../acquisition/acquisition-process-overview";

type Land = { id: number; status: number; owner: string; block_plan: string; [key: string]: unknown };
type Process = { id: number; name: string; sequence_id: number | null; sequence: number | null; approval: number | null; request_number: string | null; request_date: string | null; revision: number | null; note: string | null };
type Tab = "land" | "measurement" | "documents" | "memo";

const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");
const steps = acquisitionStages.map(([, name]) => name);
const stepKey = (name: string) => name.trim().replace(/\s+/g, " ").toLowerCase();
const label = (value: unknown) => value == null || String(value).trim() === "" ? "-" : String(value);
const money = (value: unknown) => new Intl.NumberFormat("id-ID", { maximumFractionDigits: 0 }).format(Number(value) || 0);
const area = (value: unknown) => new Intl.NumberFormat("id-ID", { maximumFractionDigits: 2 }).format(Number(value) || 0);
const total = (row: Land) => (Number(row.wet_land_wide) || 0) * (Number(row.wet_land_perha_price) || 0) + (Number(row.dry_land_wide) || 0) * (Number(row.dry_land_perha_price) || 0) + (Number(row.admin_price) || 0) + (Number(row.plant_price) || 0);
const average = (row: Land) => Number(row.land_wide) > 0 ? Math.round(total(row) / Number(row.land_wide)) : 0;

function detail(name: string, value: unknown) {
  return <div className="compensated-detail-row" key={name}><strong>{name}</strong><span>{label(value)}</span></div>;
}

export default function CompensatedDetail({ row, companyID, canEdit, apiPath, showTimeline = false, onMap, onEdit }: { row: Land; companyID: number; canEdit: boolean; apiPath: string; showTimeline?: boolean; onMap: () => void; onEdit: () => void }) {
  const [tab, setTab] = useState<Tab>("land");
  const tabID = useId();
  const [processes, setProcesses] = useState<Process[]>([]);
  const [processLoading, setProcessLoading] = useState(true);
  const [processError, setProcessError] = useState("");
  const extraSteps = [...new Set(processes.map((process) => process.name))].filter((name) => !steps.some((step) => stepKey(step) === stepKey(name)));
  const processSteps = [...steps, ...extraSteps];
  const recordedSteps = processSteps.filter((step) => processes.some((process) => stepKey(process.name) === stepKey(step))).length;

  useEffect(() => {
    let active = true;
    fetch(`${apiBase}${apiPath}/${row.id}/processes?company_id=${companyID}`, { credentials: "include", cache: "no-store" })
      .then(async (response) => {
        if (response.status === 401) {
          const refresh = await fetch(`${apiBase}/api/auth/refresh`, { method: "POST", credentials: "include" });
          if (refresh.ok) return fetch(`${apiBase}${apiPath}/${row.id}/processes?company_id=${companyID}`, { credentials: "include", cache: "no-store" });
        }
        return response;
      })
      .then(async (response) => {
        const body = await response.json() as { data?: Process[]; message?: string };
        if (!response.ok) throw new Error(body.message || "Data proses gagal dimuat.");
        if (active) setProcesses(Array.isArray(body.data) ? [...new Map(body.data.map((process) => [JSON.stringify(process), process])).values()] : []);
      })
      .catch((cause: unknown) => { if (active) setProcessError(cause instanceof Error ? cause.message : "Data proses gagal dimuat."); })
      .finally(() => { if (active) setProcessLoading(false); });
    return () => { active = false; };
  }, [row.id, companyID, apiPath]);

  const acquisitionLayout = apiPath === "/api/new-compensated-lands";
  const hasHistory = acquisitionLayout || apiPath.startsWith("/api/land-owners/");
  const tabsInOverview = hasHistory || showTimeline;
  const tabItems: Array<{ id: Tab; label: string }> = [
    { id: "land", label: "Informasi lahan" },
    { id: "measurement", label: "Pengukuran dan harga" },
    ...(hasHistory ? [{ id: "documents" as const, label: "Dokumen" }, { id: "memo" as const, label: "Memo" }] : []),
  ];
  const tabs = (
    <section className="compensated-detail-tabs" aria-label="Detail lahan">
      <div className={`compensated-tab-list ${styles.tabs}`} role="tablist" aria-label="Informasi Compensated Land">
        {tabItems.map((item, index) => <button key={item.id} type="button" id={`${tabID}-${item.id}`} role="tab" tabIndex={tab === item.id ? 0 : -1} aria-selected={tab === item.id} aria-controls={`${tabID}-panel`} onClick={() => setTab(item.id)} onKeyDown={(event) => {
          if (!["ArrowLeft", "ArrowRight", "Home", "End"].includes(event.key)) return;
          event.preventDefault();
          const next = event.key === "Home" ? 0 : event.key === "End" ? tabItems.length - 1 : (index + (event.key === "ArrowRight" ? 1 : -1) + tabItems.length) % tabItems.length;
          setTab(tabItems[next].id);
          document.getElementById(`${tabID}-${tabItems[next].id}`)?.focus();
        }}>{item.label}</button>)}
      </div>
      <div id={`${tabID}-panel`} role="tabpanel" aria-labelledby={`${tabID}-${tab}`} tabIndex={0} className={tab === "documents" || tab === "memo" ? styles.historyTab : "compensated-detail-section"}>
        {tab === "land" ? <>
          {detail("Owner Name", row.owner)}{detail("Owner KTP", row.owner_ktp)}{detail("Land ID", row.id)}{detail("Company", row.company_name)}{detail("Permit Number", row.permit_number)}{detail("Estate name", row.estate_name || row.potential_land)}{detail("Block plan", row.block_plan)}{detail("Planting division", row.plant_division)}{detail("Planting block", row.plant_block)}{detail("Village", row.village)}{detail("Alas hak jenis", row.alas_hak_jenis)}{detail("Alas hak no", row.alas_hak_no)}{detail("Land surface", row.land_surface)}{detail("Soil type", row.soil_type)}{detail("Slope percentage", row.slope_persentage)}{detail("Northern boundary", row.northern_boundary)}{detail("Eastern boundary", row.eastern_boundary)}{detail("Western boundary", row.western_boundary)}{detail("Southern boundary", row.southern_boundary)}
        </> : tab === "measurement" ? <>
          {detail("Measured by", row.measured_by)}{detail("Measurement date", row.measurement_date)}{detail("Grit date", row.grit_date)}{detail("Wet land size (ha)", area(row.wet_land_wide))}{detail("Wet price (@Ha)", money(row.wet_land_perha_price))}{detail("Dry land size (ha)", area(row.dry_land_wide))}{detail("Dry price (@Ha)", money(row.dry_land_perha_price))}{detail("Total land size (ha)", area(row.land_wide))}{detail("Admin price", money(row.admin_price))}{detail("Plant price", money(row.plant_price))}{detail("Average price (@Ha)", money(average(row)))}{detail("Total price", money(total(row)))}<h3 className="mt-5">Vegetation note</h3><div className="p-4 border rounded bg-light">{label(row.vegetation_note)}</div>
        </> : <LandHistoryPanel key={`${row.id}-${tab}`} landID={row.id} companyID={companyID} apiPath={apiPath} kind={tab} fallbackMemoNumber={processes.find((process) => process.sequence_id === 1)?.request_number ?? undefined} />}
      </div>
    </section>
  );

  return <div className="compensated-detail">
    <div className="compensated-detail-summary">
      <span className="badge badge-light-primary">Status: {row.status === 99 ? "Rejected/Canceled" : row.status}</span>
      <strong>{label(row.owner)} - {label(row.block_plan)}</strong>
      <div className="d-flex gap-2 ms-auto"><button type="button" className="btn btn-sm btn-light-primary" onClick={onMap}>Geographic Overview</button>{canEdit && <button type="button" className="btn btn-sm btn-light-warning" onClick={onEdit}>Edit</button>}</div>
    </div>
    <div className={`compensated-overview-grid${showTimeline ? " compensated-overview-grid--timeline" : ""}${acquisitionLayout ? " compensated-overview-grid--acquisition" : ""}`}>
      <aside className={`compensated-process-list ${styles.sequence}${showTimeline ? " compensated-process-list--timeline" : ""}`} aria-label="Process">
        {showTimeline ? <div className="compensated-process-overview">
          <div className="compensated-process-overview-heading"><div><span>ALUR AKUISISI</span><h3>Process</h3></div><strong>{processLoading ? "..." : `${recordedSteps}/${processSteps.length}`}</strong></div>
          <p>{processLoading ? "Memuat catatan proses..." : `${recordedSteps} dari ${processSteps.length} tahap memiliki catatan`}</p>
          <div className="compensated-process-progress" role="progressbar" aria-label="Tahap yang memiliki catatan proses" aria-valuemin={0} aria-valuemax={processSteps.length} aria-valuenow={recordedSteps}><span style={{ width: `${recordedSteps / processSteps.length * 100}%` }} /></div>
        </div> : <h3 className="fs-5 fw-bold">Process</h3>}
        {processError && <div role="alert" className="alert alert-danger py-2">{processError}</div>}
        {!processLoading && !processError && processes.length === 0 && <p className={showTimeline ? "text-muted" : "text-muted fs-7"}>Belum ada proses yang tercatat untuk lahan ini.</p>}
        <ol className="compensated-process-steps">{processSteps.map((step, index) => {
          const records = processes.filter((item) => stepKey(item.name) === stepKey(step));
          return <li key={step} className={records.length ? "has-record" : ""}>
            <span className="compensated-process-number">{index + 1}</span>
            <details className={styles.stage}>
              <summary className={styles.stageToggle}><span><strong>{step}</strong><span className={styles.stageHint}>{processLoading ? "Memuat..." : records.length ? `${records.length} catatan · klik untuk detail` : processError ? "-" : "Belum ada catatan"}</span></span><span className={`${styles.stageCount} ${records.length ? styles.recorded : ""}`}>{processLoading ? "…" : records.length}</span></summary>
              {records.length > 0 ? <div className="compensated-process-records">{records.map((process, recordIndex) => <div key={`${process.id}-${recordIndex}`} className="compensated-process-record">
                <span className="compensated-process-record-id">#{process.id}</span>
                <div className="compensated-process-meta">{process.request_number && <span>No. {process.request_number}</span>}{process.request_date && <span>{process.request_date.slice(0, 10)}</span>}{process.approval != null && <span className={`${styles.decision} ${process.approval === 1 ? styles.approved : process.approval === 0 ? styles.rejected : styles.pending}`}>{process.approval === 1 ? "Approved" : process.approval === 0 ? "Rejected" : process.approval === 2 ? "On Hold" : `Approval: ${process.approval}`}</span>}{process.revision != null && <span>Revisi: {process.revision}</span>}</div>
                {process.note?.trim() && <span className="compensated-process-note"><strong>Note:</strong> {process.note}</span>}
              </div>)}</div> : <p className={styles.noRecord}>{processLoading ? "Memuat catatan proses..." : "Belum ada catatan untuk tahap ini."}</p>}
            </details>
          </li>;
        })}</ol>
      </aside>
      <div className="compensated-overview-main">
        <div className="compensated-overview-map"><CompensatedMap id={row.id} companyID={companyID} title="Geographic Overview" editable={false} initialMode="view" apiPath={apiPath} /></div>
        {tabsInOverview && tabs}
      </div>
    </div>
    {!tabsInOverview && tabs}
  </div>;
}
