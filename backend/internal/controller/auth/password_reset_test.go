package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestResetRequestValidation(t *testing.T) {
	for _, test := range []struct {
		username, email string
		valid           bool
	}{
		{" admin ", " admin@example.com ", true},
		{"", "admin@example.com", false},
		{"admin", "bad-email", false},
		{"admin", "Admin <admin@example.com>", false},
		{"admin", "admin@example.com\r\nBcc: other@example.com", false},
		{strings.Repeat("a", 255), "admin@example.com", false},
	} {
		input := resetRequest{Username: test.username, Email: test.email}
		if input.valid() != test.valid {
			t.Errorf("unexpected validation for username=%q", test.username)
		}
	}
}

func TestResetPasswordGenerator(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		password, err := generateResetPassword()
		if err != nil {
			t.Fatal(err)
		}
		if len(password) != 16 || seen[password] {
			t.Fatal("unexpected length or repeated generated password")
		}
		if strings.ContainsAny(password, "\r\n<>\"'& ") {
			t.Fatal("unexpected character in password")
		}
		seen[password] = true
	}
}

func TestResetLimiter(t *testing.T) {
	now := time.Now()
	l := resetLimiter{entries: make(map[string]resetLimitEntry)}
	if l.allow("client-a", "Admin", now) != 0 {
		t.Fatal("first request rejected")
	}
	if l.allow("client-b", "admin", now) <= 0 {
		t.Fatal("account cooldown bypassed by case/IP change")
	}
	if l.allow("client-b", "admin", now.Add(time.Minute)) != 0 {
		t.Fatal("expired account cooldown retained")
	}
	for i := 0; i < 4; i++ {
		if l.allow("client-a", string(rune('a'+i)), now) != 0 {
			t.Fatal("request below IP threshold rejected")
		}
	}
	if l.allow("client-a", "another", now) <= 0 {
		t.Fatal("IP limit bypassed")
	}
	if l.allow("client-a", "another", now.Add(16*time.Minute)) != 0 {
		t.Fatal("expired IP limit retained")
	}
}

func TestForgotPasswordRejectsMalformedRequests(t *testing.T) {
	for _, body := range []string{
		`{`, `{}`, `{"username":"admin","email":"bad"}`,
		`{"username":"admin","email":"admin@example.com","id":2}`,
		`{"username":"admin","email":"admin@example.com"} {}`,
		`{"username":"` + strings.Repeat("a", 5000) + `","email":"admin@example.com"}`,
	} {
		r := httptest.NewRequest(http.MethodPost, "/api/auth/forgot-password", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		(&Controller{}).ForgotPassword(w, r)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("got %d, expected 400", w.Code)
		}
	}
	r := httptest.NewRequest(http.MethodPost, "/api/auth/forgot-password", strings.NewReader("username=admin"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	(&Controller{}).ForgotPassword(w, r)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("form submission: got %d", w.Code)
	}
}

func TestForgotPasswordRequiresMailConfiguration(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	r := httptest.NewRequest(http.MethodPost, "/api/auth/forgot-password", strings.NewReader(`{"username":"config-test","email":"config-test@example.com"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	(&Controller{}).ForgotPassword(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d, expected 503", w.Code)
	}
}
