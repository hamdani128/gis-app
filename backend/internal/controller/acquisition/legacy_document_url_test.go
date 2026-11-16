package acquisition

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestLegacyDocumentURLUsesRecordFolders(t *testing.T) {
	t.Setenv("LEGACY_DOCUMENT_URL", defaultLegacyDocumentURL)
	created := time.Date(2026, time.October, 21, 0, 0, 0, 0, time.UTC)
	march := time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name, link, file, folder1, folder2 string
	}{
		{"supported document", datedLegacyDocumentURL("7bccb-2.jpeg", &created), "7bccb-2.jpeg", "2026", "10"},
		{"single-digit month", datedLegacyDocumentURL("march.jpg", &march), "march.jpg", "2026", "3"},
		{"memo", memoLegacyDocumentURL("1cf24-Haryanto20261021_14243112.jpg", 5), "1cf24-Haryanto20261021_14243112.jpg", "memo", "5"},
	} {
		parsed, err := url.Parse(test.link)
		if err != nil || parsed.Host != "gis.aepplantations.id" {
			t.Fatalf("%s: invalid URL %q: %v", test.name, test.link, err)
		}
		q := parsed.Query()
		if q.Get("p_file_name") != test.file || q.Get("folder1") != test.folder1 || q.Get("folder2") != test.folder2 {
			t.Errorf("%s: wrong URL query: %v", test.name, q)
		}
	}
	if got := datedLegacyDocumentURL("../outside.pdf", &created); got != "" {
		t.Errorf("path traversal produced URL %q", got)
	}
	if got := datedLegacyDocumentURL("missing.pdf", nil); got != "" {
		t.Errorf("missing date produced URL %q", got)
	}
}

func TestHistoryFileRedirectsToLegacyURL(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/lands/31708/document", nil)
	legacy := memoLegacyDocumentURL("memo.jpg", 5)
	servePaymentHistoryFile(w, r, "", legacy, "memo.jpg")
	if w.Code != http.StatusFound || w.Header().Get("Location") != legacy {
		t.Fatalf("redirect = %d %q, want 302 %q", w.Code, w.Header().Get("Location"), legacy)
	}
}

func TestLegacyPreviewUsesDirectUploadPath(t *testing.T) {
	t.Setenv("LEGACY_DOCUMENT_PREVIEW_BASE_URL", defaultLegacyPreviewBaseURL)
	created := time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)
	if got := datedLegacyPreviewURL("memo page-1.jpg", &created); got != "http://gis.aepplantations.id/CI/assets/uploads/2026/3/memo%20page-1.jpg" {
		t.Fatalf("dated preview URL = %q", got)
	}
	if got := memoLegacyPreviewURL("memo.jpg", 5); got != "http://gis.aepplantations.id/CI/assets/uploads/memo/5/memo.jpg" {
		t.Fatalf("memo preview URL = %q", got)
	}
	if got := memoLegacyPreviewURL("../outside.jpg", 5); got != "" {
		t.Fatalf("path traversal produced preview URL %q", got)
	}
}

func TestMissingBackendPreviewDoesNotRedirectToDownload(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/payment/31708/memo/document?preview=1", nil)
	servePaymentHistoryFile(w, r, "", memoLegacyDocumentURL("memo.jpg", 5), "memo.jpg")
	if w.Code != http.StatusNotFound || w.Header().Get("Location") != "" {
		t.Fatalf("missing backend preview = %d, location %q", w.Code, w.Header().Get("Location"))
	}
}
