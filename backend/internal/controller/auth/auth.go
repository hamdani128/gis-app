package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	authModel "gis-app/backend/internal/model/auth"
	"gis-app/backend/internal/response"

	"github.com/jackc/pgx/v5"
)

const (
	sessionCookie = "gis_session"
	sessionTTL    = time.Hour
)

type Controller struct {
	Model *authModel.UserModel
}

type credentials struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

func secret() ([]byte, bool) {
	value := os.Getenv("AUTH_SESSION_SECRET")
	if len(value) < 32 {
		return nil, false
	}
	return []byte(value), true
}

func makeSession(userID int64, key []byte, expires time.Time) string {
	payload := strconv.FormatInt(userID, 10) + "." + strconv.FormatInt(expires.Unix(), 10)
	encoded := base64.RawURLEncoding.EncodeToString([]byte(payload))
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(encoded))
	return encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func readSession(token string, key []byte, now time.Time) (int64, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return 0, false
	}
	providedMAC, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, false
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(parts[0]))
	if !hmac.Equal(providedMAC, mac.Sum(nil)) {
		return 0, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return 0, false
	}
	fields := strings.Split(string(payload), ".")
	if len(fields) != 2 {
		return 0, false
	}
	userID, idErr := strconv.ParseInt(fields[0], 10, 64)
	expires, expiryErr := strconv.ParseInt(fields[1], 10, 64)
	if idErr != nil || expiryErr != nil || userID <= 0 || now.Unix() >= expires {
		return 0, false
	}
	return userID, true
}

func setSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	secure := strings.EqualFold(os.Getenv("AUTH_COOKIE_SECURE"), "true")
	// Expire sessions issued before the cookie was made available to /api.
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Path: "/api/auth", MaxAge: -1, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/api",
		Expires:  expires,
		MaxAge:   int(sessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	secure := strings.EqualFold(os.Getenv("AUTH_COOKIE_SECURE"), "true")
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Path: "/api/auth", MaxAge: -1, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/api",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (c *Controller) Login(w http.ResponseWriter, r *http.Request) {
	key, ok := secret()
	if !ok {
		response.Error(w, http.StatusInternalServerError, "AUTH_SESSION_SECRET belum diatur atau kurang dari 32 karakter")
		return
	}
	var input credentials
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		response.Error(w, http.StatusBadRequest, "Body login tidak valid")
		return
	}
	input.Identifier = strings.TrimSpace(input.Identifier)
	if input.Identifier == "" || input.Password == "" || len(input.Identifier) > 254 || len(input.Password) > 1024 {
		response.Error(w, http.StatusBadRequest, "Username/email dan kata sandi wajib diisi")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	user, err := c.Model.Authenticate(ctx, input.Identifier, input.Password)
	if errors.Is(err, pgx.ErrNoRows) {
		response.Error(w, http.StatusUnauthorized, "Username/email atau kata sandi tidak valid")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Tidak dapat memeriksa akun pengguna")
		return
	}

	expires := time.Now().Add(sessionTTL)
	setSessionCookie(w, makeSession(user.ID, key, expires), expires)
	response.Success(w, "Login berhasil", user)
}

func (c *Controller) Me(w http.ResponseWriter, r *http.Request) {
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
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	user, err := c.Model.GetActiveByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		clearSessionCookie(w)
		response.Error(w, http.StatusUnauthorized, "Akun tidak aktif")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Tidak dapat memeriksa sesi pengguna")
		return
	}
	response.Success(w, "Sesi login aktif", user)
}

func (c *Controller) MenuAccess(w http.ResponseWriter, r *http.Request) {
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
		response.Error(w, http.StatusUnauthorized, "Sesi login tidak valid atau sudah berakhir")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	user, err := c.Model.GetActiveByID(ctx, userID)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "Akun tidak aktif")
		return
	}
	codes, err := c.Model.GetVisibleModuleCodes(ctx, user.Username)
	if err != nil {
		log.Printf("Menu access: %v", err)
		response.Error(w, http.StatusInternalServerError, "Hak akses menu tidak dapat diperiksa")
		return
	}
	response.Success(w, "Hak akses menu berhasil diambil", codes)
}

func (c *Controller) Refresh(w http.ResponseWriter, r *http.Request) {
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
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if _, err := c.Model.GetActiveByID(ctx, userID); errors.Is(err, pgx.ErrNoRows) {
		clearSessionCookie(w)
		response.Error(w, http.StatusUnauthorized, "Akun tidak aktif")
		return
	} else if err != nil {
		response.Error(w, http.StatusInternalServerError, "Tidak dapat memperbarui sesi login")
		return
	}
	expires := time.Now().Add(sessionTTL)
	setSessionCookie(w, makeSession(userID, key, expires), expires)
	response.Success(w, "Sesi login diperbarui", map[string]int64{"expires_at": expires.Unix()})
}

func (c *Controller) Companies(w http.ResponseWriter, r *http.Request) {
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
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	user, err := c.Model.GetActiveByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		clearSessionCookie(w)
		response.Error(w, http.StatusUnauthorized, "Akun tidak aktif")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Tidak dapat memeriksa sesi pengguna")
		return
	}
	companies, err := c.Model.GetAccessibleCompanies(ctx, user.Username)
	if err != nil {
		log.Printf("Auth company access: %v", err)
		response.Error(w, http.StatusInternalServerError, "Tidak dapat mengambil daftar company untuk akun ini")
		return
	}
	response.Success(w, "Daftar company yang dapat diakses berhasil diambil", companies)
}

func (c *Controller) Logout(w http.ResponseWriter, _ *http.Request) {
	clearSessionCookie(w)
	response.Success(w, "Logout berhasil", nil)
}
