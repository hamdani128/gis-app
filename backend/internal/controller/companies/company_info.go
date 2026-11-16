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

type CompanyInfoController struct {
	Model *companyModel.CompanyInfoModel
}

func companyInfoError(w http.ResponseWriter, err error) {
	status, message := http.StatusInternalServerError, "Terjadi kesalahan pada database"
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		status, message = http.StatusNotFound, "Data company tidak ditemukan"
	case errors.As(err, &pgErr):
		switch pgErr.Code {
		case "23505":
			status, message = http.StatusConflict, "ID company sudah digunakan"
		case "23503":
			status, message = http.StatusConflict, "Company masih dipakai oleh employee atau data lain"
		case "22001", "23502", "23514", "22P02":
			status, message = http.StatusBadRequest, "Data tidak sesuai format atau batas kolom database"
		}
	}
	log.Printf("Company info: %v", err)
	response.Error(w, status, message)
}

func companyInfoID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 0 {
		response.Error(w, http.StatusBadRequest, "ID harus berupa angka nol atau positif")
		return 0, false
	}
	return id, true
}

func readCompanyInfoInput(w http.ResponseWriter, r *http.Request) (companyModel.CompanyInfoInput, bool) {
	var input companyModel.CompanyInfoInput
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
	companyModel.NormalizeCompanyInfoInput(&input)
	if input.Name == "" {
		response.Error(w, http.StatusBadRequest, "Name wajib diisi")
		return input, false
	}
	for _, id := range []*int32{input.ProvinceID, input.RegionID} {
		if id != nil && *id < 0 {
			response.Error(w, http.StatusBadRequest, "Province dan Region harus dipilih dari daftar yang tersedia")
			return input, false
		}
	}
	return input, true
}

func companyInfoContext(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 8*time.Second)
}

func (c *CompanyInfoController) validateRelations(ctx context.Context, w http.ResponseWriter, input companyModel.CompanyInfoInput) bool {
	if input.ProvinceID != nil {
		exists, err := c.Model.ProvinceExists(ctx, *input.ProvinceID)
		if err != nil {
			companyInfoError(w, err)
			return false
		}
		if !exists {
			response.Error(w, http.StatusBadRequest, "Province yang dipilih tidak ditemukan dalam domain PROVINCE")
			return false
		}
	}
	if input.RegionID != nil {
		exists, err := c.Model.RegionExists(ctx, *input.RegionID)
		if err != nil {
			companyInfoError(w, err)
			return false
		}
		if !exists {
			response.Error(w, http.StatusBadRequest, "Region yang dipilih tidak ditemukan")
			return false
		}
	}
	return true
}

func (c *CompanyInfoController) Index(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := companyInfoContext(r)
	defer cancel()
	items, err := c.Model.GetAll(ctx)
	if err != nil {
		companyInfoError(w, err)
		return
	}
	response.Success(w, "Data company berhasil diambil", items)
}

func (c *CompanyInfoController) Options(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := companyInfoContext(r)
	defer cancel()
	items, err := c.Model.GetOptions(ctx)
	if err != nil {
		companyInfoError(w, err)
		return
	}
	response.Success(w, "Pilihan company berhasil diambil", items)
}

func (c *CompanyInfoController) Show(w http.ResponseWriter, r *http.Request) {
	id, ok := companyInfoID(w, r)
	if !ok {
		return
	}
	ctx, cancel := companyInfoContext(r)
	defer cancel()
	item, err := c.Model.GetByID(ctx, id)
	if err != nil {
		companyInfoError(w, err)
		return
	}
	response.Success(w, "Detail company berhasil diambil", item)
}

func (c *CompanyInfoController) Store(w http.ResponseWriter, r *http.Request) {
	input, ok := readCompanyInfoInput(w, r)
	if !ok {
		return
	}
	ctx, cancel := companyInfoContext(r)
	defer cancel()
	if !c.validateRelations(ctx, w, input) {
		return
	}
	item, err := c.Model.Create(ctx, input)
	if err != nil {
		companyInfoError(w, err)
		return
	}
	if item.ID != nil {
		w.Header().Set("Location", "/api/company-info/"+strconv.Itoa(int(*item.ID)))
	}
	response.Created(w, "Company berhasil ditambahkan", item)
}

func (c *CompanyInfoController) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := companyInfoID(w, r)
	if !ok {
		return
	}
	input, ok := readCompanyInfoInput(w, r)
	if !ok {
		return
	}
	ctx, cancel := companyInfoContext(r)
	defer cancel()
	if !c.validateRelations(ctx, w, input) {
		return
	}
	item, err := c.Model.Update(ctx, id, input)
	if err != nil {
		companyInfoError(w, err)
		return
	}
	response.Success(w, "Company berhasil diperbarui", item)
}

func (c *CompanyInfoController) Destroy(w http.ResponseWriter, r *http.Request) {
	id, ok := companyInfoID(w, r)
	if !ok {
		return
	}
	ctx, cancel := companyInfoContext(r)
	defer cancel()
	if err := c.Model.Delete(ctx, id); err != nil {
		companyInfoError(w, err)
		return
	}
	response.Success(w, "Company berhasil dihapus", nil)
}
