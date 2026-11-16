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

type BusinessProcessSequenceController struct {
	Model *adminModel.BusinessProcessSequenceModel
}

func businessProcessSequenceError(w http.ResponseWriter, err error) {
	status, message := http.StatusInternalServerError, "Terjadi kesalahan pada database"
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		status, message = http.StatusNotFound, "Data business process sequence tidak ditemukan"
	case errors.As(err, &pgErr):
		switch pgErr.Code {
		case "23505":
			status, message = http.StatusConflict, "ID business process sequence sudah digunakan"
		case "23503":
			status, message = http.StatusConflict, "Process role masih dipakai oleh record lain"
		case "22001", "23502", "23514", "22P02":
			status, message = http.StatusBadRequest, "Data tidak sesuai format atau batas kolom database"
		}
	}
	log.Printf("Business process sequence: %v", err)
	response.Error(w, status, message)
}

func businessProcessSequenceID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, http.StatusBadRequest, "ID harus berupa angka positif")
		return 0, false
	}
	return id, true
}

func readBusinessProcessSequenceInput(w http.ResponseWriter, r *http.Request) (adminModel.BusinessProcessSequenceInput, bool) {
	var input adminModel.BusinessProcessSequenceInput
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
	adminModel.NormalizeBusinessProcessSequenceInput(&input)
	if input.ProcessRoleID == nil || *input.ProcessRoleID <= 0 {
		response.Error(w, http.StatusBadRequest, "Process role wajib dipilih")
		return input, false
	}
	if input.Name == "" {
		response.Error(w, http.StatusBadRequest, "Sequence name wajib diisi")
		return input, false
	}
	if input.Sequence == nil || *input.Sequence < 0 {
		response.Error(w, http.StatusBadRequest, "Sequence wajib diisi dan tidak boleh negatif")
		return input, false
	}
	return input, true
}

func (c *BusinessProcessSequenceController) validateRole(ctx context.Context, w http.ResponseWriter, roleID int32) bool {
	exists, err := c.Model.RoleExists(ctx, roleID)
	if err != nil {
		businessProcessSequenceError(w, err)
		return false
	}
	if !exists {
		response.Error(w, http.StatusBadRequest, "Process role yang dipilih tidak ditemukan")
		return false
	}
	return true
}

func (c *BusinessProcessSequenceController) Index(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	items, err := c.Model.GetAll(ctx)
	if err != nil {
		businessProcessSequenceError(w, err)
		return
	}
	response.Success(w, "Data business process sequence berhasil diambil", items)
}

func (c *BusinessProcessSequenceController) Show(w http.ResponseWriter, r *http.Request) {
	id, ok := businessProcessSequenceID(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	item, err := c.Model.GetByID(ctx, id)
	if err != nil {
		businessProcessSequenceError(w, err)
		return
	}
	response.Success(w, "Detail business process sequence berhasil diambil", item)
}

func (c *BusinessProcessSequenceController) Store(w http.ResponseWriter, r *http.Request) {
	input, ok := readBusinessProcessSequenceInput(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if !c.validateRole(ctx, w, *input.ProcessRoleID) {
		return
	}
	item, err := c.Model.Create(ctx, input)
	if err != nil {
		businessProcessSequenceError(w, err)
		return
	}
	if item.ID != nil {
		w.Header().Set("Location", "/api/business-process-sequences/"+strconv.Itoa(int(*item.ID)))
	}
	response.Created(w, "Business process sequence berhasil ditambahkan", item)
}

func (c *BusinessProcessSequenceController) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := businessProcessSequenceID(w, r)
	if !ok {
		return
	}
	input, ok := readBusinessProcessSequenceInput(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if !c.validateRole(ctx, w, *input.ProcessRoleID) {
		return
	}
	item, err := c.Model.Update(ctx, id, input)
	if err != nil {
		businessProcessSequenceError(w, err)
		return
	}
	response.Success(w, "Business process sequence berhasil diperbarui", item)
}

func (c *BusinessProcessSequenceController) Destroy(w http.ResponseWriter, r *http.Request) {
	id, ok := businessProcessSequenceID(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := c.Model.Delete(ctx, id); err != nil {
		businessProcessSequenceError(w, err)
		return
	}
	response.Success(w, "Business process sequence berhasil dihapus", nil)
}
