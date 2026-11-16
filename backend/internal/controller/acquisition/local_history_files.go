package acquisition

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gis-app/backend/internal/response"
)

func localHistoryFileName(name string) bool {
	if !validStoredName(name) {
		return false
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".pdf", ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return true
	}
	return false
}

func localHistoryFiles(landID int64) []string {
	files := []string{}
	entries, err := os.ReadDir(paymentLandUploadDir(landID))
	if err != nil {
		return files
	}
	for _, entry := range entries {
		if entry.Type().IsRegular() && localHistoryFileName(entry.Name()) {
			files = append(files, entry.Name())
		}
	}
	return files
}

func serveLocalHistoryFile(w http.ResponseWriter, r *http.Request, landID int64) {
	name := r.PathValue("name")
	if !localHistoryFileName(name) {
		response.Error(w, http.StatusNotFound, "File tidak ditemukan pada folder lahan")
		return
	}
	path := safeStoredFile(paymentLandUploadDir(landID), name)
	if path == "" {
		response.Error(w, http.StatusNotFound, "File tidak ditemukan pada folder lahan")
		return
	}
	servePaymentHistoryFile(w, r, path, "", name)
}

func (c *PaymentController) LocalHistoryFiles(w http.ResponseWriter, r *http.Request) {
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
	response.Success(w, "File pada folder lahan berhasil diambil", localHistoryFiles(landID))
}

func (c *PaymentController) LocalHistoryFile(w http.ResponseWriter, r *http.Request) {
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
	serveLocalHistoryFile(w, r, landID)
}

func (c *CompensatedHistoryController) LocalHistoryFiles(w http.ResponseWriter, r *http.Request) {
	_, cancel, landID, ok := c.params(w, r)
	if !ok {
		return
	}
	defer cancel()
	response.Success(w, "File pada folder lahan berhasil diambil", localHistoryFiles(landID))
}

func (c *CompensatedHistoryController) LocalHistoryFile(w http.ResponseWriter, r *http.Request) {
	_, cancel, landID, ok := c.params(w, r)
	if !ok {
		return
	}
	defer cancel()
	serveLocalHistoryFile(w, r, landID)
}
