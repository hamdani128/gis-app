package companies

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	companyModel "gis-app/backend/internal/model/companies"
	"gis-app/backend/internal/response"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type RegionController struct {
	Model *companyModel.RegionModel
}

func regionError(w http.ResponseWriter, err error) {
	status, message := http.StatusInternalServerError, "Terjadi kesalahan pada database"
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		status, message = http.StatusNotFound, "Data region tidak ditemukan"
	case errors.As(err, &pgErr):
		switch pgErr.Code {
		case "23505":
			status, message = http.StatusConflict, "ID region sudah digunakan"
		case "23503":
			status, message = http.StatusConflict, "Region GM masih dipakai oleh data lain"
		case "22001", "23502", "23514", "22P02":
			status, message = http.StatusBadRequest, "Data tidak sesuai format atau batas kolom database"
		}
	}
	log.Printf("Region: %v", err)
	response.Error(w, status, message)
}

func regionID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, http.StatusBadRequest, "ID harus berupa angka positif")
		return 0, false
	}
	return id, true
}

func readRegionInput(w http.ResponseWriter, r *http.Request) (companyModel.RegionInput, bool) {
	var input companyModel.RegionInput
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "Body JSON tidak valid atau berisi field yang tidak dikenal")
		return input, false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		response.Error(w, http.StatusBadRequest, "Body harus berisi satu objek JSON")
		return input, false
	}
	companyModel.NormalizeRegionInput(&input)
	if input.RegionName == "" {
		response.Error(w, http.StatusBadRequest, "Region name wajib diisi")
		return input, false
	}
	if input.EmployeeID != nil && *input.EmployeeID <= 0 {
		response.Error(w, http.StatusBadRequest, "Region GM harus berupa employee yang valid")
		return input, false
	}
	return input, true
}

func regionContext(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 8*time.Second)
}

func (c *RegionController) validateEmployee(ctx context.Context, w http.ResponseWriter, employeeID *int32) bool {
	if employeeID == nil {
		return true
	}
	exists, err := c.Model.EmployeeExists(ctx, *employeeID)
	if err != nil {
		regionError(w, err)
		return false
	}
	if !exists {
		response.Error(w, http.StatusBadRequest, "Region GM yang dipilih tidak ditemukan")
		return false
	}
	return true
}

func (c *RegionController) Index(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := regionContext(r)
	defer cancel()
	items, err := c.Model.GetAll(ctx)
	if err != nil {
		regionError(w, err)
		return
	}
	response.Success(w, "Data region berhasil diambil", items)
}

func (c *RegionController) Employees(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := regionContext(r)
	defer cancel()
	items, err := c.Model.GetEmployees(ctx)
	if err != nil {
		regionError(w, err)
		return
	}
	response.Success(w, "Pilihan Region GM berhasil diambil", items)
}

func (c *RegionController) Show(w http.ResponseWriter, r *http.Request) {
	id, ok := regionID(w, r)
	if !ok {
		return
	}
	ctx, cancel := regionContext(r)
	defer cancel()
	item, err := c.Model.GetByID(ctx, id)
	if err != nil {
		regionError(w, err)
		return
	}
	response.Success(w, "Detail region berhasil diambil", item)
}

func (c *RegionController) Store(w http.ResponseWriter, r *http.Request) {
	input, ok := readRegionInput(w, r)
	if !ok {
		return
	}
	ctx, cancel := regionContext(r)
	defer cancel()
	if !c.validateEmployee(ctx, w, input.EmployeeID) {
		return
	}
	item, err := c.Model.Create(ctx, input)
	if err != nil {
		regionError(w, err)
		return
	}
	if item.ID != nil {
		w.Header().Set("Location", "/api/regions/"+strconv.Itoa(int(*item.ID)))
	}
	response.Created(w, "Region berhasil ditambahkan", item)
}

func (c *RegionController) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := regionID(w, r)
	if !ok {
		return
	}
	input, ok := readRegionInput(w, r)
	if !ok {
		return
	}
	ctx, cancel := regionContext(r)
	defer cancel()
	if !c.validateEmployee(ctx, w, input.EmployeeID) {
		return
	}
	item, err := c.Model.Update(ctx, id, input)
	if err != nil {
		regionError(w, err)
		return
	}
	response.Success(w, "Region berhasil diperbarui", item)
}

func (c *RegionController) Destroy(w http.ResponseWriter, r *http.Request) {
	id, ok := regionID(w, r)
	if !ok {
		return
	}
	ctx, cancel := regionContext(r)
	defer cancel()
	if err := c.Model.Delete(ctx, id); err != nil {
		regionError(w, err)
		return
	}
	response.Success(w, "Region berhasil dihapus", nil)
}
