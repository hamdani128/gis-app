package acquisition

import (
	"context"
	"github.com/jackc/pgx/v5"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStageArchivesRunAccessCheckBeforeReadingFiles(t *testing.T) {
	for _, name := range []string{"documents", "file", "legacy-status", "memo", "memo-file", "local-files", "local-file"} {
		t.Run(name, func(t *testing.T) {
			checks := 0
			controller := CompensatedHistoryController{CheckAccess: func(ctx context.Context, r *http.Request, companyID, landID int64) error {
				checks++
				if companyID != 7 || landID != 42 {
					t.Fatalf("incorrect access target: company=%d land=%d", companyID, landID)
				}
				return pgx.ErrNoRows
			}}
			handlers := map[string]http.HandlerFunc{"documents": controller.SupportedDocuments, "file": controller.SupportedDocumentFile, "legacy-status": controller.SupportedDocumentLegacyStatus, "memo": controller.Memo, "memo-file": controller.MemoFile, "local-files": controller.LocalHistoryFiles, "local-file": controller.LocalHistoryFile}
			r := httptest.NewRequest("GET", "/api/gis-approval/42/memo?company_id=7", nil)
			r.SetPathValue("id", "42")
			r.SetPathValue("doc_id", "3")
			r.SetPathValue("name", "file.pdf")
			w := httptest.NewRecorder()
			handlers[name](w, r)
			if checks != 1 || w.Code != 404 {
				t.Fatalf("checks=%d status=%d", checks, w.Code)
			}
		})
	}
}
