package report

import (
	"net/http"
	"strconv"
	"strings"

	"gis-app/backend/internal/response"
)

func (c *ReportController) CompensatedLandDocument(w http.ResponseWriter, r *http.Request) {
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
	documentCount := int64(-1)
	if raw := strings.TrimSpace(r.URL.Query().Get("count_doc")); raw != "" {
		documentCount, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || documentCount < 0 {
			response.Error(w, http.StatusBadRequest, "Jumlah dokumen harus berupa angka nol atau lebih")
			return
		}
	}
	year := int64(0)
	if raw := strings.TrimSpace(r.URL.Query().Get("year")); raw != "" {
		year, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || year < 1900 || year > 2100 {
			response.Error(w, http.StatusBadRequest, "Tahun harus antara 1900 dan 2100")
			return
		}
	}
	month := int64(0)
	if raw := strings.TrimSpace(r.URL.Query().Get("month")); raw != "" {
		month, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || month < 1 || month > 12 {
			response.Error(w, http.StatusBadRequest, "Bulan harus antara 1 dan 12")
			return
		}
	}
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	if len(search) > 200 || page > 100000 {
		response.Error(w, http.StatusBadRequest, "Filter report tidak valid")
		return
	}
	data, err := c.Model.CompensatedLandDocument(ctx, companyID, documentCount, year, month, search, page, pageSize)
	if err != nil {
		reportFailure(w, err)
		return
	}
	response.Success(w, "Compensated Land Document berhasil diambil", data)
}
