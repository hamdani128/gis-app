package mapping

import (
	"context"
	"errors"
	model "gis-app/backend/internal/model/mapping"
	"gis-app/backend/internal/response"
	"github.com/jackc/pgx/v5"
	"log"
	"net/http"
	"strconv"
	"time"
)

type CompanyLandSource interface {
	List(context.Context, int64, int64, int64) (model.Layer, error)
}
type CompanyLandController struct{ Model CompanyLandSource }

func (c *CompanyLandController) Index(w http.ResponseWriter, r *http.Request) {
	company, err := strconv.ParseInt(r.URL.Query().Get("company_id"), 10, 64)
	selected, e := strconv.ParseInt(r.URL.Query().Get("selected_land_id"), 10, 64)
	after := int64(0)
	var cursorErr error
	if value := r.URL.Query().Get("after"); value != "" {
		after, cursorErr = strconv.ParseInt(value, 10, 64)
	}
	if err != nil || e != nil || cursorErr != nil || company < 0 || selected <= 0 || after < 0 {
		response.Error(w, 400, "company_id, selected_land_id, atau after tidak valid")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	layer, err := c.Model.List(ctx, company, selected, after)
	if errors.Is(err, pgx.ErrNoRows) {
		response.Error(w, 404, "Lahan tidak ditemukan pada company aktif")
		return
	}
	if err != nil {
		log.Printf("Company land map: %v", err)
		response.Error(w, 500, "Layer lahan company gagal dimuat")
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	response.Success(w, "Layer lahan company", layer)
}
