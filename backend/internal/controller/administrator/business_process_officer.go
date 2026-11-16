package administrator

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	adminModel "gis-app/backend/internal/model/administrator"
	"gis-app/backend/internal/response"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type BusinessProcessOfficerController struct {
	Model *adminModel.BusinessProcessOfficerModel
}

func businessProcessOfficerError(w http.ResponseWriter, err error) {
	status, message := http.StatusInternalServerError, "Terjadi kesalahan pada database"
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		status, message = http.StatusNotFound, "Data business process officer tidak ditemukan"
	case errors.As(err, &pgErr):
		switch pgErr.Code {
		case "23505":
			status, message = http.StatusConflict, "ID business process officer sudah digunakan"
		case "23503":
			status, message = http.StatusConflict, "Employee atau process sequence masih dipakai oleh record lain"
		case "22001", "23502", "23514", "22P02":
			status, message = http.StatusBadRequest, "Data tidak sesuai format atau batas kolom database"
		}
	}
	log.Printf("Business process officer: %v", err)
	response.Error(w, status, message)
}

func businessProcessOfficerID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, http.StatusBadRequest, "ID harus berupa angka positif")
		return 0, false
	}
	return id, true
}

func readBusinessProcessOfficerInput(w http.ResponseWriter, r *http.Request) (adminModel.BusinessProcessOfficerInput, bool) {
	var input adminModel.BusinessProcessOfficerInput
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
	if input.EmployeeID == nil || *input.EmployeeID <= 0 || input.ProcessSequenceID == nil || *input.ProcessSequenceID <= 0 {
		response.Error(w, http.StatusBadRequest, "Employee dan process sequence wajib dipilih")
		return input, false
	}
	return input, true
}

func (c *BusinessProcessOfficerController) validateRelations(ctx context.Context, w http.ResponseWriter, input adminModel.BusinessProcessOfficerInput) bool {
	employeeExists, err := c.Model.EmployeeExists(ctx, *input.EmployeeID)
	if err != nil {
		businessProcessOfficerError(w, err)
		return false
	}
	if !employeeExists {
		response.Error(w, http.StatusBadRequest, "Employee yang dipilih tidak ditemukan")
		return false
	}
	sequenceExists, err := c.Model.SequenceExists(ctx, *input.ProcessSequenceID)
	if err != nil {
		businessProcessOfficerError(w, err)
		return false
	}
	if !sequenceExists {
		response.Error(w, http.StatusBadRequest, "Process sequence yang dipilih tidak ditemukan")
		return false
	}
	return true
}

func (c *BusinessProcessOfficerController) Index(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	items, err := c.Model.GetAll(ctx)
	if err != nil {
		businessProcessOfficerError(w, err)
		return
	}
	response.Success(w, "Data business process officer berhasil diambil", items)
}

func (c *BusinessProcessOfficerController) Show(w http.ResponseWriter, r *http.Request) {
	id, ok := businessProcessOfficerID(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	item, err := c.Model.GetByID(ctx, id)
	if err != nil {
		businessProcessOfficerError(w, err)
		return
	}
	response.Success(w, "Detail business process officer berhasil diambil", item)
}

func (c *BusinessProcessOfficerController) Store(w http.ResponseWriter, r *http.Request) {
	input, ok := readBusinessProcessOfficerInput(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	if !c.validateRelations(ctx, w, input) {
		return
	}
	item, err := c.Model.Create(ctx, input)
	if err != nil {
		businessProcessOfficerError(w, err)
		return
	}
	if item.ID != nil {
		w.Header().Set("Location", "/api/business-process-officers/"+strconv.Itoa(int(*item.ID)))
	}
	response.Created(w, "Business process officer berhasil ditambahkan", item)
}

func (c *BusinessProcessOfficerController) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := businessProcessOfficerID(w, r)
	if !ok {
		return
	}
	input, ok := readBusinessProcessOfficerInput(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	if !c.validateRelations(ctx, w, input) {
		return
	}
	item, err := c.Model.Update(ctx, id, input)
	if err != nil {
		businessProcessOfficerError(w, err)
		return
	}
	response.Success(w, "Business process officer berhasil diperbarui", item)
}

func (c *BusinessProcessOfficerController) Destroy(w http.ResponseWriter, r *http.Request) {
	id, ok := businessProcessOfficerID(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	if err := c.Model.Delete(ctx, id); err != nil {
		businessProcessOfficerError(w, err)
		return
	}
	response.Success(w, "Business process officer berhasil dihapus", nil)
}
