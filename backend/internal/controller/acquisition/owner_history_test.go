package acquisition

import (
	"gis-app/backend/internal/model/lands"
	"net/http/httptest"
	"testing"
)

func TestOwnerHistoryRejectsInvalidIDsBeforeAccess(t *testing.T) {
	c := CompensatedHistoryController{OwnerModel: &lands.LandOwnerModel{}}
	for _, test := range []struct{ company, owner, land string }{
		{"bad", "1", "2"}, {"-1", "1", "2"}, {"1", "bad", "2"}, {"1", "0", "2"},
		{"1", "1", "bad"}, {"1", "1", "0"}, {"1", "1", "-1"}, {"1", "1", ""},
	} {
		r := httptest.NewRequest("GET", "/api/land-owners/1/lands/2/memo?company_id="+test.company, nil)
		r.SetPathValue("id", test.owner)
		r.SetPathValue("land_id", test.land)
		w := httptest.NewRecorder()
		_, cancel, _, ok := c.params(w, r)
		if cancel != nil {
			cancel()
		}
		if ok || w.Code != 400 {
			t.Fatalf("company=%q owner=%q land=%q: ok=%v status=%d", test.company, test.owner, test.land, ok, w.Code)
		}
	}
}
