package report

import (
	"net/http"

	"gis-app/backend/internal/response"
)

func (c *ReportController) EstateProgressReport(w http.ResponseWriter, r *http.Request) {
	c.index(w, r, c.Model.EstateProgressReport)
}

func (c *ReportController) EstateProgressReportSummary(w http.ResponseWriter, r *http.Request) {
	r.SetPathValue("code", "aestategrttreport")
	ctx, cancel, companyID, _, ok := c.params(w, r)
	if !ok {
		return
	}
	defer cancel()
	data, err := c.Model.EstateProgressReportSummary(ctx, companyID)
	if err != nil {
		reportFailure(w, err)
		return
	}
	response.Success(w, "Estate Progress Report berhasil diambil", data)
}
