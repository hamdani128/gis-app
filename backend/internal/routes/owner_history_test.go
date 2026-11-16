package routes

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOwnerArchiveRoutesRequireSession(t *testing.T) {
	t.Setenv("AUTH_SESSION_SECRET", strings.Repeat("k", 32))
	handler := Setup(nil)
	for _, suffix := range []string{"supported-documents", "supported-documents/3", "supported-documents/3/legacy-status", "memo", "memo/document", "local-files", "local-files/file.pdf"} {
		r := httptest.NewRequest("GET", "/api/land-owners/1/lands/2/"+suffix+"?company_id=1", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 401 {
			t.Errorf("%s: got %d, want 401", suffix, w.Code)
		}
	}
}
