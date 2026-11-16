"use client";

import { useEffect, useMemo, useState } from "react";
import type { ChangeEvent } from "react";
import { BASEMAPS, coordinatePairs, LeafletMapView, mapFileTextToGeoJSON, spatialTextToGeoJSON } from "../../companies/land-info";
import type { BasemapKey } from "../../companies/land-info";

type Mode = "view" | "settings";
type MapResponse = { data?: { map_data?: string; map_upload?: string }; message?: string };
const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");

function parseFeatures(raw: string): Array<{ type: "Feature"; properties: Record<string, unknown>; geometry: Record<string, unknown> }> {
  const parsed = JSON.parse(raw) as Record<string, unknown>;
  if (parsed.type === "FeatureCollection" && Array.isArray(parsed.features)) return parsed.features as ReturnType<typeof parseFeatures>;
  if (parsed.type === "Feature") return [parsed as ReturnType<typeof parseFeatures>[number]];
  if (typeof parsed.type === "string") return [{ type: "Feature", properties: {}, geometry: parsed }];
  throw new Error("File tidak berisi GeoJSON yang valid.");
}

function normalizeMapData(raw: string): string {
  if (!raw.trim()) return "";
  const geoJSON = spatialTextToGeoJSON(raw);
  return parseFeatures(geoJSON).length ? geoJSON : "";
}

function csvBoundary(geoJSON: string): string {
  const pairs = coordinatePairs(geoJSON);
  if (pairs.length < 2) return geoJSON;
  if (pairs.length === 2) return JSON.stringify({ type: "LineString", coordinates: pairs });
  const first = pairs[0], last = pairs[pairs.length - 1];
  const ring = first[0] === last[0] && first[1] === last[1] ? pairs : [...pairs, first];
  return JSON.stringify({ type: "Polygon", coordinates: [ring] });
}

