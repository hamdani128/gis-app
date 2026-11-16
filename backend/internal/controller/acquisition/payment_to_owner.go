package acquisition

import (
	"context"
	"net/http"
	"time"

	"gis-app/backend/internal/controller/auth"
	"gis-app/backend/internal/model/acquisition"
	"gis-app/backend/internal/response"
)

type PaymentToOwnerController struct {
	Model *acquisition.SubmitMemoLandModel
}

func (c *PaymentToOwnerController) Index(w http.ResponseWriter, r *http.Request) {
	companyID, _, ok := submitParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	page, err := c.Model.PaymentToOwnerList(ctx, auth.Username(r), companyID)
	if err != nil {
		paymentFailure(w, err)
		return
	}
	response.Success(w, "Data Payment to Owner berhasil diambil", page)
}

func (c *PaymentToOwnerController) Show(w http.ResponseWriter, r *http.Request) {
	companyID, landID, ok := submitParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	detail, err := c.Model.PaymentToOwnerDetail(ctx, companyID, landID)
	if err != nil {
		paymentFailure(w, err)
		return
	}
	response.Success(w, "Detail Payment to Owner berhasil diambil", detail)
}

func (c *PaymentToOwnerController) Map(w http.ResponseWriter, r *http.Request) {
	companyID, landID, ok := submitParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	data, err := c.Model.PaymentToOwnerMap(ctx, companyID, landID)
	if err != nil {
		paymentFailure(w, err)
		return
	}
	response.Success(w, "Koordinat lahan berhasil diambil", map[string]string{"map_data": data})
}

func (c *PaymentToOwnerController) DocumentTypes(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := submitParams(w, r); !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	items, err := c.Model.PaymentToOwnerDocumentTypes(ctx)
	if err != nil {
		paymentFailure(w, err)
		return
	}
	response.Success(w, "Jenis dokumen Payment to Owner berhasil diambil", items)
}

func (c *PaymentToOwnerController) Store(w http.ResponseWriter, r *http.Request) {
	storePaymentDocuments(w, r, c.Model.SavePaymentToOwner, "Payment to Owner", "8")
}
