package mapping

import (
	"context"
	"encoding/json"
	"fmt"
	"gis-app/backend/internal/model/lands"
	"github.com/jackc/pgx/v5/pgxpool"
)

const PageSize = 250

type CompanyLandModel struct{ DB *pgxpool.Pool }
type Layer struct {
	Type         string         `json:"type"`
	Features     []any          `json:"features"`
	CompanyID    int64          `json:"company_id"`
	NextCursor   int64          `json:"next_cursor"`
	Skipped      int            `json:"skipped"`
	SelectedLand map[string]any `json:"selected_land,omitempty"`
}

const companyLandFrom = ` FROM public."Gis_Compensated_Lands" l
LEFT JOIN public."Gis_Aquistion_Divisions" d ON d."id_Gis_Aqcuistion_Divisions"=l."Gis_Aquistion_Divisions_id"
LEFT JOIN public."Gis_Potential_Lands" p ON p."id_Gis_Potential_Lands"=d."Gis_Potential_Lands_id"
LEFT JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id"
LEFT JOIN public."Gis_Land_Owners" o ON o."id_Gis_Land_Owners"=l."Gis_Land_Owners_id"
LEFT JOIN public."Gis_Companies" co ON co."id_Gis_Companies"=COALESCE(lp."Gis_Companies_id",o."Gis_Companies_id") `

// Explicit popup fields keep unrelated account/identity and payment data out of map responses.
const landMapDetailsSQL = `jsonb_build_object(
'land_id',l."id_Gis_Compensated_Lands",'status',l.status,
'owner',COALESCE(o.name,''),'land_wide',l.land_wide,
'wet_land_wide',l.wet_land_wide,'dry_land_wide',l.dry_land_wide,
'village',COALESCE(l.village,''),'estate_name',COALESCE(p.estate_name,''),
'company_name',COALESCE(NULLIF(BTRIM(co.name),''),co.com_name,''),
'block_plan',COALESCE(d.division_name,''),'permit_number',COALESCE(lp.permit_number,''))`

// Every page rechecks the selected land's company; company=0 is only available
// through the route's authenticated global-company permission guard.
func (m *CompanyLandModel) List(ctx context.Context, companyID, selectedID, after int64) (Layer, error) {
	result := Layer{Type: "FeatureCollection", Features: []any{}}
	var selectedJSON []byte
	err := m.DB.QueryRow(ctx, `SELECT COALESCE(lp."Gis_Companies_id",o."Gis_Companies_id"),`+landMapDetailsSQL+companyLandFrom+`
WHERE l."id_Gis_Compensated_Lands"=$1 AND COALESCE(lp."Gis_Companies_id",o."Gis_Companies_id")>0
AND ($2::bigint=0 OR COALESCE(lp."Gis_Companies_id",o."Gis_Companies_id")=$2)`, selectedID, companyID).Scan(&result.CompanyID, &selectedJSON)
	if err != nil {
		return result, err
	}
	if err := json.Unmarshal(selectedJSON, &result.SelectedLand); err != nil {
		return result, err
	}
	fk, spatial, err := lands.CompensatedSpatialSource(ctx, &lands.CompensatedLandModel{DB: m.DB})
	if err != nil {
		return result, err
	}
	// Page by land, not spatial row: a land may contain multiple polygons.
	rows, err := m.DB.Query(ctx, `WITH page AS (
SELECT l."id_Gis_Compensated_Lands" AS id,l.status,`+landMapDetailsSQL+` AS details`+companyLandFrom+`
WHERE COALESCE(lp."Gis_Companies_id",o."Gis_Companies_id")=$1
AND l."id_Gis_Compensated_Lands">$2 AND l."id_Gis_Compensated_Lands"<>$3
ORDER BY l."id_Gis_Compensated_Lands" LIMIT 251)
SELECT page.id,COALESCE(page.status,0),page.details,`+spatial+` FROM page
LEFT JOIN public."Gis_Data_Spatials" s ON `+fk+`::text=page.id::text ORDER BY page.id`, result.CompanyID, after, selectedID)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	count, last := 0, int64(0)
	for rows.Next() {
		var id, status int64
		var raw *string
		var detailJSON []byte
		if err := rows.Scan(&id, &status, &detailJSON, &raw); err != nil {
			return result, err
		}
		if id != last {
			count++
			if count > PageSize {
				result.NextCursor = last
				break
			}
			last = id
		}
		if raw == nil || *raw == "" {
			continue
		}
		var details map[string]any
		if err := json.Unmarshal(detailJSON, &details); err != nil {
			return result, err
		}
		features, err := companyFeatures(*raw, id, status, details)
		if err != nil {
			result.Skipped++
			continue
		}
		result.Features = append(result.Features, features...)
	}
	return result, rows.Err()
}

func companyFeatures(raw string, id, status int64, details ...map[string]any) ([]any, error) {
	features, err := lands.ParseCompensatedSpatialFeatures(raw)
	if err != nil {
		return nil, err
	}
	for index, feature := range features {
		object, ok := feature.(map[string]any)
		if !ok || !validGeometry(object["geometry"]) {
			return nil, fmt.Errorf("invalid spatial geometry for land %d", id)
		}
		// Only publish geometry and known map properties from legacy files.
		properties := map[string]any{"land_id": id, "status": status}
		if len(details) > 0 {
			for _, key := range []string{"owner", "land_wide", "wet_land_wide", "dry_land_wide", "village", "estate_name", "company_name", "block_plan", "permit_number"} {
				properties[key] = details[0][key]
			}
		}
		features[index] = map[string]any{"type": "Feature", "geometry": object["geometry"], "properties": properties}
	}
	return features, nil
}

func validGeometry(value any) bool {
	geometry, ok := value.(map[string]any)
	if !ok {
		return false
	}
	if geometry["type"] == "GeometryCollection" {
		items, ok := geometry["geometries"].([]any)
		if !ok || len(items) == 0 {
			return false
		}
		for _, item := range items {
			if !validGeometry(item) {
				return false
			}
		}
		return true
	}
	depth := 0
	switch geometry["type"] {
	case "Point":
		depth = 0
	case "MultiPoint", "LineString":
		depth = 1
	case "MultiLineString", "Polygon":
		depth = 2
	case "MultiPolygon":
		depth = 3
	default:
		return false
	}
	return validCoordinates(geometry["coordinates"], depth)
}

func validCoordinates(value any, depth int) bool {
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return false
	}
	if depth == 0 {
		if len(items) < 2 {
			return false
		}
		lng, a := items[0].(float64)
		lat, b := items[1].(float64)
		return a && b && lng >= -180 && lng <= 180 && lat >= -90 && lat <= 90
	}
	for _, item := range items {
		if !validCoordinates(item, depth-1) {
			return false
		}
	}
	return true
}
