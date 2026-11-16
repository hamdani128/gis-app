"use client";

import { useEffect, useMemo, useState } from "react";
import type { ChangeEvent } from "react";
import { BASEMAPS, coordinatePairs, LeafletMapView, mapFileTextToGeoJSON, spatialTextToGeoJSON } from "../../companies/land-info";
import type { BasemapKey } from "../../companies/land-info";
import CompanyLandMap from "../../../../components/mapping/company-land-map";

type MapEnvelope = { data?: { map_data?: string }; message?: string };
type Point = { lng: string; lat: string };
const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");

function fromPoints(points: Point[]): string {
  const pairs = points.map(({ lng, lat }) => [Number(lng), Number(lat)]);
  if (!pairs.length) return JSON.stringify({ type: "FeatureCollection", features: [] });
  const geometry = pairs.length === 1 ? { type: "Point", coordinates: pairs[0] } : pairs.length === 2 ? { type: "LineString", coordinates: pairs } : { type: "Polygon", coordinates: [[...pairs, ...(pairs[0][0] === pairs[pairs.length - 1][0] && pairs[0][1] === pairs[pairs.length - 1][1] ? [] : [pairs[0]])]] };
  return JSON.stringify({ type: "FeatureCollection", features: [{ type: "Feature", properties: {}, geometry }] });
}

function validPoints(points: Point[]) {
  return points.every(({ lng, lat }) => lng.trim() !== "" && lat.trim() !== "" && Number.isFinite(Number(lng)) && Number.isFinite(Number(lat)) && Number(lng) >= -180 && Number(lng) <= 180 && Number(lat) >= -90 && Number(lat) <= 90);
}

function toDraft(data: string): Point[] {
  const pairs = coordinatePairs(data);
  if (pairs.length > 3 && pairs[0][0] === pairs[pairs.length - 1][0] && pairs[0][1] === pairs[pairs.length - 1][1]) pairs.pop();
  return pairs.map(([lng, lat]) => ({ lng: String(lng), lat: String(lat) }));
}

type CompensatedMapProps = { id: number; companyID: number; title: string; editable: boolean; initialMode: "view" | "settings"; apiPath?: string };

export default function CompensatedMap(props: CompensatedMapProps) {
  return <CompensatedMapWorkspace key={`${props.companyID}:${props.id}:${props.apiPath}`} {...props} />;
}

