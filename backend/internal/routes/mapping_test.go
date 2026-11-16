package routes

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompanyLandMapRequiresSession(t *testing.T) {
	t.Setenv("AUTH_SESSION_SECRET", strings.Repeat("k", 32))
	rec := httptest.NewRecorder()
	Setup(nil).ServeHTTP(rec, httptest.NewRequest("GET", "/api/mapping/company-lands?company_id=7&selected_land_id=24187", nil))
	if rec.Code != 401 {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
