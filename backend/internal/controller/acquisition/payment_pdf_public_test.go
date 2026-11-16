package acquisition

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublicSignaturePDF(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(paymentSignatureDir(), 0755); err != nil {
		t.Fatal(err)
	}
	const landID int64 = 31707
	token, err := paymentPublicToken(landID)
	if err != nil || len(token) != 64 {
		t.Fatalf("token creation failed: %v", err)
	}
	again, err := paymentPublicToken(landID)
	if err != nil || again != token {
		t.Fatalf("token changed: %v", err)
	}
	const data = "%PDF-1.4\npublic document\n%%EOF\n"
	if err := os.WriteFile(paymentPublicPDFPath(landID), []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	if got := paymentPublicURL(landID, token); !strings.HasSuffix(got, "/31707/"+token) {
		t.Fatalf("unexpected public URL: %q", got)
	}
	controller := &PaymentController{}
	for _, testcase := range []struct {
		name   string
		token  string
		status int
	}{
		{"valid token", token, http.StatusOK},
		{"incorrect token", strings.Repeat("0", 64), http.StatusNotFound},
		{"malformed token", "invalid", http.StatusNotFound},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/public/payment-approvals/31707/"+testcase.token, nil)
			request.SetPathValue("id", "31707")
			request.SetPathValue("token", testcase.token)
			response := httptest.NewRecorder()
			controller.PublicSignaturePDF(response, request)
			if response.Code != testcase.status {
				t.Fatalf("status = %d, want %d", response.Code, testcase.status)
			}
			if testcase.status == http.StatusOK {
				if response.Body.String() != data || response.Header().Get("Content-Disposition") != "attachment; filename=payment-approval-31707.pdf" {
					t.Fatal("public response did not serve the expected PDF")
				}
			}
		})
	}
	if _, err := os.Stat(filepath.Join(paymentSignatureDir(), "payment-approval-31707.token")); err != nil {
		t.Fatal(err)
	}
}
