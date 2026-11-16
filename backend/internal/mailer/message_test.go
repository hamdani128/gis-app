package mailer

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/mail"
	"strings"
	"testing"
)

func TestNotificationBCCIsEnvelopeOnly(t *testing.T) {
	c := Config{From: "GIS <sender@example.com>"}
	raw, recipients, err := c.encodeMessage(Message{To: []string{"user@example.com"}, CC: []string{"User@example.com", "cc@example.com"}, BCC: []string{"hidden@example.com", "cc@example.com"}, Subject: "[TESTING] GIS", HTML: "<p>MODE TESTING</p>", ID: "gis-1@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if len(recipients.To) != 1 || len(recipients.CC) != 1 || len(recipients.BCC) != 1 {
		t.Fatal("incorrect envelope deduplication")
	}
	message, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if message.Header.Get("Bcc") != "" || strings.Contains(string(raw), "hidden@example.com") {
		t.Fatal("BCC address exposed in message")
	}
	body, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, message.Body))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "<p>MODE TESTING</p>" {
		t.Fatal("MIME body changed")
	}
	if message.Header.Get("Cc") != "cc@example.com" {
		t.Fatal("missing visible CC")
	}
}

func TestNotificationHeaderValidation(t *testing.T) {
	c := Config{From: "sender@example.com"}
	for _, message := range []Message{
		{Subject: "GIS", CC: []string{"cc@example.com"}},
		{To: []string{"user@example.com\r\nBcc: other@example.com"}},
		{To: []string{"user@example.com"}, Subject: "subject\nBcc: other@example.com"},
		{To: []string{"user@example.com"}, ID: "bad id"},
	} {
		if c.ValidateMessage(message) == nil {
			t.Fatal("invalid notification accepted")
		}
	}
}
