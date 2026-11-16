package report

import (
	"net/http"
	"time"

	"gis-app/backend/internal/response"
)

func (c *ReportController) ListAcquisitionProcess(w http.ResponseWriter, r *http.Request) {
	c.index(w, r, c.Model.ListAcquisitionProcess)
}

func (c *ReportController) AcquisitionPeriod(w http.ResponseWriter, r *http.Request) {
	ctx, cancel, companyID, code, ok := c.params(w, r)
	if !ok {
		return
	}
	defer cancel()
	if code != "rlist_acq_process" {
		response.Error(w, http.StatusNotFound, "Report periode tidak ditemukan")
		return
	}
	now := time.Now()
	startDefault := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endDefault := startDefault.AddDate(0, 1, -1)
	startText := r.URL.Query().Get("start_date")
	endText := r.URL.Query().Get("end_date")
	if startText == "" {
		startText = startDefault.Format("2006-01-02")
	}
	if endText == "" {
		endText = endDefault.Format("2006-01-02")
	}
	start, startErr := time.Parse("2006-01-02", startText)
	end, endErr := time.Parse("2006-01-02", endText)
	if startErr != nil || endErr != nil || end.Before(start) || end.Sub(start) > 5*366*24*time.Hour {
		response.Error(w, http.StatusBadRequest, "Periode GRTT tidak valid; gunakan tanggal awal hingga akhir dalam rentang maksimal lima tahun")
		return
	}
	data, err := c.Model.AcquisitionPeriod(ctx, companyID, start, end)
	if err != nil {
		reportFailure(w, err)
		return
	}
	response.Success(w, "List Acquisition Progress berhasil diambil", data)
}
