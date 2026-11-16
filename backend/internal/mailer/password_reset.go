package mailer

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/mail"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Host, Port, Username, Password, TLSMode, From, LoginURL string
}

func LoadConfig() (Config, error) {
	c := Config{
		Host: strings.TrimSpace(os.Getenv("SMTP_HOST")), Port: strings.TrimSpace(os.Getenv("SMTP_PORT")),
		Username: os.Getenv("SMTP_USERNAME"), Password: os.Getenv("SMTP_PASSWORD"),
		TLSMode: strings.ToLower(strings.TrimSpace(os.Getenv("SMTP_TLS_MODE"))),
		From:    strings.TrimSpace(os.Getenv("SMTP_FROM")), LoginURL: strings.TrimRight(os.Getenv("FRONTEND_ORIGIN"), "/"),
	}
	if c.TLSMode == "" {
		c.TLSMode = "starttls"
	}
	if c.Port == "" {
		c.Port = "587"
		if c.TLSMode == "tls" {
			c.Port = "465"
		}
	}
	if c.From == "" {
		c.From = "AEP GIS System <auto.alert@aepplantations.id>"
	}
	if c.LoginURL == "" {
		c.LoginURL = "http://localhost:3000"
	}
	c.LoginURL += "/auth/login"
	port, err := strconv.Atoi(c.Port)
	if err != nil || port < 1 || port > 65535 || c.Host == "" || strings.ContainsAny(c.Host, "\r\n") {
		return Config{}, errors.New("SMTP host/port is not configured")
	}
	if c.TLSMode != "starttls" && c.TLSMode != "tls" && c.TLSMode != "none" {
		return Config{}, errors.New("SMTP_TLS_MODE must be starttls, tls, or none")
	}
	if (c.Username == "") != (c.Password == "") || (c.Username != "" && c.TLSMode == "none") {
		return Config{}, errors.New("SMTP credentials require both username/password and TLS")
	}
	if _, err := mail.ParseAddress(c.From); err != nil || strings.ContainsAny(c.From, "\r\n") {
		return Config{}, errors.New("invalid SMTP_FROM")
	}
	u, err := url.Parse(c.LoginURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
		return Config{}, errors.New("invalid FRONTEND_ORIGIN")
	}
	return c, nil
}

var resetTemplate = template.Must(template.New("reset-email").Parse(`<!doctype html>
<html lang="id"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;background:#f1f5f2;font-family:Arial,Helvetica,sans-serif;color:#324b3b">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr><td align="center" style="padding:32px 16px">
<table role="presentation" width="560" cellpadding="0" cellspacing="0" style="width:100%;max-width:560px;border:1px solid #dde8df;border-radius:14px;overflow:hidden;background:#fff">
<tr><td style="padding:26px 30px;background:#20563c;color:#fff"><p style="margin:0 0 8px;font-size:10px;letter-spacing:2px;color:#b8d9c2">AEP PLANTATIONS</p><h1 style="margin:0;font-size:22px">AEP GIS System</h1><p style="margin:8px 0 0;font-size:12px;color:#c9dfd0">ICT / Helpdesk Notification</p></td></tr>
<tr><td style="padding:30px;font-size:14px;line-height:1.8"><p>Dear <strong>{{.Username}}</strong>,</p>
<h2 style="font-size:20px;color:#294d35">Password akun Anda telah direset.</h2><p>Berikut informasi login baru untuk akun AEP GIS Anda.</p>
<table role="presentation" width="100%" cellpadding="12" cellspacing="0" style="margin:22px 0;border:1px solid #e0eae2;background:#f4f8f4"><tr><td style="font-size:12px;color:#7a9180">Username</td><td><strong>{{.Username}}</strong></td></tr><tr><td style="font-size:12px;color:#7a9180">Password baru</td><td style="font-family:monospace;font-size:17px;letter-spacing:1px"><strong>{{.Password}}</strong></td></tr></table>
<p>Silakan login dan segera ubah password melalui <strong>Pengaturan akun</strong>.<br><span style="font-size:12px;color:#7a9180">For security reasons, please sign in and change your password immediately.</span></p>
<p style="margin:26px 0"><a href="{{.LoginURL}}" style="display:inline-block;padding:12px 22px;border-radius:7px;background:#28774e;color:#fff;text-decoration:none;font-weight:bold">Masuk ke AEP GIS</a></p>
<p style="font-size:12px;color:#7a9180">Jika Anda tidak meminta reset ini, segera hubungi Departemen ICT.<br>If you did not request this reset, please contact the ICT Department.</p><p>Best regards,<br><strong>Helpdesk ICT</strong><br>AEP GIS System</p></td></tr>
<tr><td style="padding:17px 30px;border-top:1px solid #e8eee9;background:#fafcf9;font-size:11px;color:#8ba08f">Email ini dikirim otomatis. Mohon tidak membalas email ini.<br>This email is automatically generated. Please do not reply.</td></tr>
</table></td></tr></table></body></html>`))

