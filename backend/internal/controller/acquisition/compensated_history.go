package acquisition

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"gis-app/backend/internal/model/acquisition"
	"gis-app/backend/internal/model/lands"
	"gis-app/backend/internal/response"
	"github.com/jackc/pgx/v5"
)

// CompensatedHistoryController exposes acquisition archives after either the
// status-scoped land check or the owner/company/land relationship check.
type CompensatedHistoryController struct {
	Model       *acquisition.SubmitMemoLandModel
	LandModel   *lands.CompensatedLandModel
	OwnerModel  *lands.LandOwnerModel
	CheckAccess func(context.Context, *http.Request, int64, int64) error
}

func compensatedHistoryFailure(w http.ResponseWriter, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		response.Error(w, 404, "Lahan tidak ditemukan pada company aktif")
		return
	}
	log.Printf("Compensated history: %v", err)
	response.Error(w, 500, "Riwayat lahan gagal dimuat")
}

func (c *CompensatedHistoryController) params(w http.ResponseWriter, r *http.Request) (context.Context, context.CancelFunc, int64, bool) {
	companyID, landID, ok := submitParams(w, r)
	if !ok {
		return nil, nil, 0, false
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	if c.CheckAccess != nil {
		if err := c.CheckAccess(ctx, r, companyID, landID); err != nil {
			cancel()
			compensatedHistoryFailure(w, err)
			return nil, nil, 0, false
		}
		return ctx, cancel, landID, true
	}
	if c.OwnerModel != nil {
		// On owner routes {id} is the owner, and {land_id} is the land.
		ownerID := landID
		landID, err := strconv.ParseInt(r.PathValue("land_id"), 10, 64)
		if err != nil || landID <= 0 {
			cancel()
			response.Error(w, 400, "ID land tidak valid")
			return nil, nil, 0, false
		}
		if _, err := c.OwnerModel.LandDetail(ctx, companyID, ownerID, landID); err != nil {
			cancel()
			compensatedHistoryFailure(w, err)
			return nil, nil, 0, false
		}
		return ctx, cancel, landID, true
	}
	if err := c.LandModel.CheckAccess(ctx, companyID, landID); err != nil {
		cancel()
		compensatedHistoryFailure(w, err)
		return nil, nil, 0, false
	}
	return ctx, cancel, landID, true
}

func (c *CompensatedHistoryController) SupportedDocuments(w http.ResponseWriter, r *http.Request) {
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
	for i := range items {
		setSupportedDocumentAccess(landID, &items[i])
	}
	response.Success(w, "Riwayat dokumen lahan berhasil diambil", items)
}

func (c *CompensatedHistoryController) SupportedDocumentFile(w http.ResponseWriter, r *http.Request) {
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
			servePaymentHistoryFile(w, r, supportedDocumentPath(landID, item), datedLegacyDocumentURL(item.FileName, item.Created), item.FileName)
			return
		}
	}
	response.Error(w, 404, "Dokumen lahan tidak ditemukan")
}

func (c *CompensatedHistoryController) Memo(w http.ResponseWriter, r *http.Request) {
	ctx, cancel, landID, ok := c.params(w, r)
	if !ok {
		return
	}
	defer cancel()
	item, err := c.Model.PaymentMemo(ctx, landID)
	if err != nil {
		compensatedHistoryFailure(w, err)
		return
	}
	setMemoAccess(landID, &item)
	response.Success(w, "Memo lahan berhasil diambil", item)
}

func (c *CompensatedHistoryController) MemoFile(w http.ResponseWriter, r *http.Request) {
	ctx, cancel, landID, ok := c.params(w, r)
	if !ok {
		return
	}
	defer cancel()
	item, err := c.Model.PaymentMemo(ctx, landID)
	if err != nil {
		compensatedHistoryFailure(w, err)
		return
	}
	servePaymentHistoryFile(w, r, memoDocumentPath(landID, item), memoLegacyDocumentURL(item.FileName, item.CompanyID), item.FileName)
}
