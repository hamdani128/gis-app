package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	authModel "gis-app/backend/internal/model/auth"
	"gis-app/backend/internal/response"

	"github.com/jackc/pgx/v5"
)

type accountInput struct {
	Name            string `json:"name"`
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (input *accountInput) validate() string {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || utf8.RuneCountInString(input.Name) > 100 {
		return "Nama wajib diisi dan maksimal 100 karakter"
	}
	if input.CurrentPassword != "" || input.NewPassword != "" {
		if input.CurrentPassword == "" || len(input.CurrentPassword) > 1024 {
			return "Password saat ini wajib diisi"
		}
		if utf8.RuneCountInString(input.NewPassword) < 8 || utf8.RuneCountInString(input.NewPassword) > 100 || strings.TrimSpace(input.NewPassword) == "" {
			return "Password baru harus terdiri dari 8 sampai 100 karakter"
		}
		if input.CurrentPassword == input.NewPassword {
			return "Password baru harus berbeda dari password saat ini"
		}
	}
	return ""
}

// UpdateAccount always targets the signed-in user; callers cannot supply a user ID.
func (c *Controller) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	key, ok := secret()
	if !ok {
		response.Error(w, http.StatusInternalServerError, "Konfigurasi sesi belum tersedia")
		return
	}
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "Sesi login tidak ditemukan")
		return
	}
	userID, valid := readSession(cookie.Value, key, time.Now())
	if !valid {
		clearSessionCookie(w)
		response.Error(w, http.StatusUnauthorized, "Sesi login tidak valid atau sudah berakhir")
		return
	}
	// Requiring JSON prevents cross-origin form submissions from changing an account.
	if strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0]) != "application/json" {
		response.Error(w, http.StatusUnsupportedMediaType, "Content-Type harus application/json")
		return
	}
	var input accountInput
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		response.Error(w, http.StatusBadRequest, "Data pengaturan akun tidak valid")
		return
	}
	if message := input.validate(); message != "" {
		response.Error(w, http.StatusBadRequest, message)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	user, err := c.Model.UpdateAccount(ctx, userID, input.Name, input.CurrentPassword, input.NewPassword)
	if errors.Is(err, pgx.ErrNoRows) {
		clearSessionCookie(w)
		response.Error(w, http.StatusUnauthorized, "Akun tidak aktif")
		return
	}
	if errors.Is(err, authModel.ErrCurrentPassword) {
		response.Error(w, http.StatusBadRequest, "Password saat ini tidak sesuai")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Pengaturan akun gagal disimpan")
		return
	}
	response.Success(w, "Pengaturan akun berhasil diperbarui", user)
}
