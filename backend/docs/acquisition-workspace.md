# Shared acquisition detail workspace

All acquisition workspaces use four tabs: Informasi lahan, Pengukuran dan harga,
Dokumen, and Memo. Document lists and memo summaries load when their tabs are
selected. Clicking a detail button opens the shared archive preview dialog.
The acquisition timeline uses collapsible stages; the stage with an available
approval/submission form starts expanded. Existing approval, upload, memo selection,
and PD PDF actions remain inside the corresponding stage.

The layout applies to Submit Memo Land, RM, GM, GIS, Legal, Operation Control,
PD, Payment, and Payment to Owner. Acquisition Request's selected-land detail uses
the same tabs and scopes document access to the selected memo and land.

## Archive endpoints

For each prefix below, archive endpoints have the same suffixes as
[New Compensated Lands history](./new-compensated-history-api.md):
`/{id}/supported-documents`, `/{id}/supported-documents/{doc_id}`,
`/{id}/supported-documents/{doc_id}/legacy-status`, `/{id}/memo`,
`/{id}/memo/document`, `/{id}/local-files`, and `/{id}/local-files/{name}`.

- `/api/submit-memo-land`
- `/api/rm-approval`
- `/api/gm-approval`
- `/api/gis-approval`
- `/api/legal-approval`
- `/api/operation-control-approval`
- `/api/pd-approval`

These routes require a signed-in session, acquisition company access, and the
same stage-specific land check as that stage's detail endpoint. In particular,
GM and parallel approvals keep their prerequisite-process checks. Land outside
the permitted company/stage returns 404. Metadata and file routes use the same
check, including legacy-status and backend-folder access.

Acquisition Request uses prefix `/api/acquisition-memos/{memo_id}/lands` with the
same suffixes. It requires `acquisitionRequest:view`, access to the memo's company,
and proof that the land belongs to that memo. A memo ID does not grant access to
unrelated land files.

Payment and Payment to Owner retain their existing archive endpoints and access
checks. No write endpoint, approval rule, or storage location is changed.

## Validation

`go test ./internal/controller/acquisition ./internal/model/acquisition ./internal/routes`
checks archive authentication and that each archive handler invokes its stage
access check before reading metadata or files. TypeScript and focused frontend
lint cover the shared tabs and every consuming workspace.
