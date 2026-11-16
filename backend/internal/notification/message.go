package notification

import (
	"bytes"
	"fmt"
	"gis-app/backend/internal/mailer"
	"html/template"
	"strconv"
	"strings"
)

type Event struct {
	ID, LandID, CompanyID, EstateID                 int64
	Approval, Status                                int
	Date, Note, Username, OfficerName, OfficerEmail string
	Company, Estate, Owner, StatusName              string
	Area, Total                                     float64
}

type Recipients struct{ To, CC []string }

func approvalLabel(approval int) string {
	switch approval {
	case 1:
		return "Approved"
	case 2:
		return "On Hold"
	case 0:
		return "Rejected"
	default:
		return "Unknown"
	}
}

func formatNumber(number float64, places int) string {
	parts := strings.Split(strconv.FormatFloat(number, 'f', places, 64), ".")
	whole := parts[0]
	start := 0
	if strings.HasPrefix(whole, "-") {
		start = 1
	}
	for i := len(whole) - 3; i > start; i -= 3 {
		whole = whole[:i] + "." + whole[i:]
	}
	if places > 0 {
		return whole + "," + parts[1]
	}
	return whole
}

func BuildMessage(event Event, recipients Recipients, config Config) (mailer.Message, error) {
	to := append([]string(nil), recipients.To...)
	cc := append([]string(nil), recipients.CC...)
	bcc := config.BCC
	if config.TestMode {
		// Replace every envelope recipient before validation or SMTP delivery.
		to = []string{config.TestTo}
		cc = nil
		bcc = nil
	} else if event.Status == 99 {
		to = append(to, event.OfficerEmail)
		cc = append(cc, config.RejectedCC...)
	} else {
		cc = append(cc, event.OfficerEmail)
	}
	message, err := mailer.NormalizeRecipients(to, cc, bcc)
	if err != nil {
		return mailer.Message{}, err
	}
	if len(message.To) == 0 {
		if config.TestMode {
			return mailer.Message{}, fmt.Errorf("testing recipient is required")
		}
		return mailer.Message{}, fmt.Errorf("no next approver email for process %d", event.ID)
	}
	message.Subject = "[GIS] Acquisition Process (" + event.Company + ")"
	if event.Status == 99 {
		message.Subject = "[GIS] Acquisition Process [REJECTED] (" + event.Company + ")"
	}
	if config.TestMode {
		message.Subject = "[TESTING] " + message.Subject
	}
	message.ID = fmt.Sprintf("gis-acquisition-%d@aepplantations.id", event.ID)
	officer := event.Username
	if event.OfficerName != "" {
		officer += " (" + event.OfficerName + ")"
	}
	var html bytes.Buffer
	data := struct {
		Event                                    Event
		TestMode                                 bool
		Approval, Officer, Area, Total, LoginURL string
	}{event, config.TestMode, approvalLabel(event.Approval), officer, formatNumber(event.Area, 2), formatNumber(event.Total, 0), config.Mail.LoginURL}
	if err := notificationTemplate.Execute(&html, data); err != nil {
		return mailer.Message{}, err
	}
	message.HTML = html.String()
	if err := config.Mail.ValidateMessage(message); err != nil {
		return mailer.Message{}, err
	}
	return message, nil
}

