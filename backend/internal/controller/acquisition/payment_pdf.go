package acquisition

import (
	"bytes"
	"compress/zlib"
	"context"
	_ "embed"
	"fmt"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"gis-app/backend/internal/model/acquisition"
	"github.com/skip2/go-qrcode"
)

//go:embed assets/aeplogonew.png
var aepLogoPNG []byte

var pdfLogoStream = makePDFLogo()

func makePDFLogo() []byte {
	logo, err := png.Decode(bytes.NewReader(aepLogoPNG))
	if err != nil {
		return nil
	}
	const width, height = 80, 93
	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, a := logo.At(logo.Bounds().Min.X+x*logo.Bounds().Dx()/width, logo.Bounds().Min.Y+y*logo.Bounds().Dy()/height).RGBA()
			_, _ = writer.Write([]byte{byte((r + 65535 - a) / 257), byte((g + 65535 - a) / 257), byte((b + 65535 - a) / 257)})
		}
	}
	if writer.Close() != nil {
		return nil
	}
	return compressed.Bytes()
}

type paymentPDFPage struct{ content strings.Builder }

func pdfText(value any) string {
	if value == nil {
		return "-"
	}
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" {
		return "-"
	}
	return text
}

func pdfEscape(value string) string {
	var out strings.Builder
	for _, char := range value {
		switch char {
		case '\\', '(', ')':
			out.WriteByte('\\')
			out.WriteRune(char)
		case '\n', '\r', '\t':
			out.WriteByte(' ')
		default:
			if char >= 32 && char <= 126 {
				out.WriteRune(char)
			} else {
				out.WriteByte('?')
			}
		}
	}
	return out.String()
}

func (p *paymentPDFPage) text(x, y, size float64, bold bool, value string) {
	font := "F1"
	if bold {
		font = "F2"
	}
	fmt.Fprintf(&p.content, "BT /%s %.1f Tf %.1f %.1f Td (%s) Tj ET\n", font, size, x, y, pdfEscape(value))
}

func (p *paymentPDFPage) line(x1, y1, x2, y2 float64) {
	fmt.Fprintf(&p.content, "%.1f %.1f m %.1f %.1f l S\n", x1, y1, x2, y2)
}

func (p *paymentPDFPage) rect(x, y, width, height float64) {
	fmt.Fprintf(&p.content, "%.1f %.1f %.1f %.1f re S\n", x, y, width, height)
}

func (p *paymentPDFPage) fillRect(x, y, width, height float64, color string) {
	fmt.Fprintf(&p.content, "q %s rg %.1f %.1f %.1f %.1f re f Q\n", color, x, y, width, height)
}

func (p *paymentPDFPage) qrCode(value string, x, y, size float64) error {
	qr, err := qrcode.New(value, qrcode.Medium)
	if err != nil {
		return err
	}
	bitmap := qr.Bitmap()
	module := size / float64(len(bitmap))
	for row, pixels := range bitmap {
		for start := 0; start < len(pixels); {
			if !pixels[start] {
				start++
				continue
			}
			end := start + 1
			for end < len(pixels) && pixels[end] {
				end++
			}
			p.fillRect(x+float64(start)*module, y+size-float64(row+1)*module, float64(end-start)*module, module, "0 0 0")
			start = end
		}
	}
	return nil
}

func (p *paymentPDFPage) whiteText(x, y, size float64, bold bool, value string) {
	font := "F1"
	if bold {
		font = "F2"
	}
	fmt.Fprintf(&p.content, "q 1 1 1 rg BT /%s %.1f Tf %.1f %.1f Td (%s) Tj ET Q\n", font, size, x, y, pdfEscape(value))
}

func paymentHistoryHeader(p *paymentPDFPage, y float64) {
	p.fillRect(42, y-23, 511, 23, "0.14 0.34 0.18")
	for _, column := range []struct {
		x     float64
		label string
	}{{50, "Step"}, {78, "Process"}, {184, "Status"}, {242, "Officer"}, {331, "Date"}, {421, "Notes"}} {
		p.whiteText(column.x, y-15, 8, true, column.label)
	}
}

