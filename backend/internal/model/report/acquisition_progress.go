package report

import (
	"context"
	"time"
)

type AcquisitionProgressSequence struct {
	ID       int64  `json:"id"`
	Sequence int64  `json:"sequence"`
	Name     string `json:"name"`
}

type AcquisitionProgressMetric struct {
	Area       float64 `json:"area"`
	Percent    float64 `json:"percent"`
	TotalPrice float64 `json:"total_price"`
	LandCount  int64   `json:"land_count"`
}

type AcquisitionProgressStage struct {
	AcquisitionProgressSequence
	AcquisitionProgressMetric
}

type AcquisitionProgressDivision struct {
	ID                  int64                      `json:"id"`
	Name                string                     `json:"name"`
	Company             string                     `json:"company"`
	PermitNumber        string                     `json:"permit_number"`
	Area                float64                    `json:"area"`
	HasSpatialArea      bool                       `json:"has_spatial_area"`
	Stages              []AcquisitionProgressStage `json:"stages"`
	Compensated         AcquisitionProgressMetric  `json:"compensated"`
	Total               AcquisitionProgressMetric  `json:"total"`
	MissingSpatialCount int64                      `json:"missing_spatial_count"`
}

type AcquisitionProgressSummary struct {
	CompanyName string                        `json:"company_name"`
	Permits     []string                      `json:"permits"`
	GeneratedAt string                        `json:"generated_at"`
	Sequences   []AcquisitionProgressSequence `json:"sequences"`
	Divisions   []AcquisitionProgressDivision `json:"divisions"`
}

type acquisitionProgressLand struct {
	divisionID int64
	status     int64
	price      float64
	area       float64
	spatial    bool
}

func (m *ReportModel) AcquisitionProgress(ctx context.Context, companyID int64, filter ReportFilter) (ReportPage, error) {
	return m.list(ctx, companyID, filter)
}

