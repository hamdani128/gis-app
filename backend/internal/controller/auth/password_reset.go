package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"sync"
	"time"

	"gis-app/backend/internal/mailer"
	"gis-app/backend/internal/response"
	"github.com/jackc/pgx/v5"
)

type resetRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

func (input *resetRequest) valid() bool {
	input.Username = strings.TrimSpace(input.Username)
	input.Email = strings.TrimSpace(input.Email)
	address, err := mail.ParseAddress(input.Email)
	return input.Username != "" && len(input.Username) <= 254 && len(input.Email) <= 254 &&
		!strings.ContainsAny(input.Username+input.Email, "\r\n") && err == nil && address.Address == input.Email
}

func generateResetPassword() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
	password := make([]byte, 16)
	for i := range password {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		password[i] = alphabet[index.Int64()]
	}
	return string(password), nil
}

type resetLimitEntry struct {
	count   int
	expires time.Time
}
type resetLimiter struct {
	sync.Mutex
	entries map[string]resetLimitEntry
}

var passwordResetLimiter = resetLimiter{entries: make(map[string]resetLimitEntry)}

func (l *resetLimiter) allow(ip, username string, now time.Time) time.Duration {
	l.Lock()
	defer l.Unlock()
	for key, entry := range l.entries {
		if !now.Before(entry.expires) {
			delete(l.entries, key)
		}
	}
	hash := sha256.Sum256([]byte(strings.ToLower(username)))
	accountKey := "account:" + string(hash[:])
	ipKey := "ip:" + ip
	if entry := l.entries[ipKey]; entry.count >= 5 {
		return entry.expires.Sub(now)
	}
	if entry := l.entries[accountKey]; entry.count >= 1 {
		return entry.expires.Sub(now)
	}
	// Bound memory even when requests use many different account names/IPs.
	if len(l.entries) >= 10000 {
		return time.Minute
	}
	for key, window := range map[string]time.Duration{ipKey: 15 * time.Minute, accountKey: time.Minute} {
		entry := l.entries[key]
		if entry.count == 0 {
			entry.expires = now.Add(window)
		}
		entry.count++
		l.entries[key] = entry
	}
	return 0
}

func (c *Controller) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0]) != "application/json" {
		response.Error(w, http.StatusUnsupportedMediaType, "Content-Type harus application/json")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var input resetRequest
	if err := decoder.Decode(&input); err != nil || decoder.Decode(&struct{}{}) != io.EOF || !input.valid() {
		response.Error(w, http.StatusBadRequest, "Username dan email terdaftar wajib diisi dengan benar")
		return
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	if wait := passwordResetLimiter.allow(ip, input.Username, time.Now()); wait > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
		response.Error(w, http.StatusTooManyRequests, "Terlalu banyak permintaan reset. Silakan coba lagi nanti.")
		return
	}
	config, err := mailer.LoadConfig()
	if err != nil {
		log.Printf("Password reset mail configuration: %v", err)
		response.Error(w, http.StatusServiceUnavailable, "Layanan email reset belum tersedia. Silakan hubungi Departemen ICT.")
		return
	}
	password, err := generateResetPassword()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Permintaan reset belum dapat diproses")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	err = c.Model.ResetPassword(ctx, input.Username, input.Email, password, func(to, username string) error {
		return config.SendReset(ctx, to, username, password)
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		detail := strings.ReplaceAll(err.Error(), password, "[redacted]")
		if config.Password != "" {
			detail = strings.ReplaceAll(detail, config.Password, "[redacted]")
		}
		log.Printf("Password reset failed: %s", detail)
		response.Error(w, http.StatusServiceUnavailable, "Permintaan reset belum dapat diproses. Silakan coba kembali atau hubungi Departemen ICT.")
		return
	}
	// Do not reveal whether a username/email pair exists.
	response.Success(w, "Jika username dan email cocok dengan akun aktif, password baru akan dikirim ke email terdaftar.", nil)
}