func pdfMoney(value any) string {
	amount, _ := strconv.ParseFloat(pdfText(value), 64)
	negative := ""
	if amount < 0 {
		negative = "-"
		amount = -amount
	}
	digits := strconv.FormatInt(int64(amount+0.5), 10)
	for i := len(digits) - 3; i > 0; i -= 3 {
		digits = digits[:i] + "." + digits[i:]
	}
	return negative + digits
}

func pdfDate(value any) string {
	text := pdfText(value)
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"} {
		if parsed, err := time.Parse(layout, text); err == nil {
			return parsed.Format("02 Jan 2006")
		}
	}
	return text
}

func pdfDateTime(value any) string {
	text := pdfText(value)
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05", "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, text); err == nil {
			return parsed.Format("02 Jan 2006 15:04:05")
		}
	}
	return text
}

func (p *paymentPDFPage) field(x, y float64, label string, value any) {
	if label == "Company" {
		p.text(x, y, 9, false, label)
		p.text(x+82, y, 9, false, ":")
		p.text(x+91, y, 9, true, pdfText(value))
		return
	}
	p.text(x, y, 9, false, label)
	p.text(x+111, y, 9, false, ":")
	p.text(x+120, y, 9, true, pdfText(value))
}

func pdfWrap(value string, max int) []string {
	words := strings.Fields(value)
	if len(words) == 0 {
		return []string{"-"}
	}
	lines := []string{}
	line := ""
	for _, word := range words {
		if line != "" && len(line)+1+len(word) > max {
			lines = append(lines, line)
			line = ""
		}
		if len(word) > max {
			word = word[:max]
		}
		if line != "" {
			line += " "
		}
		line += word
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

func paymentSignaturePDF(detail acquisition.SubmitMemoLandDetail, publicURL string) ([]byte, error) {
	land := detail.Land
	history := make([]map[string]any, 0, len(detail.History))
	for _, record := range detail.History {
		if record["sequence_id"] != float64(6) {
			history = append(history, record)
		}
	}
	pages := []*paymentPDFPage{{}}
	p := pages[0]
	p.content.WriteString("0.14 0.34 0.18 RG 0.14 0.34 0.18 rg\n")
	p.text(42, 793, 16, true, "LAND ACQUISITION PAYMENT APPROVAL")
	if len(pdfLogoStream) > 0 {
		p.content.WriteString("q 40 0 0 47 510 769 cm /Im1 Do Q\n")
	} else {
		p.text(505, 793, 18, true, "AEP")
	}
	p.text(42, 777, 8, false, "Printed on: "+time.Now().Format("02 Jan 2006 15:04"))
	p.line(42, 764, 553, 764)
	p.text(42, 745, 11, true, "LAND INFORMATION")
	p.text(305, 745, 11, true, "MEASUREMENT DETAILS")
	p.line(42, 739, 289, 739)
	p.line(305, 739, 553, 739)

	memoNumber, memoDate := "-", "-"
	for _, record := range history {
		if record["sequence_id"] == float64(submitSequenceIDForPDF) {
			memoNumber, memoDate = pdfText(record["request_number"]), pdfDate(record["request_date"])
			break
		}
	}
	left := [][2]any{{"Land ID", land["id"]}, {"Company", land["company_name"]}, {"Estate", land["estate_name"]}, {"Permit Number", land["permit_number"]}, {"Block Plan", land["block_plan"]}, {"Land Owner", land["owner"]}, {"Village", land["village"]}, {"Memo Number", memoNumber}, {"Memo Date", memoDate}}
	for i, row := range left {
		p.field(42, 721-float64(i)*20, row[0].(string), row[1])
	}
	measure := [][2]any{{"Measurement Date", pdfDate(land["measurement_date"])}, {"Measured By", land["measured_by"]}, {"Wet Land Size", fmt.Sprintf("%v Ha", land["wet_land_wide"])}, {"Dry Land Size", fmt.Sprintf("%v Ha", land["dry_land_wide"])}, {"Total Land Size", fmt.Sprintf("%v Ha", land["land_wide"])}}
	for i, row := range measure {
		p.field(305, 721-float64(i)*23, row[0].(string), row[1])
	}
	p.text(305, 591, 11, true, "COMPENSATION CALCULATION")
	p.line(305, 585, 553, 585)
	prices := [][2]string{{"Wet Land Price (@Ha)", pdfMoney(land["wet_land_perha_price"])}, {"Dry Land Price (@Ha)", pdfMoney(land["dry_land_perha_price"])}, {"Administration Cost", pdfMoney(land["admin_price"])}, {"Plant Compensation", pdfMoney(land["plant_price"])}}
	for i, row := range prices {
		p.text(305, 567-float64(i)*22, 9, false, row[0])
		p.text(487, 567-float64(i)*22, 9, false, row[1])
	}
	p.line(305, 492, 553, 492)
	total := amount(land["wet_land_wide"])*amount(land["wet_land_perha_price"]) + amount(land["dry_land_wide"])*amount(land["dry_land_perha_price"]) + amount(land["admin_price"]) + amount(land["plant_price"])
	p.text(305, 475, 9, true, "TOTAL COMPENSATION")
	p.text(487, 475, 9, true, pdfMoney(total))
	p.text(305, 452, 9, false, "Average Price per Hectare")
	if area := amount(land["land_wide"]); area > 0 {
		p.text(487, 452, 9, false, pdfMoney(total/area))
	} else {
		p.text(487, 452, 9, false, "0")
	}
	p.rect(42, 427, 247, 115)
	p.fillRect(43, 428, 245, 113, "0.96 0.98 0.96")
	p.text(52, 520, 8, true, "ELECTRONIC PD APPROVAL")
	for _, record := range history {
		if record["sequence_id"] == float64(5) && record["approval"] == float64(1) {
			name := pdfText(record["employee_name"])
			if name == "-" {
				name = pdfText(record["username"])
			}
			p.text(52, 498, 8, true, name)
			p.text(52, 480, 7, false, pdfDateTime(record["date_created"]))
			p.text(52, 462, 7, false, fmt.Sprintf("Process ID: %v", record["id"]))
			break
		}
	}
	p.text(52, 443, 7, false, "Scan QR to download PDF")
	if err := p.qrCode(publicURL, 192, 438, 91); err != nil {
		return nil, err
	}
	p.text(42, 406, 11, true, "APPROVAL HISTORY")
	p.line(42, 399, 553, 399)
	paymentHistoryHeader(p, 389)
	y := float64(366)
	stepNumbers := map[int]string{1: "1", 9: "2", 8: "3", 2: "4", 3: "4", 10: "5", 5: "6"}
	for index, record := range history {
		processLines := pdfWrap(pdfText(record["name"]), 19)
		officerLines := pdfWrap(pdfText(record["employee_name"]), 17)
		noteLines := pdfWrap(pdfText(record["note"]), 28)
		dateLines := pdfWrap(pdfDate(record["date_created"]), 16)
		lineCount := max(len(processLines), len(officerLines), len(noteLines), len(dateLines))
		rowHeight := float64(lineCount*9 + 13)
		if y-rowHeight < 57 {
			p = &paymentPDFPage{}
			pages = append(pages, p)
			p.content.WriteString("0.14 0.34 0.18 RG 0.14 0.34 0.18 rg\n")
			p.text(42, 790, 13, true, "LAND ACQUISITION PAYMENT APPROVAL - HISTORY")
			p.line(42, 778, 553, 778)
			paymentHistoryHeader(p, 762)
			y = 739
		}
		if index%2 == 1 {
			p.fillRect(42, y-rowHeight, 511, rowHeight, "0.96 0.97 0.96")
		}
		sequence := int(amount(record["sequence_id"]))
		step := stepNumbers[sequence]
		if step == "" {
			step = strconv.Itoa(index + 1)
		}
		p.text(52, y-12, 8, true, step)
		status := "Approved"
		if record["approval"] == float64(0) {
			status = "Rejected"
		}
		if record["approval"] == nil {
			status = "Pending"
		}
		p.text(184, y-12, 7, true, status)
		for lineIndex, line := range processLines {
			p.text(78, y-12-float64(lineIndex)*9, 7, true, line)
		}
		for lineIndex, line := range officerLines {
			p.text(242, y-12-float64(lineIndex)*9, 7, false, line)
		}
		for lineIndex, line := range dateLines {
			p.text(331, y-12-float64(lineIndex)*9, 7, false, line)
		}
		for lineIndex, line := range noteLines {
			p.text(421, y-12-float64(lineIndex)*9, 7, false, line)
		}
		y -= rowHeight
		p.line(42, y, 553, y)
	}
	for pageIndex, page := range pages {
		page.line(42, 48, 553, 48)
		page.text(42, 34, 7, false, fmt.Sprintf("Document ID: %v | Generated on: %s", land["id"], time.Now().Format("20060102150405")))
		page.text(42, 23, 7, false, "Official document generated from the recorded PD approval")
		page.text(520, 34, 7, false, fmt.Sprintf("%d/%d", pageIndex+1, len(pages)))
	}
	return encodePaymentPDF(pages), nil
}

const submitSequenceIDForPDF = 1

func amount(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	default:
		parsed, _ := strconv.ParseFloat(fmt.Sprint(value), 64)
		return parsed
	}
}

func encodePaymentPDF(pages []*paymentPDFPage) []byte {
	// One catalog, one page tree, two standard fonts, then two objects per page.
	objects := []string{"", "", "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>", "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold >>"}
	imageResource := ""
	if len(pdfLogoStream) > 0 {
		objects = append(objects, fmt.Sprintf("<< /Type /XObject /Subtype /Image /Width 80 /Height 93 /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /FlateDecode /Length %d >>\nstream\n%sendstream", len(pdfLogoStream), string(pdfLogoStream)))
		imageResource = " /XObject << /Im1 5 0 R >>"
	}
	pageIDs := make([]int, len(pages))
	for i, page := range pages {
		pageIDs[i] = len(objects) + 1
		contentID := pageIDs[i] + 1
		stream := page.content.String()
		objects = append(objects,
			fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 3 0 R /F2 4 0 R >>%s >> /Contents %d 0 R >>", imageResource, contentID),
			fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(stream), stream))
	}
	refs := make([]string, len(pageIDs))
	for i, id := range pageIDs {
		refs[i] = fmt.Sprintf("%d 0 R", id)
	}
	objects[0] = "<< /Type /Catalog /Pages 2 0 R >>"
	objects[1] = fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(refs, " "), len(pageIDs))
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, object := range objects {
		offsets[i+1] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(offsets))
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&out, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&out, "trailer << /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xref)
	return out.Bytes()
}