// AcquisitionProgressSummary follows the legacy status-based report. Each land
// contributes its price once, even if Gis_Data_Spatials has multiple polygons.
func (m *ReportModel) AcquisitionProgressSummary(ctx context.Context, companyID, divisionID int64) (AcquisitionProgressSummary, error) {
	result := AcquisitionProgressSummary{
		Permits: []string{}, Sequences: []AcquisitionProgressSequence{},
		Divisions: []AcquisitionProgressDivision{}, GeneratedAt: time.Now().Format(time.RFC3339),
	}

	sequenceRows, err := m.DB.Query(ctx, `SELECT "id_Gis_Aquisition_Process_Sequences",COALESCE(sequence,0),COALESCE(name,'')
		FROM public."Gis_Aquisition_Process_Sequences"
		ORDER BY sequence,"id_Gis_Aquisition_Process_Sequences"`)
	if err != nil {
		return result, err
	}
	for sequenceRows.Next() {
		var item AcquisitionProgressSequence
		if err = sequenceRows.Scan(&item.ID, &item.Sequence, &item.Name); err != nil {
			sequenceRows.Close()
			return result, err
		}
		result.Sequences = append(result.Sequences, item)
	}
	err = sequenceRows.Err()
	sequenceRows.Close()
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
		AND ($2::bigint=0 OR d."id_Gis_Aqcuistion_Divisions"=$2)
		ORDER BY co.name,d.division_name,d."id_Gis_Aqcuistion_Divisions"`, companyID, divisionID)
	if err != nil {
		return result, err
	}
	divisionIndex := map[int64]int{}
	divisionIDs := []int64{}
	permits := map[string]bool{}
	for divisionRows.Next() {
		var item AcquisitionProgressDivision
		if err = divisionRows.Scan(&item.ID, &item.Name, &item.Company, &item.PermitNumber); err != nil {
			divisionRows.Close()
			return result, err
		}
		item.Stages = make([]AcquisitionProgressStage, len(result.Sequences))
		for i, sequence := range result.Sequences {
			item.Stages[i].AcquisitionProgressSequence = sequence
		}
		if result.CompanyName == "" {
			result.CompanyName = item.Company
		} else if result.CompanyName != item.Company {
			result.CompanyName = "Semua company"
		}
		if item.PermitNumber != "" && !permits[item.PermitNumber] {
			permits[item.PermitNumber] = true
			result.Permits = append(result.Permits, item.PermitNumber)
		}
		divisionIndex[item.ID] = len(result.Divisions)
		divisionIDs = append(divisionIDs, item.ID)
		result.Divisions = append(result.Divisions, item)
	}
	err = divisionRows.Err()
	divisionRows.Close()
	if err != nil || len(divisionIDs) == 0 {
		return result, err
	}

	spatialRows, err := m.DB.Query(ctx, `SELECT "Gis_Aqcuistion_Divisions_id",mapdata
		FROM public."Gis_Data_Spatials"
		WHERE "Gis_Aqcuistion_Divisions_id"::bigint=ANY($1::bigint[])
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
		if area, valid := SpatialAreaHectares(raw); valid {
			division := &result.Divisions[divisionIndex[id]]
			division.Area += area
			division.HasSpatialArea = true
		}
	}
	err = spatialRows.Err()
	spatialRows.Close()
	if err != nil {
		return result, err
	}

	landRows, err := m.DB.Query(ctx, `SELECT l."id_Gis_Compensated_Lands",l."Gis_Aquistion_Divisions_id",
		COALESCE(l.status,0),`+reportPrice+`,s.mapdata
		FROM public."Gis_Compensated_Lands" l
		LEFT JOIN public."Gis_Data_Spatials" s ON s."Gis_Compensated_Lands_id"=l."id_Gis_Compensated_Lands"
		WHERE l."Gis_Aquistion_Divisions_id"::bigint=ANY($1::bigint[])
		ORDER BY l."id_Gis_Compensated_Lands"`, divisionIDs)
	if err != nil {
		return result, err
	}
	lands := map[int64]*acquisitionProgressLand{}
	for landRows.Next() {
		var id, divisionID, status int64
		var price float64
		var raw *string
		if err = landRows.Scan(&id, &divisionID, &status, &price, &raw); err != nil {
			landRows.Close()
			return result, err
		}
		land := lands[id]
		if land == nil {
			land = &acquisitionProgressLand{divisionID: divisionID, status: status, price: price}
			lands[id] = land
		}
		if raw != nil {
			if area, valid := SpatialAreaHectares(*raw); valid {
				land.area += area
				land.spatial = true
			}
		}
	}
	err = landRows.Err()
	landRows.Close()
	if err != nil {
		return result, err
	}

	relevantStatuses := map[int64]bool{10: true}
	for _, sequence := range result.Sequences {
		relevantStatuses[sequence.Sequence] = true
	}
	for _, land := range lands {
		if !relevantStatuses[land.status] {
			continue
		}
		division := &result.Divisions[divisionIndex[land.divisionID]]
		if !land.spatial {
			division.MissingSpatialCount++
			continue
		}
		if land.status == 10 {
			addAcquisitionProgressMetric(&division.Compensated, land, false)
		}
		for i, sequence := range result.Sequences {
			if land.status == sequence.Sequence {
				addAcquisitionProgressMetric(&division.Stages[i].AcquisitionProgressMetric, land, true)
			}
		}
	}
	for i := range result.Divisions {
		division := &result.Divisions[i]
		for j := range division.Stages {
			metric := &division.Stages[j].AcquisitionProgressMetric
			finishAcquisitionProgressMetric(metric, division.Area)
			division.Total.Area += metric.Area
			division.Total.TotalPrice += metric.TotalPrice
			division.Total.LandCount += metric.LandCount
		}
		finishAcquisitionProgressMetric(&division.Compensated, division.Area)
		division.Total.Area += division.Compensated.Area
		division.Total.TotalPrice += division.Compensated.TotalPrice
		division.Total.LandCount += division.Compensated.LandCount
		finishAcquisitionProgressMetric(&division.Total, division.Area)
	}
	return result, nil
}

func addAcquisitionProgressMetric(metric *AcquisitionProgressMetric, land *acquisitionProgressLand, stage bool) {
	area := land.area
	if stage {
		area *= (110.93 * 110.93) / (110.57 * 110.57)
	}
	metric.Area += area
	metric.TotalPrice += land.price
	metric.LandCount++
}

func finishAcquisitionProgressMetric(metric *AcquisitionProgressMetric, divisionArea float64) {
	if divisionArea > 0 {
		metric.Percent = metric.Area / divisionArea * 100
	}
}
