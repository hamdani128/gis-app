package acquisition

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gis-app/backend/internal/controller/auth"
	"gis-app/backend/internal/model/acquisition"
	"gis-app/backend/internal/response"
)

const maxPaymentDocuments = 10
const maxPaymentDocumentSize = 15 << 20

func (c *PaymentController) Store(w http.ResponseWriter, r *http.Request) {
	storePaymentDocuments(w, r, func(ctx context.Context, username string, companyID, landID int64, transferDate time.Time, note, _ string, documents []acquisition.PaymentDocumentInput) error {
		return c.Model.SavePayment(ctx, username, companyID, landID, transferDate, note, documents)
	}, "Payment", "7")
}

func storePaymentDocuments(w http.ResponseWriter, r *http.Request, save func(context.Context, string, int64, int64, time.Time, string, string, []acquisition.PaymentDocumentInput) error, stage, nextStatus string) {
	companyID, landID, ok := submitParams(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 52<<20)
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		response.Error(w, 400, "Form atau ukuran dokumen tidak valid (total maksimal 50 MB)")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	choice := ""
	if stage == "Payment to Owner" {
		choice = strings.TrimSpace(r.FormValue("payment_choice"))
		if choice != "pay_to_owner" && choice != "cancel" {
			response.Error(w, 400, "Pilih Pay to Owner atau Cancel")
			return
		}
		if choice == "cancel" {
			nextStatus = "99"
		}
	}
	transferDate, err := time.Parse("2006-01-02", r.FormValue("transfer_date"))
	if err != nil {
		response.Error(w, 400, "Transfer Date wajib berupa tanggal yang valid")
		return
	}
	note := strings.TrimSpace(r.FormValue("note"))
	if len(note) > 10000 {
		response.Error(w, 400, "Note maksimal 10000 karakter")
		return
	}
	if choice == "cancel" && note == "" {
		response.Error(w, 400, "Alasan Cancel wajib diisi pada Note")
		return
	}
	files := r.MultipartForm.File["documents"]
	typeIDs := r.MultipartForm.Value["document_type_ids"]
	titles := r.MultipartForm.Value["document_titles"]
	docNotes := r.MultipartForm.Value["document_notes"]
	if (len(files) == 0 && choice != "cancel") || len(files) > maxPaymentDocuments || len(typeIDs) != len(files) || len(titles) != len(files) || len(docNotes) != len(files) {
		response.Error(w, 400, "Tambahkan 1–10 dokumen beserta jenis, judul, dan catatannya")
		return
	}
	uploadDir := paymentLandUploadDir(landID)
	uploadPrepared := len(files) > 0
	if uploadPrepared {
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			paymentFailure(w, err)
			return
		}
	}
	paths := []string{}
	committed := false
	defer func() {
		if !committed {
			for _, path := range paths {
				_ = os.Remove(path)
			}
			if uploadPrepared {
				_ = os.Remove(uploadDir)
			}
		}
	}()
	documents := make([]acquisition.PaymentDocumentInput, 0, len(files))
	totalSize := int64(0)
	for index, header := range files {
		typeID, err := strconv.ParseInt(typeIDs[index], 10, 64)
		title, docNote := strings.TrimSpace(titles[index]), strings.TrimSpace(docNotes[index])
		if err != nil || typeID <= 0 || len(title) > 200 || len(docNote) > 2000 {
			response.Error(w, 400, "Jenis, judul, atau catatan dokumen tidak valid")
			return
		}
		if title == "" {
			title = strings.TrimSuffix(filepath.Base(header.Filename), filepath.Ext(header.Filename))
		}
		if title == "" || len(title) > 200 {
			response.Error(w, 400, "Judul dokumen tidak valid atau melebihi 200 karakter")
			return
		}
		if header.Size <= 0 || header.Size > maxPaymentDocumentSize {
			response.Error(w, 400, "Setiap dokumen wajib berisi dan maksimal 15 MB")
			return
		}
		totalSize += header.Size
		if totalSize > 50<<20 {
			response.Error(w, 400, "Total dokumen maksimal 50 MB")
			return
		}
		ext := strings.ToLower(filepath.Ext(header.Filename))
		if ext != ".pdf" && ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
			response.Error(w, 400, "Dokumen harus PDF, PNG, atau JPEG")
			return
		}
		file, err := header.Open()
		if err != nil {
			paymentFailure(w, err)
			return
		}
		data, readErr := io.ReadAll(io.LimitReader(file, maxPaymentDocumentSize+1))
		closeErr := file.Close()
		if readErr != nil || closeErr != nil || len(data) == 0 || len(data) > maxPaymentDocumentSize {
			response.Error(w, 400, "Dokumen kosong atau terlalu besar")
			return
		}
		mime := http.DetectContentType(data)
		if (ext == ".pdf" && mime != "application/pdf") || (ext == ".png" && mime != "image/png") || ((ext == ".jpg" || ext == ".jpeg") && mime != "image/jpeg") {
			response.Error(w, 400, "Isi dokumen tidak sesuai format file")
			return
		}
		var suffix [12]byte
		if _, err := rand.Read(suffix[:]); err != nil {
			paymentFailure(w, err)
			return
		}
		fileName := fmt.Sprintf("payment-%d-%s%s", landID, hex.EncodeToString(suffix[:]), ext)
		path := filepath.Join(uploadDir, fileName)
		out, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			paymentFailure(w, err)
			return
		}
		paths = append(paths, path)
		_, writeErr := out.Write(data)
		closeErr = out.Close()
		if writeErr != nil || closeErr != nil {
			paymentFailure(w, fmt.Errorf("write payment document: %v, close: %v", writeErr, closeErr))
			return
		}
		documents = append(documents, acquisition.PaymentDocumentInput{TypeID: typeID, Title: title, Note: docNote, FileName: fileName})
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if err := save(ctx, auth.Username(r), companyID, landID, transferDate, note, choice, documents); err != nil {
		paymentFailure(w, err)
		return
	}
	committed = true
	response.Success(w, stage+" berhasil disimpan; lahan lanjut ke status "+nextStatus, map[string]int{"documents": len(documents)})
}
