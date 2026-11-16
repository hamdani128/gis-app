package acquisition

import (
	"path/filepath"
	"testing"
)

func TestAcquisitionUploadRootUsesConfiguredMount(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ACQUISITION_UPLOADS_DIR", root)
	for _, test := range []struct {
		name string
		got  string
		want string
	}{
		{"payment documents", paymentUploadDir(), filepath.Join(root, "acquisition-land")},
		{"memos", memoUploadDir(), filepath.Join(root, "acquisition-memos")},
		{"PD PDFs", paymentSignatureDir(), filepath.Join(root, "payment-signatures")},
	} {
		if test.got != test.want {
			t.Errorf("%s: got %q, want %q", test.name, test.got, test.want)
		}
	}
}
