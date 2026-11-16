package acquisition

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gis-app/backend/internal/controller/auth"
	"gis-app/backend/internal/model/acquisition"
	"gis-app/backend/internal/response"
	"github.com/jackc/pgx/v5"
)

type PaymentController struct {
	Model *acquisition.SubmitMemoLandModel
}

func paymentFailure(w http.ResponseWriter, err error) {
	if errors.Is(err, acquisition.ErrNotOfficer) {
		response.Error(w, 403, "Akun ini bukan petugas Payment untuk company tersebut")
		return
	}
	if errors.Is(err, acquisition.ErrPaymentNotReady) || errors.Is(err, acquisition.ErrPaymentAlreadyProcessed) {
		response.Error(w, 409, err.Error())
		return
	}
	if errors.Is(err, acquisition.ErrPaymentToOwnerNotReady) || errors.Is(err, acquisition.ErrPaymentToOwnerAlreadyProcessed) {
		response.Error(w, 409, err.Error())
		return
	}
	if errors.Is(err, acquisition.ErrInvalidPaymentDocumentType) || errors.Is(err, acquisition.ErrPaymentDocumentsRequired) {
		response.Error(w, 400, err.Error())
		return
	}
	if errors.Is(err, acquisition.ErrInvalidPaymentToOwnerChoice) || errors.Is(err, acquisition.ErrPaymentToOwnerCancelReasonRequired) {
		response.Error(w, 400, err.Error())
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		response.Error(w, 404, "Lahan yang sudah disetujui PD tidak ditemukan pada company ini")
		return
	}
	log.Printf("Payment: %v", err)
	response.Error(w, 500, "Gagal memuat data Payment")
}

func paymentUploadDir() string {
	return filepath.Join(acquisitionUploadRoot(), "acquisition-land")
}

func (c *PaymentController) Index(w http.ResponseWriter, r *http.Request) {
	companyID, _, ok := submitParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	page, err := c.Model.PaymentList(ctx, auth.Username(r), companyID)
	if err != nil {
		paymentFailure(w, err)
		return
	}
	response.Success(w, "Data Payment berhasil diambil", page)
}

func (c *PaymentController) Show(w http.ResponseWriter, r *http.Request) {
	companyID, landID, ok := submitParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	detail, err := c.Model.PaymentDetail(ctx, companyID, landID)
	if err != nil {
		paymentFailure(w, err)
		return
	}
	response.Success(w, "Detail Payment berhasil diambil", detail)
}

func (c *PaymentController) Map(w http.ResponseWriter, r *http.Request) {
	companyID, landID, ok := submitParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	data, err := c.Model.PaymentMap(ctx, companyID, landID)
	if err != nil {
		paymentFailure(w, err)
		return
	}
	response.Success(w, "Koordinat lahan berhasil diambil", map[string]string{"map_data": data})
}

func (c *PaymentController) Document(w http.ResponseWriter, r *http.Request) {
	companyID, landID, ok := submitParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	detail, err := c.Model.PaymentDocumentDetail(ctx, companyID, landID)
	if err != nil {
		paymentFailure(w, err)
		return
	}
	if detail.Land["status"] != float64(7) {
		response.Error(w, 404, "Dokumen Payment belum tersedia")
		return
	}
	for _, ext := range []string{".pdf", ".png", ".jpg", ".jpeg"} {
		path := filepath.Join(paymentUploadDir(), fmt.Sprintf("payment-%d%s", landID, ext))
		if _, err := os.Stat(path); err == nil {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=payment-%d%s", landID, ext))
			http.ServeFile(w, r, path)
			return
		}
	}
	response.Error(w, 404, "Dokumen Payment tidak ditemukan pada server")
}
