"use client";

import TableLoadingRows from "../../../components/ui/table-loading-rows";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { ChangeEvent, FormEvent } from "react";
import { useUserProfile } from "../../../components/auth/user-profile-context";

type Permit = {
  id: number; company_id: number; company_name: string; permit_number: string; permit_date: string;
  province_id: number | null; province: string; address: string; permit_agency: string; agency_phone: string;
  agency_address: string; signatory_name: string; signatory_position: string; signatory_place: string;
  permit_document: string; map_data: string; map_file: string;
};
type Option = { id: number; name: string };
type Options = { companies: Option[]; provinces: Option[] };
type APIEnvelope<T> = { message?: string; data?: T };
type LeafletBounds = { isValid: () => boolean };
export type LeafletFeatureLayer = { bindPopup: (content: HTMLElement) => unknown; on: (event: string, handler: () => void) => unknown; setStyle: (style: Record<string, unknown>) => unknown; bringToFront: () => unknown };
type LeafletLayer = { addTo: (map: LeafletMap) => LeafletLayer; getBounds: () => LeafletBounds; on: (event: string, handler: () => void) => LeafletLayer; bringToFront: () => unknown };
type LeafletMap = {
  setView: (center: [number, number], zoom: number) => LeafletMap;
  setZoom: (zoom: number) => LeafletMap;
  getZoom: () => number;
  fitBounds: (bounds: LeafletBounds, options?: { padding: [number, number]; maxZoom?: number }) => LeafletMap;
  invalidateSize: (options?: { pan?: boolean }) => LeafletMap;
  on: (event: string, handler: () => void) => LeafletMap;
  removeLayer: (layer: LeafletLayer) => LeafletMap;
  remove: () => void;
};
type LeafletAPI = {
  map: (element: HTMLElement, options?: Record<string, unknown>) => LeafletMap;
  tileLayer: (url: string, options: Record<string, unknown>) => LeafletLayer;
  geoJSON: (data: unknown, options?: { style?: (feature?: unknown) => Record<string, unknown>; onEachFeature?: (feature: unknown, layer: LeafletFeatureLayer) => void; pointToLayer?: (_feature: unknown, latlng: unknown) => unknown }) => LeafletLayer;
  circleMarker: (latlng: unknown, options: Record<string, unknown>) => unknown;
};
declare global { interface Window { leaflet?: LeafletAPI } }
export type BasemapKey = "osm" | "esri-street" | "esri-topo" | "esri-imagery";
export const BASEMAPS: Record<BasemapKey, { label: string; url: string; attribution: string; maxZoom: number }> = {
  osm: { label: "OpenStreetMap", url: "https://tile.openstreetmap.org/{z}/{x}/{y}.png", attribution: '&copy; <a href="https://www.openstreetmap.org/copyright" target="_blank" rel="noreferrer">OpenStreetMap</a> contributors', maxZoom: 19 },
  "esri-street": { label: "Esri Street", url: "https://services.arcgisonline.com/ArcGIS/rest/services/World_Street_Map/MapServer/tile/{z}/{y}/{x}", attribution: 'Sources: <a href="https://www.esri.com/" target="_blank" rel="noreferrer">Esri</a>, DeLorme, HERE, USGS, Intermap, iPC, NRCAN, Esri Japan, METI, Esri China (Hong Kong), Esri (Thailand), MapmyIndia, TomTom', maxZoom: 19 },
  "esri-topo": { label: "Esri Topographic", url: "https://services.arcgisonline.com/ArcGIS/rest/services/World_Topo_Map/MapServer/tile/{z}/{y}/{x}", attribution: 'Sources: <a href="https://www.esri.com/" target="_blank" rel="noreferrer">Esri</a>, DeLorme, HERE, TomTom, Intermap, increment P Corp., GEBCO, USGS, FAO, NPS, NRCAN, GeoBase, IGN, Kadaster NL, Ordnance Survey, Esri Japan, METI, Esri China (Hong Kong), swisstopo, MapmyIndia, GIS User Community', maxZoom: 19 },
  "esri-imagery": { label: "Esri Imagery", url: "https://services.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}", attribution: 'Sources: <a href="https://www.esri.com/" target="_blank" rel="noreferrer">Esri</a>, DigitalGlobe, GeoEye, i-cubed, USDA FSA, USGS, AEX, Getmapping, Aerogrid, IGN, IGP, swisstopo, GIS User Community', maxZoom: 19 },
};
type Form = Omit<Permit, "id" | "company_name" | "province" | "permit_document" | "map_file"> & { permit_document: string; map_file: string };
const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");
const blankForm = (): Form => ({ company_id: 0, permit_number: "", permit_date: "", province_id: null, address: "", permit_agency: "", agency_phone: "", agency_address: "", signatory_name: "", signatory_position: "", signatory_place: "", permit_document: "", map_data: "", map_file: "" });

function normalizePermit(value: unknown): Permit | null {
  if (!value || typeof value !== "object") return null;
  const row = value as Record<string, unknown>;
  const text = (field: string) => typeof row[field] === "string" || typeof row[field] === "number" ? String(row[field]) : "";
  const mapData = row.map_data;
  return {
    id: Number(row.id) || 0,
    company_id: Number(row.company_id) || 0,
    company_name: text("company_name"),
    permit_number: text("permit_number"),
    permit_date: text("permit_date"),
    province_id: row.province_id === null || row.province_id === undefined || row.province_id === "" ? null : Number(row.province_id) || null,
    province: text("province"), address: text("address"), permit_agency: text("permit_agency"),
    agency_phone: text("agency_phone"), agency_address: text("agency_address"),
    signatory_name: text("signatory_name"), signatory_position: text("signatory_position"), signatory_place: text("signatory_place"),
    permit_document: text("permit_document"), map_data: typeof mapData === "string" ? mapData : mapData ? JSON.stringify(mapData) : "", map_file: text("map_file"),
  };
}

