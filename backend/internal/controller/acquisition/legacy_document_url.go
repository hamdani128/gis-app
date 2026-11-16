package acquisition

import (
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultLegacyDocumentURL = "http://gis.aepplantations.id/doc/download_doc.php"
const defaultLegacyPreviewBaseURL = "http://gis.aepplantations.id/CI/assets/uploads"

func legacyPreviewURL(name, folder1, folder2 string) string {
	if !validStoredName(name) || folder1 == "" || folder2 == "" {
		return ""
	}
	base := strings.TrimSpace(os.Getenv("LEGACY_DOCUMENT_PREVIEW_BASE_URL"))
	if base == "" {
		base = defaultLegacyPreviewBaseURL
	}
	u, err := url.Parse(base)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	return strings.TrimRight(u.String(), "/") + "/" + url.PathEscape(folder1) + "/" + url.PathEscape(folder2) + "/" + url.PathEscape(name)
}

func legacyDocumentURL(name, folder1, folder2 string) string {
	if !validStoredName(name) || folder1 == "" || folder2 == "" {
		return ""
	}
	base := strings.TrimSpace(os.Getenv("LEGACY_DOCUMENT_URL"))
	if base == "" {
		base = defaultLegacyDocumentURL
	}
	u, err := url.Parse(base)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	q := u.Query()
	q.Set("p_file_name", name)
	q.Set("folder1", folder1)
	q.Set("folder2", folder2)
	u.RawQuery = q.Encode()
	return u.String()
}

func datedLegacyDocumentURL(name string, created *time.Time) string {
	if created == nil {
		return ""
	}
	return legacyDocumentURL(name, created.Format("2006"), strconv.Itoa(int(created.Month())))
}

func datedLegacyPreviewURL(name string, created *time.Time) string {
	if created == nil {
		return ""
	}
	return legacyPreviewURL(name, created.Format("2006"), strconv.Itoa(int(created.Month())))
}

func memoLegacyDocumentURL(name string, companyID int64) string {
	if companyID <= 0 {
		return ""
	}
	return legacyDocumentURL(name, "memo", strconv.FormatInt(companyID, 10))
}

func memoLegacyPreviewURL(name string, companyID int64) string {
	if companyID <= 0 {
		return ""
	}
	return legacyPreviewURL(name, "memo", strconv.FormatInt(companyID, 10))
}
