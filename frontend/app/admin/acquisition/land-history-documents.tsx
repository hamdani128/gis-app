"use client";
/* eslint-disable @next/next/no-img-element -- Legacy GIS images need the browser's session cookie. */

import { useCallback, useEffect, useId, useRef, useState } from "react";
import type { ReactNode } from "react";
import { createPortal } from "react-dom";
import styles from "./land-history-documents.module.css";

type SupportedDocument = {
  id: number;
  title: string;
  type_name: string;
  file_name: string;
  doc_group: string;
  note: string;
  available: boolean;
  legacy_url?: string;
  preview_url?: string;
};

type Memo = {
  id: string;
  memo_number: string;
  memo_date: string;
  memo_note: string;
  memo_doc: string;
  found: boolean;
  available: boolean;
  legacy_url?: string;
  preview_url?: string;
};

type Props = { landID: number; companyID: number; apiPath: string; fallbackMemoNumber?: string };
type Envelope<T> = { data?: T; message?: string };
const apiBase = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:9000").replace(/\/$/, "");
const isPreviewImage = (name: string) => /\.(png|jpe?g|gif|webp)$/i.test(name);

async function getData<T>(path: string): Promise<T> {
  let response = await fetch(`${apiBase}${path}`, { credentials: "include", cache: "no-store" });
  if (response.status === 401) {
    const refresh = await fetch(`${apiBase}/api/auth/refresh`, { method: "POST", credentials: "include" });
    if (refresh.ok) response = await fetch(`${apiBase}${path}`, { credentials: "include", cache: "no-store" });
  }
  const result = await response.json().catch(() => null) as Envelope<T> | null;
  if (!response.ok) throw new Error(result?.message || "Data gagal dimuat.");
  return result?.data as T;
}

type HistoryPreviewProps = {
  fileName: string;
  primaryURL?: string;
  legacyStatusPath?: string;
  recordURL: string;
  folderListPath: string;
  folderFileURL: (name: string) => string;
  alt: string;
};

