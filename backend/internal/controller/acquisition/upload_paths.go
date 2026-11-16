package acquisition

import (
	"os"
	"path/filepath"
	"strings"
)

func acquisitionUploadRoot() string {
	if configured := strings.TrimSpace(os.Getenv("ACQUISITION_UPLOADS_DIR")); configured != "" {
		return filepath.Clean(configured)
	}
	cwd, _ := os.Getwd()
	if filepath.Base(cwd) == "backend" {
		return "uploads"
	}
	return filepath.Join("backend", "uploads")
}
