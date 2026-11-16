package mapping

import (
	"context"
	model "gis-app/backend/internal/model/mapping"
	"github.com/jackc/pgx/v5"
	"net/http/httptest"
	"testing"
)

type source struct {
	company, selected, after int64
	calls                    int
	err                      error
}

func (s *source) List(_ context.Context, company, selected, after int64) (model.Layer, error) {
	s.calls++
	s.company = company
	s.selected = selected
	s.after = after
	return model.Layer{Type: "FeatureCollection", Features: []any{}, CompanyID: company}, s.err
}
func TestMapParametersAndCompanyScope(t *testing.T) {
	for _, query := range []string{"company_id=-1&selected_land_id=5", "company_id=2&selected_land_id=0", "company_id=2&selected_land_id=5&after=-1", "company_id=x&selected_land_id=5", "company_id=2&selected_land_id=5&after=x"} {
		stub := &source{}
		rec := httptest.NewRecorder()
		(&CompanyLandController{Model: stub}).Index(rec, httptest.NewRequest("GET", "/?"+query, nil))
		if rec.Code != 400 || stub.calls != 0 {
			t.Fatalf("invalid query reached model: %s", query)
		}
	}
	stub := &source{}
	rec := httptest.NewRecorder()
	(&CompanyLandController{Model: stub}).Index(rec, httptest.NewRequest("GET", "/?company_id=9&selected_land_id=24187&after=300", nil))
	if rec.Code != 200 || stub.company != 9 || stub.selected != 24187 || stub.after != 300 {
		t.Fatal("company and selected land scope were not preserved")
	}
	stub.err = pgx.ErrNoRows
	rec = httptest.NewRecorder()
	(&CompanyLandController{Model: stub}).Index(rec, httptest.NewRequest("GET", "/?company_id=4&selected_land_id=24187", nil))
	if rec.Code != 404 {
		t.Fatal("land outside company must return 404")
	}
}
