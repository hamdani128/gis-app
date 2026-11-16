package mailer

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net/mail"
	"strings"
	"time"
)

type Message struct {
	To, CC, BCC       []string
	Subject, HTML, ID string
}

// NormalizeRecipients removes duplicates across To, CC, then BCC. BCC addresses
// are envelope-only and are never exposed in message headers.
func NormalizeRecipients(to, cc, bcc []string) (Message, error) {
	result := Message{}
	seen := make(map[string]bool)
	for _, group := range []struct {
		source []string
		target *[]string
	}{{to, &result.To}, {cc, &result.CC}, {bcc, &result.BCC}} {
		for _, raw := range group.source {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				continue
			}
			address, err := mail.ParseAddress(raw)
			if err != nil || strings.ContainsAny(raw, "\r\n") {
				return Message{}, errors.New("invalid notification email address")
			}
			key := strings.ToLower(address.Address)
			if seen[key] {
				continue
			}
			seen[key] = true
			*group.target = append(*group.target, address.Address)
		}
	}
	return result, nil
}

func (c Config) encodeMessage(input Message) ([]byte, Message, error) {
	message, err := NormalizeRecipients(input.To, input.CC, input.BCC)
	if err != nil {
		return nil, message, err
	}
	if len(message.To) == 0 {
		return nil, message, errors.New("notification has no To recipients")
	}
	if strings.ContainsAny(input.Subject+input.ID+c.From, "\r\n") {
		return nil, message, errors.New("invalid message header")
	}
	if _, err := mail.ParseAddress(c.From); err != nil {
		return nil, message, errors.New("invalid sender")
	}
	var output bytes.Buffer
	output.WriteString("From: " + c.From + "\r\nTo: " + strings.Join(message.To, ", ") + "\r\n")
	if len(message.CC) > 0 {
		output.WriteString("Cc: " + strings.Join(message.CC, ", ") + "\r\n")
	}
	output.WriteString("Subject: " + mime.QEncoding.Encode("UTF-8", input.Subject) + "\r\n")
	if input.ID != "" {
		if strings.ContainsAny(input.ID, "<> \t") {
			return nil, message, errors.New("invalid message ID")
		}
		output.WriteString("Message-ID: <" + input.ID + ">\r\n")
	}
	output.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n")
	encoded := base64.StdEncoding.EncodeToString([]byte(input.HTML))
	for len(encoded) > 76 {
		output.WriteString(encoded[:76] + "\r\n")
		encoded = encoded[76:]
	}
	output.WriteString(encoded + "\r\n")
	return output.Bytes(), message, nil
}

func (c Config) SendHTML(ctx context.Context, message Message) error {
	body, recipients, err := c.encodeMessage(message)
	if err != nil {
		return err
	}
	from, err := mail.ParseAddress(c.From)
	if err != nil {
		return err
	}
	client, err := c.connect(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	if err := client.Mail(from.Address); err != nil {
		return fmt.Errorf("SMTP sender: %w", err)
	}
	for _, group := range [][]string{recipients.To, recipients.CC, recipients.BCC} {
		for _, address := range group {
			if err := client.Rcpt(address); err != nil {
				return fmt.Errorf("SMTP recipient: %w", err)
			}
		}
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA: %w", err)
	}
	if _, err := writer.Write(body); err != nil {
		return fmt.Errorf("SMTP body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("SMTP acceptance: %w", err)
	}
	_ = client.Quit()
	return nil
}

func (c Config) ValidateMessage(message Message) error {
	_, _, err := c.encodeMessage(message)
	return err
}
