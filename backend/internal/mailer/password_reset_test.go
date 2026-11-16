package mailer

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/mail"
	"strings"
	"testing"
)

func TestResetEmailEscapesHTMLAndRejectsHeaderInjection(t *testing.T) {
	c := Config{From: "AEP GIS <auto.alert@example.com>", LoginURL: "https://gis.example.com/auth/login"}
	raw, err := c.message("user@example.com", `<script>alert('x')</script>`, "safe-password")
	if err != nil {
		t.Fatal(err)
	}
	message, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, message.Body))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "<script>") || !strings.Contains(string(body), "&lt;script&gt;") {
		t.Fatal("username was not escaped")
	}
	if !strings.Contains(string(body), c.LoginURL) || !strings.Contains(string(body), "safe-password") {
		t.Fatal("missing reset details")
	}
	if message.Header.Get("Subject") != "AEP GIS - Password Reset Notification" {
		t.Fatal("missing notification subject")
	}
	if _, err := c.message("user@example.com\r\nBcc: other@example.com", "user", "password"); err == nil {
		t.Fatal("header injection accepted")
	}
}

func TestSMTPConfig(t *testing.T) {
	for key, value := range map[string]string{
		"SMTP_HOST": "smtp.example.com", "SMTP_PORT": "587", "SMTP_TLS_MODE": "starttls",
		"SMTP_USERNAME": "", "SMTP_PASSWORD": "", "SMTP_FROM": "AEP GIS <auto.alert@example.com>",
		"FRONTEND_ORIGIN": "https://gis.example.com",
	} {
		t.Setenv(key, value)
	}
	c, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if c.LoginURL != "https://gis.example.com/auth/login" {
		t.Fatal("unexpected login URL")
	}
	t.Setenv("SMTP_TLS_MODE", "none")
	t.Setenv("SMTP_USERNAME", "user")
	t.Setenv("SMTP_PASSWORD", "secret")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("authenticated SMTP without TLS accepted")
	}
	t.Setenv("SMTP_TLS_MODE", "tls")
	t.Setenv("SMTP_PORT", "bad")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("invalid SMTP port accepted")
	}
	t.Setenv("SMTP_PORT", "465")
	t.Setenv("SMTP_FROM", "sender@example.com\r\nBcc: other@example.com")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("invalid sender accepted")
	}
}
