package acquisition

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"gis-app/backend/internal/controller/auth"
	"gis-app/backend/internal/model/acquisition"
	"gis-app/backend/internal/response"
	"github.com/jackc/pgx/v5"
)

type GMApprovalController struct {
	Model *acquisition.SubmitMemoLandModel
}

func gmFailure(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, acquisition.ErrNotOfficer):
		response.Error(w, 403, "Akun ini bukan petugas GM Approval untuk company tersebut")
	case errors.Is(err, acquisition.ErrGMAlreadyProcessed), errors.Is(err, acquisition.ErrGMNotReady):
		response.Error(w, 409, err.Error())
	case errors.Is(err, pgx.ErrNoRows):
		response.Error(w, 404, "Lahan tidak ditemukan pada company atau tahap GM Approval")
	default:
		log.Printf("GM Approval: %v", err)
		response.Error(w, 500, "Gagal memproses GM Approval")
	}
}

func (c *GMApprovalController) Index(w http.ResponseWriter, r *http.Request) {
	companyID, _, ok := submitParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	page, err := c.Model.GMList(ctx, auth.Username(r), companyID)
	if err != nil {
		gmFailure(w, err)
		return
	}
	response.Success(w, "Data GM Approval berhasil diambil", page)
}

func (c *GMApprovalController) Show(w http.ResponseWriter, r *http.Request) {
	companyID, landID, ok := submitParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	detail, err := c.Model.GMDetail(ctx, auth.Username(r), companyID, landID)
	if err != nil {
		gmFailure(w, err)
		return
	}
	response.Success(w, "Detail GM Approval berhasil diambil", detail)
}

func (c *GMApprovalController) Map(w http.ResponseWriter, r *http.Request) {
	companyID, landID, ok := submitParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	data, err := c.Model.GMMap(ctx, auth.Username(r), companyID, landID)
	if err != nil {
		gmFailure(w, err)
		return
	}
	response.Success(w, "Koordinat lahan berhasil diambil", map[string]string{"map_data": data})
}

func (c *GMApprovalController) Store(w http.ResponseWriter, r *http.Request) {
	companyID, landID, ok := submitParams(w, r)
	if !ok {
		return
	}
	var input acquisition.GMApprovalInput
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		response.Error(w, 400, "Data form tidak valid")
		return
	}
	if input.Approved == nil || len(input.Note) > 10000 || (!*input.Approved && strings.TrimSpace(input.Note) == "") {
		response.Error(w, 400, "Note penolakan wajib diisi dan maksimal 10000 karakter")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	if err := c.Model.GMApprove(ctx, auth.Username(r), companyID, landID, input); err != nil {
		gmFailure(w, err)
		return
	}
	response.Success(w, "GM Approval berhasil disimpan", nil)
}
