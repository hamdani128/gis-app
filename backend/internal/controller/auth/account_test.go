package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAccountInputValidation(t *testing.T) {
	tests := []struct {
		name  string
		input accountInput
		valid bool
	}{
		{"name only", accountInput{Name: "  New Name  "}, true},
		{"empty name", accountInput{Name: "   "}, false},
		{"long name", accountInput{Name: strings.Repeat("a", 101)}, false},
		{"password change", accountInput{Name: "Name", CurrentPassword: "old", NewPassword: "new password"}, true},
		{"missing current", accountInput{Name: "Name", NewPassword: "new password"}, false},
		{"missing new", accountInput{Name: "Name", CurrentPassword: "old"}, false},
		{"short new", accountInput{Name: "Name", CurrentPassword: "old", NewPassword: "short"}, false},
		{"blank new", accountInput{Name: "Name", CurrentPassword: "old", NewPassword: "        "}, false},
		{"unchanged password", accountInput{Name: "Name", CurrentPassword: "same-password", NewPassword: "same-password"}, false},
		{"long new", accountInput{Name: "Name", CurrentPassword: "old", NewPassword: strings.Repeat("a", 101)}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := test.input
			if message := input.validate(); (message == "") != test.valid {
				t.Fatalf("valid=%v, message=%q", test.valid, message)
			}
			if test.valid && input.Name != strings.TrimSpace(test.input.Name) {
				t.Fatal("name was not trimmed")
			}
			if input.CurrentPassword != test.input.CurrentPassword || input.NewPassword != test.input.NewPassword {
				t.Fatal("password content was altered")
			}
		})
	}
}

func TestUpdateAccountRejectsInvalidRequests(t *testing.T) {
	key := strings.Repeat("k", 32)
	t.Setenv("AUTH_SESSION_SECRET", key)
	tests := []struct {
		name, body, token, contentType string
		status                         int
	}{
		{"no session", `{}`, "", "application/json", http.StatusUnauthorized},
		{"tampered session", `{}`, "invalid", "application/json", http.StatusUnauthorized},
		{"expired session", `{}`, makeSession(1, []byte(key), time.Now().Add(-time.Minute)), "application/json", http.StatusUnauthorized},
		{"form submission", `name=New`, "valid", "application/x-www-form-urlencoded", http.StatusUnsupportedMediaType},
		{"malformed JSON", `{`, "valid", "application/json", http.StatusBadRequest},
		{"other account ID", `{"name":"New","id":2}`, "valid", "application/json", http.StatusBadRequest},
		{"role injection", `{"name":"New","username":"admin"}`, "valid", "application/json", http.StatusBadRequest},
		{"trailing JSON", `{"name":"New"} {}`, "valid", "application/json", http.StatusBadRequest},
		{"missing name", `{}`, "valid", "application/json", http.StatusBadRequest},
		{"missing current password", `{"name":"New","new_password":"new-password"}`, "valid", "application/json", http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPut, "/api/auth/me", strings.NewReader(test.body))
			r.Header.Set("Content-Type", test.contentType)
			if test.token != "" {
				token := test.token
				if token == "valid" {
					token = makeSession(1, []byte(key), time.Now().Add(time.Minute))
				}
				r.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
			}
			w := httptest.NewRecorder()
			// A nil model ensures all rejected requests stop before database access.
			(&Controller{}).UpdateAccount(w, r)
			if w.Code != test.status {
				t.Fatalf("got %d, want %d: %s", w.Code, test.status, w.Body.String())
			}
		})
	}
}
