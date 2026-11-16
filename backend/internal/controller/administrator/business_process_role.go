package administrator

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

	adminModel "gis-app/backend/internal/model/administrator"
	"gis-app/backend/internal/response"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type BusinessProcessRoleController struct {
	Model *adminModel.BusinessProcessRoleModel
}

func businessProcessRoleError(w http.ResponseWriter, err error) {
	status, message := http.StatusInternalServerError, "Terjadi kesalahan pada database"
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		status, message = http.StatusNotFound, "Data business process role tidak ditemukan"
	case errors.As(err, &pgErr):
		switch pgErr.Code {
		case "23505":
			status, message = http.StatusConflict, "ID business process role sudah digunakan"
		case "23503":
			status, message = http.StatusConflict, "Data masih dipakai oleh record lain"
		case "22001", "23502", "23514", "22P02", "22007", "22008":
			status, message = http.StatusBadRequest, "Data tidak sesuai format atau batas kolom database"
		}
	}
	log.Printf("Business process role: %v", err)
	response.Error(w, status, message)
}

func businessProcessRoleID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, http.StatusBadRequest, "ID harus berupa angka positif")
		return 0, false
	}
	return id, true
}

func normalizeRoleDateTime(value *string) bool {
	if value == nil {
		return true
	}
	valid := false
	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02T15:04:05", time.RFC3339} {
		if _, err := time.Parse(layout, *value); err == nil {
			valid = true
			break
		}
	}
	if !valid {
		return false
	}
	if strings.Contains(*value, "T") {
		valueWithoutZone := strings.Replace(*value, "T", " ", 1)
		if len(valueWithoutZone) > 19 {
			valueWithoutZone = valueWithoutZone[:19]
		}
		*value = valueWithoutZone
	}
	return true
}

func readBusinessProcessRoleInput(w http.ResponseWriter, r *http.Request) (adminModel.BusinessProcessRoleInput, bool) {
	var input adminModel.BusinessProcessRoleInput
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

	adminModel.NormalizeBusinessProcessRoleInput(&input)
	if input.RoleName == "" {
		response.Error(w, http.StatusBadRequest, "Role name wajib diisi")
		return input, false
	}
	if !normalizeRoleDateTime(input.ValidFrom) {
		response.Error(w, http.StatusBadRequest, "Valid from harus berupa tanggal dan waktu yang valid")
		return input, false
	}
	if !normalizeRoleDateTime(input.ValidUntil) {
		response.Error(w, http.StatusBadRequest, "Valid until harus berupa tanggal dan waktu yang valid")
		return input, false
	}
	return input, true
}

func (c *BusinessProcessRoleController) Index(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	items, err := c.Model.GetAll(ctx)
	if err != nil {
		businessProcessRoleError(w, err)
		return
	}
	response.Success(w, "Data business process role berhasil diambil", items)
}

func (c *BusinessProcessRoleController) Show(w http.ResponseWriter, r *http.Request) {
	id, ok := businessProcessRoleID(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	item, err := c.Model.GetByID(ctx, id)
	if err != nil {
		businessProcessRoleError(w, err)
		return
	}
	response.Success(w, "Detail business process role berhasil diambil", item)
}

func (c *BusinessProcessRoleController) Store(w http.ResponseWriter, r *http.Request) {
	input, ok := readBusinessProcessRoleInput(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	item, err := c.Model.Create(ctx, input)
	if err != nil {
		businessProcessRoleError(w, err)
		return
	}
	if item.ID != nil {
		w.Header().Set("Location", "/api/business-process-roles/"+strconv.Itoa(int(*item.ID)))
	}
	response.Created(w, "Business process role berhasil ditambahkan", item)
}

func (c *BusinessProcessRoleController) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := businessProcessRoleID(w, r)
	if !ok {
		return
	}
	input, ok := readBusinessProcessRoleInput(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	item, err := c.Model.Update(ctx, id, input)
	if err != nil {
		businessProcessRoleError(w, err)
		return
	}
	response.Success(w, "Business process role berhasil diperbarui", item)
}

func (c *BusinessProcessRoleController) Destroy(w http.ResponseWriter, r *http.Request) {
	id, ok := businessProcessRoleID(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := c.Model.Delete(ctx, id); err != nil {
		businessProcessRoleError(w, err)
		return
	}
	response.Success(w, "Business process role berhasil dihapus", nil)
}
