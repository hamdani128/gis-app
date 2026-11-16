"use client";

import { useRef, useState, type FormEvent } from "react";
import { createPortal } from "react-dom";

export type PaymentDocumentType = { id: number; name: string; file_type: string };
export type PendingPaymentDocument = { id: number; typeID: number; typeName: string; title: string; note: string; file: File };

type Props = {
  landID: number;
  stageName?: string;
  officerName: string;
  saving: boolean;
  loadTypes: () => Promise<PaymentDocumentType[]>;
  onSubmit: (transferDate: string, note: string, documents: PendingPaymentDocument[], choice: "pay_to_owner" | "cancel") => Promise<void>;
};

export default function PaymentDocumentUpload({ landID, stageName = "Payment", officerName, saving, loadTypes, onSubmit }: Props) {
  const ownerStage = stageName === "Payment to Owner";
  const nextID = useRef(0);
  const [documents, setDocuments] = useState<PendingPaymentDocument[]>([]);
  const [transferDate, setTransferDate] = useState("");
  const [note, setNote] = useState("");
  const [paymentChoice, setPaymentChoice] = useState<"pay_to_owner" | "cancel">("pay_to_owner");
  const [modalOpen, setModalOpen] = useState(false);
  const [modalView, setModalView] = useState<"list" | "form">("list");
  const [types, setTypes] = useState<PaymentDocumentType[]>([]);
  const [typesLoading, setTypesLoading] = useState(false);
  const [typeID, setTypeID] = useState("");
  const [title, setTitle] = useState("");
  const [docNote, setDocNote] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [uploadError, setUploadError] = useState("");

  async function openModal() {
    setModalView("list");
    setModalOpen(true);
    setUploadError("");
    if (types.length) return;
    setTypesLoading(true);
    try { setTypes(await loadTypes()); }
    catch (cause) { setUploadError(cause instanceof Error ? cause.message : "Jenis dokumen gagal dimuat."); }
    finally { setTypesLoading(false); }
  }

  function openForm() {
    setTypeID(""); setTitle(""); setDocNote(""); setFile(null); setUploadError("");
    setModalView("form");
  }

  function addDocument(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    event.stopPropagation();
    const selectedType = types.find((item) => item.id === Number(typeID));
    if (!selectedType || !file) { setUploadError("Pilih jenis dokumen dan berkas."); return; }
    if (file.size === 0 || file.size > 15 * 1024 * 1024) { setUploadError("Ukuran berkas harus 1 byte hingga 15 MB."); return; }
    if (documents.reduce((total, item) => total + item.file.size, file.size) > 50 * 1024 * 1024) { setUploadError("Total seluruh berkas maksimal 50 MB."); return; }
    if (!/\.(pdf|png|jpe?g)$/i.test(file.name)) { setUploadError("Pilih berkas PDF, PNG, atau JPEG."); return; }
    if (documents.length >= 10) { setUploadError("Maksimal 10 dokumen."); return; }
    nextID.current += 1;
    setDocuments((current) => [...current, { id: nextID.current, typeID: selectedType.id, typeName: selectedType.name, title: title.trim() || file.name.replace(/\.[^.]+$/, ""), note: docNote.trim(), file }]);
    setUploadError("");
    setModalView("list");
  }

  return <>
    <form onSubmit={(event) => { event.preventDefault(); void onSubmit(transferDate, note, documents, paymentChoice); }}>
      <strong className="d-block mb-2">Need your Approval</strong>
      <label className="form-label d-block">{ownerStage ? "Payment Document" : "Approval Document"}</label>
      <button type="button" className="btn btn-sm btn-light-primary mb-2" onClick={() => void openModal()} disabled={saving}>{ownerStage ? "Upload Payment Document" : "Upload Approval Document"}</button>
      <div className="text-muted fs-7 mb-3">{documents.length ? `${documents.length} dokumen siap disimpan` : ownerStage && paymentChoice === "cancel" ? "Dokumen pembatalan boleh ditambahkan bila ada." : "Belum ada dokumen. Tambahkan minimal satu."}</div>
      {ownerStage && <><label className="form-label" htmlFor="payment-to-owner-method">Payment</label><select id="payment-to-owner-method" className="form-select form-select-sm mb-3" value={paymentChoice} onChange={(event) => setPaymentChoice(event.target.value as "pay_to_owner" | "cancel")}><option value="pay_to_owner">Pay to Owner</option><option value="cancel">Cancel</option></select></>}
      <label className="form-label" htmlFor="payment-transfer-date">{ownerStage ? "Payment Date" : "Transfer Date"}</label>
      <input id="payment-transfer-date" type="date" className="form-control form-control-sm mb-3" value={transferDate} onChange={(event) => setTransferDate(event.target.value)} required />
      <label className="form-label" htmlFor="payment-note">Note</label>
      <textarea id="payment-note" className="form-control mb-3" rows={4} maxLength={10000} value={note} onChange={(event) => setNote(event.target.value)} />
      <button type="submit" className="btn btn-sm btn-primary" disabled={saving || !transferDate || (documents.length === 0 && (!ownerStage || paymentChoice !== "cancel")) || (ownerStage && paymentChoice === "cancel" && !note.trim())}>{saving ? "Menyimpan..." : `Submit ${stageName}`}</button>
      <div className="text-muted fs-7 mt-3">{officerName}</div>
    </form>
    {modalOpen && createPortal(<div className="modal d-block show" style={{ zIndex: 1120, backgroundColor: "rgba(0, 0, 0, 0.65)" }} role="dialog" aria-modal="true" aria-label={`Dokumen ${stageName} · Land #${landID}`}>
      <div className="modal-dialog modal-dialog-centered modal-xl" style={{ maxWidth: ownerStage ? "min(900px, 96vw)" : "min(1100px, 96vw)" }}>
        <div className="modal-content" style={{ maxHeight: "92vh" }}>
          <div className="modal-header"><h2 className="modal-title fs-4">{ownerStage ? "Payment Document" : "Approval Document"} · Land #{landID}</h2><button type="button" className="btn-close" aria-label="Tutup" onClick={() => setModalOpen(false)} /></div>
          {modalView === "list" ? <>
            <div className="modal-body overflow-auto">
              {uploadError && <div role="alert" className="alert alert-danger">{uploadError}</div>}
              <div className="d-flex align-items-center justify-content-between gap-3 mb-4"><div className="text-muted">Dokumen yang akan disimpan bersama {stageName}</div><button type="button" className="btn btn-sm btn-primary" onClick={openForm} disabled={saving || typesLoading || types.length === 0 || documents.length >= 10}>+ Add Doc</button></div>
              <div className="table-responsive"><table className="table table-row-bordered align-middle"><thead><tr><th>No.</th><th>Document Title</th><th>Doc type</th><th>Doc file</th><th>Doc note</th><th>Actions</th></tr></thead><tbody>
                {documents.length ? documents.map((document, index) => <tr key={document.id}><td>{index + 1}</td><td className="fw-semibold">{document.title}</td><td>{document.typeName}</td><td className="text-break">{document.file.name}</td><td>{document.note || "—"}</td><td><button type="button" className="btn btn-sm btn-light-danger" disabled={saving} onClick={() => setDocuments((current) => current.filter((item) => item.id !== document.id))}>Hapus</button></td></tr>) : <tr><td colSpan={6} className="text-center text-muted py-8">Belum ada dokumen.</td></tr>}
              </tbody></table></div>
              <div className="text-muted fs-7">Maksimal 10 dokumen. Berkas dalam tabel disimpan saat {stageName} di-Submit.</div>
            </div>
            <div className="modal-footer"><button type="button" className="btn btn-light" onClick={() => setModalOpen(false)}>Selesai</button></div>
          </> : <form onSubmit={addDocument}>
            <div className="modal-body overflow-auto">
              {uploadError && <div role="alert" className="alert alert-danger">{uploadError}</div>}
              <h3 className="fs-5 mb-4">Add Doc</h3>
              <div className="row g-4">
                <div className={ownerStage ? "col-12" : "col-md-6"}><label className="form-label">Land ID</label><input className="form-control" value={landID} readOnly /></div>
                <div className={ownerStage ? "col-12" : "col-md-6"}><label className="form-label" htmlFor="payment-doc-title">Document title</label><input id="payment-doc-title" className="form-control" maxLength={200} value={title} onChange={(event) => setTitle(event.target.value)} placeholder="Otomatis dari nama file bila kosong" /></div>
                <div className={ownerStage ? "col-12" : "col-md-6"}><label className="form-label" htmlFor="payment-doc-type">Doc type</label><select id="payment-doc-type" className="form-select" value={typeID} onChange={(event) => setTypeID(event.target.value)} required><option value="">Pilih jenis dokumen</option>{types.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></div>
                <div className={ownerStage ? "col-12" : "col-md-6"}><label className="form-label">Doc group</label><input className="form-control" value="Payment" readOnly /></div>
                <div className="col-12"><label className="form-label" htmlFor="payment-doc-file">Doc file</label><input id="payment-doc-file" type="file" className="form-control" accept=".pdf,.png,.jpg,.jpeg" onChange={(event) => setFile(event.target.files?.[0] ?? null)} required /><div className="form-text">PDF, PNG, atau JPEG; maksimal 15 MB per dokumen dan 50 MB seluruhnya.</div></div>
                <div className="col-12"><label className="form-label" htmlFor="payment-doc-note">Doc note</label><textarea id="payment-doc-note" className="form-control" rows={3} maxLength={2000} value={docNote} onChange={(event) => setDocNote(event.target.value)} /></div>
              </div>
            </div>
            <div className="modal-footer"><button type="button" className="btn btn-light" onClick={() => { setUploadError(""); setModalView("list"); }}>Kembali ke daftar</button><button type="submit" className="btn btn-primary">Tambahkan</button></div>
          </form>}
        </div>
      </div>
    </div>, document.body)}
  </>;
}
