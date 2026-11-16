package acquisition

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"gis-app/backend/internal/controller/auth"
	"gis-app/backend/internal/model/acquisition"
	"gis-app/backend/internal/response"
	"github.com/jackc/pgx/v5"
)

type SubmitMemoLandController struct {
	Model *acquisition.SubmitMemoLandModel
}

func submitParams(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	companyID, err := strconv.ParseInt(r.URL.Query().Get("company_id"), 10, 64)
	if err != nil || companyID < 0 {
		response.Error(w, 400, "company_id tidak valid")
		return 0, 0, false
	}
	var landID int64
	if r.PathValue("id") != "" {
		landID, err = strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || landID <= 0 {
			response.Error(w, 400, "ID lahan tidak valid")
			return 0, 0, false
		}
	}
	return companyID, landID, true
}

func submitFailure(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, acquisition.ErrNotOfficer):
		response.Error(w, 403, err.Error())
	case errors.Is(err, acquisition.ErrAlreadySubmitted):
		response.Error(w, 409, err.Error())
	case errors.Is(err, pgx.ErrNoRows):
		response.Error(w, 404, "Lahan atau memo tidak ditemukan pada company petugas")
	default:
		log.Printf("Submit Memo Land: %v", err)
		response.Error(w, 500, "Gagal memproses Submit Memo Land")
	}
}

func (c *SubmitMemoLandController) Index(w http.ResponseWriter, r *http.Request) {
	companyID, _, ok := submitParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	page, err := c.Model.List(ctx, auth.Username(r), companyID)
	if err != nil {
		submitFailure(w, err)
		return
	}
	response.Success(w, "Data Submit Memo Land berhasil diambil", page)
}

func (c *SubmitMemoLandController) Show(w http.ResponseWriter, r *http.Request) {
	companyID, landID, ok := submitParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	detail, err := c.Model.Detail(ctx, auth.Username(r), companyID, landID)
	if err != nil {
		submitFailure(w, err)
		return
	}
	response.Success(w, "Detail Submit Memo Land berhasil diambil", detail)
}

func (c *SubmitMemoLandController) Store(w http.ResponseWriter, r *http.Request) {
	companyID, landID, ok := submitParams(w, r)
	if !ok {
		return
	}
	var input acquisition.SubmitMemoLandInput
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		response.Error(w, 400, "Data form tidak valid")
		return
	}
	if input.MemoID <= 0 || len(input.Note) > 10000 {
		response.Error(w, 400, "Memo wajib dipilih dan note maksimal 10000 karakter")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	if err := c.Model.Submit(ctx, auth.Username(r), companyID, landID, input); err != nil {
		submitFailure(w, err)
		return
	}
	response.Success(w, "Submit Memo Land berhasil disimpan", nil)
}

func (c *SubmitMemoLandController) Map(w http.ResponseWriter, r *http.Request) {
	companyID, landID, ok := submitParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	data, err := c.Model.Map(ctx, auth.Username(r), companyID, landID)
	if err != nil {
		submitFailure(w, err)
		return
	}
	response.Success(w, "Koordinat lahan berhasil diambil", map[string]string{"map_data": data})
}
