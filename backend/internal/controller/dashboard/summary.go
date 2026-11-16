package dashboard

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"gis-app/backend/internal/model/dashboard"
	"gis-app/backend/internal/response"
)

type Controller struct{ Model *dashboard.Model }

func (c *Controller) Summary(w http.ResponseWriter, r *http.Request) {
	companyID, err := strconv.ParseInt(r.URL.Query().Get("company_id"), 10, 64)
	if err != nil || companyID < 0 {
		response.Error(w, http.StatusBadRequest, "company_id tidak valid")
		return
	}
	year := time.Now().Year()
	if value := r.URL.Query().Get("year"); value != "" {
		year, err = strconv.Atoi(value)
		if err != nil || year < 2000 || year > 2100 {
			response.Error(w, http.StatusBadRequest, "year tidak valid")
			return
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	data, err := c.Model.Summary(ctx, companyID, year)
	if err != nil {
		log.Printf("Dashboard: %v", err)
		response.Error(w, http.StatusInternalServerError, "Data dashboard gagal dimuat")
		return
	}
	response.Success(w, "Dashboard berhasil diambil", data)
}
