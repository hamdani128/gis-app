package report

import (
	"context"
	"sort"
)

type EstateProgressMonth struct {
	EstateID int64   `json:"estate_id"`
	Year     int     `json:"year"`
	Month    int     `json:"month"`
	Area     float64 `json:"area"`
	Price    float64 `json:"price"`
	Average  float64 `json:"average"`
}

type EstateProgressSummary struct {
	AllEstateSummary
	Monthly []EstateProgressMonth `json:"monthly"`
}

func (m *ReportModel) EstateProgressReport(ctx context.Context, companyID int64, filter ReportFilter) (ReportPage, error) {
	return m.list(ctx, companyID, filter)
}

// EstateProgressReportSummary adds the legacy monthly GRTT price table to the
// estate status, permit and HCV summary shared with All Estate Progress.
func (m *ReportModel) EstateProgressReportSummary(ctx context.Context, companyID int64) (EstateProgressSummary, error) {
	result := EstateProgressSummary{Monthly: []EstateProgressMonth{}}
	base, err := m.AllEstateProgressSummary(ctx, companyID, nil, nil)
	if err != nil {
		return result, err
	}
	result.AllEstateSummary = base
	estateIDs := []int64{}
	estates := map[int64]*AllEstateRow{}
	for companyIndex := range result.Companies {
		for estateIndex := range result.Companies[companyIndex].Estates {
			estate := &result.Companies[companyIndex].Estates[estateIndex]
			estateIDs = append(estateIDs, estate.ID)
			estates[estate.ID] = estate
			estate.AveragePriceByYear = map[int]float64{}
		}
	}
	if len(estateIDs) == 0 {
		return result, nil
	}

	rows, err := m.DB.Query(ctx, `SELECT d."Gis_Potential_Lands_id"::bigint,
		EXTRACT(YEAR FROM l.grtt_date)::int, EXTRACT(MONTH FROM l.grtt_date)::int,
		COALESCE(SUM(COALESCE(l.dry_land_wide,0)+COALESCE(l.wet_land_wide,0)),0),
		COALESCE(SUM(`+reportPrice+`),0)
		FROM public."Gis_Compensated_Lands" l
		JOIN public."Gis_Aquistion_Divisions" d
			ON d."id_Gis_Aqcuistion_Divisions"=l."Gis_Aquistion_Divisions_id"
		WHERE d."Gis_Potential_Lands_id"::bigint=ANY($1::bigint[])
			AND l.status IN (6,7,8,10) AND l.grtt_date IS NOT NULL
		GROUP BY d."Gis_Potential_Lands_id", EXTRACT(YEAR FROM l.grtt_date), EXTRACT(MONTH FROM l.grtt_date)
		ORDER BY d."Gis_Potential_Lands_id", 2, 3`, estateIDs)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	years := map[int]bool{}
	for _, year := range result.Years {
		years[year] = true
	}
	yearly := map[int64]map[int]*allEstateAverage{}
	for rows.Next() {
		var item EstateProgressMonth
		var year, month int32
		if err = rows.Scan(&item.EstateID, &year, &month, &item.Area, &item.Price); err != nil {
			return result, err
		}
		item.Year, item.Month = int(year), int(month)
		if item.Area > 0 {
			item.Average = item.Price / item.Area
		}
		result.Monthly = append(result.Monthly, item)
		years[item.Year] = true
		if yearly[item.EstateID] == nil {
			yearly[item.EstateID] = map[int]*allEstateAverage{}
		}
		if yearly[item.EstateID][item.Year] == nil {
			yearly[item.EstateID][item.Year] = &allEstateAverage{}
		}
		yearly[item.EstateID][item.Year].wide += item.Area
		yearly[item.EstateID][item.Year].price += item.Price
	}
	if err = rows.Err(); err != nil {
		return result, err
	}
	result.Years = result.Years[:0]
	for year := range years {
		result.Years = append(result.Years, year)
	}
	sort.Ints(result.Years)
	for estateID, byYear := range yearly {
		for year, totals := range byYear {
			if totals.wide > 0 {
				estates[estateID].AveragePriceByYear[year] = totals.price / totals.wide
			}
		}
	}
	return result, nil
}
