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

type ModuleController struct {
	Model *adminModel.ModuleModel
}

func moduleError(w http.ResponseWriter, err error) {
	status, message := http.StatusInternalServerError, "Terjadi kesalahan pada database"
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		status, message = http.StatusNotFound, "Data modul tidak ditemukan"
	case errors.As(err, &pgErr):
		switch pgErr.Code {
		case "23505":
			status, message = http.StatusConflict, "ID modul sudah digunakan"
		case "23503":
			status, message = http.StatusConflict, "Modul masih dipakai oleh hak akses user"
		case "22001", "23502", "23514", "22P02":
			status, message = http.StatusBadRequest, "Data tidak sesuai format atau batas kolom database"
		}
	}
	log.Printf("Module: %v", err)
	response.Error(w, status, message)
}

func moduleID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, http.StatusBadRequest, "ID harus berupa angka positif")
		return 0, false
	}
	return id, true
}

func readModuleInput(w http.ResponseWriter, r *http.Request) (adminModel.ModuleInput, bool) {
	var input adminModel.ModuleInput
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
	adminModel.NormalizeModuleInput(&input)
	if input.ModuleName == "" || input.ModuleCode == "" {
		response.Error(w, http.StatusBadRequest, "Module name dan module code wajib diisi")
		return input, false
	}
	if input.Sequence != nil && *input.Sequence < 0 {
		response.Error(w, http.StatusBadRequest, "Seq tidak boleh bernilai negatif")
		return input, false
	}
	if input.PrivType != nil {
		value := strings.ToLower(*input.PrivType)
		if value != "all" && value != "edit" && value != "view" {
			response.Error(w, http.StatusBadRequest, "Privtype harus berupa view, edit, atau all")
			return input, false
		}
		input.PrivType = &value
	}
	return input, true
}

func moduleContext(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 8*time.Second)
}

func (c *ModuleController) Index(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := moduleContext(r)
	defer cancel()
	items, err := c.Model.GetAll(ctx)
	if err != nil {
		moduleError(w, err)
		return
	}
	response.Success(w, "Data modul berhasil diambil", items)
}

func (c *ModuleController) Options(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := moduleContext(r)
	defer cancel()
	options, err := c.Model.GetOptions(ctx)
	if err != nil {
		moduleError(w, err)
		return
	}
	response.Success(w, "Pilihan modul berhasil diambil", options)
}

func (c *ModuleController) Show(w http.ResponseWriter, r *http.Request) {
	id, ok := moduleID(w, r)
	if !ok {
		return
	}
	ctx, cancel := moduleContext(r)
	defer cancel()
	item, err := c.Model.GetByID(ctx, id)
	if err != nil {
		moduleError(w, err)
		return
	}
	response.Success(w, "Detail modul berhasil diambil", item)
}

func (c *ModuleController) Store(w http.ResponseWriter, r *http.Request) {
	input, ok := readModuleInput(w, r)
	if !ok {
		return
	}
	ctx, cancel := moduleContext(r)
	defer cancel()
	item, err := c.Model.Create(ctx, input)
	if err != nil {
		moduleError(w, err)
		return
	}
	if item.ID != nil {
		w.Header().Set("Location", "/api/modules/"+strconv.Itoa(int(*item.ID)))
	}
	response.Created(w, "Modul berhasil ditambahkan", item)
}

func (c *ModuleController) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := moduleID(w, r)
	if !ok {
		return
	}
	input, ok := readModuleInput(w, r)
	if !ok {
		return
	}
	ctx, cancel := moduleContext(r)
	defer cancel()
	item, err := c.Model.Update(ctx, id, input)
	if err != nil {
		moduleError(w, err)
		return
	}
	response.Success(w, "Modul berhasil diperbarui", item)
}

func (c *ModuleController) Destroy(w http.ResponseWriter, r *http.Request) {
	id, ok := moduleID(w, r)
	if !ok {
		return
	}
	ctx, cancel := moduleContext(r)
	defer cancel()
	if err := c.Model.Delete(ctx, id); err != nil {
		moduleError(w, err)
		return
	}
	response.Success(w, "Modul berhasil dihapus", nil)
}
