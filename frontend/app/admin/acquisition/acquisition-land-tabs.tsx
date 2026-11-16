"use client";

import { useId, useState } from "react";
import { LandHistoryPanel } from "./land-history-documents";
import styles from "./acquisition-workspace.module.css";

type Land = { id: number; company_id: number; [key: string]: unknown };
type Tab = "land" | "measurement" | "documents" | "memo";
const tabs: Array<[Tab, string]> = [["land", "Informasi lahan"], ["measurement", "Pengukuran dan harga"], ["documents", "Dokumen"], ["memo", "Memo"]];
const text = (value: unknown) => value == null || String(value).trim() === "" ? "—" : String(value);
const number = (value: unknown, digits = 0) => new Intl.NumberFormat("id-ID", { maximumFractionDigits: digits }).format(Number(value) || 0);
const date = (value: unknown) => { if (!value) return "—"; const parsed = new Date(String(value)); return Number.isNaN(parsed.getTime()) ? text(value) : parsed.toLocaleDateString("id-ID"); };

export default function AcquisitionLandTabs({ land, apiPath, memoNumber }: { land: Land; apiPath: string; memoNumber?: string }) {
  const [tab, setTab] = useState<Tab>("land");
  const id = useId();
  const total = Number(land.wet_land_wide || 0) * Number(land.wet_land_perha_price || 0) + Number(land.dry_land_wide || 0) * Number(land.dry_land_perha_price || 0) + Number(land.admin_price || 0) + Number(land.plant_price || 0);
  const fields: Array<[string, unknown]> = tab === "land" ? [
    ["Owner", land.owner], ["KTP owner", land.owner_ktp], ["Land ID", land.id], ["Company", land.company_name], ["Status", land.status_name || land.status], ["Estate", land.estate_name || land.potential_land], ["Permit number", land.permit_number], ["Block plan", land.block_plan || land.division], ["Village", land.village], ["Planting division", land.plant_division], ["Planting block", land.plant_block], ["Alas hak jenis", land.alas_hak_jenis], ["Alas hak no", land.alas_hak_no], ["Land surface", land.land_surface], ["Soil type", land.soil_type], ["Northern boundary", land.northern_boundary], ["Eastern boundary", land.eastern_boundary], ["Western boundary", land.western_boundary], ["Southern boundary", land.southern_boundary],
  ] : [
    ["Measured by", land.measured_by], ["Measurement date", date(land.measurement_date)], ["Grit date", date(land.grit_date)], ["Land size (ha)", number(land.land_wide, 2)], ["Wet size (ha)", number(land.wet_land_wide, 2)], ["Wet price (@Ha)", number(land.wet_land_perha_price)], ["Dry size (ha)", number(land.dry_land_wide, 2)], ["Dry price (@Ha)", number(land.dry_land_perha_price)], ["Admin price", number(land.admin_price)], ["Plant price", number(land.plant_price)], ["Average price (@Ha)", number(Number(land.land_wide) > 0 ? total / Number(land.land_wide) : 0)], ["Total price", number(total)],
  ];
  return <section className={styles.tabsCard} aria-label="Detail lahan dan arsip akuisisi">
    <div className={styles.tabList} role="tablist" aria-label="Detail lahan">
      {tabs.map(([key, title], index) => <button key={key} type="button" id={`${id}-${key}`} role="tab" aria-selected={tab === key} tabIndex={tab === key ? 0 : -1} aria-controls={`${id}-panel`} onClick={() => setTab(key)} onKeyDown={(event) => {
        if (!["ArrowRight", "ArrowLeft", "Home", "End"].includes(event.key)) return;
        event.preventDefault();
        const next = event.key === "Home" ? 0 : event.key === "End" ? tabs.length - 1 : (index + (event.key === "ArrowRight" ? 1 : -1) + tabs.length) % tabs.length;
        setTab(tabs[next][0]); document.getElementById(`${id}-${tabs[next][0]}`)?.focus();
      }}>{title}</button>)}
    </div>
    <div id={`${id}-panel`} role="tabpanel" aria-labelledby={`${id}-${tab}`} tabIndex={0} className={styles.tabPanel}>
      {tab === "documents" || tab === "memo" ? <LandHistoryPanel key={`${land.id}-${tab}`} landID={land.id} companyID={land.company_id} apiPath={apiPath} fallbackMemoNumber={memoNumber} kind={tab} /> : <div className={styles.infoFields}>
        {fields.map(([label, value]) => <div className={styles.infoRow} key={label}><strong>{label}</strong><span>{text(value)}</span></div>)}
        {tab === "measurement" && <div className={styles.note}><strong>Vegetation note</strong><p>{text(land.vegetation_note)}</p></div>}
      </div>}
    </div>
  </section>;
}
