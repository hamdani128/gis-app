package acquisition

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gis-app/backend/internal/model/acquisition"
)

func TestPaymentStoreRejectsDocumentWithoutType(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("transfer_date", "2026-11-03"); err != nil {
		t.Fatal(err)
	}
	file, err := writer.CreateFormFile("documents", "approval.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("%PDF-1.4\n%%EOF\n")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/payment/31707?company_id=78", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.SetPathValue("id", "31707")
	response := httptest.NewRecorder()
	(&PaymentController{}).Store(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}

func TestPaymentToOwnerCancelAcceptsReasonWithoutDocument(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range map[string]string{
		"payment_choice": "cancel",
		"transfer_date":  "2026-11-03",
		"note":           "Pemilik membatalkan pembayaran",
	} {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/payment-to-owner/31708?company_id=5", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.SetPathValue("id", "31708")
	response := httptest.NewRecorder()
	called := false
	save := func(_ context.Context, _ string, companyID, landID int64, _ time.Time, note, choice string, docs []acquisition.PaymentDocumentInput) error {
		called = true
		if companyID != 5 || landID != 31708 || note != "Pemilik membatalkan pembayaran" || choice != "cancel" || len(docs) != 0 {
			t.Fatalf("unexpected cancellation values: company=%d land=%d note=%q choice=%q docs=%d", companyID, landID, note, choice, len(docs))
		}
		return nil
	}
	storePaymentDocuments(response, request, save, "Payment to Owner", "8")
	if response.Code != http.StatusOK || !called {
		t.Fatalf("status=%d save_called=%v, want 200 and true", response.Code, called)
	}
}

func TestPaymentToOwnerCancelRequiresReason(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range map[string]string{"payment_choice": "cancel", "transfer_date": "2026-11-03"} {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/payment-to-owner/31708?company_id=5", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.SetPathValue("id", "31708")
	response := httptest.NewRecorder()
	storePaymentDocuments(response, request, nil, "Payment to Owner", "8")
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", response.Code)
	}
}
