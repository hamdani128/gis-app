package companies

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	companyModel "gis-app/backend/internal/model/companies"
	"gis-app/backend/internal/response"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type LandPermitController struct{ Model *companyModel.LandPermitModel }

func landPermitErr(w http.ResponseWriter, err error) {
	status, msg := http.StatusInternalServerError, "Gagal memproses data land permit"
	var pg *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		status, msg = http.StatusNotFound, "Land permit tidak ditemukan pada company aktif"
	case errors.As(err, &pg):
		if pg.Code == "23503" {
			status, msg = http.StatusBadRequest, "Company atau province yang dipilih tidak valid"
		} else if pg.Code == "22001" || pg.Code == "23502" || pg.Code == "22P02" {
			status, msg = http.StatusBadRequest, "Data tidak sesuai format database"
		}
	}
	log.Printf("Land permit: %v", err)
	response.Error(w, status, msg)
}
func landPermitIDs(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	companyID, e := strconv.ParseInt(r.URL.Query().Get("company_id"), 10, 64)
	if e != nil || companyID <= 0 {
		response.Error(w, 400, "company_id wajib berupa angka positif")
		return 0, 0, false
	}
	var id int64
	if r.PathValue("id") != "" {
		id, e = strconv.ParseInt(r.PathValue("id"), 10, 64)
		if e != nil || id <= 0 {
			response.Error(w, 400, "ID permit tidak valid")
			return 0, 0, false
		}
	}
	return companyID, id, true
}
func permitContext(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 15*time.Second)
}
func (c *LandPermitController) Index(w http.ResponseWriter, r *http.Request) {
	companyID, _, ok := landPermitIDs(w, r)
	if !ok {
		return
	}
	ctx, cancel := permitContext(r)
	defer cancel()
	items, e := c.Model.GetAll(ctx, companyID)
	if e != nil {
		landPermitErr(w, e)
		return
	}
	response.Success(w, "Data land permit berhasil diambil", items)
}
func (c *LandPermitController) Options(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := permitContext(r)
	defer cancel()
	items, e := c.Model.GetOptions(ctx)
	if e != nil {
		landPermitErr(w, e)
		return
	}
	response.Success(w, "Pilihan land permit berhasil diambil", items)
}

func storePermitFile(file multipart.File, header *multipart.FileHeader, kind string) (string, error) {
	defer file.Close()
	if header.Size > 20*1024*1024 {
		return "", fmt.Errorf("Ukuran file %s maksimal 20 MB", kind)
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".pdf": true, ".png": true, ".jpg": true, ".jpeg": true, ".json": true, ".geojson": true, ".csv": true, ".zip": true, ".mif": true, ".mid": true}
	if !allowed[ext] {
		return "", fmt.Errorf("Format file %s tidak didukung", kind)
	}
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", err
	}
	dir := filepath.Join("uploads", "land-permits")
	cwd, _ := os.Getwd()
	if filepath.Base(cwd) != "backend" {
		dir = filepath.Join("backend", dir)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	name := kind + "-" + hex.EncodeToString(token[:]) + ext
	path := filepath.Join(dir, name)
	out, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return "", err
	}
	if _, err = io.Copy(out, file); err != nil {
		out.Close()
		os.Remove(path)
		return "", err
	}
	if err = out.Close(); err != nil {
		return "", err
	}
	return "/api/land-permit-files/" + name, nil
}

func ServeLandPermitFile(w http.ResponseWriter, r *http.Request) {
	name := filepath.Base(r.PathValue("name"))
	if name == "." || name == "/" {
		http.NotFound(w, r)
		return
	}
	cwd, _ := os.Getwd()
	dir := filepath.Join("backend", "uploads", "land-permits")
	if filepath.Base(cwd) == "backend" {
		dir = filepath.Join("uploads", "land-permits")
	}
	http.ServeFile(w, r, filepath.Join(dir, name))
}

func readPermitInput(w http.ResponseWriter, r *http.Request) (companyModel.LandPermitInput, bool) {
	var in companyModel.LandPermitInput
	r.Body = http.MaxBytesReader(w, r.Body, 45*1024*1024)
	if err := r.ParseMultipartForm(45 * 1024 * 1024); err != nil {
		response.Error(w, 400, "Form atau ukuran upload tidak valid")
		return in, false
	}
	if err := json.Unmarshal([]byte(r.FormValue("data")), &in); err != nil {
		response.Error(w, 400, "Data form tidak valid")
		return in, false
	}
	if in.CompanyID <= 0 || strings.TrimSpace(in.PermitNumber) == "" {
		response.Error(w, 400, "Company dan permit number wajib diisi")
		return in, false
	}
	if f, h, e := r.FormFile("permit_document"); e == nil {
		path, err := storePermitFile(f, h, "permit")
		if err != nil {
			response.Error(w, 400, err.Error())
			return in, false
		}
		in.PermitDocument = path
	}
	if f, h, e := r.FormFile("map_file"); e == nil {
		path, err := storePermitFile(f, h, "map")
		if err != nil {
			response.Error(w, 400, err.Error())
			return in, false
		}
		in.MapFile = path
	}
	return in, true
}
func (c *LandPermitController) Store(w http.ResponseWriter, r *http.Request) {
	in, ok := readPermitInput(w, r)
	if !ok {
		return
	}
	ctx, cancel := permitContext(r)
	defer cancel()
	item, e := c.Model.Save(ctx, in.CompanyID, nil, in)
	if e != nil {
		landPermitErr(w, e)
		return
	}
	response.Created(w, "Land permit berhasil ditambahkan", item)
}
func (c *LandPermitController) Update(w http.ResponseWriter, r *http.Request) {
	companyID, id, ok := landPermitIDs(w, r)
	if !ok {
		return
	}
	in, ok := readPermitInput(w, r)
	if !ok {
		return
	}
	if in.CompanyID != companyID {
		response.Error(w, 400, "Company form tidak sama dengan company aktif")
		return
	}
	ctx, cancel := permitContext(r)
	defer cancel()
	item, e := c.Model.Save(ctx, companyID, &id, in)
	if e != nil {
		landPermitErr(w, e)
		return
	}
	response.Success(w, "Land permit berhasil diperbarui", item)
}
func (c *LandPermitController) Destroy(w http.ResponseWriter, r *http.Request) {
	companyID, id, ok := landPermitIDs(w, r)
	if !ok {
		return
	}
	ctx, cancel := permitContext(r)
	defer cancel()
	if e := c.Model.Delete(ctx, companyID, id); e != nil {
		landPermitErr(w, e)
		return
	}
	response.Success(w, "Land permit berhasil dihapus", nil)
}