func (c Config) message(to, username, password string) ([]byte, error) {
	address, err := mail.ParseAddress(to)
	if err != nil || address.Address != to || strings.ContainsAny(to, "\r\n") {
		return nil, errors.New("invalid recipient")
	}
	var html bytes.Buffer
	if err := resetTemplate.Execute(&html, struct{ Username, Password, LoginURL string }{username, password, c.LoginURL}); err != nil {
		return nil, err
	}
	var message bytes.Buffer
	message.WriteString("From: " + c.From + "\r\nTo: " + address.String() + "\r\nSubject: AEP GIS - Password Reset Notification\r\n")
	message.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n")
	encoded := base64.StdEncoding.EncodeToString(html.Bytes())
	for len(encoded) > 76 {
		message.WriteString(encoded[:76] + "\r\n")
		encoded = encoded[76:]
	}
	message.WriteString(encoded + "\r\n")
	return message.Bytes(), nil
}

// SendReset returns success after the SMTP server accepts DATA. A QUIT failure
// after that point must not cause a delivered password to be rolled back.
func (c Config) SendReset(ctx context.Context, to, username, password string) error {
	message, err := c.message(to, username, password)
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
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP recipient: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA: %w", err)
	}
	if _, err := writer.Write(message); err != nil {
		return fmt.Errorf("SMTP body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("SMTP acceptance: %w", err)
	}
	_ = client.Quit()
	return nil
}

// CheckConnection verifies TLS and authentication without sending any email.
func (c Config) CheckConnection(ctx context.Context) error {
	client, err := c.connect(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	return client.Quit()
}

func (c Config) connect(ctx context.Context) (*smtp.Client, error) {
	deadline := time.Now().Add(8 * time.Second)
	if expiry, ok := ctx.Deadline(); ok && expiry.Before(deadline) {
		deadline = expiry
	}
	dialer := &net.Dialer{Deadline: deadline}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(c.Host, c.Port))
	if err != nil {
		return nil, fmt.Errorf("SMTP connection: %w", err)
	}
	ok := false
	defer func() {
		if !ok {
			conn.Close()
		}
	}()
	if err := conn.SetDeadline(deadline); err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12}
	if c.TLSMode == "tls" {
		tlsConn := tls.Client(conn, tlsConfig)
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			return nil, fmt.Errorf("SMTP TLS: %w", err)
		}
		conn = tlsConn
	}
	client, err := smtp.NewClient(conn, c.Host)
	if err != nil {
		return nil, fmt.Errorf("SMTP greeting: %w", err)
	}
	if c.TLSMode == "starttls" {
		if err := client.StartTLS(tlsConfig); err != nil {
			return nil, fmt.Errorf("SMTP STARTTLS: %w", err)
		}
	}
	if c.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", c.Username, c.Password, c.Host)); err != nil {
			return nil, fmt.Errorf("SMTP authentication: %w", err)
		}
	}
	ok = true
	return client, nil
}
