package report

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	acquisitionController "gis-app/backend/internal/controller/acquisition"
	"gis-app/backend/internal/controller/auth"
	"gis-app/backend/internal/model/acquisition"
	authModel "gis-app/backend/internal/model/auth"
	reportModel "gis-app/backend/internal/model/report"
	"gis-app/backend/internal/response"

	"github.com/jackc/pgx/v5"
)

type ReportController struct {
	Model     *reportModel.ReportModel
	Documents *acquisition.SubmitMemoLandModel
	Access    *authModel.UserModel
	Files     acquisitionController.ReportHistoryFiles
}

var reportCodes = map[string]bool{
	"rlist_acq_process": true,
	"mapreport":         true,
	"acqprogress":       true,
	"detaillandacq":     true,
	"rpt_all_progress":  true,
	"plantmaprep":       true,
	"reportdocland":     true,
	"aestategrttreport": true,
}

func (c *ReportController) params(w http.ResponseWriter, r *http.Request) (context.Context, context.CancelFunc, int64, string, bool) {
	code := strings.ToLower(strings.TrimSpace(r.PathValue("code")))
	if !reportCodes[code] {
		response.Error(w, http.StatusNotFound, "Report tidak ditemukan")
		return nil, nil, 0, "", false
	}
	companyID, err := strconv.ParseInt(r.URL.Query().Get("company_id"), 10, 64)
	if err != nil || companyID < 0 {
		response.Error(w, http.StatusBadRequest, "company_id tidak valid")
		return nil, nil, 0, "", false
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	codes, err := c.Access.GetVisibleModuleCodes(ctx, auth.Username(r))
	if err != nil {
		cancel()
		response.Error(w, http.StatusInternalServerError, "Hak akses report gagal diperiksa")
		return nil, nil, 0, "", false
	}
	for _, visible := range codes {
		if visible == code {
			return ctx, cancel, companyID, code, true
		}
	}
	cancel()
	response.Error(w, http.StatusForbidden, "Tidak memiliki hak akses report")
	return nil, nil, 0, "", false
}

func reportLandID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, http.StatusBadRequest, "ID lahan tidak valid")
		return 0, false
	}
	return id, true
}

func (c *ReportController) land(w http.ResponseWriter, ctx context.Context, companyID int64, r *http.Request) (int64, bool) {
	id, ok := reportLandID(w, r)
	if !ok {
		return 0, false
	}
	if _, err := c.Model.Land(ctx, companyID, id); err != nil {
		reportFailure(w, err)
		return 0, false
	}
	return id, true
}

func reportFailure(w http.ResponseWriter, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		response.Error(w, http.StatusNotFound, "Lahan tidak ditemukan pada company aktif")
		return
	}
	log.Printf("Report: %v", err)
	response.Error(w, http.StatusInternalServerError, "Data report gagal dimuat")
}

func (c *ReportController) Index(w http.ResponseWriter, r *http.Request) {
	switch strings.ToLower(strings.TrimSpace(r.PathValue("code"))) {
	case "rlist_acq_process":
		c.ListAcquisitionProcess(w, r)
	case "mapreport":
		c.MapAcquisitionProgress(w, r)
	case "acqprogress":
		c.AcquisitionProgress(w, r)
	case "detaillandacq":
		c.DetailLandAcquisition(w, r)
	case "rpt_all_progress":
		c.AllEstateProgress(w, r)
	case "plantmaprep":
		c.MapPlanting(w, r)
	case "reportdocland":
		c.CompensatedLandDocument(w, r)
	case "aestategrttreport":
		c.EstateProgressReport(w, r)
	default:
		response.Error(w, http.StatusNotFound, "Report tidak ditemukan")
	}
}

type reportListFunc func(context.Context, int64, reportModel.ReportFilter) (reportModel.ReportPage, error)

func (c *ReportController) index(w http.ResponseWriter, r *http.Request, list reportListFunc) {
	ctx, cancel, companyID, _, ok := c.params(w, r)
	if !ok {
		return
	}
	defer cancel()
	page, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.ParseInt(r.URL.Query().Get("page_size"), 10, 64)
	if err != nil || pageSize < 1 {
		pageSize = 25
	}
	if pageSize > 100 {
		pageSize = 100
	}
	status := int64(-1)
	if raw := strings.TrimSpace(r.URL.Query().Get("status")); raw != "" {
		status, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || status < 0 {
			response.Error(w, http.StatusBadRequest, "status tidak valid")
			return
		}
	}
	filter := reportModel.ReportFilter{
		Search: r.URL.Query().Get("search"), Estate: r.URL.Query().Get("estate"),
		Status: status, Page: page, PageSize: pageSize,
	}
	if len(filter.Search) > 200 || len(filter.Estate) > 200 || page > 100000 {
		response.Error(w, http.StatusBadRequest, "Filter report tidak valid")
		return
	}
	data, err := list(ctx, companyID, filter)
	if err != nil {
		reportFailure(w, err)
		return
	}
	response.Success(w, "Data report berhasil diambil", data)
}

func (c *ReportController) History(w http.ResponseWriter, r *http.Request) {
	ctx, cancel, companyID, _, ok := c.params(w, r)
	if !ok {
		return
	}
	defer cancel()
	landID, ok := c.land(w, ctx, companyID, r)
	if !ok {
		return
	}
	items, err := c.Model.History(ctx, landID)
	if err != nil {
		reportFailure(w, err)
		return
	}
	response.Success(w, "Riwayat akuisisi berhasil diambil", items)
}

