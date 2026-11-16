"use client";

import { useState } from "react";

export default function ApprovalControls({ id, stageName, memoNumber, officerName, saving, onSubmit }: {
  id: string;
  stageName: string;
  memoNumber?: string | null;
  officerName: string;
  saving: boolean;
  onSubmit: (approved: boolean, note: string) => void;
}) {
  const [decision, setDecision] = useState("");
  const [note, setNote] = useState("");
  const canSubmit = !!decision && !saving && (decision !== "rejected" || !!note.trim());

  return <div>
    <strong className="d-block mb-2">{stageName}</strong>
    <div className="fs-7 mb-3">Memo: {memoNumber || "—"}</div>
    <strong className="d-block mb-2">Need your Approval</strong>
    <label className="form-label" htmlFor={`${id}-decision`}>Approval</label>
    <select id={`${id}-decision`} className="form-select mb-3" value={decision} onChange={(event) => setDecision(event.target.value)}>
      <option value="">-- select --</option>
      <option value="approved">Approve</option>
      <option value="rejected">Reject</option>
    </select>
    <label className="form-label" htmlFor={`${id}-note`}>Note</label>
    <textarea id={`${id}-note`} className="form-control mb-3" rows={4} maxLength={10000} value={note} onChange={(event) => setNote(event.target.value)} />
    <button type="button" className="btn btn-sm btn-primary" onClick={() => onSubmit(decision === "approved", note)} disabled={!canSubmit}>{saving ? "Menyimpan..." : "Submit"}</button>
    <div className="text-muted fs-7 mt-3">{officerName}</div>
  </div>;
}
