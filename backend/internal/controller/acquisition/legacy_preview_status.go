package acquisition

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gis-app/backend/internal/response"
)

var legacyStatusClient = &http.Client{Timeout: 4 * time.Second}

func checkLegacyPreviewStatus(ctx context.Context, previewURL string) *bool {
	if previewURL == "" {
		missing := false
		return &missing
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, previewURL, nil)
	if err != nil {
		return nil
	}
	res, err := legacyStatusClient.Do(req)
	if err != nil {
		return nil
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound || res.StatusCode == http.StatusGone {
		missing := false
		return &missing
	}
	if res.StatusCode != http.StatusOK {
		return nil
	}
	contentType := strings.ToLower(strings.TrimSpace(strings.SplitN(res.Header.Get("Content-Type"), ";", 2)[0]))
	available := res.ContentLength != 0 && (strings.HasPrefix(contentType, "image/") || contentType == "application/pdf")
	return &available
}

func (c *PaymentController) SupportedDocumentLegacyStatus(w http.ResponseWriter, r *http.Request) {
	companyID, landID, ok := submitParams(w, r)
	if !ok {
		return
	}
	docID, err := strconv.ParseInt(r.PathValue("doc_id"), 10, 64)
	if err != nil || docID <= 0 {
		response.Error(w, 400, "ID dokumen tidak valid")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if _, err := c.Model.PaymentDocumentDetail(ctx, companyID, landID); err != nil {
		paymentFailure(w, err)
		return
	}
	items, err := c.Model.PaymentSupportedDocuments(ctx, landID)
	if err != nil {
		paymentFailure(w, err)
		return
	}
	for _, item := range items {
		if item.ID == docID {
			response.Success(w, "Status preview lama berhasil diperiksa", struct {
				Available *bool `json:"available"`
			}{checkLegacyPreviewStatus(ctx, datedLegacyPreviewURL(item.FileName, item.Created))})
			return
		}
	}
	response.Error(w, 404, "Dokumen lahan tidak ditemukan")
}

func (c *CompensatedHistoryController) SupportedDocumentLegacyStatus(w http.ResponseWriter, r *http.Request) {
	docID, err := strconv.ParseInt(r.PathValue("doc_id"), 10, 64)
	if err != nil || docID <= 0 {
		response.Error(w, 400, "ID dokumen tidak valid")
		return
	}
	ctx, cancel, landID, ok := c.params(w, r)
	if !ok {
		return
	}
	defer cancel()
	items, err := c.Model.PaymentSupportedDocuments(ctx, landID)
	if err != nil {
		compensatedHistoryFailure(w, err)
		return
	}
	for _, item := range items {
		if item.ID == docID {
			response.Success(w, "Status preview lama berhasil diperiksa", struct {
				Available *bool `json:"available"`
			}{checkLegacyPreviewStatus(ctx, datedLegacyPreviewURL(item.FileName, item.Created))})
			return
		}
	}
	response.Error(w, 404, "Dokumen lahan tidak ditemukan")
}
