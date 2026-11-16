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

type DocTypeController struct {
	Model *adminModel.DocTypeModel
}

func docTypeError(w http.ResponseWriter, err error) {
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
			message = "ID tipe dokumen sudah digunakan"
		case "23503":
			status = http.StatusConflict
			message = "Data terkait dengan record lain"
		case "22001", "23502", "23514", "22P02":
			status = http.StatusBadRequest
			message = "Data tidak sesuai format atau batas kolom database"
		}
	}

	log.Printf("Doc type: %v", err)
	response.Error(w, status, message)
}

func readDocTypeInput(w http.ResponseWriter, r *http.Request) (adminModel.DocTypeInput, bool) {
	var input adminModel.DocTypeInput
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

	input.TypeName = strings.TrimSpace(input.TypeName)
	input.FileType = strings.TrimSpace(input.FileType)
	if input.TypeDocGroup != nil {
		value := strings.TrimSpace(*input.TypeDocGroup)
		if value == "" {
			input.TypeDocGroup = nil
		} else {
			input.TypeDocGroup = &value
		}
	}
	if input.TypeName == "" || input.FileType == "" {
		response.Error(w, http.StatusBadRequest, "Type name dan file type wajib diisi")
		return input, false
	}
	if input.Mandatory != 0 && input.Mandatory != 1 {
		response.Error(w, http.StatusBadRequest, "Mandatory harus bernilai active atau inactive")
		return input, false
	}
	if input.SeqRelated != nil && *input.SeqRelated < 0 {
		response.Error(w, http.StatusBadRequest, "Seq related tidak boleh bernilai negatif")
		return input, false
	}
	return input, true
}

func docTypeID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, http.StatusBadRequest, "ID harus berupa angka positif")
		return 0, false
	}
	return id, true
}

func (c *DocTypeController) Index(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	items, err := c.Model.GetAll(ctx)
	if err != nil {
		docTypeError(w, err)
		return
	}
	response.Success(w, "Data tipe dokumen berhasil diambil", items)
}

func (c *DocTypeController) Show(w http.ResponseWriter, r *http.Request) {
	id, ok := docTypeID(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	item, err := c.Model.GetByID(ctx, id)
	if err != nil {
		docTypeError(w, err)
		return
	}
	response.Success(w, "Detail tipe dokumen berhasil diambil", item)
}

func (c *DocTypeController) Store(w http.ResponseWriter, r *http.Request) {
	input, ok := readDocTypeInput(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	item, err := c.Model.Create(ctx, input)
	if err != nil {
		docTypeError(w, err)
		return
	}
	w.Header().Set("Location", "/api/doc-types/"+strconv.FormatInt(item.ID, 10))
	response.Created(w, "Tipe dokumen berhasil ditambahkan", item)
}

func (c *DocTypeController) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := docTypeID(w, r)
	if !ok {
		return
	}
	input, ok := readDocTypeInput(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	item, err := c.Model.Update(ctx, id, input)
	if err != nil {
		docTypeError(w, err)
		return
	}
	response.Success(w, "Tipe dokumen berhasil diperbarui", item)
}

func (c *DocTypeController) Destroy(w http.ResponseWriter, r *http.Request) {
	id, ok := docTypeID(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := c.Model.Delete(ctx, id); err != nil {
		docTypeError(w, err)
		return
	}
	response.Success(w, "Tipe dokumen berhasil dihapus", nil)
}
