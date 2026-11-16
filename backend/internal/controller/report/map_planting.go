package report

import (
	"net/http"
	"strconv"
	"strings"

	"gis-app/backend/internal/response"
)

func (c *ReportController) MapPlanting(w http.ResponseWriter, r *http.Request) {
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
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	if len(search) > 200 || page > 100000 {
		response.Error(w, http.StatusBadRequest, "Filter report tidak valid")
		return
	}
	data, err := c.Model.MapPlanting(ctx, companyID, search, page, pageSize)
	if err != nil {
		reportFailure(w, err)
		return
	}
	response.Success(w, "Map Planting berhasil diambil", data)
}

func (c *ReportController) PlantingMap(w http.ResponseWriter, r *http.Request) {
	r.SetPathValue("code", "plantmaprep")
	ctx, cancel, companyID, _, ok := c.params(w, r)
	if !ok {
		return
	}
	defer cancel()
	potentialID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || potentialID <= 0 {
		response.Error(w, http.StatusBadRequest, "ID potential land tidak valid")
		return
	}
	data, err := c.Model.PlantingMap(ctx, companyID, potentialID)
	if err != nil {
		reportFailure(w, err)
		return
	}
	response.Success(w, "Peta planting berhasil diambil", data)
}
