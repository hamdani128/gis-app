package acquisition

import (
	"bytes"
	"os"
	"strconv"
	"strings"
	"testing"

	acquisitionModel "gis-app/backend/internal/model/acquisition"
)

func TestPaymentSignaturePDF(t *testing.T) {
	detail := acquisitionModel.SubmitMemoLandDetail{
		Land: map[string]any{
			"id": float64(31707), "company_name": "PT. AEP Pinago Plantations Tbk.",
			"estate_name": "Inti 2 Estate", "permit_number": "HGU INTI 2 TAMBAHAN",
			"block_plan": "Inti 2 Block Plan", "owner": "Afizh Nuryanto", "village": "Keban 1",
			"measurement_date": "2026-10-01T00:00:00", "measured_by": "Haidar", "land_wide": float64(1.43),
			"wet_land_wide": float64(0), "dry_land_wide": float64(0),
			"wet_land_perha_price": float64(0), "dry_land_perha_price": float64(0),
			"admin_price": float64(3509000), "plant_price": float64(111540000),
		},
		History: []map[string]any{
			{"id": float64(11), "sequence_id": float64(1), "name": "Submit Memo Land", "approval": float64(1), "request_number": "MEMO-01", "request_date": "2026-10-20T08:54:47", "employee_name": "Haidar", "date_created": "2026-10-20T09:00:00", "note": "Mohon persetujuan bapak terhadap pengajuan lahan Afizh Nuryanto untuk diproses ke tahap selanjutnya."},
			{"id": float64(12), "sequence_id": float64(9), "name": "RM Approval", "approval": float64(1), "employee_name": "Irwan", "date_created": "2026-10-21T10:00:00", "note": "Lahan strategis karena menyatukan dua bidang lahan yang selama ini terpisah."},
			{"id": float64(13), "sequence_id": float64(8), "name": "GM Approval", "approval": float64(1), "employee_name": "Sarwedi Pangaribuan", "date_created": "2026-10-22T10:00:00", "note": "Areal GRTT untuk menyatukan dengan areal yang telah di GRTT sebelumnya."},
			{"id": float64(14), "sequence_id": float64(2), "name": "GIS Approval", "approval": float64(1), "employee_name": "Suharianto", "date_created": "2026-10-23T10:00:00", "note": "Pengajuan berada di dalam HGU. Data dan peta sudah sesuai dengan pengajuan."},
			{"id": float64(15), "sequence_id": float64(3), "name": "Legal Approval", "approval": float64(1), "employee_name": "Riza Zulfan", "date_created": "2026-10-24T10:00:00", "note": "Dokumen telah dilengkapi dan diperiksa."},
			{"id": float64(16), "sequence_id": float64(10), "name": "Operation Control Approval", "approval": float64(1), "employee_name": "Ronald Sihombing", "date_created": "2026-10-25T10:00:00", "note": "Recommended"},
			{"id": float64(22), "sequence_id": float64(5), "name": "PD Approval", "approval": float64(1), "employee_name": "Budi Purwanto", "date_created": "2026-10-29T16:26:59", "note": "Recommended"},
		},
	}
	data, err := paymentSignaturePDF(detail, "https://example.com/api/public/payment-approvals/31707/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-1.4\n")) || !bytes.HasSuffix(data, []byte("%%EOF\n")) {
		t.Fatal("PDF header or trailer is invalid")
	}
	for _, expected := range []string{"Budi Purwanto", "Process ID: 22", "115.049.000", "/Subtype /Image", "Step", "/Count 1", "Scan QR to download PDF"} {
		if !bytes.Contains(data, []byte(expected)) {
			t.Errorf("PDF is missing %q", expected)
		}
	}
	startMarker := []byte("startxref\n")
	start := bytes.LastIndex(data, startMarker)
	if start < 0 {
		t.Fatal("PDF has no cross-reference pointer")
	}
	value := strings.SplitN(string(data[start+len(startMarker):]), "\n", 2)[0]
	offset, err := strconv.Atoi(value)
	if err != nil || offset < 0 || offset >= len(data) || !bytes.HasPrefix(data[offset:], []byte("xref\n")) {
		t.Fatal("PDF cross-reference pointer is invalid")
	}
	if path := os.Getenv("PAYMENT_PDF_PREVIEW"); path != "" {
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
}
