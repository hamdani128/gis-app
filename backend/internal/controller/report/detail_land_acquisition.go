package report

import "net/http"

func (c *ReportController) DetailLandAcquisition(w http.ResponseWriter, r *http.Request) {
	c.index(w, r, c.Model.DetailLandAcquisition)
}
