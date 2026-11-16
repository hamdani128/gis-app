package report

import (
	"net/http"
	"time"

	"gis-app/backend/internal/response"
)

func (c *ReportController) AllEstateProgress(w http.ResponseWriter, r *http.Request) {
	c.index(w, r, c.Model.AllEstateProgress)
}

func (c *ReportController) AllEstateProgressSummary(w http.ResponseWriter, r *http.Request) {
	r.SetPathValue("code", "rpt_all_progress")
	ctx, cancel, companyID, code, ok := c.params(w, r)
	if !ok {
		return
	}
	defer cancel()
	if code != "rpt_all_progress" {
		response.Error(w, http.StatusNotFound, "Report estate tidak ditemukan")
		return
	}
	startText := r.URL.Query().Get("start_date")
	endText := r.URL.Query().Get("end_date")
	var start, end *time.Time
	if startText != "" || endText != "" {
		startValue, startErr := time.Parse("2006-01-02", startText)
		endValue, endErr := time.Parse("2006-01-02", endText)
		if startErr != nil || endErr != nil || endValue.Before(startValue) || endValue.Sub(startValue) > 5*366*24*time.Hour {
			response.Error(w, http.StatusBadRequest, "Periode GRTT tidak valid; isi tanggal awal dan akhir dalam rentang maksimal lima tahun")
			return
		}
		start, end = &startValue, &endValue
	}
	data, err := c.Model.AllEstateProgressSummary(ctx, companyID, start, end)
	if err != nil {
		reportFailure(w, err)
		return
	}
	response.Success(w, "All Estate Progress berhasil diambil", data)
}
