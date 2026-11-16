"use client";

import { useEffect, useMemo, useState } from "react";
import type { Estate } from "./page";
import { BASEMAPS, LeafletMapView, spatialTextToGeoJSON } from "../../companies/land-info";
import type { BasemapKey } from "../../companies/land-info";

type Layer = { kind: string; id: number; name: string; map_data: string; area: number };
type MapReport = { estate_id: number; company_name: string; permit_number: string; potential_name: string; estate_name: string; layers: Layer[] };
type Envelope<T> = { data?: T; message?: string };
const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");
const kinds = [
  { key: "permit", label: "Permit Land", color: "#2563eb" },
  { key: "potential", label: "Potential Land", color: "#d97706" },
  { key: "hcv", label: "HCV Land", color: "#16a34a" },
  { key: "division", label: "Planting Division", color: "#7c3aed" },
  { key: "block", label: "Planting Block", color: "#dc2626" },
  { key: "acquisition", label: "Acquisition Area", color: "#0d9488" },
] as const;
const colorByKind = Object.fromEntries(kinds.map((item) => [item.key, item.color])) as Record<string, string>;
const number = (value: number) => new Intl.NumberFormat("id-ID", { maximumFractionDigits: 2 }).format(value || 0);
type Feature = { type: "Feature"; properties: Record<string, unknown>; geometry: unknown };

function featureStyle(feature?: unknown): Record<string, unknown> {
  const kind = (feature as { properties?: { kind?: string } } | undefined)?.properties?.kind ?? "";
  const color = colorByKind[kind] || "#334155";
  return { color, weight: kind === "potential" ? 4 : kind === "permit" ? 3 : 2, opacity: 0.95, fillColor: color, fillOpacity: kind === "potential" ? 0.06 : kind === "permit" ? 0.08 : 0.24 };
}

function featuresFromLayer(layer: Layer): Feature[] {
  const parsed = JSON.parse(spatialTextToGeoJSON(layer.map_data)) as { type?: string; features?: Feature[]; geometry?: unknown };
  const features = parsed.type === "FeatureCollection" ? parsed.features ?? [] : parsed.type === "Feature" ? [parsed as Feature] : [{ type: "Feature" as const, properties: {}, geometry: parsed }];
  return features.filter((feature) => feature.geometry).map((feature) => ({
    type: "Feature", geometry: feature.geometry,
    properties: { ...feature.properties, kind: layer.kind, label: layer.name },
  }));
}

export default function PlantingMapDialog({ estate, companyID, onClose }: { estate: Estate; companyID: number; onClose: () => void }) {
  const [report, setReport] = useState<MapReport | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [visible, setVisible] = useState<Set<string>>(() => new Set(kinds.map((item) => item.key)));
  const [basemap, setBasemap] = useState<BasemapKey>("esri-imagery");
  const [zoom, setZoom] = useState(5);

  useEffect(() => {
    let active = true;
    const url = `${apiBase}/api/reports/plantmaprep/${estate.id}/planting-map?company_id=${companyID}`;
    const load = async () => {
      let response = await fetch(url, { credentials: "include", cache: "no-store" });
      if (response.status === 401) {
        const refresh = await fetch(`${apiBase}/api/auth/refresh`, { method: "POST", credentials: "include" });
        if (refresh.ok) response = await fetch(url, { credentials: "include", cache: "no-store" });
      }
      const body = await response.json().catch(() => null) as Envelope<MapReport> | null;
      if (!response.ok) throw new Error(body?.message || "Peta planting gagal dimuat.");
      if (active) setReport(body?.data ?? null);
    };
    void load().catch((cause) => { if (active) setError(cause instanceof Error ? cause.message : "Peta planting gagal dimuat."); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [estate.id, companyID]);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => { if (event.key === "Escape") onClose(); };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  const { geoJSON, invalid } = useMemo(() => {
    const features: Feature[] = [];
    let invalid = 0;
    const ordered = [...(report?.layers ?? [])].sort((left, right) => Number(left.kind === "potential") - Number(right.kind === "potential"));
    for (const layer of ordered) {
      if (!visible.has(layer.kind)) continue;
      try { features.push(...featuresFromLayer(layer)); } catch { invalid += 1; }
    }
    return { geoJSON: features.length ? JSON.stringify({ type: "FeatureCollection", features }) : "", invalid };
  }, [report, visible]);

  const toggle = (key: string) => setVisible((current) => {
    const next = new Set(current);
    if (next.has(key)) next.delete(key); else next.add(key);
    return next;
  });

  return <div className="planting-map-dialog" role="dialog" aria-modal="true" aria-label={`Peta planting ${estate.potential_name}`}>
    <div className="planting-map-header"><div><span className="text-muted fs-8 text-uppercase fw-semibold">Map Planting Report</span><h2 className="fs-4 fw-bold mb-0">{estate.potential_name || estate.estate_name || `Estate #${estate.id}`}</h2><span className="text-muted fs-7">{report?.company_name || estate.company_name} · Permit {report?.permit_number || estate.permit_number || "—"}</span></div><button type="button" className="btn btn-sm btn-light" onClick={onClose}>Tutup</button></div>
    {error && <div className="alert alert-danger rounded-0 mb-0" role="alert">{error}</div>}
    <div className="planting-map-body">
      <aside className="planting-map-legend"><h3 className="fw-bold fs-5">Legend & layer</h3><p className="text-muted fs-7">Tampilkan atau sembunyikan batas pada peta.</p>
        {kinds.map((kind) => {
          const layers = (report?.layers ?? []).filter((layer) => layer.kind === kind.key);
          const area = layers.reduce((sum, layer) => sum + layer.area, 0);
          return <label key={kind.key} className="planting-layer-row"><input type="checkbox" checked={visible.has(kind.key)} onChange={() => toggle(kind.key)} /><span className="planting-layer-swatch" style={{ backgroundColor: kind.color }} /><span><strong>{kind.label}</strong><small className="d-block text-muted">{layers.length} polygon · {number(area)} ha</small></span></label>;
        })}
        {invalid > 0 && <div className="alert alert-warning py-2 fs-8 mt-3">{invalid} geometri tidak dapat ditampilkan.</div>}
      </aside>
      <div className="planting-map-canvas map-modal-canvas"><LeafletMapView value={geoJSON} zoom={zoom} onZoomChange={setZoom} basemap={basemap} featureStyle={featureStyle} focusKind="potential" emptyMessage={loading ? "Memuat peta..." : "Belum ada polygon potential land untuk estate ini."} />
        <label className="map-basemap-control"><span>Basemap</span><select className="form-select form-select-sm" value={basemap} onChange={(event) => setBasemap(event.target.value as BasemapKey)}>{Object.entries(BASEMAPS).map(([key, option]) => <option key={key} value={key}>{option.label}</option>)}</select></label>
        <div className="map-zoom-control"><button type="button" className="btn btn-sm btn-light-primary" aria-label="Perbesar peta" onClick={() => setZoom((value) => Math.min(19, value + 1))}>+</button><input aria-label="Zoom peta" type="range" min="1" max="19" value={zoom} onChange={(event) => setZoom(Number(event.target.value))} /><button type="button" className="btn btn-sm btn-light-primary" aria-label="Perkecil peta" onClick={() => setZoom((value) => Math.max(1, value - 1))}>−</button></div>
      </div>
    </div>
  </div>;
}
