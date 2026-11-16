package notification

import (
	"gis-app/backend/internal/mailer"
	"strings"
	"testing"
)

func testConfig() Config {
	return Config{TestMode: true, TestTo: "tester@example.com", Mail: mailer.Config{From: "GIS <sender@example.com>", LoginURL: "https://gis.example.com/auth/login"}, BCC: []string{"observer@example.com"}, RejectedCC: []string{"rejection@example.com"}}
}

func TestTestingMessageAndRecipients(t *testing.T) {
	event := Event{ID: 17, LandID: 45, Company: "Example Estate", Status: 1, Approval: 1, OfficerEmail: "author@example.com", Area: 12.34, Total: 123456789}
	m, err := BuildMessage(event, Recipients{To: []string{"approver@example.com"}, CC: []string{"next@example.com"}}, testConfig())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(m.Subject, "[TESTING] [GIS]") || !strings.Contains(m.HTML, "MODE TESTING") {
		t.Fatal("missing testing markers")
	}
	if strings.Join(m.To, ",") != "tester@example.com" || len(m.CC) != 0 || len(m.BCC) != 0 {
		t.Fatal("unexpected recipients")
	}
	if !strings.Contains(m.HTML, "12,34 ha") || !strings.Contains(m.HTML, "123.456.789") {
		t.Fatal("unexpected amount formatting")
	}
	if m.ID != "gis-acquisition-17@aepplantations.id" {
		t.Fatal("missing stable process message ID")
	}
}

func TestRejectedRecipientsAndEscapedContent(t *testing.T) {
	event := Event{ID: 9, Company: "<script>bad</script>", Note: "<img src=x onerror=bad>", Status: 99, Approval: 0, OfficerEmail: "author@example.com"}
	config := testConfig()
	config.TestMode = false
	m, err := BuildMessage(event, Recipients{To: []string{"submitter@example.com"}, CC: []string{"next@example.com"}}, config)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(m.To, ",") != "submitter@example.com,author@example.com" || strings.Join(m.CC, ",") != "next@example.com,rejection@example.com" {
		t.Fatal("rejected recipient routing changed")
	}
	if !strings.Contains(m.Subject, "[REJECTED]") || strings.Contains(m.HTML, "<script>") || strings.Contains(m.HTML, "<img src=x") {
		t.Fatal("unsafe content or missing rejected subject")
	}
}

func TestProductionAndDeduplication(t *testing.T) {
	c := testConfig()
	c.TestMode = false
	e := Event{ID: 1, Company: "Company", OfficerEmail: "APPROVER@example.com"}
	m, err := BuildMessage(e, Recipients{To: []string{"approver@example.com", "approver@example.com"}, CC: []string{"observer@example.com"}}, c)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(m.Subject, "TESTING") || strings.Contains(m.HTML, "MODE TESTING") {
		t.Fatal("testing markers remain in production")
	}
	if len(m.To) != 1 || len(m.CC) != 1 || len(m.BCC) != 0 {
		t.Fatal("duplicate recipients were not removed")
	}
}

func TestNoApproverAndHeaderInjectionFail(t *testing.T) {
	for _, e := range []Event{{ID: 1, Company: "Company"}, {ID: 2, Company: "Company\r\nBcc: other@example.com"}} {
		recipients := Recipients{}
		if e.ID == 2 {
			recipients.To = []string{"approver@example.com"}
		}
		config := testConfig()
		config.TestMode = false
		if _, err := BuildMessage(e, recipients, config); err == nil {
			t.Fatal("invalid notification accepted")
		}
	}
}

func TestApprovalLabels(t *testing.T) {
	for value, want := range map[int]string{1: "Approved", 2: "On Hold", 0: "Rejected", -1: "Unknown"} {
		if approvalLabel(value) != want {
			t.Fatal("wrong approval label")
		}
	}
}

func TestTestingSuppressesEveryProductionRecipient(t *testing.T) {
	for _, status := range []int{1, 99} {
		config := testConfig()
		event := Event{ID: 12, Company: "Company", Status: status, OfficerEmail: "author@example.com"}
		for _, recipients := range []Recipients{{}, {To: []string{"bad email"}, CC: []string{"other@example.com"}}} {
			message, err := BuildMessage(event, recipients, config)
			if err != nil {
				t.Fatal(err)
			}
			if len(message.To) != 1 || message.To[0] != config.TestTo || len(message.CC) != 0 || len(message.BCC) != 0 {
				t.Fatal("production recipient leaked into testing delivery")
			}
		}
		config.TestTo = ""
		if _, err := BuildMessage(event, Recipients{To: []string{"approver@example.com"}}, config); err == nil {
			t.Fatal("missing testing address fell back to real recipients")
		}
	}
}

func TestConfigDefaultsToTesting(t *testing.T) {
	for key, value := range map[string]string{"SMTP_HOST": "smtp.example.com", "SMTP_PORT": "465", "SMTP_TLS_MODE": "tls", "SMTP_USERNAME": "", "SMTP_PASSWORD": "", "SMTP_FROM": "sender@example.com", "FRONTEND_ORIGIN": "https://gis.example.com", "NOTIFICATION_TEST_MODE": "", "NOTIFICATION_TEST_TO": "tester@example.com", "NOTIFICATION_BATCH_SIZE": "", "NOTIFICATION_BCC": "copy@example.com; second@example.com", "NOTIFICATION_REJECTED_CC": "copy@example.com"} {
		t.Setenv(key, value)
	}
	c, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !c.TestMode || c.BatchSize != 1 || len(c.BCC) != 2 || len(c.RejectedCC) != 1 {
		t.Fatal("unexpected defaults or recipients")
	}
	t.Setenv("NOTIFICATION_TEST_TO", "")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("missing test address accepted")
	}
	t.Setenv("NOTIFICATION_TEST_MODE", "false")
	c, err = LoadConfig()
	if err != nil || c.TestMode {
		t.Fatal("explicit production mode not applied")
	}
	t.Setenv("NOTIFICATION_TEST_MODE", "typo")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("invalid test mode accepted")
	}
}
