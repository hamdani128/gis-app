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

type RefValueSettingController struct {
	Model *adminModel.RefValueSettingModel
}

// Mengubah error database menjadi respons API.
func refError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "Terjadi kesalahan pada database"

	var pgErr *pgconn.PgError

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		status = http.StatusNotFound
		message = "Data tidak ditemukan"

	case errors.As(err, &pgErr):
		switch pgErr.Code {
		case "23505":
			status = http.StatusConflict
			message = "Data melanggar aturan unik database"

		case "23503":
			status = http.StatusConflict
			message = "Data terkait dengan record lain"

		case "22001", "23502", "23514":
			status = http.StatusBadRequest
			message = "Data tidak sesuai batasan kolom database"
		}
	}

	log.Printf("Ref value setting: %v", err)
	response.Error(w, status, message)
}

// Membaca dan memvalidasi ID dari URL.
func refID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)

	if err != nil || id <= 0 {
		response.Error(
			w,
			http.StatusBadRequest,
			"ID harus berupa angka positif",
		)
		return 0, false
	}

	return id, true
}

// Membaca dan memvalidasi body POST / PUT.
func readRefInput(
	w http.ResponseWriter,
	r *http.Request,
) (adminModel.RefValueSettingInput, bool) {
	var input adminModel.RefValueSettingInput

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			"Body JSON tidak valid atau berisi field yang tidak dikenal",
		)
		return input, false
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		response.Error(
			w,
			http.StatusBadRequest,
			"Body harus berisi satu objek JSON",
		)
		return input, false
	}

	input.Domain = strings.TrimSpace(input.Domain)
	input.LowValue = strings.TrimSpace(input.LowValue)

	if input.Domain == "" || input.LowValue == "" {
		response.Error(
			w,
			http.StatusBadRequest,
			"Domain dan low_value wajib diisi",
		)
		return input, false
	}

	return input, true
}

// GET /api/ref-value-settings
func (c *RefValueSettingController) Index(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items, err := c.Model.GetAll(ctx)
	if err != nil {
		refError(w, err)
		return
	}

	response.Success(w, "Data berhasil diambil", items)
}

// GET /api/ref-value-settings/{id}
func (c *RefValueSettingController) Show(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, ok := refID(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := c.Model.GetByID(ctx, id)
	if err != nil {
		refError(w, err)
		return
	}

	response.Success(w, "Detail data berhasil diambil", item)
}

// POST /api/ref-value-settings
func (c *RefValueSettingController) Store(
	w http.ResponseWriter,
	r *http.Request,
) {
	input, ok := readRefInput(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := c.Model.Create(ctx, input)
	if err != nil {
		refError(w, err)
		return
	}

	w.Header().Set(
		"Location",
		"/api/ref-value-settings/"+strconv.FormatInt(item.ID, 10),
	)

	response.Created(w, "Data berhasil ditambahkan", item)
}

// PUT /api/ref-value-settings/{id}
func (c *RefValueSettingController) Update(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, ok := refID(w, r)
	if !ok {
		return
	}

	input, ok := readRefInput(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	item, err := c.Model.Update(ctx, id, input)
	if err != nil {
		refError(w, err)
		return
	}

	response.Success(w, "Data berhasil diperbarui", item)
}

// DELETE /api/ref-value-settings/{id}
func (c *RefValueSettingController) Destroy(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, ok := refID(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := c.Model.Delete(ctx, id); err != nil {
		refError(w, err)
		return
	}

	response.Success(w, "Data berhasil dihapus", nil)
}
