package lands

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	landModel "gis-app/backend/internal/model/lands"
	"gis-app/backend/internal/response"
	"github.com/jackc/pgx/v5"
)

type ProposeLandController struct{ Model *landModel.ProposeLandModel }

func proposeLandParams(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	companyID, err := strconv.ParseInt(r.URL.Query().Get("company_id"), 10, 64)
	if err != nil || companyID < 0 {
		response.Error(w, 400, "company_id wajib berupa angka nol atau positif")
		return 0, 0, false
	}
	var id int64
	if r.PathValue("id") != "" {
		id, err = strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			response.Error(w, 400, "ID Propose Land tidak valid")
			return 0, 0, false
		}
	}
	return companyID, id, true
}

func proposeLandFailure(w http.ResponseWriter, err error) {
	status, msg := http.StatusInternalServerError, "Gagal memproses data Propose Land"
	if errors.Is(err, pgx.ErrNoRows) {
		status, msg = http.StatusNotFound, "Propose Land tidak ditemukan pada company aktif"
	} else if strings.Contains(err.Error(), "harus") || strings.Contains(err.Error(), "tidak valid") || strings.Contains(err.Error(), "tidak ditemukan") {
		status, msg = http.StatusBadRequest, err.Error()
	}
	log.Printf("Propose Land: %v", err)
	response.Error(w, status, msg)
}

func (c *ProposeLandController) Index(w http.ResponseWriter, r *http.Request) {
	companyID, _, ok := proposeLandParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	items, err := c.Model.GetAll(ctx, companyID)
	if err != nil {
		proposeLandFailure(w, err)
		return
	}
	response.Success(w, "Data Propose Land berhasil diambil", items)
}

func (c *ProposeLandController) Options(w http.ResponseWriter, r *http.Request) {
	companyID, _, ok := proposeLandParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	items, err := c.Model.GetOptions(ctx, companyID)
	if err != nil {
		proposeLandFailure(w, err)
		return
	}
	response.Success(w, "Pilihan Propose Land berhasil diambil", items)
}

func (c *ProposeLandController) save(w http.ResponseWriter, r *http.Request, id *int64) {
	companyID, _, ok := proposeLandParams(w, r)
	if !ok {
		return
	}
	var in landModel.ProposeLandInput
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&in); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		response.Error(w, 400, "Data form tidak valid")
		return
	}
	if in.GisAquistionDivisionsID <= 0 || in.GisLandOwnersID <= 0 {
		response.Error(w, 400, "Block Plan dan Owner wajib diisi")
		return
	}
	if in.DryLandWide < 0 || in.WetLandWide < 0 || in.LandWide < 0 || in.DryLandPerhaPrice < 0 || in.WetLandPerhaPrice < 0 || in.AdminPrice < 0 || in.PlantPrice < 0 {
		response.Error(w, 400, "Luas dan harga tidak boleh negatif")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	item, err := c.Model.Save(ctx, companyID, id, in)
	if err != nil {
		proposeLandFailure(w, err)
		return
	}
	if id == nil {
		response.Created(w, "Propose Land berhasil ditambahkan", item)
	} else {
		response.Success(w, "Propose Land berhasil diperbarui", item)
	}
}

func (c *ProposeLandController) Store(w http.ResponseWriter, r *http.Request) { c.save(w, r, nil) }
func (c *ProposeLandController) Update(w http.ResponseWriter, r *http.Request) {
	_, id, ok := proposeLandParams(w, r)
	if !ok {
		return
	}
	c.save(w, r, &id)
}
func (c *ProposeLandController) Destroy(w http.ResponseWriter, r *http.Request) {
	companyID, id, ok := proposeLandParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if err := c.Model.Delete(ctx, companyID, id); err != nil {
		proposeLandFailure(w, err)
		return
	}
	response.Success(w, "Propose Land berhasil dihapus", nil)
}
