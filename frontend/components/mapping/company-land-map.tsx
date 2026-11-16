"use client";

import { useEffect, useMemo, useState } from "react";
import { LeafletMapView } from "../../app/admin/companies/land-info";
import type { BasemapKey, LeafletFeatureLayer } from "../../app/admin/companies/land-info";
import styles from "./company-land-map.module.css";

type LandProperties = { land_id?: number; status?: number; selected?: boolean; owner?: string; land_wide?: number | null; wet_land_wide?: number | null; dry_land_wide?: number | null; village?: string; estate_name?: string; company_name?: string; block_plan?: string; permit_number?: string };
type Feature = { properties?: LandProperties; [key: string]: unknown };
type Layer = { features: Feature[]; next_cursor: number; skipped: number; selected_land?: LandProperties };
const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");
const empty = JSON.stringify({ type: "FeatureCollection", features: [] });
const statusNames: Record<number, string> = { 0: "Submit memo", 1: "RM approval", 2: "GM approval", 3: "GIS / Legal approval", 4: "Operation control", 5: "PD approval", 6: "Payment", 7: "Payment to owner", 8: "New compensated land", 10: "Compensated land", 99: "Canceled" };

function companyStyle(feature?: unknown) {
  const status = (feature as Feature)?.properties?.status;
  const completed = status === 8 || status === 10;
  return { color: completed ? "#15803d" : "#ca8a04", fillColor: completed ? "#22c55e" : "#facc15", weight: 2, opacity: .9, fillOpacity: .25 };
}

function bindLandPopup(feature: unknown, layer: LeafletFeatureLayer) {
  const properties = (feature as Feature)?.properties;
  if (!properties?.land_id) return;
  const content = document.createElement("div");
  content.className = styles.popup;
  const title = document.createElement("strong");
  title.className = styles.popupTitle;
  title.textContent = `Land #${properties.land_id}`;
  const detail = document.createElement("div");
  detail.className = styles.popupStatus;
  detail.textContent = `${properties.selected ? "Lahan terpilih · " : ""}${statusNames[properties.status ?? -1] ?? (properties.status == null ? "Memuat detail…" : `Status ${properties.status}`)}`;
  content.append(title, detail);
  const area = (value: number | null | undefined) => value == null || !Number.isFinite(Number(value)) ? "—" : `${new Intl.NumberFormat("id-ID", { maximumFractionDigits: 4 }).format(Number(value))} ha`;
  const rows: [string, string][] = [
    ["Pemilik", properties.owner?.trim() || "—"],
    ["Luas lahan", area(properties.land_wide)],
    ["Lahan basah", area(properties.wet_land_wide)],
    ["Lahan kering", area(properties.dry_land_wide)],
    ["Company", properties.company_name?.trim() || "—"],
    ["Estate", properties.estate_name?.trim() || "—"],
    ["Desa", properties.village?.trim() || "—"],
    ["Block plan", properties.block_plan?.trim() || "—"],
    ["Nomor izin", properties.permit_number?.trim() || "—"],
  ];
  const list = document.createElement("dl");
  list.className = styles.popupDetails;
  for (const [label, value] of rows) {
    const term = document.createElement("dt"); term.textContent = label;
    const definition = document.createElement("dd"); definition.textContent = value;
    list.append(term, definition);
  }
  content.append(list);
  layer.bindPopup(content);
  if (properties.selected) return;
  const style = companyStyle(feature);
  layer.on("mouseover", () => layer.setStyle({ weight: 4, fillOpacity: .45 }));
  layer.on("mouseout", () => layer.setStyle(style));
}

export default function CompanyLandMap({ id, companyID, value, zoom, onZoomChange, basemap, emptyMessage }: {
  id: number; companyID: number; value: string; zoom: number; onZoomChange: (zoom: number) => void; basemap: BasemapKey; emptyMessage: string;
}) {
  const [result, setResult] = useState<{ key: string; features: Feature[]; loading: boolean; error: string; skipped: number; selectedLand?: LandProperties }>();
  const [visible, setVisible] = useState(true);
  const [retry, setRetry] = useState(0);
  const key = `${companyID}:${id}`;
  useEffect(() => {
    const controller = new AbortController();
    async function load() {
      const features: Feature[] = [];
      let after = 0, skipped = 0;
      let selectedLand: LandProperties | undefined;
      setResult({ key, features: [], loading: true, error: "", skipped: 0 });
      try {
        do {
          const response = await fetch(`${apiBase}/api/mapping/company-lands?company_id=${companyID}&selected_land_id=${id}&after=${after}`, { credentials: "include", cache: "no-store", signal: controller.signal });
          const body = await response.json() as { data?: Layer; message?: string };
          if (!response.ok || !body.data) throw new Error(body.message || "Layer company gagal dimuat.");
          features.push(...body.data.features); skipped += body.data.skipped;
          selectedLand ??= body.data.selected_land;
          const next = body.data.next_cursor;
          if (next !== 0 && next <= after) throw new Error("Cursor layer company tidak valid.");
          after = next;
          if (!controller.signal.aborted) setResult({ key, features: [...features], loading: after > 0, error: "", skipped, selectedLand });
        } while (after > 0 && !controller.signal.aborted);
      } catch (error) {
        if (!controller.signal.aborted) setResult({ key, features: [...features], loading: false, error: error instanceof Error ? error.message : "Layer company gagal dimuat.", skipped, selectedLand });
      }
    }
    void load();
    return () => controller.abort();
  }, [id, companyID, key, retry]);
  const current = result?.key === key ? result : undefined;
  const selected = useMemo(() => {
    if (!value.trim()) return value;
    try {
      const data = JSON.parse(value) as { type: string; features?: Feature[] };
      if (data.type === "FeatureCollection") return JSON.stringify({ ...data, features: data.features?.map((feature) => ({ ...feature, properties: { ...current?.selectedLand, land_id: id, selected: true } })) ?? [] });
      return value;
    } catch { return value; }
  }, [value, id, current?.selectedLand]);
  const background = useMemo(() => JSON.stringify({ type: "FeatureCollection", features: current?.features ?? [] }), [current?.features]);
  return <>
    <LeafletMapView value={selected} zoom={zoom} onZoomChange={onZoomChange} basemap={basemap} emptyMessage={emptyMessage} backgroundValue={visible ? background : empty} backgroundStyle={companyStyle} onEachFeature={bindLandPopup} />
    <div className={styles.legend}>
      <span><i style={{ background: "#e63946" }} />Land #{id} terpilih</span>
      <span><i style={{ background: "#22c55e" }} />Lahan selesai</span>
      <span><i style={{ background: "#facc15" }} />Lahan lainnya</span>
      <label><input type="checkbox" checked={visible} onChange={(event) => setVisible(event.target.checked)} /> Lahan satu company</label>
      <small role="status">{current?.error ? <>{current.error} <button type="button" onClick={() => setRetry((value) => value + 1)}>Coba lagi</button></> : !current || current.loading ? "Memuat lahan company…" : "Klik polygon untuk Land ID · Geser peta untuk lahan lain"}{!!current?.skipped && ` · ${current.skipped} data spasial tidak valid dilewati`}</small>
    </div>
  </>;
}
