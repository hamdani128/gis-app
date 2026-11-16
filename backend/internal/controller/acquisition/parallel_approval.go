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

type ParallelApprovalController struct {
	Model *acquisition.SubmitMemoLandModel
	Stage acquisition.ParallelApprovalStage
}

func (c *ParallelApprovalController) failure(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, acquisition.ErrNotOfficer):
		response.Error(w, 403, "Akun ini bukan petugas "+c.Stage.Name+" untuk company tersebut")
	case errors.Is(err, acquisition.ErrParallelAlreadyProcessed), errors.Is(err, acquisition.ErrParallelNotReady):
		response.Error(w, 409, err.Error())
	case errors.Is(err, pgx.ErrNoRows):
		response.Error(w, 404, "Lahan tidak ditemukan pada company atau tahap "+c.Stage.Name)
	default:
		log.Printf("%s: %v", c.Stage.Name, err)
		response.Error(w, 500, "Gagal memproses "+c.Stage.Name)
	}
}

func (c *ParallelApprovalController) Index(w http.ResponseWriter, r *http.Request) {
	companyID, _, ok := submitParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	page, err := c.Model.ParallelList(ctx, auth.Username(r), companyID, c.Stage)
	if err != nil {
		c.failure(w, err)
		return
	}
	response.Success(w, "Data "+c.Stage.Name+" berhasil diambil", page)
}

func (c *ParallelApprovalController) Show(w http.ResponseWriter, r *http.Request) {
	companyID, landID, ok := submitParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	detail, err := c.Model.ParallelDetail(ctx, auth.Username(r), companyID, landID, c.Stage)
	if err != nil {
		c.failure(w, err)
		return
	}
	response.Success(w, "Detail "+c.Stage.Name+" berhasil diambil", detail)
}

func (c *ParallelApprovalController) Map(w http.ResponseWriter, r *http.Request) {
	companyID, landID, ok := submitParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	data, err := c.Model.ParallelMap(ctx, auth.Username(r), companyID, landID, c.Stage)
	if err != nil {
		c.failure(w, err)
		return
	}
	response.Success(w, "Koordinat lahan berhasil diambil", map[string]string{"map_data": data})
}

func (c *ParallelApprovalController) Store(w http.ResponseWriter, r *http.Request) {
	companyID, landID, ok := submitParams(w, r)
	if !ok {
		return
	}
	var input acquisition.RMApprovalInput
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
	if err := c.Model.ParallelApprove(ctx, auth.Username(r), companyID, landID, c.Stage, input); err != nil {
		c.failure(w, err)
		return
	}
	response.Success(w, c.Stage.Name+" berhasil disimpan", nil)
}