export default function HCVMap({ id, name, permit, companyID, initialMode }: { id: number; name: string; permit: string; companyID: number; initialMode: Mode }) {
  const [mode, setMode] = useState<Mode>(initialMode);
  const [mapData, setMapData] = useState("");
  const [spatialMapData, setSpatialMapData] = useState("");
  const [mapUpload, setMapUpload] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [manualChanged, setManualChanged] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [basemap, setBasemap] = useState<BasemapKey>("osm");
  const [zoom, setZoom] = useState(5);
  const [point, setPoint] = useState({ lng: "", lat: "" });
  const coordinates = useMemo(() => coordinatePairs(mapData), [mapData]);

  useEffect(() => {
    let active = true;
    fetch(`${apiBase}/api/hcv-lands/${id}/map?company_id=${companyID}`, { credentials: "include", cache: "no-store" })
      .then(async (response) => {
        const body = await response.json() as MapResponse;
        if (!response.ok) throw new Error(body.message || "Peta HCV gagal dimuat.");
        const spatial = normalizeMapData(body.data?.map_data ?? "");
        const path = body.data?.map_upload ?? "";
        if (active) { setSpatialMapData(spatial); setMapData(spatial); setMapUpload(path); }
        if (path) {
          const uploadedResponse = await fetch(`${apiBase}/api/hcv-lands/${id}/map-upload?company_id=${companyID}`, { credentials: "include", cache: "no-store" });
          if (uploadedResponse.ok) {
            try { const converted = mapFileTextToGeoJSON(path, await uploadedResponse.text()); const uploaded = normalizeMapData(/\.csv(?:$|[?#])/i.test(path) ? csvBoundary(converted) : converted); if (active && uploaded) setMapData(uploaded); }
            catch { /* Unreadable legacy uploads fall back to spatial rows. */ }
          }
        }
      })
      .catch((cause: unknown) => { if (active) setError(cause instanceof Error ? cause.message : "Peta HCV gagal dimuat."); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [companyID, id]);

  async function handleFile(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;
    setError(""); setMessage("");
    try { const converted = mapFileTextToGeoJSON(file.name, await file.text()); const value = normalizeMapData(/\.csv$/i.test(file.name) ? csvBoundary(converted) : converted); setFile(file); setManualChanged(false); setMapData(value); setMessage(`${file.name} siap disimpan.`); }
    catch (cause) { setError(cause instanceof Error ? cause.message : "File koordinat tidak dapat dibaca."); }
    event.target.value = "";
  }

  function addPoint() {
    const lng = Number(point.lng), lat = Number(point.lat);
    if (!point.lng.trim() || !point.lat.trim() || !Number.isFinite(lng) || !Number.isFinite(lat) || lng < -180 || lng > 180 || lat < -90 || lat > 90) {
      setError("Isi longitude (-180 sampai 180) dan latitude (-90 sampai 90) yang valid."); return;
    }
    try {
      const features = mapData.trim() ? parseFeatures(mapData) : [];
      const polygon = features.find((feature) => feature.geometry.type === "Polygon");
      const rings = polygon?.geometry.coordinates as Array<Array<[number, number]>> | undefined;
      if (rings?.[0]?.length) {
        const ring = rings[0];
        const first = ring[0], last = ring[ring.length - 1];
        ring.splice(first[0] === last[0] && first[1] === last[1] ? ring.length - 1 : ring.length, 0, [lng, lat]);
        if (ring[0][0] !== ring[ring.length - 1][0] || ring[0][1] !== ring[ring.length - 1][1]) ring.push([...ring[0]]);
      } else features.push({ type: "Feature", properties: {}, geometry: { type: "Point", coordinates: [lng, lat] } });
      setMapData(JSON.stringify({ type: "FeatureCollection", features }));
      setManualChanged(true);
      setPoint({ lng: "", lat: "" }); setError(""); setMessage("Titik ditambahkan. Klik Simpan peta untuk menyimpan ke database.");
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Mapdata tidak valid."); }
  }

  async function saveMap() {
    setSaving(true); setError(""); setMessage("");
    try {
      const bodyData = new FormData();
      bodyData.set("map_data", mapData);
      if (file) bodyData.set("map_upload", manualChanged ? new File([mapData], "hcv-coordinates.geojson", { type: "application/geo+json" }) : file);
      else if (manualChanged) bodyData.set("clear_upload", "true");
      const response = await fetch(`${apiBase}/api/hcv-lands/${id}/map?company_id=${companyID}`, {
        method: "PUT", credentials: "include", body: bodyData,
      });
      const body = await response.json() as MapResponse;
      if (!response.ok) throw new Error(body.message || "Peta HCV gagal disimpan.");
      const saved = normalizeMapData(body.data?.map_data ?? "");
      setSpatialMapData(saved); setMapData(saved); setMapUpload(body.data?.map_upload ?? ""); setFile(null); setManualChanged(false);
      setMessage("Peta HCV berhasil disimpan."); setMode("view");
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Peta HCV gagal disimpan."); }
    finally { setSaving(false); }
  }

  return <div className="hcv-map-workspace">
    <div className="hcv-map-toolbar">
      <div><strong>{name}</strong><span className="ms-3 text-muted">Land Permit: {permit}</span></div>
      <div className="d-flex align-items-center gap-2">
        <button type="button" className={`btn btn-sm ${mode === "view" ? "btn-primary" : "btn-light"}`} onClick={() => setMode("view")}>Lihat peta</button>
        <button type="button" className={`btn btn-sm ${mode === "settings" ? "btn-primary" : "btn-light"}`} onClick={() => setMode("settings")}>Atur koordinat</button>
      </div>
    </div>
    {(error || message) && <div className={`alert ${error ? "alert-danger" : "alert-success"} rounded-0 mb-0 py-2`} role="status">{error || message}</div>}
    <div className={`hcv-map-layout ${mode === "settings" ? "hcv-map-layout--settings" : ""}`}>
      <div className="map-modal-canvas">
        <LeafletMapView value={mode === "view" ? (spatialMapData || mapData) : mapData} zoom={zoom} onZoomChange={setZoom} basemap={basemap} />
        {loading && !spatialMapData && <div className="map-empty-hint">Memuat data spasial HCV...</div>}
        <label className="map-basemap-control"><span>Basemap</span><select className="form-select form-select-sm" aria-label="Pilih jenis basemap" value={basemap} onChange={(event) => setBasemap(event.target.value as BasemapKey)}>{Object.entries(BASEMAPS).map(([key, option]) => <option key={key} value={key}>{option.label}</option>)}</select></label>
        <div className="map-zoom-control"><button type="button" className="btn btn-sm btn-light-primary" aria-label="Perbesar peta" onClick={() => setZoom((value) => Math.min(19, value + 1))}>+</button><input aria-label="Zoom peta" type="range" min="1" max="19" value={zoom} onChange={(event) => setZoom(Number(event.target.value))} /><button type="button" className="btn btn-sm btn-light-primary" aria-label="Perkecil peta" onClick={() => setZoom((value) => Math.max(1, value - 1))}>−</button></div>
      </div>
      {mode === "settings" && <aside className="hcv-coordinate-panel">
        <h3 className="fs-5 fw-bold">Form Edit Coordinate</h3>
        {mapUpload && <small className="d-block text-muted mb-2">Sumber file: {mapUpload.split("/").pop()}</small>}
        <label className="form-label fw-semibold" htmlFor="hcv-csv-file">Upload CSV / GeoJSON</label>
        <input id="hcv-csv-file" className="form-control form-control-sm mb-3" type="file" accept=".csv,.json,.geojson,.wkt,.txt" disabled={loading} onChange={(event) => void handleFile(event)} />
        <label className="form-label fw-semibold" htmlFor="hcv-mapinfo-file">Upload MapInfo</label>
        <input id="hcv-mapinfo-file" className="form-control form-control-sm mb-3" type="file" accept=".mif" disabled={loading} onChange={(event) => void handleFile(event)} />
        <div className="d-flex justify-content-between border-bottom pb-2"><strong>Coordinates</strong><span className="text-muted">{coordinates.length} points</span></div>
        <div className="hcv-coordinate-list">{coordinates.length === 0 ? <p className="text-muted p-3">Belum ada koordinat HCV.</p> : coordinates.slice(0, 300).map(([lng, lat], index) => <div className="map-coordinate-row" key={`${index}-${lng}-${lat}`}><span className="text-muted">{index + 1}</span><input className="form-control form-control-sm" aria-label={`Longitude titik ${index + 1}`} value={lng} readOnly /><input className="form-control form-control-sm" aria-label={`Latitude titik ${index + 1}`} value={lat} readOnly /></div>)}{coordinates.length > 300 && <small className="text-muted">Menampilkan 300 dari {coordinates.length} titik.</small>}</div>
        <div className="border-top pt-3 mt-2"><strong>Add Point</strong><div className="d-flex gap-2 mt-2"><input className="form-control form-control-sm" type="number" step="any" placeholder="Longitude" aria-label="Longitude" value={point.lng} onChange={(event) => setPoint({ ...point, lng: event.target.value })} /><input className="form-control form-control-sm" type="number" step="any" placeholder="Latitude" aria-label="Latitude" value={point.lat} onChange={(event) => setPoint({ ...point, lat: event.target.value })} /><button type="button" className="btn btn-sm btn-light-primary" onClick={addPoint}>Add</button></div></div>
        <button type="button" className="btn btn-primary w-100 mt-4" onClick={() => void saveMap()} disabled={saving || loading}>{saving ? "Menyimpan..." : "Simpan peta"}</button>
      </aside>}
    </div>
  </div>;
}
