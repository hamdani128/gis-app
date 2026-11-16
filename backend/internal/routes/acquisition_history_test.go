package routes

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAcquisitionArchivesRequireSession(t *testing.T) {
	t.Setenv("AUTH_SESSION_SECRET", strings.Repeat("k", 32))
	handler := Setup(nil)
	for _, prefix := range []string{"/api/submit-memo-land", "/api/rm-approval", "/api/gm-approval", "/api/gis-approval", "/api/legal-approval", "/api/operation-control-approval", "/api/pd-approval", "/api/acquisition-memos/9/lands"} {
		for _, suffix := range []string{"supported-documents", "supported-documents/3", "supported-documents/3/legacy-status", "memo", "memo/document", "local-files", "local-files/file.pdf"} {
			request := httptest.NewRequest("GET", prefix+"/42/"+suffix+"?company_id=7", nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != 401 {
				t.Errorf("%s/%s: expected 401, got %d", prefix, suffix, response.Code)
			}
		}
	}
}
