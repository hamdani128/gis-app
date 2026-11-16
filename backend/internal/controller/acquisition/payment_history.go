package acquisition

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gis-app/backend/internal/model/acquisition"
	"gis-app/backend/internal/response"
)

func legacyUploadDir(envKey string) string {
	if configured := strings.TrimSpace(os.Getenv(envKey)); configured != "" {
		return configured
	}
	return `\\10.20.5.22\aepgis\CI\assets\uploads`
}

func validStoredName(name string) bool {
	return name != "" && name != "." && name != ".." && filepath.Base(name) == name && !strings.ContainsAny(name, `/\`)
}

func safeStoredFile(dir, name string) string {
	if !validStoredName(name) {
		return ""
	}
	path := filepath.Join(dir, name)
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return ""
	}
	return path
}

func paymentDocumentPathIn(root string, landID int64, name string) string {
	if landID <= 0 {
		return ""
	}
	if path := safeStoredFile(filepath.Join(root, strconv.FormatInt(landID, 10)), name); path != "" {
		return path
	}
	return safeStoredFile(root, name)
}

func paymentLandUploadDir(landID int64) string {
	return filepath.Join(paymentUploadDir(), strconv.FormatInt(landID, 10))
}

func paymentStoredFile(landID int64, name string) string {
	if path := paymentDocumentPathIn(paymentUploadDir(), landID, name); path != "" {
		return path
	}
	return paymentDocumentPathIn(filepath.Join(filepath.Dir(paymentUploadDir()), "payment-approvals"), landID, name)
}

func datedLegacyFile(dir, name string, created *time.Time) string {
	if created == nil {
		return ""
	}
	year := created.Format("2006")
	if path := safeStoredFile(filepath.Join(dir, year, strconv.Itoa(int(created.Month()))), name); path != "" {
		return path
	}
	return safeStoredFile(filepath.Join(dir, year, created.Format("01")), name)
}

func supportedDocumentPath(landID int64, item acquisition.PaymentSupportedDocument) string {
	if path := safeStoredFile(paymentLandUploadDir(landID), item.FileName); path != "" {
		return path
	}
	if strings.EqualFold(strings.TrimSpace(item.Group), "payment") || strings.EqualFold(strings.TrimSpace(item.Group), "payment to owner") {
		if path := paymentStoredFile(landID, item.FileName); path != "" {
			return path
		}
	}
	return datedLegacyFile(legacyUploadDir("LEGACY_SUPPORTED_DOC_DIR"), item.FileName, item.Created)
}

func memoDocumentPath(landID int64, item acquisition.PaymentMemo) string {
	if !item.Found || item.CompanyID <= 0 {
		return ""
	}
	if path := safeStoredFile(paymentLandUploadDir(landID), item.FileName); path != "" {
		return path
	}
	if path := safeStoredFile(memoUploadDir(), item.FileName); path != "" {
		return path
	}
	memoRoot := strings.TrimSpace(os.Getenv("LEGACY_MEMO_DIR"))
	if memoRoot == "" {
		memoRoot = filepath.Join(legacyUploadDir("LEGACY_SUPPORTED_DOC_DIR"), "memo")
	}
	return safeStoredFile(filepath.Join(memoRoot, strconv.FormatInt(item.CompanyID, 10)), item.FileName)
}

func servePaymentHistoryFile(w http.ResponseWriter, r *http.Request, path, fallbackURL, name string) {
	if path == "" {
		if fallbackURL == "" || r.URL.Query().Get("preview") == "1" {
			response.Error(w, 404, "File lama belum tersedia pada server ini")
			return
		}
		http.Redirect(w, r, fallbackURL, http.StatusFound)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	disposition := "attachment"
	switch strings.ToLower(filepath.Ext(name)) {
	case ".pdf", ".png", ".jpg", ".jpeg", ".gif", ".webp":
		disposition = "inline"
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`%s; filename=%q`, disposition, filepath.Base(name)))
	http.ServeFile(w, r, path)
}

func setSupportedDocumentAccess(landID int64, item *acquisition.PaymentSupportedDocument) {
	path := supportedDocumentPath(landID, *item)
	item.PreviewURL = datedLegacyPreviewURL(item.FileName, item.Created)
	if path == "" {
		item.LegacyURL = datedLegacyDocumentURL(item.FileName, item.Created)
	}
	item.Available = path != "" || item.LegacyURL != "" || item.PreviewURL != ""
}

func setMemoAccess(landID int64, item *acquisition.PaymentMemo) {
	path := memoDocumentPath(landID, *item)
	if path == "" && item.Found {
		item.LegacyURL = memoLegacyDocumentURL(item.FileName, item.CompanyID)
		item.PreviewURL = memoLegacyPreviewURL(item.FileName, item.CompanyID)
	}
	item.Available = path != "" || item.LegacyURL != "" || item.PreviewURL != ""
}

func (c *PaymentController) SupportedDocuments(w http.ResponseWriter, r *http.Request) {
	companyID, landID, ok := submitParams(w, r)
	if !ok {
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
	for i := range items {
		setSupportedDocumentAccess(landID, &items[i])
	}
	response.Success(w, "Riwayat dokumen lahan berhasil diambil", items)
}

func (c *PaymentController) SupportedDocumentFile(w http.ResponseWriter, r *http.Request) {
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
			servePaymentHistoryFile(w, r, supportedDocumentPath(landID, item), datedLegacyDocumentURL(item.FileName, item.Created), item.FileName)
			return
		}
	}
	response.Error(w, 404, "Dokumen lahan tidak ditemukan")
}

func (c *PaymentController) Memo(w http.ResponseWriter, r *http.Request) {
	companyID, landID, ok := submitParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if _, err := c.Model.PaymentDocumentDetail(ctx, companyID, landID); err != nil {
		paymentFailure(w, err)
		return
	}
	item, err := c.Model.PaymentMemo(ctx, landID)
	if err != nil {
		paymentFailure(w, err)
		return
	}
	setMemoAccess(landID, &item)
	response.Success(w, "Memo lahan berhasil diambil", item)
}

func (c *PaymentController) MemoFile(w http.ResponseWriter, r *http.Request) {
	companyID, landID, ok := submitParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if _, err := c.Model.PaymentDocumentDetail(ctx, companyID, landID); err != nil {
		paymentFailure(w, err)
		return
	}
	item, err := c.Model.PaymentMemo(ctx, landID)
	if err != nil {
		paymentFailure(w, err)
		return
	}
	servePaymentHistoryFile(w, r, memoDocumentPath(landID, item), memoLegacyDocumentURL(item.FileName, item.CompanyID), item.FileName)
}
