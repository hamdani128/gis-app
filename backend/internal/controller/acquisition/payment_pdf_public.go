package acquisition

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func paymentPublicPDFPath(landID int64) string {
	return filepath.Join(paymentSignatureDir(), fmt.Sprintf("payment-approval-%d-qr.pdf", landID))
}

func paymentPublicTokenPath(landID int64) string {
	return filepath.Join(paymentSignatureDir(), fmt.Sprintf("payment-approval-%d.token", landID))
}

func validPaymentToken(token string) bool {
	if len(token) != 64 {
		return false
	}
	_, err := hex.DecodeString(token)
	return err == nil
}

func paymentPublicToken(landID int64) (string, error) {
	path := paymentPublicTokenPath(landID)
	read := func() (string, error) {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		token := strings.TrimSpace(string(data))
		if !validPaymentToken(token) {
			return "", errors.New("invalid stored payment PDF token")
		}
		return token, nil
	}
	if token, err := read(); err == nil || !errors.Is(err, os.ErrNotExist) {
		return token, err
	}
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return "", err
	}
	token := hex.EncodeToString(secret[:])
	file, err := os.CreateTemp(paymentSignatureDir(), ".payment-token-*")
	if err != nil {
		return "", err
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)
	if _, err = file.WriteString(token); err != nil {
		file.Close()
		return "", err
	}
	if err = file.Close(); err != nil {
		return "", err
	}
	if err = os.Link(tempPath, path); errors.Is(err, os.ErrExist) {
		return read()
	} else if err != nil {
		return "", err
	}
	return token, nil
}

func paymentPublicURL(landID int64, token string) string {
	origin := strings.TrimRight(os.Getenv("PUBLIC_API_ORIGIN"), "/")
	if origin == "" {
		origin = "http://localhost:9000"
	}
	return fmt.Sprintf("%s/api/public/payment-approvals/%d/%s", origin, landID, token)
}

func (c *PaymentController) PublicSignaturePDF(w http.ResponseWriter, r *http.Request) {
	landID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	token := r.PathValue("token")
	if err != nil || landID <= 0 || !validPaymentToken(token) {
		http.NotFound(w, r)
		return
	}
	stored, err := os.ReadFile(paymentPublicTokenPath(landID))
	if err != nil || subtle.ConstantTimeCompare([]byte(strings.TrimSpace(string(stored))), []byte(token)) != 1 {
		http.NotFound(w, r)
		return
	}
	path := paymentPublicPDFPath(landID)
	if _, err := os.Stat(path); err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=payment-approval-%d.pdf", landID))
	http.ServeFile(w, r, path)
}
