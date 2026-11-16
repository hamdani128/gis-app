package report

import (
	"context"
	"math"
	"strconv"
	"strings"
	"time"
)

func (m *ReportModel) ListAcquisitionProcess(ctx context.Context, companyID int64, filter ReportFilter) (ReportPage, error) {
	return m.list(ctx, companyID, filter)
}

type ReportPeriodSequence struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type ReportPeriodLand struct {
	ID              int64            `json:"id"`
	Owner           string           `json:"owner"`
	Status          int64            `json:"status"`
	DateCreated     string           `json:"date_created"`
	GRTTDate        string           `json:"grtt_date"`
	StageDates      map[int64]string `json:"stage_dates"`
	Area            float64          `json:"area"`
	AreaSource      string           `json:"area_source"`
	TotalPrice      float64          `json:"total_price"`
	PricePerHectare float64          `json:"price_per_hectare"`
}

type ReportPeriodDivision struct {
	ID              int64              `json:"id"`
	Name            string             `json:"name"`
	Company         string             `json:"company"`
	PermitNumber    string             `json:"permit_number"`
	Area            float64            `json:"area"`
	HasSpatialArea  bool               `json:"has_spatial_area"`
	Lands           []ReportPeriodLand `json:"lands"`
	TotalArea       float64            `json:"total_area"`
	TotalPrice      float64            `json:"total_price"`
	PricePerHectare float64            `json:"price_per_hectare"`
	ProgressPercent float64            `json:"progress_percent"`
}

type ReportPeriod struct {
	StartDate string                 `json:"start_date"`
	EndDate   string                 `json:"end_date"`
	Sequences []ReportPeriodSequence `json:"sequences"`
	Divisions []ReportPeriodDivision `json:"divisions"`
	LandCount int                    `json:"land_count"`
	Truncated bool                   `json:"truncated"`
}

const legacyAreaFactor = 110.57 * 110.57 * 100

// SpatialAreaHectares follows the old report's square-degree conversion.
// A malformed WKT row is ignored; callers can use the recorded land area.
func SpatialAreaHectares(raw string) (float64, bool) {
	raw = strings.TrimSpace(raw)
	upper := strings.ToUpper(raw)
	ringDepth := 0
	switch {
	case strings.HasPrefix(upper, "MULTIPOLYGON"):
		ringDepth = 3
	case strings.HasPrefix(upper, "POLYGON"):
		ringDepth = 2
	default:
		return 0, false
	}
	depth, start, ringIndex := 0, -1, 0
	areaDegrees := 0.0
	for index, char := range raw {
		switch char {
		case '(':
			depth++
			if depth == ringDepth {
				start = index + 1
			}
			if ringDepth == 3 && depth == 2 {
				ringIndex = 0
			}
		case ')':
			if depth == ringDepth && start >= 0 {
				value, ok := wktRingArea(raw[start:index])
				if !ok {
					return 0, false
				}
				if ringIndex == 0 {
					areaDegrees += value
				} else {
					areaDegrees -= value
				}
				ringIndex++
				start = -1
			}
			depth--
			if depth < 0 {
				return 0, false
			}
		}
	}
	if depth != 0 || areaDegrees <= 0 || math.IsNaN(areaDegrees) || math.IsInf(areaDegrees, 0) {
		return 0, false
	}
	return areaDegrees * legacyAreaFactor, true
}

func wktRingArea(raw string) (float64, bool) {
	points := strings.Split(raw, ",")
	if len(points) < 3 {
		return 0, false
	}
	type pair struct{ x, y float64 }
	parsed := make([]pair, 0, len(points))
	for _, point := range points {
		fields := strings.Fields(point)
		if len(fields) < 2 {
			return 0, false
		}
		x, errX := strconv.ParseFloat(fields[0], 64)
		y, errY := strconv.ParseFloat(fields[1], 64)
		if errX != nil || errY != nil || math.Abs(x) > 180 || math.Abs(y) > 90 {
			return 0, false
		}
		parsed = append(parsed, pair{x, y})
	}
	area := 0.0
	for i := range parsed {
		next := parsed[(i+1)%len(parsed)]
		area += parsed[i].x*next.y - next.x*parsed[i].y
	}
	return math.Abs(area) / 2, true
}

