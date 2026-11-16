package report

import "net/http"

func (c *ReportController) MapAcquisitionProgress(w http.ResponseWriter, r *http.Request) {
	c.index(w, r, c.Model.MapAcquisitionProgress)
}
