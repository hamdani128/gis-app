package acquisition

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckLegacyPreviewStatus(t *testing.T) {
	status, contentType := http.StatusOK, "application/pdf"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("method = %s", r.Method)
		}
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(status)
	}))
	defer server.Close()
	for _, test := range []struct {
		status      int
		contentType string
		want        *bool
	}{
		{http.StatusOK, "application/pdf", boolPtr(true)},
		{http.StatusNotFound, "text/html", boolPtr(false)},
		{http.StatusOK, "text/html", boolPtr(false)},
		{http.StatusServiceUnavailable, "text/html", nil},
	} {
		status, contentType = test.status, test.contentType
		got := checkLegacyPreviewStatus(context.Background(), server.URL)
		if (got == nil) != (test.want == nil) || (got != nil && *got != *test.want) {
			t.Errorf("status %d type %s: got %v, want %v", status, contentType, got, test.want)
		}
	}
}

func boolPtr(value bool) *bool { return &value }