func (m *ReportModel) AcquisitionPeriod(ctx context.Context, companyID int64, start, end time.Time) (ReportPeriod, error) {
	result := ReportPeriod{StartDate: start.Format("2006-01-02"), EndDate: end.Format("2006-01-02"), Sequences: []ReportPeriodSequence{}, Divisions: []ReportPeriodDivision{}}
	seqRows, err := m.DB.Query(ctx, `SELECT "id_Gis_Aquisition_Process_Sequences",COALESCE(name,'')
		FROM public."Gis_Aquisition_Process_Sequences" WHERE sequence>0 ORDER BY sequence,"id_Gis_Aquisition_Process_Sequences"`)
	if err != nil {
		return result, err
	}
	for seqRows.Next() {
		var item ReportPeriodSequence
		if err = seqRows.Scan(&item.ID, &item.Name); err != nil {
			seqRows.Close()
			return result, err
		}
		result.Sequences = append(result.Sequences, item)
	}
	err = seqRows.Err()
	seqRows.Close()
	if err != nil {
		return result, err
	}

	divisionRows, err := m.DB.Query(ctx, `SELECT d."id_Gis_Aqcuistion_Divisions",COALESCE(d.division_name,''),
		COALESCE(NULLIF(BTRIM(co.name),''),NULLIF(BTRIM(co.com_name),''),''),COALESCE(lp.permit_number,'')
		FROM public."Gis_Aquistion_Divisions" d
		JOIN public."Gis_Potential_Lands" p ON p."id_Gis_Potential_Lands"=d."Gis_Potential_Lands_id"
		JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id"
		JOIN public."Gis_Companies" co ON co."id_Gis_Companies"=lp."Gis_Companies_id"
		WHERE ($1::bigint=0 OR lp."Gis_Companies_id"=$1)
		ORDER BY co.name,d.division_name,d."id_Gis_Aqcuistion_Divisions"`, companyID)
	if err != nil {
		return result, err
	}
	divisionIndex := map[int64]int{}
	divisionIDs := []int64{}
	for divisionRows.Next() {
		var item ReportPeriodDivision
		if err = divisionRows.Scan(&item.ID, &item.Name, &item.Company, &item.PermitNumber); err != nil {
			divisionRows.Close()
			return result, err
		}
		item.Lands = []ReportPeriodLand{}
		divisionIndex[item.ID] = len(result.Divisions)
		divisionIDs = append(divisionIDs, item.ID)
		result.Divisions = append(result.Divisions, item)
	}
	err = divisionRows.Err()
	divisionRows.Close()
	if err != nil {
		return result, err
	}
	if len(divisionIDs) == 0 {
		return result, nil
	}

	spatialRows, err := m.DB.Query(ctx, `SELECT "Gis_Aqcuistion_Divisions_id",mapdata
		FROM public."Gis_Data_Spatials" WHERE "Gis_Aqcuistion_Divisions_id"::bigint=ANY($1::bigint[])
		AND NULLIF(BTRIM(mapdata),'') IS NOT NULL`, divisionIDs)
	if err != nil {
		return result, err
	}
	for spatialRows.Next() {
		var id int64
		var raw string
		if err = spatialRows.Scan(&id, &raw); err != nil {
			spatialRows.Close()
			return result, err
		}
		if value, ok := SpatialAreaHectares(raw); ok {
			index := divisionIndex[id]
			result.Divisions[index].Area += value
			result.Divisions[index].HasSpatialArea = true
		}
	}
	err = spatialRows.Err()
	spatialRows.Close()
	if err != nil {
		return result, err
	}

	landRows, err := m.DB.Query(ctx, `SELECT l."id_Gis_Compensated_Lands",l."Gis_Aquistion_Divisions_id",
		COALESCE(o.name,''),COALESCE(l.status,0),l.date_created,l.grtt_date,
		COALESCE(l.land_wide,0),`+reportPrice+`
		FROM public."Gis_Compensated_Lands" l
		JOIN public."Gis_Aquistion_Divisions" d ON d."id_Gis_Aqcuistion_Divisions"=l."Gis_Aquistion_Divisions_id"
		JOIN public."Gis_Potential_Lands" p ON p."id_Gis_Potential_Lands"=d."Gis_Potential_Lands_id"
		JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id"
		LEFT JOIN public."Gis_Land_Owners" o ON o."id_Gis_Land_Owners"=l."Gis_Land_Owners_id"
		WHERE ($1::bigint=0 OR lp."Gis_Companies_id"=$1)
		AND l.grtt_date >= $2::date AND l.grtt_date < $3::date + INTERVAL '1 day'
		ORDER BY l."Gis_Aquistion_Divisions_id",l.status DESC,l."id_Gis_Compensated_Lands"
		LIMIT 10001`, companyID, start, end)
	if err != nil {
		return result, err
	}
	landIDs := []int64{}
	landPositions := map[int64][2]int{}
	for landRows.Next() {
		var item ReportPeriodLand
		var divisionID int64
		var created, grtt *time.Time
		if err = landRows.Scan(&item.ID, &divisionID, &item.Owner, &item.Status, &created, &grtt, &item.Area, &item.TotalPrice); err != nil {
			landRows.Close()
			return result, err
		}
		if len(landIDs) == 10000 {
			result.Truncated = true
			break
		}
		if created != nil {
			item.DateCreated = created.Format("2006-01-02")
		}
		if grtt != nil {
			item.GRTTDate = grtt.Format("2006-01-02")
		}
		item.AreaSource = "recorded"
		item.StageDates = map[int64]string{}
		if len(result.Sequences) > 0 {
			item.StageDates[result.Sequences[0].ID] = item.DateCreated
		}
		index := divisionIndex[divisionID]
		position := len(result.Divisions[index].Lands)
		result.Divisions[index].Lands = append(result.Divisions[index].Lands, item)
		landPositions[item.ID] = [2]int{index, position}
		landIDs = append(landIDs, item.ID)
	}
	err = landRows.Err()
	landRows.Close()
	if err != nil {
		return result, err
	}
	result.LandCount = len(landIDs)
	if len(landIDs) == 0 {
		return result, nil
	}

	areaRows, err := m.DB.Query(ctx, `SELECT "Gis_Compensated_Lands_id",mapdata FROM public."Gis_Data_Spatials"
		WHERE "Gis_Compensated_Lands_id"::bigint=ANY($1::bigint[])
		AND NULLIF(BTRIM(mapdata),'') IS NOT NULL`, landIDs)
	if err != nil {
		return result, err
	}
	spatialAreas := map[int64]float64{}
	for areaRows.Next() {
		var id int64
		var raw string
		if err = areaRows.Scan(&id, &raw); err != nil {
			areaRows.Close()
			return result, err
		}
		if value, ok := SpatialAreaHectares(raw); ok {
			spatialAreas[id] += value
		}
	}
	err = areaRows.Err()
	areaRows.Close()
	if err != nil {
		return result, err
	}
	for id, value := range spatialAreas {
		position := landPositions[id]
		item := &result.Divisions[position[0]].Lands[position[1]]
		item.Area = value
		item.AreaSource = "spatial"
	}

	processRows, err := m.DB.Query(ctx, `SELECT "Gis_Compensated_Lands_id","Gis_Aquisition_Process_Sequences_id",
		MAX(date_created) FROM public."Gis_Aquisition_Processes"
		WHERE "Gis_Compensated_Lands_id"::bigint=ANY($1::bigint[]) AND approval=1
		GROUP BY "Gis_Compensated_Lands_id","Gis_Aquisition_Process_Sequences_id"`, landIDs)
	if err != nil {
		return result, err
	}
	for processRows.Next() {
		var landID, sequenceID int64
		var approved *time.Time
		if err = processRows.Scan(&landID, &sequenceID, &approved); err != nil {
			processRows.Close()
			return result, err
		}
		if approved != nil {
			position := landPositions[landID]
			result.Divisions[position[0]].Lands[position[1]].StageDates[sequenceID] = approved.Format("2006-01-02")
		}
	}
	err = processRows.Err()
	processRows.Close()
	if err != nil {
		return result, err
	}
	for i := range result.Divisions {
		division := &result.Divisions[i]
		for j := range division.Lands {
			item := &division.Lands[j]
			if len(result.Sequences) > 0 {
				last := result.Sequences[len(result.Sequences)-1].ID
				if item.StageDates[last] == "" {
					item.StageDates[last] = item.GRTTDate
				}
			}
			if item.Area > 0 {
				item.PricePerHectare = item.TotalPrice / item.Area
			}
			division.TotalArea += item.Area
			division.TotalPrice += item.TotalPrice
		}
		if division.TotalArea > 0 {
			division.PricePerHectare = division.TotalPrice / division.TotalArea
		}
		if division.HasSpatialArea && division.Area > 0 {
			division.ProgressPercent = division.TotalArea / division.Area * 100
		}
	}
	return result, nil
}