var notificationTemplate = template.Must(template.New("acquisition").Parse(`<!doctype html>
<html lang="id"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;background:#f1f5f2;color:#324b3b;font-family:Arial,Helvetica,sans-serif">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr><td align="center" style="padding:28px 14px">
<table role="presentation" width="650" cellpadding="0" cellspacing="0" style="width:100%;max-width:650px;background:#fff;border:1px solid #dce7de;border-radius:14px;overflow:hidden">
{{if .TestMode}}<tr><td style="padding:18px 26px;background:#fff0c2;color:#805b13;border-bottom:1px solid #eedb9d"><strong>MODE TESTING — NOTIFIKASI UJI COBA</strong><p style="margin:6px 0 0;font-size:12px;line-height:1.7">Email ini dikirim dalam tahap pengujian fitur notifikasi GIS. Data di bawah berasal dari sistem. Verifikasi kebutuhan tindak lanjut di aplikasi.<br>This is a test notification from the GIS notification system.</p></td></tr>{{end}}
<tr><td style="padding:25px 28px;background:#20563c;color:#fff"><p style="margin:0 0 8px;color:#b9d9c4;font-size:10px;letter-spacing:2px">AEP PLANTATIONS · GIS</p><h1 style="margin:0;font-size:22px">{{.Event.Company}}</h1><p style="margin:10px 0 0;color:#d1e4d7;font-size:13px">Land Acquisition Process Notification</p></td></tr>
<tr><td style="padding:26px 28px;font-size:13px;line-height:1.8"><p>Dengan Hormat,<br><em>Dear All,</em></p><p>Berikut pembaruan proses approval akuisisi lahan.<br><span style="color:#879c8d">The land acquisition approval process has been updated.</span></p>
<h2 style="font-size:15px;color:#37664a;margin-top:24px">Informasi proses / Process information</h2>
<table width="100%" cellpadding="10" cellspacing="0" style="background:#f3f8f3;border:1px solid #e1eae2;font-size:12px">
<tr><td width="35%" style="color:#7d9383">Approval status</td><td><strong>{{.Approval}}</strong></td></tr>
<tr><td style="color:#7d9383">Tanggal / Date</td><td>{{.Event.Date}}</td></tr>
<tr><td style="color:#7d9383">Oleh / By</td><td>{{.Officer}}</td></tr>
<tr><td style="color:#7d9383;vertical-align:top">Catatan / Note</td><td style="white-space:pre-wrap">{{.Event.Note}}</td></tr></table>
<h2 style="font-size:15px;color:#37664a;margin-top:24px">Informasi lahan / Land information</h2>
<table width="100%" cellpadding="10" cellspacing="0" style="border:1px solid #e1eae2;font-size:12px">
<tr><td width="35%" style="color:#7d9383">Land ID</td><td><strong>{{.Event.LandID}}</strong></td></tr>
<tr style="background:#fafcf9"><td style="color:#7d9383">Company</td><td>{{.Event.Company}}</td></tr>
<tr><td style="color:#7d9383">Estate</td><td>{{.Event.Estate}}</td></tr>
<tr style="background:#fafcf9"><td style="color:#7d9383">Owner</td><td>{{.Event.Owner}}</td></tr>
<tr><td style="color:#7d9383">Land size</td><td>{{.Area}} ha</td></tr>
<tr style="background:#fafcf9"><td style="color:#7d9383">Total price</td><td>Rp {{.Total}}</td></tr>
<tr><td style="color:#7d9383">Status</td><td><strong>{{.Event.StatusName}}</strong></td></tr></table>
<p style="margin-top:24px">Silakan buka aplikasi GIS untuk meninjau informasi dan melanjutkan proses sesuai hak akses Anda.<br><span style="font-size:12px;color:#879c8d">Please sign in to review this land acquisition process.</span></p>
<p style="margin:24px 0"><a href="{{.LoginURL}}" style="display:inline-block;background:#28774e;color:#fff;border-radius:7px;padding:12px 22px;text-decoration:none;font-weight:bold">Buka AEP GIS</a></p>
<p>Regards,<br><strong>GIS Administrator</strong><br>AEP Plantations</p></td></tr>
<tr><td style="padding:16px 28px;background:#fafcf9;border-top:1px solid #e6eee7;color:#91a396;font-size:11px">Email ini dikirim otomatis. Mohon tidak membalas email ini.<br>This message is sent automatically. Please do not reply. · Process #{{.Event.ID}}</td></tr>
</table></td></tr></table></body></html>`))
