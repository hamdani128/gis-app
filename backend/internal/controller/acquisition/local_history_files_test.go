package acquisition

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalHistoryFilesStayWithinLandFolder(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ACQUISITION_UPLOADS_DIR", root)
	dir := filepath.Join(root, "acquisition-land", "31708")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"memo.jpg": "image", "approval.pdf": "pdf", "ignore.txt": "text"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	files := localHistoryFiles(31708)
	if len(files) != 2 || files[0] != "approval.pdf" || files[1] != "memo.jpg" {
		t.Fatalf("local files = %v", files)
	}
	if got := localHistoryFiles(31709); len(got) != 0 {
		t.Fatalf("another land's files = %v", got)
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/local-files/memo.jpg", nil)
	r.SetPathValue("name", "memo.jpg")
	serveLocalHistoryFile(w, r, 31708)
	if w.Code != http.StatusOK || w.Body.String() != "image" {
		t.Fatalf("local file response = %d %q", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	r.SetPathValue("name", "../memo.jpg")
	serveLocalHistoryFile(w, r, 31708)
	if w.Code != http.StatusNotFound {
		t.Fatalf("traversal response = %d", w.Code)
	}
}
