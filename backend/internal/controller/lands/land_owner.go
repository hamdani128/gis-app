package lands

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	landModel "gis-app/backend/internal/model/lands"
	"gis-app/backend/internal/response"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type LandOwnerController struct{ Model *landModel.LandOwnerModel }

func ownerParams(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	companyID, err := strconv.ParseInt(r.URL.Query().Get("company_id"), 10, 64)
	if err != nil || companyID < 0 {
		response.Error(w, 400, "company_id wajib berupa angka nol atau positif")
		return 0, 0, false
	}
	var id int64
	if r.PathValue("id") != "" {
		id, err = strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			response.Error(w, 400, "ID Land Owner tidak valid")
			return 0, 0, false
		}
	}
	return companyID, id, true
}

func ownerFailure(w http.ResponseWriter, err error) {
	status, msg := 500, "Gagal memproses Land Owner"
	if errors.Is(err, pgx.ErrNoRows) {
		status, msg = 404, "Land Owner tidak ditemukan pada company aktif"
	} else {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			status, msg = 409, "Land Owner masih digunakan oleh data land atau relasi lain"
		} else if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			status, msg = 409, "Nomor KTP atau identitas owner sudah digunakan"
		} else if errors.As(err, &pgErr) && pgErr.Code == "23502" {
			status, msg = 400, "Kolom wajib pada Land Owner belum lengkap"
		} else if err.Error() == "company tidak valid untuk pilihan aktif" || err.Error() == "company tidak ditemukan" || err.Error() == "name wajib diisi" {
			status, msg = 400, err.Error()
		}
	}
	log.Printf("Land Owner: %v", err)
	response.Error(w, status, msg)
}

func (c *LandOwnerController) Index(w http.ResponseWriter, r *http.Request) {
	companyID, _, ok := ownerParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	items, err := c.Model.GetAll(ctx, companyID)
	if err != nil {
		ownerFailure(w, err)
		return
	}
	response.Success(w, "Data Land Owners berhasil diambil", items)
}
func (c *LandOwnerController) Options(w http.ResponseWriter, r *http.Request) {
	companyID, _, ok := ownerParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	items, err := c.Model.Companies(ctx, companyID)
	if err != nil {
		ownerFailure(w, err)
		return
	}
	response.Success(w, "Pilihan company berhasil diambil", items)
}
func (c *LandOwnerController) Lands(w http.ResponseWriter, r *http.Request) {
	companyID, id, ok := ownerParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	items, err := c.Model.Lands(ctx, companyID, id)
	if err != nil {
		ownerFailure(w, err)
		return
	}
	response.Success(w, "Daftar land milik owner berhasil diambil", items)
}
func ownerLandID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("land_id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, 400, "ID land tidak valid")
		return 0, false
	}
	return id, true
}
func (c *LandOwnerController) LandDetail(w http.ResponseWriter, r *http.Request) {
	companyID, ownerID, ok := ownerParams(w, r)
	if !ok {
		return
	}
	landID, ok := ownerLandID(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	item, err := c.Model.LandDetail(ctx, companyID, ownerID, landID)
	if err != nil {
		ownerFailure(w, err)
		return
	}
	response.Success(w, "Detail land berhasil diambil", item)
}
func (c *LandOwnerController) LandProcesses(w http.ResponseWriter, r *http.Request) {
	companyID, ownerID, ok := ownerParams(w, r)
	if !ok {
		return
	}
	landID, ok := ownerLandID(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	items, err := c.Model.LandProcesses(ctx, companyID, ownerID, landID)
	if err != nil {
		ownerFailure(w, err)
		return
	}
	response.Success(w, "Proses land berhasil diambil", items)
}
func (c *LandOwnerController) LandMap(w http.ResponseWriter, r *http.Request) {
	companyID, ownerID, ok := ownerParams(w, r)
	if !ok {
		return
	}
	landID, ok := ownerLandID(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	mapData, err := c.Model.LandMap(ctx, companyID, ownerID, landID)
	if err != nil {
		ownerFailure(w, err)
		return
	}
	response.Success(w, "Peta land berhasil diambil", map[string]string{"map_data": mapData})
}
func (c *LandOwnerController) save(w http.ResponseWriter, r *http.Request, id *int64) {
	companyID, _, ok := ownerParams(w, r)
	if !ok {
		return
	}
	var input landModel.LandOwnerInput
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		response.Error(w, 400, "Data form tidak valid")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	item, err := c.Model.Save(ctx, companyID, id, input)
	if err != nil {
		ownerFailure(w, err)
		return
	}
	if id == nil {
		response.Created(w, "Land Owner berhasil ditambahkan", item)
	} else {
		response.Success(w, "Land Owner berhasil diperbarui", item)
	}
}
func (c *LandOwnerController) Store(w http.ResponseWriter, r *http.Request) { c.save(w, r, nil) }
func (c *LandOwnerController) Update(w http.ResponseWriter, r *http.Request) {
	_, id, ok := ownerParams(w, r)
	if ok {
		c.save(w, r, &id)
	}
}
func (c *LandOwnerController) Destroy(w http.ResponseWriter, r *http.Request) {
	companyID, id, ok := ownerParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if err := c.Model.Delete(ctx, companyID, id); err != nil {
		ownerFailure(w, err)
		return
	}
	response.Success(w, "Land Owner berhasil dihapus", nil)
}