func (c *ReportController) Map(w http.ResponseWriter, r *http.Request) {
	ctx, cancel, companyID, _, ok := c.params(w, r)
	if !ok {
		return
	}
	defer cancel()
	landID, ok := c.land(w, ctx, companyID, r)
	if !ok {
		return
	}
	data, err := c.Model.Map(ctx, companyID, landID)
	if err != nil {
		reportFailure(w, err)
		return
	}
	response.Success(w, "Koordinat lahan berhasil diambil", map[string]string{"map_data": data})
}

func (c *ReportController) SupportedDocuments(w http.ResponseWriter, r *http.Request) {
	ctx, cancel, companyID, _, ok := c.params(w, r)
	if !ok {
		return
	}
	defer cancel()
	landID, ok := c.land(w, ctx, companyID, r)
	if !ok {
		return
	}
	items, err := c.Documents.PaymentSupportedDocuments(ctx, landID)
	if err != nil {
		reportFailure(w, err)
		return
	}
	for i := range items {
		c.Files.SetSupportedDocumentAccess(landID, &items[i])
	}
	response.Success(w, "Dokumen lahan berhasil diambil", items)
}

func (c *ReportController) document(w http.ResponseWriter, r *http.Request) (context.Context, context.CancelFunc, int64, acquisition.PaymentSupportedDocument, bool) {
	ctx, cancel, companyID, _, ok := c.params(w, r)
	if !ok {
		return nil, nil, 0, acquisition.PaymentSupportedDocument{}, false
	}
	landID, ok := c.land(w, ctx, companyID, r)
	if !ok {
		cancel()
		return nil, nil, 0, acquisition.PaymentSupportedDocument{}, false
	}
	docID, err := strconv.ParseInt(r.PathValue("doc_id"), 10, 64)
	if err != nil || docID <= 0 {
		cancel()
		response.Error(w, 400, "ID dokumen tidak valid")
		return nil, nil, 0, acquisition.PaymentSupportedDocument{}, false
	}
	items, err := c.Documents.PaymentSupportedDocuments(ctx, landID)
	if err != nil {
		cancel()
		reportFailure(w, err)
		return nil, nil, 0, acquisition.PaymentSupportedDocument{}, false
	}
	for _, item := range items {
		if item.ID == docID {
			return ctx, cancel, landID, item, true
		}
	}
	cancel()
	response.Error(w, 404, "Dokumen lahan tidak ditemukan")
	return nil, nil, 0, acquisition.PaymentSupportedDocument{}, false
}

func (c *ReportController) SupportedDocumentFile(w http.ResponseWriter, r *http.Request) {
	_, cancel, landID, item, ok := c.document(w, r)
	if !ok {
		return
	}
	defer cancel()
	c.Files.ServeSupportedDocument(w, r, landID, item)
}

func (c *ReportController) SupportedDocumentLegacyStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel, _, item, ok := c.document(w, r)
	if !ok {
		return
	}
	defer cancel()
	response.Success(w, "Status preview lama berhasil diperiksa", map[string]*bool{
		"available": c.Files.SupportedDocumentLegacyStatus(ctx, item),
	})
}

func (c *ReportController) Memo(w http.ResponseWriter, r *http.Request) {
	ctx, cancel, companyID, _, ok := c.params(w, r)
	if !ok {
		return
	}
	defer cancel()
	landID, ok := c.land(w, ctx, companyID, r)
	if !ok {
		return
	}
	item, err := c.Documents.PaymentMemo(ctx, landID)
	if err != nil {
		reportFailure(w, err)
		return
	}
	c.Files.SetMemoAccess(landID, &item)
	response.Success(w, "Memo lahan berhasil diambil", item)
}

func (c *ReportController) MemoFile(w http.ResponseWriter, r *http.Request) {
	ctx, cancel, companyID, _, ok := c.params(w, r)
	if !ok {
		return
	}
	defer cancel()
	landID, ok := c.land(w, ctx, companyID, r)
	if !ok {
		return
	}
	item, err := c.Documents.PaymentMemo(ctx, landID)
	if err != nil {
		reportFailure(w, err)
		return
	}
	c.Files.ServeMemo(w, r, landID, item)
}

func (c *ReportController) MemoLegacyStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel, companyID, _, ok := c.params(w, r)
	if !ok {
		return
	}
	defer cancel()
	landID, ok := c.land(w, ctx, companyID, r)
	if !ok {
		return
	}
	item, err := c.Documents.PaymentMemo(ctx, landID)
	if err != nil {
		reportFailure(w, err)
		return
	}
	response.Success(w, "Status preview memo lama berhasil diperiksa", map[string]*bool{
		"available": c.Files.MemoLegacyStatus(ctx, item),
	})
}

func (c *ReportController) LocalHistoryFiles(w http.ResponseWriter, r *http.Request) {
	ctx, cancel, companyID, _, ok := c.params(w, r)
	if !ok {
		return
	}
	defer cancel()
	landID, ok := c.land(w, ctx, companyID, r)
	if !ok {
		return
	}
	response.Success(w, "File folder lahan berhasil diambil", c.Files.LocalFiles(landID))
}

func (c *ReportController) LocalHistoryFile(w http.ResponseWriter, r *http.Request) {
	ctx, cancel, companyID, _, ok := c.params(w, r)
	if !ok {
		return
	}
	defer cancel()
	landID, ok := c.land(w, ctx, companyID, r)
	if !ok {
		return
	}
	c.Files.ServeLocalFile(w, r, landID)
}
