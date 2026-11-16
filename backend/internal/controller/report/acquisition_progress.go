package report

import (
	"net/http"
	"strconv"
	"strings"

	"gis-app/backend/internal/response"
)

func (c *ReportController) AcquisitionProgress(w http.ResponseWriter, r *http.Request) {
	c.index(w, r, c.Model.AcquisitionProgress)
}

func (c *ReportController) AcquisitionProgressSummary(w http.ResponseWriter, r *http.Request) {
	ctx, cancel, companyID, code, ok := c.params(w, r)
	if !ok {
		return
	}
	defer cancel()
	if code != "acqprogress" {
		response.Error(w, http.StatusNotFound, "Report progress tidak ditemukan")
		return
	}
	divisionID := int64(0)
	if raw := strings.TrimSpace(r.URL.Query().Get("division_id")); raw != "" {
		var err error
		divisionID, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || divisionID < 0 {
			response.Error(w, http.StatusBadRequest, "division_id tidak valid")
			return
		}
	}
	data, err := c.Model.AcquisitionProgressSummary(ctx, companyID, divisionID)
	if err != nil {
		reportFailure(w, err)
		return
	}
	response.Success(w, "Land Acquisition Progress berhasil diambil", data)
}
