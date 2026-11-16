package acquisition

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"gis-app/backend/internal/response"
)

func (c *PaymentController) DocumentTypes(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := submitParams(w, r); !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	items, err := c.Model.PaymentDocumentTypes(ctx)
	if err != nil {
		paymentFailure(w, err)
		return
	}
	response.Success(w, "Jenis dokumen Payment berhasil diambil", items)
}

func (c *PaymentController) Documents(w http.ResponseWriter, r *http.Request) {
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
	items, err := c.Model.PaymentDocuments(ctx, landID)
	if err != nil {
		paymentFailure(w, err)
		return
	}
	response.Success(w, "Dokumen Payment berhasil diambil", items)
}

func (c *PaymentController) DocumentFile(w http.ResponseWriter, r *http.Request) {
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
	items, err := c.Model.PaymentDocuments(ctx, landID)
	if err != nil {
		paymentFailure(w, err)
		return
	}
	for _, item := range items {
		if item.ID != docID {
			continue
		}
		path := paymentStoredFile(landID, item.FileName)
		if path == "" {
			path = datedLegacyFile(legacyUploadDir("LEGACY_SUPPORTED_DOC_DIR"), item.FileName, item.Created)
		}
		if path != "" {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=payment-document-%d%s", docID, filepath.Ext(item.FileName)))
			http.ServeFile(w, r, path)
			return
		}
		if remote := datedLegacyDocumentURL(item.FileName, item.Created); remote != "" {
			http.Redirect(w, r, remote, http.StatusFound)
			return
		}
		break
	}
	response.Error(w, 404, "Dokumen Payment tidak ditemukan")
}
