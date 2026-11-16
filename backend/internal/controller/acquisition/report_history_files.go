package acquisition

import (
	"context"
	"net/http"

	"gis-app/backend/internal/model/acquisition"
)

// ReportHistoryFiles shares acquisition's legacy URL and backend-folder fallback
// with the report controllers without duplicating document storage rules.
type ReportHistoryFiles struct{}

func (ReportHistoryFiles) SetSupportedDocumentAccess(landID int64, item *acquisition.PaymentSupportedDocument) {
	setSupportedDocumentAccess(landID, item)
}

func (ReportHistoryFiles) ServeSupportedDocument(w http.ResponseWriter, r *http.Request, landID int64, item acquisition.PaymentSupportedDocument) {
	servePaymentHistoryFile(w, r, supportedDocumentPath(landID, item), datedLegacyDocumentURL(item.FileName, item.Created), item.FileName)
}

func (ReportHistoryFiles) SupportedDocumentLegacyStatus(ctx context.Context, item acquisition.PaymentSupportedDocument) *bool {
	return checkLegacyPreviewStatus(ctx, datedLegacyPreviewURL(item.FileName, item.Created))
}

func (ReportHistoryFiles) SetMemoAccess(landID int64, item *acquisition.PaymentMemo) {
	setMemoAccess(landID, item)
	if item.Found {
		item.PreviewURL = memoLegacyPreviewURL(item.FileName, item.CompanyID)
	}
}

func (ReportHistoryFiles) MemoLegacyStatus(ctx context.Context, item acquisition.PaymentMemo) *bool {
	return checkLegacyPreviewStatus(ctx, memoLegacyPreviewURL(item.FileName, item.CompanyID))
}

func (ReportHistoryFiles) ServeMemo(w http.ResponseWriter, r *http.Request, landID int64, item acquisition.PaymentMemo) {
	servePaymentHistoryFile(w, r, memoDocumentPath(landID, item), memoLegacyDocumentURL(item.FileName, item.CompanyID), item.FileName)
}

func (ReportHistoryFiles) LocalFiles(landID int64) []string {
	return localHistoryFiles(landID)
}

func (ReportHistoryFiles) ServeLocalFile(w http.ResponseWriter, r *http.Request, landID int64) {
	serveLocalHistoryFile(w, r, landID)
}
