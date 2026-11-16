package acquisition

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gis-app/backend/internal/model/acquisition"
)

func TestMemoDocumentPathUsesMemoCompanyFolder(t *testing.T) {
	root := t.TempDir()
	t.Setenv("LEGACY_MEMO_DIR", root)
	companyDir := filepath.Join(root, "4")
	if err := os.MkdirAll(companyDir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(companyDir, "memo-example.pdf")
	if err := os.WriteFile(path, []byte("PDF"), 0644); err != nil {
		t.Fatal(err)
	}

	item := acquisition.PaymentMemo{CompanyID: 4, FileName: "memo-example.pdf", Found: true}
	if got := memoDocumentPath(31708, item); got != path {
		t.Fatalf("memo path = %q, want %q", got, path)
	}
	item.CompanyID = 5
	if got := memoDocumentPath(31708, item); got != "" {
		t.Fatalf("memo from another company folder should not resolve: %q", got)
	}
}

func TestHistoryPathsPreferBackendLandFolder(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ACQUISITION_UPLOADS_DIR", root)
	landID := int64(31708)
	landDir := filepath.Join(root, "acquisition-land", "31708")
	if err := os.MkdirAll(landDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"historic.jpg", "memo.jpg"} {
		if err := os.WriteFile(filepath.Join(landDir, name), []byte("image"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	created := time.Date(2026, time.October, 21, 0, 0, 0, 0, time.UTC)
	doc := acquisition.PaymentSupportedDocument{FileName: "historic.jpg", Group: "Payment", Created: &created}
	if got := supportedDocumentPath(landID, doc); got != filepath.Join(landDir, "historic.jpg") {
		t.Fatalf("supported document path = %q", got)
	}
	memo := acquisition.PaymentMemo{CompanyID: 5, FileName: "memo.jpg", Found: true}
	if got := memoDocumentPath(landID, memo); got != filepath.Join(landDir, "memo.jpg") {
		t.Fatalf("memo path = %q", got)
	}
	setSupportedDocumentAccess(landID, &doc)
	setMemoAccess(landID, &memo)
	if !doc.Available || doc.PreviewURL != "http://gis.aepplantations.id/CI/assets/uploads/2026/10/historic.jpg" || doc.LegacyURL != "" || !memo.Available || memo.PreviewURL != "" {
		t.Fatalf("backend files should be served locally: doc=%+v memo=%+v", doc, memo)
	}
}

func TestReportMemoChecksLegacyPreviewBeforeBackendFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ACQUISITION_UPLOADS_DIR", root)
	landID := int64(31708)
	landDir := filepath.Join(root, "acquisition-land", "31708")
	if err := os.MkdirAll(landDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(landDir, "memo.jpg"), []byte("image"), 0644); err != nil {
		t.Fatal(err)
	}
	item := acquisition.PaymentMemo{CompanyID: 5, FileName: "memo.jpg", Found: true}
	(ReportHistoryFiles{}).SetMemoAccess(landID, &item)
	if !item.Available || item.PreviewURL != "http://gis.aepplantations.id/CI/assets/uploads/memo/5/memo.jpg" {
		t.Fatalf("report memo should expose the legacy preview with a local fallback: %+v", item)
	}
}

func TestPaymentDocumentPathSeparatesLandFoldersAndReadsOldFlatFiles(t *testing.T) {
	root := t.TempDir()
	landDir := filepath.Join(root, "31708")
	if err := os.MkdirAll(landDir, 0755); err != nil {
		t.Fatal(err)
	}
	newPath := filepath.Join(landDir, "new.pdf")
	if err := os.WriteFile(newPath, []byte("PDF"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := paymentDocumentPathIn(root, 31708, "new.pdf"); got != newPath {
		t.Fatalf("new file path = %q", got)
	}
	if got := paymentDocumentPathIn(root, 31709, "new.pdf"); got != "" {
		t.Fatalf("another land resolved file: %q", got)
	}
	oldPath := filepath.Join(root, "old.pdf")
	if err := os.WriteFile(oldPath, []byte("PDF"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := paymentDocumentPathIn(root, 31708, "old.pdf"); got != oldPath {
		t.Fatalf("old file path = %q", got)
	}
	if got := paymentDocumentPathIn(root, 31708, "../old.pdf"); got != "" {
		t.Fatalf("traversal resolved file: %q", got)
	}
}

func TestSupportedDocumentPathStillReadsLegacyDatedFolder(t *testing.T) {
	root := t.TempDir()
	t.Setenv("LEGACY_SUPPORTED_DOC_DIR", root)
	created := time.Date(2026, time.March, 21, 0, 0, 0, 0, time.UTC)
	dir := filepath.Join(root, "2026", "3")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "historic.jpg")
	if err := os.WriteFile(path, []byte("JPEG"), 0644); err != nil {
		t.Fatal(err)
	}
	item := acquisition.PaymentSupportedDocument{FileName: "historic.jpg", Group: "Land", Created: &created}
	if got := supportedDocumentPath(31708, item); got != path {
		t.Fatalf("legacy file path = %q", got)
	}
}