function HistoryPreview({ fileName, primaryURL, legacyStatusPath, recordURL, folderListPath, folderFileURL, alt }: HistoryPreviewProps) {
  const [source, setSource] = useState<"legacy" | "record" | "checking" | "folder" | "choose">(!fileName ? "checking" : primaryURL ? "legacy" : "record");
  const [folderFiles, setFolderFiles] = useState<string[]>([]);
  const [selectedFile, setSelectedFile] = useState("");

  const applyFolderFiles = useCallback((files: string[]) => {
    setFolderFiles(files);
    const exactMatch = files.find((name) => name.toLowerCase() === fileName.toLowerCase());
    if (exactMatch || files.length === 1) {
      setSelectedFile(exactMatch || files[0]);
      setSource("folder");
    } else {
      setSource("choose");
    }
  }, [fileName]);

  const checkFolder = useCallback(async () => {
    setSource("checking");
    try { applyFolderFiles(await getData<string[]>(folderListPath)); }
    catch { setFolderFiles([]); setSource("choose"); }
  }, [applyFolderFiles, folderListPath]);

  useEffect(() => {
    if (fileName) return;
    let active = true;
    void getData<string[]>(folderListPath).then((files) => {
      if (active) applyFolderFiles(files);
    }).catch(() => {
      if (active) { setFolderFiles([]); setSource("choose"); }
    });
    return () => { active = false; };
  }, [fileName, folderListPath, applyFolderFiles]);

  useEffect(() => {
    if (!primaryURL || !legacyStatusPath || isPreviewImage(fileName)) return;
    let active = true;
    void getData<{ available: boolean | null }>(legacyStatusPath).then((result) => {
      if (active && result.available === false) setSource((current) => current === "legacy" ? "record" : current);
    }).catch(() => { /* Keep the direct legacy preview when its status cannot be checked. */ });
    return () => { active = false; };
  }, [fileName, primaryURL, legacyStatusPath]);

  useEffect(() => {
    if (source !== "record" || !recordURL) return;
    let active = true;
    void (async () => {
      let response = await fetch(recordURL, { method: "HEAD", credentials: "include", cache: "no-store" });
      if (response.status === 401) {
        const refresh = await fetch(`${apiBase}/api/auth/refresh`, { method: "POST", credentials: "include" });
        if (refresh.ok) response = await fetch(recordURL, { method: "HEAD", credentials: "include", cache: "no-store" });
      }
      if (active && !response.ok) void checkFolder();
    })().catch(() => { if (active) void checkFolder(); });
    return () => { active = false; };
  }, [source, recordURL, checkFolder]);

  function onPreviewError() {
    if (source === "legacy") setSource("record");
    else if (source === "record") void checkFolder();
    else if (source === "folder") setSource("choose");
  }

  if (source === "checking") return <p className="text-muted p-4">Memeriksa folder backend untuk lahan ini...</p>;
  if (source === "choose") return <div className="p-4">
    <div className="alert alert-warning">Preview dokumen yang tercatat tidak tersedia. {folderFiles.length ? "Pilih file yang sesuai dari folder lahan:" : "Tidak ada file yang dapat dipreview di folder lahan ini."}</div>
    {folderFiles.length > 0 && <div className="d-flex flex-wrap gap-2">{folderFiles.map((name) => <button key={name} type="button" className="btn btn-sm btn-light-primary" onClick={() => { setSelectedFile(name); setSource("folder"); }}>{name}</button>)}</div>}
  </div>;

  const currentFile = source === "folder" ? selectedFile : fileName;
  const currentURL = source === "legacy" && primaryURL ? primaryURL : source === "folder" ? folderFileURL(selectedFile) : recordURL;
  return <div>
    {source === "folder" && <div className="d-flex flex-wrap align-items-center gap-3 px-4 pt-3"><strong>File folder lahan: {selectedFile}</strong><a href={currentURL} target="_blank" rel="noreferrer">Buka file ini</a>{folderFiles.length > 1 && <button type="button" className="btn btn-sm btn-light" onClick={() => setSource("choose")}>Pilih file lain</button>}</div>}
    {isPreviewImage(currentFile) ? <img src={currentURL} alt={alt} onError={onPreviewError} style={{ display: "block", maxWidth: "100%", maxHeight: "min(65vh, 760px)", margin: "0 auto", objectFit: "contain" }} /> : <iframe src={currentURL} title={alt} onError={onPreviewError} style={{ display: "block", width: "100%", height: "min(65vh, 760px)", border: 0 }} />}
    {source !== "folder" && <div className="p-3 text-end"><button type="button" className="btn btn-sm btn-light-primary" onClick={() => void checkFolder()}>Cari di folder backend</button></div>}
  </div>;
}

function HistoryDetailDialog({ title, onClose, children, fileURL }: { title: string; onClose: () => void; children: ReactNode; fileURL?: string }) {
  const ref = useRef<HTMLDialogElement>(null);
  const titleID = useId();
  useEffect(() => {
    const dialog = ref.current;
    const previousFocus = document.activeElement;
    dialog?.showModal();
    return () => { dialog?.close(); if (previousFocus instanceof HTMLElement) previousFocus.focus(); };
  }, []);
  return createPortal(<dialog ref={ref} className={styles.dialog} aria-labelledby={titleID} onCancel={(event) => { event.preventDefault(); onClose(); }}>
    <header className={styles.dialogHeader}><div><span>ARSIP AKUISISI LAHAN</span><h2 id={titleID}>{title}</h2></div><button type="button" className="btn btn-sm btn-light" aria-label="Tutup detail dokumen" onClick={onClose}>Tutup</button></header>
    <div className={styles.dialogBody}>{children}</div>
    <footer className={styles.dialogFooter}>{fileURL && <a className="btn btn-sm btn-primary" href={fileURL} target="_blank" rel="noreferrer">Buka file</a>}<button type="button" className="btn btn-sm btn-light" onClick={onClose}>Tutup</button></footer>
  </dialog>, document.body);
}

