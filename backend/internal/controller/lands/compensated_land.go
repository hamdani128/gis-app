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

type CompensatedLandController struct {
	Model *landModel.CompensatedLandModel
}

func compensatedParams(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	companyID, err := strconv.ParseInt(r.URL.Query().Get("company_id"), 10, 64)
	if err != nil || companyID < 0 {
		response.Error(w, 400, "company_id wajib berupa angka nol atau positif")
		return 0, 0, false
	}
	var id int64
	if r.PathValue("id") != "" {
		id, err = strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			response.Error(w, 400, "ID Compensated Land tidak valid")
			return 0, 0, false
		}
	}
	return companyID, id, true
}

func compensatedFailure(w http.ResponseWriter, err error) {
	status, message := http.StatusInternalServerError, "Gagal memproses Compensated Land"
	if errors.Is(err, pgx.ErrNoRows) {
		status, message = 404, "Compensated Land tidak ditemukan pada company aktif"
	} else if strings.Contains(err.Error(), "tidak valid") || strings.Contains(err.Error(), "harus") || strings.Contains(err.Error(), "GeoJSON") {
		status, message = 400, err.Error()
	}
	log.Printf("Compensated Land: %v", err)
	response.Error(w, status, message)
}

func (c *CompensatedLandController) Index(w http.ResponseWriter, r *http.Request) {
	companyID, _, ok := compensatedParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	items, err := c.Model.GetAll(ctx, companyID)
	if err != nil {
		compensatedFailure(w, err)
		return
	}
	response.Success(w, "Data Compensated Land berhasil diambil", items)
}

func (c *CompensatedLandController) Options(w http.ResponseWriter, r *http.Request) {
	companyID, _, ok := compensatedParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	items, err := c.Model.GetOptions(ctx, companyID)
	if err != nil {
		compensatedFailure(w, err)
		return
	}
	response.Success(w, "Pilihan Compensated Land berhasil diambil", items)
}

func (c *CompensatedLandController) Processes(w http.ResponseWriter, r *http.Request) {
	companyID, id, ok := compensatedParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	items, err := c.Model.GetProcesses(ctx, companyID, id)
	if err != nil {
		compensatedFailure(w, err)
		return
	}
	response.Success(w, "Proses Compensated Land berhasil diambil", items)
}

func (c *CompensatedLandController) Update(w http.ResponseWriter, r *http.Request) {
	companyID, id, ok := compensatedParams(w, r)
	if !ok {
		return
	}
	var in landModel.CompensatedLandInput
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
	item, err := c.Model.Update(ctx, companyID, id, in)
	if err != nil {
		compensatedFailure(w, err)
		return
	}
	response.Success(w, "Compensated Land berhasil diperbarui", item)
}

func (c *CompensatedLandController) Map(w http.ResponseWriter, r *http.Request) {
	companyID, id, ok := compensatedParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	data, err := c.Model.GetMap(ctx, companyID, id)
	if err != nil {
		compensatedFailure(w, err)
		return
	}
	response.Success(w, "Koordinat Compensated Land berhasil diambil", map[string]string{"map_data": data})
}

func (c *CompensatedLandController) SaveMap(w http.ResponseWriter, r *http.Request) {
	companyID, id, ok := compensatedParams(w, r)
	if !ok {
		return
	}
	var in struct {
		MapData string `json:"map_data"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 20*1024*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&in); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		response.Error(w, 400, "Data koordinat tidak valid")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	data, err := c.Model.SaveMap(ctx, companyID, id, in.MapData)
	if err != nil {
		compensatedFailure(w, err)
		return
	}
	response.Success(w, "Koordinat Compensated Land berhasil disimpan", map[string]string{"map_data": data})
}