function CompensatedMapWorkspace({ id, companyID, title, editable, initialMode, apiPath = "/api/compensated-lands" }: CompensatedMapProps) {
  const [mode, setMode] = useState(initialMode);
  const [mapData, setMapData] = useState("");
  const [draft, setDraft] = useState<Point[]>([]);
  const [fileName, setFileName] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [basemap, setBasemap] = useState<BasemapKey>("osm");
  const [zoom, setZoom] = useState(5);
  const pointCount = useMemo(() => coordinatePairs(mapData).length, [mapData]);

  useEffect(() => {
    let active = true;
    fetch(`${apiBase}${apiPath}/${id}/map?company_id=${companyID}`, { credentials: "include", cache: "no-store" })
      .then(async (response) => {
        const body = await response.json() as MapEnvelope;
        if (!response.ok) throw new Error(body.message || "Koordinat gagal dimuat.");
        const raw = body.data?.map_data?.trim() ?? "";
        const data = raw ? spatialTextToGeoJSON(raw) : JSON.stringify({ type: "FeatureCollection", features: [] });
        if (active) { setMapData(data); setDraft(toDraft(data)); }
      })
      .catch((cause: unknown) => { if (active) setError(cause instanceof Error ? cause.message : "Koordinat gagal dimuat."); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [id, companyID, apiPath]);

  function editPoint(index: number, key: keyof Point, value: string) {
    const next = draft.map((point, at) => at === index ? { ...point, [key]: value } : point);
    setDraft(next);
    if (validPoints(next)) setMapData(fromPoints(next));
  }

  async function upload(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;
    setError(""); setMessage("");
    try {
      const converted = spatialTextToGeoJSON(mapFileTextToGeoJSON(file.name, await file.text()));
      const pairs = coordinatePairs(converted);
      if (!pairs.length) throw new Error("File tidak berisi titik koordinat.");
      const data = /\.csv$/i.test(file.name) ? fromPoints(pairs.map(([lng, lat]) => ({ lng: String(lng), lat: String(lat) }))) : converted;
      setMapData(data); setDraft(toDraft(data));
      setFileName(file.name); setMessage(`${file.name} siap disimpan.`);
    } catch (cause) { setError(cause instanceof Error ? cause.message : "File koordinat tidak dapat dibaca."); }
    event.target.value = "";
  }

  async function save() {
    if (!validPoints(draft)) { setError("Longitude harus -180 sampai 180 dan latitude -90 sampai 90."); return; }
    setSaving(true); setError(""); setMessage("");
    try {
      const response = await fetch(`${apiBase}${apiPath}/${id}/map?company_id=${companyID}`, { method: "PUT", headers: { "Content-Type": "application/json" }, credentials: "include", body: JSON.stringify({ map_data: mapData }) });
      const body = await response.json() as MapEnvelope;
      if (!response.ok) throw new Error(body.message || "Koordinat gagal disimpan.");
      const data = spatialTextToGeoJSON(body.data?.map_data || "");
      setMapData(data); setDraft(toDraft(data));
      setMessage("Koordinat berhasil disimpan."); setMode("view"); setFileName("");
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Koordinat gagal disimpan."); }
    finally { setSaving(false); }
  }

  return <div className="hcv-map-workspace compensated-map">
    <div className="hcv-map-toolbar"><strong>{title}</strong><div className="d-flex gap-2"><button type="button" className={`btn btn-sm ${mode === "view" ? "btn-primary" : "btn-light"}`} onClick={() => setMode("view")}>Lihat peta</button>{editable && <button type="button" className={`btn btn-sm ${mode === "settings" ? "btn-primary" : "btn-light"}`} onClick={() => setMode("settings")}>Atur koordinat</button>}</div></div>
    {(error || message) && <div role="status" className={`alert ${error ? "alert-danger" : "alert-success"} rounded-0 mb-0 py-2`}>{error || message}</div>}
    <div className={`hcv-map-layout ${mode === "settings" ? "hcv-map-layout--settings" : ""}`}>
      <div className="map-modal-canvas">{mode === "view" ? <CompanyLandMap key={`${companyID}:${id}`} id={id} companyID={companyID} value={mapData} zoom={zoom} onZoomChange={setZoom} basemap={basemap} emptyMessage={loading ? "Memuat koordinat..." : "Belum ada koordinat untuk lahan ini."} /> : <LeafletMapView value={mapData} zoom={zoom} onZoomChange={setZoom} basemap={basemap} emptyMessage={loading ? "Memuat koordinat..." : "Belum ada koordinat untuk lahan ini."} />}<label className="map-basemap-control"><span>Basemap</span><select className="form-select form-select-sm" value={basemap} onChange={(event) => setBasemap(event.target.value as BasemapKey)}>{Object.entries(BASEMAPS).map(([key, item]) => <option key={key} value={key}>{item.label}</option>)}</select></label><div className="map-zoom-control"><button type="button" className="btn btn-sm btn-light-primary" aria-label="Perbesar peta" onClick={() => setZoom((value) => Math.min(19, value + 1))}>+</button><input aria-label="Zoom peta" type="range" min="1" max="19" value={zoom} onChange={(event) => setZoom(Number(event.target.value))} /><button type="button" className="btn btn-sm btn-light-primary" aria-label="Perkecil peta" onClick={() => setZoom((value) => Math.max(1, value - 1))}>-</button></div></div>
      {mode === "settings" && <aside className="hcv-coordinate-panel"><h3 className="fs-5 fw-bold">Form Edit Coordinate</h3><p className="text-muted fs-7">Upload file atau ubah titik, lalu simpan koordinat.</p><label className="form-label fw-semibold" htmlFor="compensated-file">Upload koordinat</label><input id="compensated-file" className="form-control form-control-sm mb-2" type="file" accept=".csv,.json,.geojson,.mif,.wkt,.txt" onChange={(event) => void upload(event)} disabled={loading || saving} />{fileName && <small className="text-muted d-block mb-2">{fileName}</small>}<div className="d-flex justify-content-between border-bottom pb-2"><strong>Coordinates</strong><span className="text-muted">{pointCount} points</span></div><div className="hcv-coordinate-list">{draft.map((point, index) => <div className="map-coordinate-row" key={index}><span className="text-muted">{index + 1}</span><input className="form-control form-control-sm" aria-label={`Longitude titik ${index + 1}`} value={point.lng} onChange={(event) => editPoint(index, "lng", event.target.value)} /><input className="form-control form-control-sm" aria-label={`Latitude titik ${index + 1}`} value={point.lat} onChange={(event) => editPoint(index, "lat", event.target.value)} /><button type="button" className="btn btn-sm btn-light-danger" aria-label={`Hapus titik ${index + 1}`} onClick={() => { const next = draft.filter((_, at) => at !== index); setDraft(next); setMapData(fromPoints(next)); }}>×</button></div>)}</div><div className="d-flex gap-2 mt-3"><button type="button" className="btn btn-sm btn-light-primary" onClick={() => setDraft((current) => [...current, { lng: "", lat: "" }])}>+ Titik</button><button type="button" className="btn btn-sm btn-light" onClick={() => { setDraft([]); setMapData(fromPoints([])); }}>Kosongkan</button></div><button type="button" className="btn btn-primary w-100 mt-4" disabled={saving || loading} onClick={() => void save()}>{saving ? "Menyimpan..." : "Simpan koordinat"}</button></aside>}
    </div>
  </div>;
}
