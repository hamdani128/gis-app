package lands

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

	landModel "gis-app/backend/internal/model/lands"
	"gis-app/backend/internal/response"
	"github.com/jackc/pgx/v5"
)

type PotentialLandController struct{ Model *landModel.PotentialLandModel }

func potentialLandContext(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 15*time.Second)
}
func potentialLandCompany(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	companyID, err := strconv.ParseInt(r.URL.Query().Get("company_id"), 10, 64)
	if err != nil || companyID < 0 {
		response.Error(w, 400, "company_id wajib berupa angka nol atau positif")
		return 0, 0, false
	}
	var id int64
	if r.PathValue("id") != "" {
		id, err = strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			response.Error(w, 400, "ID Potential land tidak valid")
			return 0, 0, false
		}
	}
	return companyID, id, true
}
func potentialLandFailure(w http.ResponseWriter, err error) {
	status, msg := http.StatusInternalServerError, "Gagal memproses data Potential land"
	if errors.Is(err, pgx.ErrNoRows) {
		status, msg = http.StatusNotFound, "Data Potential land tidak ditemukan pada company aktif"
	} else if strings.Contains(err.Error(), "Gis_Data_Spatials harus") || strings.Contains(err.Error(), "Gis_Potential_Lands harus") {
		status, msg = http.StatusUnprocessableEntity, err.Error()
	} else if strings.Contains(err.Error(), "GeoJSON") || strings.Contains(err.Error(), "mapdata harus") {
		status, msg = http.StatusBadRequest, err.Error()
	}
	log.Printf("Potential land: %v", err)
	response.Error(w, status, msg)
}
func (c *PotentialLandController) Index(w http.ResponseWriter, r *http.Request) {
	companyID, _, ok := potentialLandCompany(w, r)
	if !ok {
		return
	}
	ctx, cancel := potentialLandContext(r)
	defer cancel()
	items, err := c.Model.GetAll(ctx, companyID)
	if err != nil {
		potentialLandFailure(w, err)
		return
	}
	response.Success(w, "Data Potential land berhasil diambil", items)
}
func (c *PotentialLandController) Options(w http.ResponseWriter, r *http.Request) {
	companyID, _, ok := potentialLandCompany(w, r)
	if !ok {
		return
	}
	ctx, cancel := potentialLandContext(r)
	defer cancel()
	items, err := c.Model.GetOptions(ctx, companyID)
	if err != nil {
		potentialLandFailure(w, err)
		return
	}
	response.Success(w, "Pilihan Potential land berhasil diambil", items)
}
func (c *PotentialLandController) Map(w http.ResponseWriter, r *http.Request) {
	companyID, id, ok := potentialLandCompany(w, r)
	if !ok {
		return
	}
	ctx, cancel := potentialLandContext(r)
	defer cancel()
	uploadPath, err := c.Model.GetMapUpload(ctx, companyID, id)
	if err != nil {
		potentialLandFailure(w, err)
		return
	}
	mapData, err := c.Model.GetMap(ctx, companyID, id)
	if err != nil {
		if uploadPath == "" {
			potentialLandFailure(w, err)
			return
		}
		mapData = `{"type":"FeatureCollection","features":[]}`
	}
	response.Success(w, "Peta Potential land berhasil diambil", map[string]string{"map_data": mapData, "map_upload": uploadPath})
}

func potentialUploadDir() string {
	cwd, _ := os.Getwd()
	if filepath.Base(cwd) == "backend" {
		return filepath.Join("uploads", "potential-lands")
	}
	return filepath.Join("backend", "uploads", "potential-lands")
}

func storePotentialMapFile(file multipart.File, header *multipart.FileHeader) (string, string, error) {
	defer file.Close()
	if header.Size > 20*1024*1024 {
		return "", "", fmt.Errorf("Ukuran file koordinat maksimal 20 MB")
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !map[string]bool{".csv": true, ".json": true, ".geojson": true, ".mif": true, ".wkt": true, ".txt": true}[ext] {
		return "", "", fmt.Errorf("Format file koordinat tidak didukung")
	}
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", "", err
	}
	dir := potentialUploadDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", "", err
	}
	name := "map-" + hex.EncodeToString(token[:]) + ext
	fullPath := filepath.Join(dir, name)
	out, err := os.OpenFile(fullPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return "", "", err
	}
	if _, err = io.Copy(out, file); err != nil {
		out.Close()
		os.Remove(fullPath)
		return "", "", err
	}
	if err = out.Close(); err != nil {
		os.Remove(fullPath)
		return "", "", err
	}
	return "/api/potential-map-files/" + name, fullPath, nil
}