var signaturePDFMu sync.Mutex

func paymentSignatureDir() string {
	return filepath.Join(acquisitionUploadRoot(), "payment-signatures")
}

func (c *PaymentController) SignaturePDF(w http.ResponseWriter, r *http.Request) {
	companyID, landID, ok := submitParams(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	detail, err := c.Model.PaymentDocumentDetail(ctx, companyID, landID)
	if err != nil {
		paymentFailure(w, err)
		return
	}
	if err := os.MkdirAll(paymentSignatureDir(), 0755); err != nil {
		paymentFailure(w, err)
		return
	}
	path := paymentPublicPDFPath(landID)
	signaturePDFMu.Lock()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		token, tokenErr := paymentPublicToken(landID)
		if tokenErr != nil {
			signaturePDFMu.Unlock()
			paymentFailure(w, tokenErr)
			return
		}
		data, pdfErr := paymentSignaturePDF(detail, paymentPublicURL(landID, token))
		if pdfErr != nil {
			signaturePDFMu.Unlock()
			paymentFailure(w, pdfErr)
			return
		}
		file, createErr := os.CreateTemp(paymentSignatureDir(), ".payment-approval-*.pdf")
		if createErr != nil {
			signaturePDFMu.Unlock()
			paymentFailure(w, createErr)
			return
		}
		tempPath := file.Name()
		_, writeErr := file.Write(data)
		closeErr := file.Close()
		if writeErr != nil || closeErr != nil {
			os.Remove(tempPath)
			signaturePDFMu.Unlock()
			if writeErr != nil {
				paymentFailure(w, writeErr)
			} else {
				paymentFailure(w, closeErr)
			}
			return
		}
		linkErr := os.Link(tempPath, path)
		os.Remove(tempPath)
		if linkErr != nil && !os.IsExist(linkErr) {
			signaturePDFMu.Unlock()
			paymentFailure(w, linkErr)
			return
		}
	} else if err != nil {
		signaturePDFMu.Unlock()
		paymentFailure(w, err)
		return
	}
	signaturePDFMu.Unlock()
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "application/pdf")
	disposition := "attachment"
	if r.URL.Query().Get("preview") == "1" {
		disposition = "inline"
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=payment-approval-%d.pdf", disposition, landID))
	http.ServeFile(w, r, path)
}