function formatPermitDate(value: string): string {
  if (!value) return "—";
  const normalized = /^\d{4}-\d{2}-\d{2}$/.test(value) ? `${value}T00:00:00` : value;
  const date = new Date(normalized);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleDateString("id-ID");
}
function permitMapFileURL(value: string): string {
  const path = value.trim();
  if (!path) return "";
  if (/^https?:\/\//i.test(path)) return path;
  const fileName = path.split(/[?#]/, 1)[0].split(/[\\/]/).pop();
  return fileName ? `${apiBase}/api/land-permit-files/${encodeURIComponent(fileName)}` : "";
}
export function mapFileTextToGeoJSON(fileName: string, text: string): string {
  if (/\.(geojson|json)(?:$|[?#])/i.test(fileName)) {
    return spatialTextToGeoJSON(text);
  }
  if (/\.(wkt|txt)(?:$|[?#])/i.test(fileName)) return spatialTextToGeoJSON(text);
  if (/\.csv(?:$|[?#])/i.test(fileName)) {
    const lines = text.split(/\r?\n/).filter((line) => line.trim());
    if (lines.length < 2) throw new Error("CSV harus memiliki header dan setidaknya satu baris koordinat.");
    const cells = (line: string) => line.split(",").map((cell) => cell.trim().replace(/^['\"]|['\"]$/g, ""));
    const headers = cells(lines[0]).map((header) => header.toLowerCase());
    const latIndex = headers.findIndex((header) => ["lat", "latitude", "y"].includes(header));
    const lngIndex = headers.findIndex((header) => ["lon", "lng", "longitude", "x"].includes(header));
    const features = lines.slice(1).flatMap((line, index) => {
      const row = cells(line);
      const lat = Number(row[latIndex >= 0 ? latIndex : 0]);
      const lng = Number(row[lngIndex >= 0 ? lngIndex : 1]);
      if (!Number.isFinite(lat) || !Number.isFinite(lng) || lat < -90 || lat > 90 || lng < -180 || lng > 180) return [];
      return [{ type: "Feature", properties: { name: `Point ${index + 1}` }, geometry: { type: "Point", coordinates: [lng, lat] } }];
    });
    if (!features.length) throw new Error("Koordinat CSV tidak valid. Gunakan kolom lat/lon atau latitude/longitude.");
    return JSON.stringify({ type: "FeatureCollection", features });
  }
  if (/\.mif(?:$|[?#])/i.test(fileName)) {
    const lines = text.split(/\r?\n/).map((line) => line.trim());
    const features: Array<{ type: string; properties: Record<string, string>; geometry: { type: string; coordinates: unknown } }> = [];
    const readCoords = (start: number, count: number) => {
      const coordinates: number[][] = [];
      for (let lineIndex = start; lineIndex < lines.length && coordinates.length < count; lineIndex += 1) {
        const cells = lines[lineIndex].split(/[\s,]+/).map(Number);
        if (cells.length >= 2 && Number.isFinite(cells[0]) && Number.isFinite(cells[1])) coordinates.push([cells[0], cells[1]]);
      }
      return coordinates;
    };
    for (let index = 0; index < lines.length; index += 1) {
      const pointMatch = lines[index].match(/^Point\s+([-+\d.eE]+)\s+([-+\d.eE]+)/i);
      if (pointMatch) {
        features.push({ type: "Feature", properties: {}, geometry: { type: "Point", coordinates: [Number(pointMatch[1]), Number(pointMatch[2])] } });
        continue;
      }
      const multipleMatch = lines[index].match(/^Pline\s+Multiple\s+(\d+)/i);
      if (multipleMatch) {
        let cursor = index + 1;
        for (let part = 0; part < Number(multipleMatch[1]); part += 1) {
          const count = Number(lines[cursor]);
          if (!Number.isFinite(count) || count <= 0) break;
          cursor += 1;
          const coordinates = readCoords(cursor, count);
          if (coordinates.length) features.push({ type: "Feature", properties: {}, geometry: { type: "LineString", coordinates } });
          cursor += count;
        }
        index = cursor - 1;
        continue;
      }
      const regionMatch = lines[index].match(/^Region\s+(\d+)/i);
      const lineMatch = lines[index].match(/^Pline\s+(\d+)/i);
      const match = regionMatch ?? lineMatch;
      if (!match) continue;
      let cursor = index + 1;
      const ringCount = Number(lines[cursor]);
      if (!Number.isFinite(ringCount) || ringCount <= 0) continue;
      cursor += 1;
      const rings: number[][][] = [];
      const first = readCoords(cursor, ringCount);
      if (first.length) rings.push(first);
      cursor += ringCount;
      if (regionMatch) {
        while (cursor < lines.length && /^\d+$/.test(lines[cursor])) {
          const holeCount = Number(lines[cursor]);
          cursor += 1;
          const hole = readCoords(cursor, holeCount);
          if (hole.length) rings.push(hole);
          cursor += holeCount;
        }
      }
      const coordinates = regionMatch ? rings : rings[0];
      if (coordinates?.length) features.push({ type: "Feature", properties: {}, geometry: { type: regionMatch ? "Polygon" : "LineString", coordinates } });
      index = cursor - 1;
    }
    if (!features.length) throw new Error("Tidak ditemukan geometri Point, Pline, atau Region di file MapInfo.");
    return JSON.stringify({ type: "FeatureCollection", features });
  }
  throw new Error("Format file peta tidak didukung untuk preview.");
}
export function coordinatePairs(value: string): Array<[number, number]> {
  try {
    const parsed = JSON.parse(value) as { type?: string; features?: Array<{ geometry?: { coordinates?: unknown; geometries?: Array<{ coordinates?: unknown }> } }>; geometry?: { coordinates?: unknown; geometries?: Array<{ coordinates?: unknown }> }; coordinates?: unknown } | unknown[];
    const object = Array.isArray(parsed) ? null : parsed as { type?: string; features?: Array<{ geometry?: { coordinates?: unknown; geometries?: Array<{ coordinates?: unknown }> } }>; geometry?: { coordinates?: unknown; geometries?: Array<{ coordinates?: unknown }> }; coordinates?: unknown };
    const geometries = Array.isArray(parsed) ? [parsed] : object?.type === "FeatureCollection" ? (object.features ?? []).map((feature) => feature.geometry).filter(Boolean) : [object?.type === "Feature" ? object.geometry : object];
    const pairs: Array<[number, number]> = [];
    const collect = (input: unknown) => {
      if (!Array.isArray(input)) return;
      if (input.length >= 2 && input.every((part) => typeof part === "number")) {
        for (let index = 0; index + 1 < input.length; index += 2) pairs.push([input[index] as number, input[index + 1] as number]);
        return;
      }
      if (input.length >= 2 && typeof input[0] === "number" && typeof input[1] === "number") { pairs.push([input[0], input[1]]); return; }
      input.forEach(collect);
    };
    for (const geometry of geometries) {
      if (!geometry) continue;
      if (Array.isArray(geometry)) collect(geometry);
      else {
        collect(geometry.coordinates);
        (geometry as { geometries?: Array<{ coordinates?: unknown }> }).geometries?.forEach((nested) => collect(nested.coordinates));
      }
    }
    return pairs;
  } catch { return []; }
}

export function spatialTextToGeoJSON(value: string): string {
  const raw = value.trim().replace(/^SRID=\d+;/i, "");
  try {
    const parsed = JSON.parse(raw) as { type?: string };
    if (typeof parsed === "string") return spatialTextToGeoJSON(parsed);
    if (parsed && typeof parsed === "object" && typeof parsed.type === "string") return JSON.stringify(parsed);
    const pairs = coordinatePairs(raw);
    if (pairs.length) {
      const closed = pairs.length > 3 && pairs[0][0] === pairs[pairs.length - 1][0] && pairs[0][1] === pairs[pairs.length - 1][1];
      const geometry = pairs.length === 1 ? { type: "Point", coordinates: pairs[0] } : closed ? { type: "Polygon", coordinates: [pairs] } : { type: "LineString", coordinates: pairs };
      return JSON.stringify({ type: "FeatureCollection", features: [{ type: "Feature", properties: {}, geometry }] });
    }
  } catch { /* WKT is handled below. */ }
  const match = raw.match(/^(POINT|LINESTRING|POLYGON|MULTILINESTRING|MULTIPOLYGON)\s*\(([\s\S]*)\)$/i);
  if (!match) throw new Error("Format koordinat tidak dikenali. Gunakan GeoJSON atau WKT.");
  const pairs = (input: string): Array<[number, number]> => input.split(",").map((part) => {
    const numbers = part.trim().split(/\s+/).map(Number);
    if (numbers.length < 2 || !Number.isFinite(numbers[0]) || !Number.isFinite(numbers[1])) throw new Error("Koordinat WKT tidak valid.");
    return [numbers[0], numbers[1]];
  });
  const groups = (input: string): string[] => {
    const result: string[] = [];
    let depth = 0, start = -1;
    for (let index = 0; index < input.length; index += 1) {
      if (input[index] === "(") { if (depth === 0) start = index + 1; depth += 1; }
      else if (input[index] === ")") { depth -= 1; if (depth === 0 && start >= 0) result.push(input.slice(start, index)); if (depth < 0) throw new Error("Kurung WKT tidak valid."); }
    }
    if (depth !== 0 || !result.length) throw new Error("Kurung WKT tidak valid.");
    return result;
  };
  let geometry: { type: string; coordinates: unknown };
  if (match[1].toUpperCase() === "POINT") geometry = { type: "Point", coordinates: pairs(match[2])[0] };
  else if (match[1].toUpperCase() === "LINESTRING") geometry = { type: "LineString", coordinates: pairs(match[2]) };
  else if (match[1].toUpperCase() === "POLYGON") geometry = { type: "Polygon", coordinates: groups(match[2]).map(pairs) };
  else if (match[1].toUpperCase() === "MULTILINESTRING") geometry = { type: "MultiLineString", coordinates: groups(match[2]).map(pairs) };
  else geometry = { type: "MultiPolygon", coordinates: groups(match[2]).map((polygon) => groups(polygon).map(pairs)) };
  return JSON.stringify({ type: "FeatureCollection", features: [{ type: "Feature", properties: {}, geometry }] });
}

function asGeoJSON(value: string): unknown {
  const parsed = JSON.parse(spatialTextToGeoJSON(value)) as { type?: string; features?: unknown[]; geometry?: unknown; coordinates?: unknown };
  if (["FeatureCollection", "Feature", "Point", "MultiPoint", "LineString", "MultiLineString", "Polygon", "MultiPolygon", "GeometryCollection"].includes(parsed.type ?? "")) return parsed;
  const coordinates = parsed.coordinates ?? parsed;
  const pairs = coordinatePairs(JSON.stringify(coordinates));
  if (!pairs.length) throw new Error("Tidak ada pasangan koordinat pada mapdata.");
  let geometry: { type: string; coordinates: unknown };
  if (pairs.length === 1) geometry = { type: "Point", coordinates: pairs[0] };
  else if (pairs.length > 3 && pairs[0][0] === pairs[pairs.length - 1][0] && pairs[0][1] === pairs[pairs.length - 1][1]) geometry = { type: "Polygon", coordinates: [pairs] };
  else geometry = { type: "LineString", coordinates: pairs };
  return { type: "FeatureCollection", features: [{ type: "Feature", properties: {}, geometry }] };
}

let leafletLoading: Promise<LeafletAPI> | null = null;

export function preloadLeaflet(): Promise<LeafletAPI> {
  const cssPath = "/metronic/assets/plugins/custom/leaflet/leaflet.bundle.css";
  if (!document.querySelector(`link[href="${cssPath}"]`)) { const link = document.createElement("link"); link.rel = "stylesheet"; link.href = cssPath; document.head.appendChild(link); }
  if (window.leaflet) return Promise.resolve(window.leaflet);
  if (leafletLoading) return leafletLoading;
  leafletLoading = new Promise((resolve, reject) => {
    const scriptPath = "/metronic/assets/plugins/custom/leaflet/leaflet.bundle.js";
    const existing = document.querySelector<HTMLScriptElement>(`script[src="${scriptPath}"]`);
    const script = existing ?? document.createElement("script");
    const failed = (message: string) => { script.remove(); leafletLoading = null; reject(new Error(message)); };
    const loaded = () => window.leaflet ? resolve(window.leaflet) : failed("Library peta Leaflet tidak tersedia.");
    script.addEventListener("load", loaded, { once: true });
    script.addEventListener("error", () => failed("Library peta Leaflet gagal dimuat."), { once: true });
    if (!existing) { script.src = scriptPath; script.async = true; document.body.appendChild(script); }
  });
  return leafletLoading;
}

export function LeafletMapView({ value, zoom, onZoomChange, basemap, emptyMessage = "Menunggu mapdata atau file GeoJSON/JSON", featureStyle, focusKind, backgroundValue, backgroundStyle, onEachFeature }: { value: string; zoom: number; onZoomChange: (zoom: number) => void; basemap: BasemapKey; emptyMessage?: string; featureStyle?: (feature?: unknown) => Record<string, unknown>; focusKind?: string; backgroundValue?: string; backgroundStyle?: (feature?: unknown) => Record<string, unknown>; onEachFeature?: (feature: unknown, layer: LeafletFeatureLayer) => void }) {
  const elementRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<LeafletMap | null>(null);
  const layerRef = useRef<LeafletLayer | null>(null);
  const backgroundRef = useRef<LeafletLayer | null>(null);
  const baseLayerRef = useRef<LeafletLayer | null>(null);
  const onZoomChangeRef = useRef(onZoomChange);
  const lastZoomAppliedRef = useRef(zoom);
  const [ready, setReady] = useState(false);
  const [mapError, setMapError] = useState("");
  const [tileError, setTileError] = useState(false);
  const [tileStatus, setTileStatus] = useState<"loading" | "loaded" | "timeout">("loading");
  const tileLoadedRef = useRef(false);
  useEffect(() => { onZoomChangeRef.current = onZoomChange; }, [onZoomChange]);
  useEffect(() => {
    let active = true;
    let resizeTimer: ReturnType<typeof setTimeout>;
    let resizeObserver: ResizeObserver | undefined;
    let invalidateMapSize = () => {};
    void preloadLeaflet().then((L) => {
      if (!active || !elementRef.current) return;
      const container = elementRef.current;
      const map = L.map(container, { zoomControl: false, scrollWheelZoom: true, preferCanvas: true, fadeAnimation: false, zoomAnimation: false, markerZoomAnimation: false }).setView([-2, 118], 5);
      lastZoomAppliedRef.current = 5;
      onZoomChangeRef.current(5);
      map.on("zoomend", () => onZoomChangeRef.current(map.getZoom()));
      mapRef.current = map;
      setReady(true);
      // The modal and its resizable coordinate panel can change the map's
      // dimensions after Leaflet has measured it. Keep Leaflet's cached size
      // in sync so tile requests and the vector layer don't disappear after
      // the initial paint.
      resizeObserver = new ResizeObserver(() => {
        if (!active || mapRef.current !== map) return;
        requestAnimationFrame(() => {
          if (!active || mapRef.current !== map) return;
          map.invalidateSize({ pan: false });
        });
      });
      resizeObserver.observe(container);
      invalidateMapSize = () => {
        if (!active || mapRef.current !== map) return;
        map.invalidateSize({ pan: false });
      };
      window.addEventListener("resize", invalidateMapSize);
      resizeTimer = setTimeout(() => {
        map.invalidateSize({ pan: false });
      }, 150);
    }).catch((error: unknown) => { if (active) setMapError(error instanceof Error ? error.message : "Peta gagal dimuat."); });
    return () => {
      active = false;
      clearTimeout(resizeTimer);
      resizeObserver?.disconnect();
      window.removeEventListener("resize", invalidateMapSize);
      mapRef.current?.remove(); mapRef.current = null; layerRef.current = null; baseLayerRef.current = null; backgroundRef.current = null;
    };
  }, []);
  useEffect(() => {
    const map = mapRef.current, L = window.leaflet;
    if (!ready || !map || !L) return;
    if (baseLayerRef.current) { map.removeLayer(baseLayerRef.current); baseLayerRef.current = null; }
    const config = BASEMAPS[basemap];
    tileLoadedRef.current = false;
    setTileStatus("loading");
    setTileError(false);
    const tileTimer = setTimeout(() => {
      if (!tileLoadedRef.current) setTileStatus("timeout");
    }, 12000);
    const baseLayer = L.tileLayer(config.url, { maxZoom: config.maxZoom, attribution: config.attribution })
      .on("tileload", () => { tileLoadedRef.current = true; setTileStatus("loaded"); })
      .on("tileerror", () => setTileError(true))
      .addTo(map);
    baseLayerRef.current = baseLayer;
    return () => {
      clearTimeout(tileTimer);
      if (mapRef.current === map) map.removeLayer(baseLayer);
      if (baseLayerRef.current === baseLayer) baseLayerRef.current = null;
    };
  }, [basemap, ready]);
  useEffect(() => {
    const map = mapRef.current, L = window.leaflet;
    if (!ready || !map || !L) return;
    if (layerRef.current) { map.removeLayer(layerRef.current); layerRef.current = null; }
    setMapError("");
    if (!value.trim()) return;
    try {
      const data = asGeoJSON(value);
      const style = featureStyle ?? (() => ({ color: "#d62828", weight: 3, opacity: 0.95, fillColor: "#e63946", fillOpacity: 0.3 }));
      const pointToLayer = (_feature: unknown, latlng: unknown) => L.circleMarker(latlng, { radius: 7, color: "#fff", weight: 2, fillColor: "#d62828", fillOpacity: 1 });
      let layer = L.geoJSON(data, { style, pointToLayer, onEachFeature }).addTo(map);
      let bounds = layer.getBounds();
      if (focusKind && data && typeof data === "object" && "features" in data && Array.isArray(data.features)) {
        const matching = data.features.filter((feature: unknown) => (feature as { properties?: { kind?: string } })?.properties?.kind === focusKind);
        if (matching.length) {
          const focusBounds = L.geoJSON({ type: "FeatureCollection", features: matching }).getBounds();
          if (focusBounds.isValid()) bounds = focusBounds;
        }
      }
      if (!bounds.isValid()) {
        map.removeLayer(layer);
        const pairs = coordinatePairs(value);
        if (pairs.length) {
          const closed = pairs.length > 3 && pairs[0][0] === pairs[pairs.length - 1][0] && pairs[0][1] === pairs[pairs.length - 1][1];
          const fallbackGeometry = pairs.length === 1
            ? { type: "Point", coordinates: pairs[0] }
            : closed ? { type: "Polygon", coordinates: [pairs] } : { type: "LineString", coordinates: pairs };
          layer = L.geoJSON({ type: "Feature", properties: {}, geometry: fallbackGeometry }, { style, pointToLayer }).addTo(map);
          bounds = layer.getBounds();
        }
      }
      layerRef.current = layer;
      if (bounds.isValid()) {
        requestAnimationFrame(() => {
          if (mapRef.current !== map) return;
          map.invalidateSize({ pan: false });
          map.fitBounds(bounds, { padding: [36, 36], maxZoom: 17 });
          onZoomChangeRef.current(map.getZoom());
        });
      } else {
        setMapError("Koordinat terbaca, tetapi bentuk geometri tidak dikenali untuk digambar.");
      }
    } catch (error) { setMapError(error instanceof Error ? error.message : "GeoJSON tidak dapat ditampilkan."); }
  }, [ready, value, featureStyle, focusKind, onEachFeature]);
  useEffect(() => {
    const map = mapRef.current, L = window.leaflet;
    if (!ready || !map || !L || !backgroundValue) return;
    const layer = L.geoJSON(JSON.parse(backgroundValue), {
      style: backgroundStyle, onEachFeature,
      pointToLayer: (feature, latlng) => L.circleMarker(latlng, { radius: 6, ...backgroundStyle?.(feature) }),
    }).addTo(map);
    backgroundRef.current = layer;
    layerRef.current?.bringToFront();
    return () => {
      if (mapRef.current === map) map.removeLayer(layer);
      if (backgroundRef.current === layer) backgroundRef.current = null;
    };
  }, [ready, backgroundValue, backgroundStyle, onEachFeature]);
  useEffect(() => {
    if (!ready || zoom === lastZoomAppliedRef.current) return;
    mapRef.current?.setZoom(zoom);
    lastZoomAppliedRef.current = zoom;
  }, [ready, zoom]);
  return <div className="leaflet-map-shell">
    <div className="leaflet-map" ref={elementRef} />
    {!value.trim() && !mapError && <div className="map-empty-hint">{emptyMessage}</div>}
    {mapError && <div className="map-error-hint">{mapError}</div>}
    {tileError && <div className="map-tile-warning">Sebagian tile {BASEMAPS[basemap].label} gagal dimuat. Periksa koneksi atau pemblokiran jaringan.</div>}
    {tileStatus === "timeout" && !tileError && <div className="map-tile-warning">Basemap {BASEMAPS[basemap].label} belum menerima tile. Periksa request tile di tab Network browser.</div>}
  </div>;
}

async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  if (init?.body && !(init.body instanceof FormData) && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  const response = await fetch(`${apiBase}${path}`, { ...init, credentials: "include", cache: "no-store", headers });
  const result = (await response.json().catch(() => null)) as APIEnvelope<T> | null;
  if (!response.ok) throw new Error(result?.message || "Permintaan gagal. Silakan coba lagi.");
  return result?.data as T;
}
function EditIcon() { return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m14 5 5 5M4 20l4.5-1 10.8-10.8a2.12 2.12 0 0 0-3-3L5.5 16 4 20Z" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>; }
function TrashIcon() { return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M4 7h16M10 11v6m4-6v6M6 7l1 14h10l1-14M9 7V4h6v3" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" /></svg>; }
function MapIcon() { return <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m3 6 6-3 6 3 6-3v15l-6 3-6-3-6 3V6Zm6-3v15m6-12v15" stroke="currentColor" strokeWidth="1.8" strokeLinejoin="round" /></svg>; }
function toForm(item: Permit): Form { return { company_id: item.company_id, permit_number: item.permit_number, permit_date: item.permit_date?.slice(0, 10) ?? "", province_id: item.province_id, address: item.address, permit_agency: item.permit_agency, agency_phone: item.agency_phone, agency_address: item.agency_address, signatory_name: item.signatory_name, signatory_position: item.signatory_position, signatory_place: item.signatory_place, permit_document: item.permit_document, map_data: item.map_data, map_file: item.map_file }; }

function PermitTableRow({ item, companyName, onMap, onEdit, onDelete }: { item: Permit; companyName: string; onMap: (item: Permit) => void; onEdit: (item: Permit) => void; onDelete: (item: Permit) => void }) {
  const permitNumber = typeof item.permit_number === "string" ? item.permit_number : "";
  const documentPath = typeof item.permit_document === "string" ? item.permit_document : "";
  const province = typeof item.province === "string" ? item.province : "";
  const displayedCompany = typeof item.company_name === "string" ? item.company_name : companyName;
  return <tr>
    <td className="text-gray-900 fw-bold">{permitNumber || "—"}</td>
    <td>{formatPermitDate(typeof item.permit_date === "string" ? item.permit_date : "")}</td>
    <td>{displayedCompany || "—"}</td>
    <td>{province || "—"}</td>
    <td>{documentPath ? <a href={`${apiBase}${documentPath}`} target="_blank" rel="noreferrer">Lihat file</a> : "—"}</td>
    <td><div className="admin-table-actions">
      <button type="button" className="btn btn-light-info" title="Map / set koordinat" aria-label={`Atur peta ${permitNumber}`} onClick={() => onMap(item)}><MapIcon /></button>
      <button type="button" className="btn btn-light-primary" title="Edit" aria-label={`Edit ${permitNumber}`} onClick={() => onEdit(item)}><EditIcon /></button>
      <button type="button" className="btn btn-light-danger" title="Hapus" aria-label={`Hapus ${permitNumber}`} onClick={() => onDelete(item)}><TrashIcon /></button>
    </div></td>
  </tr>;
}

function GeoJSONPreview({ value, large = false, zoom = 1 }: { value: string; large?: boolean; zoom?: number }) {
  const paths = useMemo(() => {
    if (!value.trim()) return [] as Array<{ kind: "point" | "path"; points: Array<[number, number]> }>;
    try {
      const parsed = JSON.parse(value) as { type?: string; features?: Array<{ geometry?: { type?: string; coordinates?: unknown; geometries?: Array<{ type?: string; coordinates?: unknown }> } }>; geometry?: { type?: string; coordinates?: unknown; geometries?: Array<{ type?: string; coordinates?: unknown }> }; coordinates?: unknown };
      const features = parsed.type === "FeatureCollection" ? parsed.features ?? [] : [parsed.type === "Feature" ? { geometry: parsed.geometry } : { geometry: parsed }];
      const output: Array<{ kind: "point" | "path"; points: Array<[number, number]> }> = [];
      const getCoordinates = (input: unknown, found: Array<[number, number]> = []): Array<[number, number]> => {
        if (!Array.isArray(input)) return found;
        if (input.length >= 2 && typeof input[0] === "number" && typeof input[1] === "number") found.push([input[0], input[1]]);
        else input.forEach((child) => getCoordinates(child, found));
        return found;
      };
      for (const feature of features) {
        const geometry = feature.geometry;
        if (!geometry) continue;
        if (geometry.type === "GeometryCollection") {
          for (const nested of (geometry as { geometries?: Array<{ type?: string; coordinates?: unknown }> }).geometries ?? []) {
            const nestedPoints = getCoordinates(nested.coordinates);
            if (nestedPoints.length) output.push({ kind: nested.type === "Point" ? "point" : "path", points: nestedPoints });
          }
          continue;
        }
        const points = getCoordinates(geometry.coordinates);
        if (points.length) output.push({ kind: geometry.type === "Point" ? "point" : "path", points });
      }
      return output;
    } catch { return []; }
  }, [value]);
  const coords = paths.flatMap((shape) => shape.points);
  if (!coords.length) return <div className="d-flex align-items-center justify-content-center bg-light rounded border text-muted" style={{ height: large ? "100%" : 250, minHeight: large ? 500 : undefined }}>Belum ada geometri GeoJSON untuk ditampilkan.</div>;
  const lngs = coords.map(([lng]) => lng), lats = coords.map(([, lat]) => lat);
  const minLng = Math.min(...lngs), maxLng = Math.max(...lngs), minLat = Math.min(...lats), maxLat = Math.max(...lats);
  const scale = Math.min(752 / (maxLng - minLng || 1), 432 / (maxLat - minLat || 1));
  const centerLng = (minLng + maxLng) / 2, centerLat = (minLat + maxLat) / 2;
  const project = ([lng, lat]: [number, number]) => [400 + (lng - centerLng) * scale * zoom, 240 - (lat - centerLat) * scale * zoom] as const;
  return <svg viewBox="0 0 800 480" preserveAspectRatio="xMidYMid meet" role="img" aria-label="Preview bentuk geometri permit" className="w-100 rounded border bg-light" style={{ height: large ? "100%" : 250, minHeight: large ? 500 : undefined }}>
    <defs><pattern id="map-grid" width="40" height="40" patternUnits="userSpaceOnUse"><path d="M 40 0 L 0 0 0 40" fill="none" stroke="#dce3e8" strokeWidth="1" /></pattern></defs><rect width="800" height="480" fill="url(#map-grid)" />
    {paths.map((shape, index) => shape.kind === "point" ? shape.points.map((point, pointIndex) => { const [x, y] = project(point); return <g key={`${index}-${pointIndex}`}><circle cx={x} cy={y} r="8" fill="#dc3545" stroke="white" strokeWidth="3" /><circle cx={x} cy={y} r="13" fill="none" stroke="#dc3545" strokeOpacity=".4" /></g>; }) : <polyline key={index} points={shape.points.map((point) => project(point).join(",")).join(" ")} fill={shape.points.length > 2 ? "rgba(25,135,84,.18)" : "none"} stroke="#198754" strokeWidth="4" strokeLinejoin="round" />)}
    <text x="16" y="466" fontSize="12" fill="#667085">{coords.length} koordinat</text>
  </svg>;
}

export default function LandInfoPage() {
  const profile = useUserProfile();
  const [companyID, setCompanyID] = useState(0);
  const [companyName, setCompanyName] = useState("");
  const [items, setItems] = useState<Permit[]>([]);
  const [options, setOptions] = useState<Options>({ companies: [], provinces: [] });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [pageSize, setPageSize] = useState(25);
  const [page, setPage] = useState(1);
  const [modal, setModal] = useState<"form" | "map" | null>(null);
  const [editing, setEditing] = useState<Permit | null>(null);
  const [form, setForm] = useState<Form>(blankForm());
  const [mapPoint, setMapPoint] = useState({ lat: "", lng: "" });
  const [permitFile, setPermitFile] = useState<File | null>(null);
  const [mapFile, setMapFile] = useState<File | null>(null);
  const [mapMeasurement, setMapMeasurement] = useState(false);
  const [mapZoom, setMapZoom] = useState(1);
  const [basemap, setBasemap] = useState<BasemapKey>("osm");
  const [mapPanelWidth, setMapPanelWidth] = useState(360);
  const [draggingMapSplitter, setDraggingMapSplitter] = useState(false);
  const requestSequence = useRef(0);

  useEffect(() => {
    const storageKey = `gis.selected-company.${profile.username.toLowerCase()}`;
    const current = () => {
      const id = Number(window.localStorage.getItem(storageKey));
      const option = options.companies.find((company) => company.id === id);
      setCompanyID(Number.isFinite(id) ? id : 0);
      if (option) setCompanyName(option.name);
    };
    const onChange = (event: Event) => {
      const detail = (event as CustomEvent<{ id?: number; name?: string }>).detail;
      if (detail?.id) { setCompanyID(detail.id); setCompanyName(detail.name ?? ""); }
      else current();
    };
    current();
    window.addEventListener("gis:company-change", onChange);
    window.addEventListener("storage", current);
    return () => { window.removeEventListener("gis:company-change", onChange); window.removeEventListener("storage", current); };
  }, [options.companies, profile.username]);

  const loadOptions = useCallback(async () => {
    try { const data = await apiRequest<Options>("/api/land-permits/options"); setOptions({ companies: data?.companies ?? [], provinces: data?.provinces ?? [] }); }
    catch (e) { setError(e instanceof Error ? e.message : "Pilihan form gagal dimuat."); }
  }, []);
  useEffect(() => { void loadOptions(); }, [loadOptions]);
  useEffect(() => {
    if (modal !== "map" || !editing || form.map_data.trim() || !editing.map_file || !/\.(geojson|json|csv|mif)(?:$|[?#])/i.test(editing.map_file)) return;
    let active = true;
    const fileURL = permitMapFileURL(editing.map_file);
    if (!fileURL) return;
    fetch(fileURL, { credentials: "include", cache: "no-store" })
      .then(async (response) => {
        if (!response.ok) throw new Error("File peta tidak dapat dibaca.");
        const text = await response.text();
        const mapData = mapFileTextToGeoJSON(editing.map_file, text);
        if (active) setForm((current) => ({ ...current, map_data: mapData }));
      })
      .catch((e) => {
        if (!active) return;
        if (editing.map_data.trim()) setForm((current) => ({ ...current, map_data: current.map_data || editing.map_data }));
        else setError(e instanceof Error ? e.message : "File peta gagal dibaca.");
      });
    return () => { active = false; };
  }, [editing, form.map_data, modal]);
  useEffect(() => {
    if (!draggingMapSplitter) return;
    const move = (event: PointerEvent) => setMapPanelWidth(Math.max(320, Math.min(760, window.innerWidth - event.clientX)));
    const stop = () => setDraggingMapSplitter(false);
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", stop, { once: true });
    return () => { window.removeEventListener("pointermove", move); window.removeEventListener("pointerup", stop); };
  }, [draggingMapSplitter]);
  useEffect(() => {
    const company = options.companies.find((option) => option.id === companyID);
    if (company) setCompanyName(company.name);
  }, [companyID, options.companies]);
  const loadItems = useCallback(async () => {
    const sequence = ++requestSequence.current;
    if (!companyID) { setItems([]); setLoading(false); return; }
    setLoading(true); setError("");
    try { const data = await apiRequest<unknown>(`/api/land-permits?company_id=${companyID}`); if (sequence === requestSequence.current) setItems(Array.isArray(data) ? data.map(normalizePermit).filter((item): item is Permit => item !== null) : []); }
    catch (e) { if (sequence === requestSequence.current) { setItems([]); setError(e instanceof Error ? e.message : "Data land permit gagal dimuat."); } }
    finally { if (sequence === requestSequence.current) setLoading(false); }
  }, [companyID]);
  useEffect(() => { void loadItems(); }, [loadItems]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return q ? items.filter((item) => [item.permit_number, item.permit_date, item.company_name, item.province, item.permit_document].some((value) => String(value ?? "").toLowerCase().includes(q))) : items;
  }, [items, query]);
  const pageCount = Math.max(1, Math.ceil(filtered.length / pageSize));
  const currentPage = Math.min(page, pageCount);
  const shown = filtered.slice((currentPage - 1) * pageSize, currentPage * pageSize);

  function resetForm() { setEditing(null); setForm(blankForm()); setPermitFile(null); setMapFile(null); }
  function openCreate() { if (!companyID) { setError("Pilih company aktif terlebih dahulu."); return; } resetForm(); setForm({ ...blankForm(), company_id: companyID }); setError(""); setModal("form"); }
  function openEdit(item: Permit) { setEditing(item); setForm(toForm(item)); setPermitFile(null); setMapFile(null); setError(""); setModal("form"); }
  function openMapWorkspace(item: Permit) {
    const fileFirst = /\.(geojson|json|csv|mif)(?:$|[?#])/i.test(item.map_file);
    setEditing(item);
    setForm({ ...toForm(item), map_data: fileFirst ? "" : item.map_data });
    setPermitFile(null); setMapFile(null); setError(""); setMapZoom(5); setModal("map"); setMapMeasurement(false);
  }
  function closeModal() { if (!saving) { setModal(null); resetForm(); setError(""); } }
  function updateField<K extends keyof Form>(key: K, value: Form[K]) { setForm((current) => ({ ...current, [key]: value })); }
  async function handleMapFile(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0] ?? null; setMapFile(file);
    if (!file) return;
    if (/\.(geojson|json)$/i.test(file.name)) {
      try { const text = await file.text(); JSON.parse(text); setForm((current) => ({ ...current, map_data: text })); }
      catch { setError("File GeoJSON/JSON tidak valid."); }
    } else if (/\.csv$/i.test(file.name)) {
      const lines = (await file.text()).split(/\r?\n/).filter((line) => line.trim());
      if (lines.length < 2) { setError("CSV harus memiliki header dan setidaknya satu baris koordinat."); return; }
      const cells = (line: string) => line.split(",").map((cell) => cell.trim().replace(/^['\"]|['\"]$/g, ""));
      const headers = cells(lines[0]).map((header) => header.toLowerCase());
      const latIndex = headers.findIndex((header) => ["lat", "latitude", "y"].includes(header));
      const lngIndex = headers.findIndex((header) => ["lon", "lng", "longitude", "x"].includes(header));
      const features = lines.slice(1).flatMap((line, index) => {
        const row = cells(line);
        const lat = Number(row[latIndex >= 0 ? latIndex : 0]);
        const lng = Number(row[lngIndex >= 0 ? lngIndex : 1]);
        if (!Number.isFinite(lat) || !Number.isFinite(lng) || lat < -90 || lat > 90 || lng < -180 || lng > 180) return [];
        return [{ type: "Feature", properties: { name: `Point ${index + 1}` }, geometry: { type: "Point", coordinates: [lng, lat] } }];
      });
      if (!features.length) { setError("Koordinat CSV tidak valid. Gunakan kolom lat/lon atau latitude/longitude."); return; }
      setForm((current) => ({ ...current, map_data: JSON.stringify({ type: "FeatureCollection", features }, null, 2) })); setError("");
    } else if (/\.mif$/i.test(file.name)) {
      const lines = (await file.text()).split(/\r?\n/).map((line) => line.trim());
      const features: Array<{ type: string; properties: Record<string, string>; geometry: { type: string; coordinates: unknown } }> = [];
      const readCoords = (start: number, count: number) => {
        const coordinates: number[][] = [];
        for (let lineIndex = start; lineIndex < lines.length && coordinates.length < count; lineIndex += 1) {
          const cells = lines[lineIndex].split(/[\s,]+/).map(Number);
          if (cells.length >= 2 && Number.isFinite(cells[0]) && Number.isFinite(cells[1])) coordinates.push([cells[0], cells[1]]);
        }
        return coordinates;
      };
      for (let index = 0; index < lines.length; index += 1) {
        const pointMatch = lines[index].match(/^Point\s+([-+\d.eE]+)\s+([-+\d.eE]+)/i);
        if (pointMatch) {
          const lng = Number(pointMatch[1]), lat = Number(pointMatch[2]);
          features.push({ type: "Feature", properties: {}, geometry: { type: "Point", coordinates: [lng, lat] } });
          continue;
        }
        const multipleMatch = lines[index].match(/^Pline\s+Multiple\s+(\d+)/i);
        if (multipleMatch) {
          let cursor = index + 1;
          for (let part = 0; part < Number(multipleMatch[1]); part += 1) {
            const count = Number(lines[cursor]);
            if (!Number.isFinite(count) || count <= 0) break;
            cursor += 1;
            const coordinates = readCoords(cursor, count);
            if (coordinates.length) features.push({ type: "Feature", properties: {}, geometry: { type: "LineString", coordinates } });
            cursor += count;
          }
          index = cursor - 1;
          continue;
        }
        const regionMatch = lines[index].match(/^Region\s+(\d+)/i);
        const lineMatch = lines[index].match(/^Pline\s+(\d+)/i);
        const match = regionMatch ?? lineMatch;
        if (!match) continue;
        let cursor = index + 1;
        const ringCount = Number(lines[cursor]);
        if (!Number.isFinite(ringCount) || ringCount <= 0) continue;
        cursor += 1;
        const rings: number[][][] = [];
        const first = readCoords(cursor, ringCount); if (first.length) rings.push(first);
        cursor += ringCount;
        if (regionMatch) {
          while (cursor < lines.length && /^\d+$/.test(lines[cursor])) {
            const holeCount = Number(lines[cursor]); cursor += 1;
            const hole = readCoords(cursor, holeCount); if (hole.length) rings.push(hole);
            cursor += holeCount;
          }
        }
        const coordinates = regionMatch ? rings : rings[0];
        if (coordinates?.length) features.push({ type: "Feature", properties: {}, geometry: { type: regionMatch ? "Polygon" : "LineString", coordinates } });
        index = cursor - 1;
      }
      if (!features.length) { setError("Tidak ditemukan geometri Point, Pline, atau Region di file MapInfo."); return; }
      setForm((current) => ({ ...current, map_data: JSON.stringify({ type: "FeatureCollection", features }, null, 2) })); setError("");
    }
  }
  async function savePermit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setSaving(true); setError("");
    const payload = { ...form, company_id: companyID, province_id: form.province_id ? Number(form.province_id) : null };
    const body = new FormData(); body.set("data", JSON.stringify(payload)); if (permitFile) body.set("permit_document", permitFile); if (mapFile) body.set("map_file", mapFile);
    try { await apiRequest<Permit>(editing ? `/api/land-permits/${editing.id}?company_id=${companyID}` : "/api/land-permits", { method: editing ? "PUT" : "POST", body }); setModal(null); resetForm(); await loadItems(); }
    catch (e) { setError(e instanceof Error ? e.message : "Land permit gagal disimpan."); }
    finally { setSaving(false); }
  }
  async function saveMap(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); if (!editing) return; setSaving(true); setError("");
    const body = new FormData(); body.set("data", JSON.stringify({ ...toForm(editing), ...form, company_id: companyID, province_id: form.province_id ? Number(form.province_id) : null })); if (mapFile) body.set("map_file", mapFile);
    try {
      const result = await apiRequest<unknown>(`/api/land-permits/${editing.id}?company_id=${companyID}`, { method: "PUT", body });
      const updated = normalizePermit(result);
      if (updated) { setEditing(updated); setForm(toForm(updated)); }
      setModal(null); setMapFile(null); await loadItems();
    }
    catch (e) { setError(e instanceof Error ? e.message : "Data peta gagal disimpan."); }
    finally { setSaving(false); }
  }
  function addPoint() {
    if (!mapPoint.lat.trim() || !mapPoint.lng.trim()) { setError("Isi latitude dan longitude terlebih dahulu."); return; }
    const lat = Number(mapPoint.lat), lng = Number(mapPoint.lng);
    if (!Number.isFinite(lat) || !Number.isFinite(lng) || lat < -90 || lat > 90 || lng < -180 || lng > 180) { setError("Latitude harus -90..90 dan longitude -180..180."); return; }
    let collection: { type: string; features: unknown[] } = { type: "FeatureCollection", features: [] };
    try { if (form.map_data.trim()) { const parsed = JSON.parse(form.map_data); if (parsed.type === "FeatureCollection" && Array.isArray(parsed.features)) collection = parsed; else collection.features = [{ type: "Feature", properties: {}, geometry: parsed }]; } }
    catch { setError("GeoJSON belum valid. Perbaiki isinya sebelum menambah titik."); return; }
    collection.features.push({ type: "Feature", properties: { name: `Point ${collection.features.length + 1}` }, geometry: { type: "Point", coordinates: [lng, lat] } });
    setForm((current) => ({ ...current, map_data: JSON.stringify(collection, null, 2) })); setMapPoint({ lat: "", lng: "" }); setError("");
  }
  async function removePermit(item: Permit) {
    const confirmed = window.Swal ? (await window.Swal.fire({ title: "Hapus land permit?", text: `${item.permit_number} akan dihapus.`, icon: "warning", showCancelButton: true, confirmButtonText: "Ya, hapus", cancelButtonText: "Batal" })).isConfirmed : window.confirm(`Hapus permit ${item.permit_number}?`);
    if (!confirmed) return;
    try { await apiRequest<null>(`/api/land-permits/${item.id}?company_id=${companyID}`, { method: "DELETE" }); await loadItems(); }
    catch (e) { setError(e instanceof Error ? e.message : "Land permit gagal dihapus."); }
  }

  const formFields: Array<{ key: keyof Form; label: string; type?: string }> = [
    { key: "permit_number", label: "Permit number" }, { key: "permit_date", label: "Permit date", type: "date" },
    { key: "address", label: "Address" }, { key: "permit_agency", label: "Permit agency" }, { key: "agency_phone", label: "Agency phone" },
    { key: "agency_address", label: "Agency address" }, { key: "signatory_name", label: "Signature officer name" },
    { key: "signatory_position", label: "Signature officer position" }, { key: "signatory_place", label: "Signature place" },
  ];
  const mapCoordinates = coordinatePairs(form.map_data);

  return <div className="d-flex flex-column gap-6">
    <nav aria-label="Breadcrumb"><ol className="breadcrumb breadcrumb-separatorless fw-semibold fs-7 mb-0"><li className="breadcrumb-item"><Link href="/admin/dashboard" className="text-muted text-hover-primary">Admin</Link></li><li className="breadcrumb-item"><span className="bullet bg-gray-300 w-5px h-2px" /></li><li className="breadcrumb-item text-muted">Companies</li><li className="breadcrumb-item"><span className="bullet bg-gray-300 w-5px h-2px" /></li><li className="breadcrumb-item text-muted" aria-current="page">Land Permits</li></ol></nav>
    <div className="d-flex flex-wrap align-items-center justify-content-between gap-4"><div><h1 className="text-gray-900 fw-bolder fs-2 mb-1">Land Permits</h1><p className="text-muted fw-semibold mb-0">Kelola izin lahan untuk company aktif: <strong>{companyName || "Belum dipilih"}</strong></p></div><button type="button" className="btn btn-dark" onClick={openCreate}><span className="me-2 fs-3 lh-1">+</span>Add Land Permit</button></div>
    {error && !modal && <div className="alert alert-danger d-flex align-items-center justify-content-between mb-0" role="alert"><span>{error}</span><button type="button" className="btn-close" aria-label="Tutup pesan error" onClick={() => setError("")} /></div>}
    <section className="card border shadow-sm"><div className="card-header flex-wrap gap-4 py-4"><span className="text-muted fw-semibold fs-7">Total data: {items.length}</span><label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7 ms-auto">Search<input className="form-control form-control-sm form-control-solid ref-table-search" type="search" placeholder="Permit number, province..." value={query} onChange={(e) => { setQuery(e.target.value); setPage(1); }} /></label><label className="d-flex align-items-center gap-2 text-muted fw-semibold fs-7">Show<select className="form-select form-select-sm form-select-solid w-auto" value={pageSize} onChange={(e) => { setPageSize(Number(e.target.value)); setPage(1); }}><option>10</option><option>25</option><option>50</option><option>100</option></select>entries</label><button type="button" className="btn btn-icon btn-sm btn-light" onClick={() => void loadItems()} disabled={loading} aria-label="Muat ulang"><span className={loading ? "spinner-border spinner-border-sm" : "fs-4"}>{loading ? "" : "↻"}</span></button></div>
      <div className="card-body pt-0"><div className="table-responsive"><table className="table admin-data-table align-middle">
        <thead className="bg-dark"><tr className="text-white fw-bold text-uppercase fs-8"><th className="bg-dark text-white">Permit number</th><th className="bg-dark text-white">Permit date</th><th className="bg-dark text-white">Company name</th><th className="bg-dark text-white">Province</th><th className="bg-dark text-white">Permit doc</th><th className="bg-dark text-white text-end">Actions</th></tr></thead>
        <tbody>
          {loading ? <TableLoadingRows columns={6} />
            : !companyID ? <tr><td colSpan={6} className="text-center py-12 text-muted">Pilih company aktif di header untuk melihat data.</td></tr>
            : shown.length === 0 ? <tr><td colSpan={6} className="text-center py-12 text-muted">{query ? "Data tidak ditemukan." : "Belum ada land permit."}</td></tr>
            : shown.map((item, index) => <PermitTableRow key={item.id > 0 ? item.id : `${companyID}-${item.permit_number || "permit"}-${index}`} item={item} companyName={companyName} onMap={openMapWorkspace} onEdit={openEdit} onDelete={(row) => void removePermit(row)} />)}
        </tbody>
      </table></div></div>
      <div className="card-footer d-flex flex-wrap align-items-center justify-content-between gap-3 py-4"><span className="text-muted fw-semibold fs-7">Showing {filtered.length ? (currentPage - 1) * pageSize + 1 : 0}–{Math.min(currentPage * pageSize, filtered.length)} of {filtered.length} entries</span><div className="d-flex align-items-center gap-2"><button type="button" className="btn btn-sm btn-light" onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={currentPage <= 1}>Previous</button><span className="text-muted fw-semibold fs-7">{currentPage} / {pageCount}</span><button type="button" className="btn btn-sm btn-light" onClick={() => setPage((p) => Math.min(pageCount, p + 1))} disabled={currentPage >= pageCount}>Next</button></div></div>
    </section>

    {modal && <div className={`ref-modal-backdrop ${modal === "map" ? "ref-modal-backdrop--full" : ""}`} onMouseDown={(e) => { if (e.target === e.currentTarget) closeModal(); }}><section className={`card ref-modal ${modal === "map" ? "ref-modal--map-full" : "ref-modal--wide"}`} role="dialog" aria-modal="true" aria-labelledby="permit-modal-title">
      {modal === "map" ? <div className="map-modal-header"><div><h2 id="permit-modal-title" className="fs-5 fw-bold mb-1">Form Edit Coordinate</h2><div className="fs-7">Company: <strong>{companyName} ({companyID})</strong><span className="mx-3">|</span>Permit Number: <strong>{editing?.permit_number}</strong></div></div><div className="d-flex align-items-center gap-2"><button type="button" className={`btn btn-sm ${mapMeasurement ? "btn-primary" : "btn-light"}`} onClick={() => setMapMeasurement((value) => !value)}>Measurement</button><button type="button" className="btn btn-sm btn-light" onClick={closeModal}>Back</button><button type="button" className="btn btn-icon btn-sm btn-light" onClick={closeModal} disabled={saving} aria-label="Tutup peta">×</button></div></div> : <div className={`card-header min-h-70px form-modal-header ${editing ? "form-modal-header--edit" : "form-modal-header--add"}`}><div className="card-title"><h2 id="permit-modal-title" className="fs-4 fw-bold mb-0">{editing ? `Edit Land Permit #${editing.id}` : "Add Land Permit"}</h2></div><button type="button" className="btn btn-light form-modal-close" onClick={closeModal} disabled={saving} aria-label="Tutup modal">×</button></div>}
      <form onSubmit={modal === "map" ? saveMap : savePermit}><div className="card-body p-0">
        {error && <div className="alert alert-danger m-4 mb-0" role="alert">{error}</div>}
        {modal === "form" ? <>
          {formFields.slice(0, 2).map(({ key, label, type }) => <div className="doc-type-form-row" key={key}><label className="form-label mb-0" htmlFor={`permit-${key}`}>{label}:</label><input id={`permit-${key}`} className="form-control form-control-solid doc-type-input" type={type ?? "text"} value={String(form[key] ?? "")} onChange={(e) => updateField(key, e.target.value as never)} required={key === "permit_number"} /></div>)}
          <div className="doc-type-form-row"><label className="form-label mb-0">Company name:</label><input className="form-control form-control-solid doc-type-input" value={companyName} readOnly /></div>
          <div className="doc-type-form-row"><label className="form-label mb-0" htmlFor="permit-province">Province:</label><select id="permit-province" className="form-select form-select-solid doc-type-select" value={form.province_id ?? ""} onChange={(e) => updateField("province_id", e.target.value ? Number(e.target.value) : null)}><option value="">Select Province</option>{options.provinces.map((o) => <option key={o.id} value={o.id}>{o.name}</option>)}</select></div>
          {formFields.slice(2).map(({ key, label }) => <div className="doc-type-form-row" key={key}><label className="form-label mb-0" htmlFor={`permit-${key}`}>{label}:</label><input id={`permit-${key}`} className="form-control form-control-solid doc-type-input" value={String(form[key] ?? "")} onChange={(e) => updateField(key, e.target.value as never)} /></div>)}
          <div className="doc-type-form-row"><label className="form-label mb-0" htmlFor="permit-doc">Permit Doc:</label><div><input id="permit-doc" type="file" className="form-control form-control-sm doc-type-input" accept=".pdf,.png,.jpg,.jpeg" onChange={(e) => setPermitFile(e.target.files?.[0] ?? null)} />{permitFile && <small className="text-muted">{permitFile.name}</small>}{!permitFile && form.permit_document && <a className="d-block fs-8" href={`${apiBase}${form.permit_document}`} target="_blank" rel="noreferrer">File tersimpan</a>}</div></div>
          <div className="doc-type-form-row"><label className="form-label mb-0" htmlFor="permit-map-file">Coordinate file:</label><div><input id="permit-map-file" type="file" className="form-control form-control-sm doc-type-input" accept=".json,.geojson,.csv,.zip,.mif,.mid" onChange={(e) => void handleMapFile(e)} /><small className="text-muted">GeoJSON, CSV, MapInfo (MIF/MID), atau ZIP</small></div></div>
          <div className="doc-type-form-row align-items-start"><label className="form-label mb-0 pt-2" htmlFor="permit-map-data">Map GeoJSON:</label><textarea id="permit-map-data" className="form-control form-control-solid" rows={5} value={form.map_data} onChange={(e) => updateField("map_data", e.target.value)} placeholder='{"type":"FeatureCollection","features":[]}' /></div>
        </> : <>
          <div className="map-modal-layout" style={{ gridTemplateColumns: `minmax(0, 1fr) 8px ${mapPanelWidth}px` }}>
            <div className="map-modal-canvas"><LeafletMapView value={form.map_data} zoom={mapZoom} onZoomChange={setMapZoom} basemap={basemap} /><label className="map-basemap-control"><span>Basemap</span><select className="form-select form-select-sm" aria-label="Pilih jenis basemap" value={basemap} onChange={(event) => setBasemap(event.target.value as BasemapKey)}>{Object.entries(BASEMAPS).map(([key, option]) => <option key={key} value={key}>{option.label}</option>)}</select></label>
              <div className="map-zoom-control"><button type="button" className="btn btn-sm btn-light-primary" aria-label="Perbesar peta" onClick={() => setMapZoom((value) => Math.min(19, value + 1))}>+</button><input aria-label="Zoom peta" type="range" min="1" max="19" step="1" value={mapZoom} onChange={(event) => setMapZoom(Number(event.target.value))} /><button type="button" className="btn btn-sm btn-light-primary" aria-label="Perkecil peta" onClick={() => setMapZoom((value) => Math.max(1, value - 1))}>−</button></div>
              {mapMeasurement && <div className="map-measurement-hint">Mode measurement aktif · klik Add Point untuk menambahkan titik.</div>}
            </div>
            <div className="map-modal-splitter" role="separator" aria-orientation="vertical" aria-label="Ubah lebar panel koordinat" onPointerDown={(event) => { event.preventDefault(); setDraggingMapSplitter(true); }} />
            <aside className="map-coordinate-panel"><div className="fw-bold mb-2">Form Edit Coordinate:</div><label className="btn btn-sm btn-light-primary w-100 mb-2 text-start" htmlFor="map-csv-file">Upload csv Format</label><input id="map-csv-file" type="file" className="form-control form-control-sm mb-3" accept=".csv,.json,.geojson" onChange={(event) => void handleMapFile(event)} /><label className="btn btn-sm btn-light-primary w-100 mb-2 text-start" htmlFor="map-info-file">Upload mapinfo Format</label><input id="map-info-file" type="file" className="form-control form-control-sm mb-3" accept=".mif,.mid,.zip,.json,.geojson" onChange={(event) => void handleMapFile(event)} />{mapFile && <small className="d-block text-primary mb-2">{mapFile.name} siap dipakai. Klik Submit untuk menyimpan koordinat pengganti.</small>}
              <div className="d-flex justify-content-between align-items-center border-bottom py-2"><strong>Coordinates</strong><span className="text-muted fs-8">{mapCoordinates.length} points</span></div><div className="map-coordinate-list">{mapCoordinates.length === 0 ? <div className="text-muted p-3">Mapdata belum berisi koordinat.</div> : mapCoordinates.slice(0, 300).map(([lng, lat], index) => <div className="map-coordinate-row" key={`${index}-${lng}-${lat}`}><span className="text-muted">{index + 1}</span><input className="form-control form-control-sm" aria-label={`Longitude titik ${index + 1}`} value={lng} readOnly /><input className="form-control form-control-sm" aria-label={`Latitude titik ${index + 1}`} value={lat} readOnly /></div>)}{mapCoordinates.length > 300 && <div className="text-muted fs-8 p-2">Menampilkan 300 dari {mapCoordinates.length} koordinat. Semua titik tetap digunakan untuk menggambar peta.</div>}</div>
              <div className="border-top pt-3 mt-2"><div className="fw-bold mb-2">Add Point</div><div className="d-flex gap-2"><input className="form-control form-control-sm" aria-label="Latitude" placeholder="Latitude" type="number" step="any" value={mapPoint.lat} onChange={(event) => setMapPoint((value) => ({ ...value, lat: event.target.value }))} /><input className="form-control form-control-sm" aria-label="Longitude" placeholder="Longitude" type="number" step="any" value={mapPoint.lng} onChange={(event) => setMapPoint((value) => ({ ...value, lng: event.target.value }))} /><button type="button" className="btn btn-sm btn-light-primary" onClick={addPoint}>Add</button></div></div>
              <small className="d-block text-muted mt-2">Mapdata dibaca dari Gis_Data_Spatials; GeoJSON/JSON unggahan menjadi fallback jika belum ada data spasial.</small>
            </aside>
          </div>
        </>}
      </div><div className="card-footer d-flex flex-wrap justify-content-end gap-3 py-3"><button type="button" className="btn btn-light" onClick={closeModal} disabled={saving}>Cancel</button><button type="submit" className={`btn ${editing ? "btn-warning" : "btn-dark"}`} disabled={saving}>{saving && <span className="spinner-border spinner-border-sm me-2" aria-hidden="true" />}{saving ? "Saving..." : modal === "map" ? "Submit" : "Save"}</button></div></form>
    </section></div>}
  </div>;
}