func (c *PotentialLandController) MapUpload(w http.ResponseWriter, r *http.Request) {
	companyID, id, ok := potentialLandCompany(w, r)
	if !ok {
		return
	}
	ctx, cancel := potentialLandContext(r)
	defer cancel()
	path, err := c.Model.GetMapUpload(ctx, companyID, id)
	if err != nil {
		potentialLandFailure(w, err)
		return
	}
	if path == "" {
		http.NotFound(w, r)
		return
	}
	name := filepath.Base(strings.ReplaceAll(path, "\\", "/"))
	if name == "." || name == "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join(potentialUploadDir(), name))
}

func (c *PotentialLandController) SaveMap(w http.ResponseWriter, r *http.Request) {
	companyID, id, ok := potentialLandCompany(w, r)
	if !ok {
		return
	}
	var input struct {
		MapData     string `json:"map_data"`
		ClearUpload bool   `json:"clear_upload"`
	}
	var uploadPath *string
	var storedPath string
	if strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "multipart/form-data") {
		r.Body = http.MaxBytesReader(w, r.Body, 25*1024*1024)
		if err := r.ParseMultipartForm(25 * 1024 * 1024); err != nil {
			response.Error(w, 400, "Form upload tidak valid")
			return
		}
		input.MapData = r.FormValue("map_data")
		input.ClearUpload = r.FormValue("clear_upload") == "true"
		file, header, err := r.FormFile("map_upload")
		if err == nil {
			path, fullPath, storeErr := storePotentialMapFile(file, header)
			if storeErr != nil {
				response.Error(w, 400, storeErr.Error())
				return
			}
			uploadPath, storedPath = &path, fullPath
		} else if !errors.Is(err, http.ErrMissingFile) {
			response.Error(w, 400, "File koordinat tidak valid")
			return
		}
	} else if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 20*1024*1024)).Decode(&input); err != nil {
		response.Error(w, 400, "Data peta tidak valid")
		return
	}
	if uploadPath == nil && input.ClearUpload {
		empty := ""
		uploadPath = &empty
	}
	if storedPath != "" && strings.TrimSpace(input.MapData) == "" {
		os.Remove(storedPath)
		response.Error(w, 400, "File koordinat tidak menghasilkan data peta")
		return
	}
	ctx, cancel := potentialLandContext(r)
	defer cancel()
	path, err := c.Model.GetMapUpload(ctx, companyID, id)
	if err != nil {
		if storedPath != "" {
			os.Remove(storedPath)
		}
		potentialLandFailure(w, err)
		return
	}
	mapData, err := c.Model.SaveMap(ctx, companyID, id, input.MapData, uploadPath)
	if err != nil {
		if storedPath != "" {
			os.Remove(storedPath)
		}
		potentialLandFailure(w, err)
		return
	}
	if uploadPath != nil {
		path = *uploadPath
	}
	response.Success(w, "Peta Potential land berhasil disimpan", map[string]string{"map_data": mapData, "map_upload": path})
}
func (c *PotentialLandController) save(w http.ResponseWriter, r *http.Request, id *int64) {
	companyID, _, ok := potentialLandCompany(w, r)
	if !ok {
		return
	}
	var in landModel.PotentialLandInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024*1024)).Decode(&in); err != nil {
		response.Error(w, 400, "Data form tidak valid")
		return
	}
	if strings.TrimSpace(in.PotentialName) == "" || in.LandPermitID <= 0 {
		response.Error(w, 400, "Potential name dan Land Permit wajib diisi")
		return
	}
	ctx, cancel := potentialLandContext(r)
	defer cancel()
	item, err := c.Model.Save(ctx, companyID, id, in)
	if err != nil {
		potentialLandFailure(w, err)
		return
	}
	if id == nil {
		response.Created(w, "Potential land berhasil ditambahkan", item)
	} else {
		response.Success(w, "Potential land berhasil diperbarui", item)
	}
}
func (c *PotentialLandController) Store(w http.ResponseWriter, r *http.Request) { c.save(w, r, nil) }
func (c *PotentialLandController) Update(w http.ResponseWriter, r *http.Request) {
	_, id, ok := potentialLandCompany(w, r)
	if !ok {
		return
	}
	c.save(w, r, &id)
}
func (c *PotentialLandController) Destroy(w http.ResponseWriter, r *http.Request) {
	companyID, id, ok := potentialLandCompany(w, r)
	if !ok {
		return
	}
	ctx, cancel := potentialLandContext(r)
	defer cancel()
	if err := c.Model.Delete(ctx, companyID, id); err != nil {
		potentialLandFailure(w, err)
		return
	}
	response.Success(w, "Potential land berhasil dihapus", nil)
}
