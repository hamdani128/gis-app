package acquisition

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	acquisitionModel "gis-app/backend/internal/model/acquisition"
	"gis-app/backend/internal/response"
	"github.com/jackc/pgx/v5"
)

type MemoController struct{ Model *acquisitionModel.MemoModel }

func memoParams(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	companyID, err := strconv.ParseInt(r.URL.Query().Get("company_id"), 10, 64)
	if err != nil || companyID < 0 {
		response.Error(w, 400, "company_id wajib berupa angka nol atau positif")
		return 0, 0, false
	}
	var id int64
	if r.PathValue("id") != "" {
		id, err = strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			response.Error(w, 400, "ID memo tidak valid")
			return 0, 0, false
		}
	}
	return companyID, id, true
}

func memoFailure(w http.ResponseWriter, err error) {
	status, message := http.StatusInternalServerError, "Gagal memproses Acquisition Request"
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		status, message = http.StatusNotFound, "Memo tidak ditemukan pada company aktif"
	case errors.Is(err, acquisitionModel.ErrMemoHasLands):
		status, message = http.StatusConflict, "Memo masih memiliki Propose Land dan tidak dapat dihapus"
	case strings.Contains(err.Error(), "tidak dapat diubah"):
		status, message = http.StatusConflict, err.Error()
	case strings.Contains(err.Error(), "wajib") || strings.Contains(err.Error(), "harus") || strings.Contains(err.Error(), "tidak ditemukan"):
		status, message = http.StatusBadRequest, err.Error()
	}
	log.Printf("Acquisition memo: %v", err)
	response.Error(w, status, message)
}

func (c *MemoController) Index(w http.ResponseWriter, r *http.Request) {
	companyID, _, ok := memoParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	items, err := c.Model.GetAll(ctx, companyID)
	if err != nil {
		memoFailure(w, err)
		return
	}
	response.Success(w, "Data Acquisition Request berhasil diambil", items)
}

func (c *MemoController) Show(w http.ResponseWriter, r *http.Request) {
	companyID, id, ok := memoParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	item, err := c.Model.GetByID(ctx, companyID, id)
	if err != nil {
		memoFailure(w, err)
		return
	}
	response.Success(w, "Detail memo berhasil diambil", item)
}

func (c *MemoController) Lands(w http.ResponseWriter, r *http.Request) {
	companyID, id, ok := memoParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	items, err := c.Model.GetLands(ctx, companyID, id)
	if err != nil {
		memoFailure(w, err)
		return
	}
	response.Success(w, "Propose Land pada memo berhasil diambil", items)
}

func memoUploadDir() string {
	return filepath.Join(acquisitionUploadRoot(), "acquisition-memos")
}

func storeMemoFile(file io.Reader, filename string) (string, string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	allowed := map[string]bool{".pdf": true, ".png": true, ".jpg": true, ".jpeg": true, ".doc": true, ".docx": true}
	if !allowed[ext] {
		return "", "", fmt.Errorf("format dokumen tidak didukung")
	}
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", "", err
	}
	dir := memoUploadDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", "", err
	}
	name := hex.EncodeToString(token[:]) + ext
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
	return name, fullPath, nil
}

func parseMemoDate(value string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04"} {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("tanggal memo tidak valid")
}

func (c *MemoController) save(w http.ResponseWriter, r *http.Request, id *int64) {
	companyID, _, ok := memoParams(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<20)
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		response.Error(w, 400, "Form atau ukuran dokumen tidak valid (maksimal 15 MB)")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	var input acquisitionModel.MemoInput
	if err := json.Unmarshal([]byte(r.FormValue("data")), &input); err != nil {
		response.Error(w, 400, "Data form tidak valid")
		return
	}
	if input.CompanyID <= 0 || strings.TrimSpace(input.MemoNumber) == "" {
		response.Error(w, 400, "Company dan nomor memo wajib diisi")
		return
	}
	memoDate, err := parseMemoDate(input.MemoDate)
	if err != nil {
		response.Error(w, 400, err.Error())
		return
	}
	var docPath *string
	var storedPath string
	if input.ClearDoc {
		empty := ""
		docPath = &empty
	}
	file, header, err := r.FormFile("memo_doc")
	if err == nil {
		defer file.Close()
		if header.Size > 15<<20 {
			response.Error(w, 400, "Dokumen maksimal 15 MB")
			return
		}
		name, path, storeErr := storeMemoFile(file, header.Filename)
		if storeErr != nil {
			response.Error(w, 400, storeErr.Error())
			return
		}
		storedPath = path
		value := name
		docPath = &value
	} else if !errors.Is(err, http.ErrMissingFile) {
		response.Error(w, 400, "Dokumen memo tidak valid")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	item, err := c.Model.Save(ctx, companyID, id, input, memoDate, docPath)
	if err != nil {
		if storedPath != "" {
			_ = os.Remove(storedPath)
		}
		memoFailure(w, err)
		return
	}
	if id == nil {
		response.Created(w, "Acquisition Request berhasil ditambahkan", item)
	} else {
		response.Success(w, "Acquisition Request berhasil diperbarui", item)
	}
}

func (c *MemoController) Store(w http.ResponseWriter, r *http.Request) { c.save(w, r, nil) }

func (c *MemoController) Update(w http.ResponseWriter, r *http.Request) {
	_, id, ok := memoParams(w, r)
	if !ok {
		return
	}
	c.save(w, r, &id)
}

func (c *MemoController) Destroy(w http.ResponseWriter, r *http.Request) {
	companyID, id, ok := memoParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if err := c.Model.Delete(ctx, companyID, id); err != nil {
		memoFailure(w, err)
		return
	}
	response.Success(w, "Acquisition Request berhasil dihapus", nil)
}

func (c *MemoController) Document(w http.ResponseWriter, r *http.Request) {
	companyID, id, ok := memoParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	item, err := c.Model.GetByID(ctx, companyID, id)
	if err != nil {
		memoFailure(w, err)
		return
	}
	name := item.MemoDoc
	if !validStoredName(name) {
		response.Error(w, 404, "Dokumen memo belum tersedia")
		return
	}
	path := safeStoredFile(memoUploadDir(), name)
	if path == "" {
		if remote := memoLegacyDocumentURL(name, item.CompanyID); remote != "" {
			http.Redirect(w, r, remote, http.StatusFound)
			return
		}
		response.Error(w, 404, "File memo lama belum tersedia pada server ini")
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", name))
	http.ServeFile(w, r, path)
}