export function LandHistoryPanel({ landID, companyID, apiPath, fallbackMemoNumber, kind }: Props & { kind: "documents" | "memo" }) {
  const [documents, setDocuments] = useState<SupportedDocument[]>([]);
  const [memo, setMemo] = useState<Memo | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [preview, setPreview] = useState<SupportedDocument | "memo" | "folder" | null>(null);
  const base = `${apiPath}/${landID}`;
  const company = `company_id=${companyID}`;
  useEffect(() => {
    let active = true;
    const request = kind === "memo"
      ? getData<Memo>(`${base}/memo?${company}`).then((value) => { if (active) setMemo(value); })
      : getData<SupportedDocument[]>(`${base}/supported-documents?${company}`).then((value) => { if (active) setDocuments(Array.isArray(value) ? value : []); });
    void request.catch((cause: unknown) => { if (active) setError(cause instanceof Error ? cause.message : "Data gagal dimuat."); }).finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [base, company, kind]);

  const folderListPath = `${base}/local-files?${company}`;
  const folderFileURL = (name: string) => `${apiBase}${base}/local-files/${encodeURIComponent(name)}?${company}`;
  const selectedDoc = preview && typeof preview === "object" ? preview : null;
  const fileURL = preview === "memo" ? `${apiBase}${base}/memo/document?${company}` : selectedDoc ? `${apiBase}${base}/supported-documents/${selectedDoc.id}?${company}` : undefined;
  const previewTitle = preview === "memo" ? `Memo ${memo?.memo_number || fallbackMemoNumber || `Land #${landID}`}` : selectedDoc?.title || selectedDoc?.type_name || `Dokumen Land #${landID}`;

  return <div className={styles.panel}>
    <div className={styles.heading}><div><h3>{kind === "memo" ? "Memo akuisisi" : "Dokumen pendukung"}</h3><p>{kind === "memo" ? "Ringkasan memo yang terhubung dengan lahan ini." : "Arsip dokumen dan berkas pendukung untuk lahan ini."} Klik detail untuk melihat file.</p></div><span className={styles.count}>{loading ? "..." : kind === "documents" ? `${documents.length} dokumen` : "Memo"}</span></div>
    {loading ? <p className="text-muted py-5" role="status">Memuat {kind === "memo" ? "memo" : "dokumen"}...</p> : error ? <div className="alert alert-danger" role="alert">{error}</div> : kind === "memo" ? <div className={styles.memoCard}>
      <div className={styles.fileIcon} aria-hidden="true">M</div><div className={styles.memoInfo}><strong>{memo?.memo_number || fallbackMemoNumber || "Nomor memo belum tersedia"}</strong><span>{memo?.memo_date || "Tanggal memo belum tersedia"}</span>{memo?.memo_note && <p>{memo.memo_note}</p>}{!memo?.found && <small>Record memo belum ditemukan. Periksa file pada folder lahan melalui detail.</small>}</div><button type="button" className="btn btn-sm btn-light-primary" onClick={() => setPreview("memo")}>Detail memo</button>
    </div> : documents.length ? <div className="table-responsive"><table className={`table align-middle mb-0 ${styles.table}`}><thead><tr><th>Dokumen</th><th>Jenis / Grup</th><th>File</th><th className="text-end">Aksi</th></tr></thead><tbody>{documents.map((doc) => <tr key={doc.id}><td><strong>{doc.title || doc.type_name || `Dokumen #${doc.id}`}</strong>{doc.note && <small>{doc.note}</small>}</td><td><span>{doc.type_name || "—"}</span><small>{doc.doc_group || "—"}</small></td><td><span className={styles.fileName}>{doc.file_name || "File belum tercatat"}</span></td><td className="text-end"><button type="button" className="btn btn-sm btn-light-primary text-nowrap" onClick={() => setPreview(doc)}>Detail dokumen</button></td></tr>)}</tbody></table></div> : <div className={styles.empty}><strong>Belum ada dokumen tercatat</strong><p>Anda tetap dapat memeriksa berkas yang tersimpan di folder lahan.</p><button type="button" className="btn btn-sm btn-light-primary" onClick={() => setPreview("folder")}>Periksa folder lahan</button></div>}
    {preview && <HistoryDetailDialog title={previewTitle} onClose={() => setPreview(null)} fileURL={fileURL}>
      {preview === "memo" && memo && <div className={styles.modalSummary}><div><small>Nomor memo</small><strong>{memo.memo_number || fallbackMemoNumber || "—"}</strong></div><div><small>Tanggal memo</small><strong>{memo.memo_date || "—"}</strong></div>{memo.memo_note && <p>{memo.memo_note}</p>}</div>}
      <HistoryPreview key={preview === "memo" ? `memo-${landID}` : selectedDoc?.id || "folder"} fileName={preview === "memo" ? memo?.memo_doc || "" : selectedDoc?.file_name || ""} primaryURL={preview === "memo" ? memo?.preview_url : selectedDoc?.preview_url} legacyStatusPath={selectedDoc ? `${base}/supported-documents/${selectedDoc.id}/legacy-status?${company}` : undefined} recordURL={fileURL ? `${fileURL}&preview=1` : ""} folderListPath={folderListPath} folderFileURL={folderFileURL} alt={previewTitle} />
    </HistoryDetailDialog>}
  </div>;
}

export default function LandHistoryDocuments({ landID, companyID, apiPath, fallbackMemoNumber }: Props) {
  const [open, setOpen] = useState<"memo" | "documents" | null>(null);
  const [documents, setDocuments] = useState<SupportedDocument[]>([]);
  const [memo, setMemo] = useState<Memo | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [preview, setPreview] = useState<SupportedDocument | null>(null);
  const base = `${apiPath}/${landID}`;
  const company = `company_id=${companyID}`;

  async function show(kind: "memo" | "documents") {
    setOpen(kind); setPreview(null); setLoading(true); setError("");
    try {
      if (kind === "memo") setMemo(await getData<Memo>(`${base}/memo?${company}`));
      else setDocuments(await getData<SupportedDocument[]>(`${base}/supported-documents?${company}`));
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Data gagal dimuat."); }
    finally { setLoading(false); }
  }

  const memoFileURL = `${apiBase}${base}/memo/document?${company}`;
  const documentURL = (doc: SupportedDocument) => `${apiBase}${base}/supported-documents/${doc.id}?${company}`;
  const memoLocalPreviewURL = `${apiBase}${base}/memo/document?${company}&preview=1`;
  const memoLegacyStatusPath = apiPath.startsWith("/api/reports/") ? `${base}/memo/legacy-status?${company}` : undefined;
  const documentLocalPreviewURL = (doc: SupportedDocument) => `${apiBase}${base}/supported-documents/${doc.id}?${company}&preview=1`;
  const documentLegacyStatusPath = (doc: SupportedDocument) => `${base}/supported-documents/${doc.id}/legacy-status?${company}`;
  const folderListPath = `${base}/local-files?${company}`;
  const folderFileURL = (name: string) => `${apiBase}${base}/local-files/${encodeURIComponent(name)}?${company}`;

  return <>
    <div className="d-flex flex-wrap gap-2 mb-4">
      <button type="button" className="btn btn-sm btn-light-primary" onClick={() => void show("memo")}>Show Memo</button>
      <button type="button" className="btn btn-sm btn-light-primary" onClick={() => void show("documents")}>Supported Document</button>
    </div>
    {open && <div className="modal d-block show" style={{ zIndex: 1200, background: "rgba(0,0,0,.65)" }} role="dialog" aria-modal="true" aria-label={`${open === "memo" ? "Memo" : "Supported Document"} Land #${landID}`}>
      <div className="modal-dialog modal-xl modal-dialog-centered" style={{ maxWidth: "min(1100px, 96vw)" }}>
        <div className="modal-content" style={{ maxHeight: "94vh" }}>
          <div className="modal-header"><h2 className="modal-title fs-4">{open === "memo" ? "Memo" : "Supported Document"} · Land #{landID}</h2><button type="button" className="btn-close" aria-label="Tutup" onClick={() => setOpen(null)} /></div>
          <div className="modal-body overflow-auto">
            {loading ? <p className="text-muted">Memuat data...</p> : error ? <div className="alert alert-danger" role="alert">{error}</div> : open === "memo" ? memo && <>
              <div className="row g-3 mb-4">
                <div className="col-md-6"><div className="text-muted fs-7">Memo Number</div><strong>{memo.memo_number || fallbackMemoNumber || "—"}</strong></div>
                <div className="col-md-6"><div className="text-muted fs-7">Memo Date</div><strong>{memo.memo_date || "—"}</strong></div>
                {memo.memo_note && <div className="col-12"><div className="text-muted fs-7">Note</div><span>{memo.memo_note}</span></div>}
              </div>
              {!memo.found && <div className="alert alert-warning">Record memo {memo.id ? `#${memo.id} ` : ""}tidak ditemukan di Gis_Aquisition_memo. Memeriksa folder lahan.</div>}
              {memo.found && !memo.available && <div className="alert alert-warning">File memo yang tercatat belum tersedia. Periksa folder lahan melalui preview di bawah.</div>}
              <HistoryPreview key={`memo-${landID}-${memo.id}`} fileName={memo.found ? memo.memo_doc : ""} primaryURL={memo.preview_url} legacyStatusPath={memoLegacyStatusPath} recordURL={memoLocalPreviewURL} folderListPath={folderListPath} folderFileURL={folderFileURL} alt={`Memo Land #${landID}`} />
            </> : <>
              <div className="table-responsive"><table className="table table-row-bordered align-middle"><thead><tr><th>Document Title</th><th>Doc type</th><th>Doc file</th><th>Doc group</th><th>Doc note</th></tr></thead><tbody>
                {documents.length ? documents.map((doc) => <tr key={doc.id}><td>{doc.title || "—"}</td><td>{doc.type_name || "—"}</td><td><button type="button" className="btn btn-sm btn-light-primary" onClick={() => setPreview(doc)}>{doc.available ? "View Doc" : "Cari File"}</button></td><td>{doc.doc_group || "—"}</td><td>{doc.note || "—"}</td></tr>) : <tr><td colSpan={5} className="text-center text-muted py-5">Belum ada dokumen untuk lahan ini.</td></tr>}
              </tbody></table></div>
              {!documents.length && <HistoryPreview key={`documents-folder-${landID}`} fileName="" recordURL="" folderListPath={folderListPath} folderFileURL={folderFileURL} alt={`Dokumen Land #${landID}`} />}
              {documents.some((doc) => !doc.available) && <p className="text-muted fs-7 mb-0">Beberapa file yang tercatat belum tersedia. Gunakan Cari File untuk memeriksa folder lahan.</p>}
            </>}
          </div>
          <div className="modal-footer">{open === "memo" && memo?.available && !loading && <a className="btn btn-primary" href={memoFileURL} target="_blank" rel="noreferrer">Buka Dokumen</a>}<button type="button" className="btn btn-light" onClick={() => setOpen(null)}>Tutup</button></div>
        </div>
      </div>
    </div>}
    {preview && <div className="modal d-block show" style={{ zIndex: 1210, background: "rgba(0,0,0,.75)" }} role="dialog" aria-modal="true" aria-label={`Dokumen ${preview.title}`}>
      <div className="modal-dialog modal-xl modal-dialog-centered" style={{ maxWidth: "min(1100px, 96vw)" }}><div className="modal-content" style={{ height: "94vh" }}>
        <div className="modal-header"><h2 className="modal-title fs-4">{preview.title || preview.type_name}</h2><button type="button" className="btn-close" aria-label="Tutup preview" onClick={() => setPreview(null)} /></div>
        <div className="modal-body p-0 overflow-auto"><HistoryPreview key={`doc-${preview.id}`} fileName={preview.file_name} primaryURL={preview.preview_url} legacyStatusPath={documentLegacyStatusPath(preview)} recordURL={documentLocalPreviewURL(preview)} folderListPath={folderListPath} folderFileURL={folderFileURL} alt={preview.title || preview.file_name} /></div>
        <div className="modal-footer"><button type="button" className="btn btn-light" onClick={() => setPreview(null)}>Tutup</button><a className="btn btn-primary" href={documentURL(preview)} target="_blank" rel="noreferrer">Buka Dokumen</a></div>
      </div></div>
    </div>}
  </>;
}
