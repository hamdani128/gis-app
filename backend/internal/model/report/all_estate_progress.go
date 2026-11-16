package report

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type AllEstateMetric struct {
	Area       float64 `json:"area"`
	Percent    float64 `json:"percent"`
	TotalPrice float64 `json:"total_price"`
	LandCount  int64   `json:"land_count"`
}

type AllEstateRow struct {
	ID                  int64           `json:"id"`
	Name                string          `json:"name"`
	PotentialArea       float64         `json:"potential_area"`
	HasSpatialArea      bool            `json:"has_spatial_area"`
	Compensated         AllEstateMetric `json:"compensated"`
	PaymentApproval     AllEstateMetric `json:"payment_approval"`
	OnProgress          AllEstateMetric `json:"on_progress"`
	AveragePriceByYear  map[int]float64 `json:"average_price_by_year"`
	MissingSpatialCount int64           `json:"missing_spatial_count"`
}

type AllEstateCompany struct {
	ID         int64          `json:"id"`
	Name       string         `json:"name"`
	PermitArea float64        `json:"permit_area"`
	HCVArea    float64        `json:"hcv_area"`
	Estates    []AllEstateRow `json:"estates"`
}

type AllEstateSummary struct {
	GeneratedAt string             `json:"generated_at"`
	StartDate   string             `json:"start_date"`
	EndDate     string             `json:"end_date"`
	Years       []int              `json:"years"`
	Companies   []AllEstateCompany `json:"companies"`
}

type allEstateLand struct {
	estateID  int64
	status    int64
	price     float64
	area      float64
	spatial   bool
	grttYear  int
	wide      float64
	landPrice float64
}

type allEstateAverage struct{ wide, price float64 }

func (m *ReportModel) AllEstateProgress(ctx context.Context, companyID int64, filter ReportFilter) (ReportPage, error) {
	return m.list(ctx, companyID, filter)
}

// AllEstateProgressSummary groups current land status by estate, matching the
// three categories and the GRTT-year price calculation in the legacy report.
func (m *ReportModel) AllEstateProgressSummary(ctx context.Context, companyID int64, start, end *time.Time) (AllEstateSummary, error) {
	result := AllEstateSummary{GeneratedAt: time.Now().Format(time.RFC3339), Years: []int{}, Companies: []AllEstateCompany{}}
	if start != nil && end != nil {
		result.StartDate = start.Format("2006-01-02")
		result.EndDate = end.Format("2006-01-02")
	}
	companies, err := m.DB.Query(ctx, `SELECT "id_Gis_Companies",COALESCE(NULLIF(BTRIM(name),''),NULLIF(BTRIM(com_name),''),'')
		FROM public."Gis_Companies" WHERE "id_Gis_Companies"<>0
		AND ($1::bigint=0 OR "id_Gis_Companies"=$1) ORDER BY name,"id_Gis_Companies"`, companyID)
	if err != nil {
		return result, err
	}
	companyIndex := map[int64]int{}
	companyIDs := []int64{}
	for companies.Next() {
		var company AllEstateCompany
		if err = companies.Scan(&company.ID, &company.Name); err != nil {
			companies.Close()
			return result, err
		}
		company.Estates = []AllEstateRow{}
		companyIndex[company.ID] = len(result.Companies)
		companyIDs = append(companyIDs, company.ID)
		result.Companies = append(result.Companies, company)
	}
	err = companies.Err()
	companies.Close()
	if err != nil || len(companyIDs) == 0 {
		return result, err
	}

	permitRows, err := m.DB.Query(ctx, `SELECT "id_Land_Permits","Gis_Companies_id"
		FROM public."Gis_Land_Permits" WHERE "Gis_Companies_id"::bigint=ANY($1::bigint[])`, companyIDs)
	if err != nil {
		return result, err
	}
	permitCompany := map[int64]int64{}
	permitIDs := []int64{}
	for permitRows.Next() {
		var permitID, ownerID int64
		if err = permitRows.Scan(&permitID, &ownerID); err != nil {
			permitRows.Close()
			return result, err
		}
		permitCompany[permitID] = ownerID
		permitIDs = append(permitIDs, permitID)
	}
	err = permitRows.Err()
	permitRows.Close()
	if err != nil {
		return result, err
	}

	estateRows, err := m.DB.Query(ctx, `SELECT p."id_Gis_Potential_Lands",COALESCE(p.estate_name,''),lp."Gis_Companies_id"
		FROM public."Gis_Potential_Lands" p
		JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id"
		WHERE lp."Gis_Companies_id"::bigint=ANY($1::bigint[])
		ORDER BY lp."Gis_Companies_id",p.estate_name,p."id_Gis_Potential_Lands"`, companyIDs)
	if err != nil {
		return result, err
	}
	estateIndex := map[int64][2]int{}
	estateIDs := []int64{}
	for estateRows.Next() {
		var estate AllEstateRow
		var ownerID int64
		if err = estateRows.Scan(&estate.ID, &estate.Name, &ownerID); err != nil {
			estateRows.Close()
			return result, err
		}
		estate.AveragePriceByYear = map[int]float64{}
		companyPosition := companyIndex[ownerID]
		estateIndex[estate.ID] = [2]int{companyPosition, len(result.Companies[companyPosition].Estates)}
		estateIDs = append(estateIDs, estate.ID)
		result.Companies[companyPosition].Estates = append(result.Companies[companyPosition].Estates, estate)
	}
	err = estateRows.Err()
	estateRows.Close()
	if err != nil {
		return result, err
	}

	spatialFKs, err := m.allEstateSpatialFKs(ctx)
	if err != nil {
		return result, err
	}
	if len(permitIDs) > 0 {
		areas, err := m.allEstateSpatialAreas(ctx, spatialFKs["permit"], permitIDs)
		if err != nil {
			return result, err
		}
		for id, area := range areas {
			result.Companies[companyIndex[permitCompany[id]]].PermitArea += area
		}
	}
	if len(estateIDs) > 0 {
		areas, err := m.allEstateSpatialAreas(ctx, spatialFKs["estate"], estateIDs)
		if err != nil {
			return result, err
		}
		for id, area := range areas {
			position := estateIndex[id]
			estate := &result.Companies[position[0]].Estates[position[1]]
			estate.PotentialArea += area
			estate.HasSpatialArea = true
		}
	}

	if len(permitIDs) > 0 {
		hcvIDColumn, err := m.allEstateColumn(ctx, "Gis_HCV_Lands", "id_Gis_HCV_Lands", "idGis_HCV_Lands", "id_hcv_lands", "id")
		if err != nil {
			return result, err
		}
		hcvPermitColumn, err := m.allEstateColumn(ctx, "Gis_HCV_Lands", "Gis_Land_Permits_id", "Gis_Land_Permits_id_Gis_Land_Permits", "id_Gis_Land_Permits", "land_permit_id")
		if err != nil {
			return result, err
		}
		hcvRows, err := m.DB.Query(ctx, `SELECT `+reportIdentifier(hcvIDColumn)+`::bigint,`+reportIdentifier(hcvPermitColumn)+`::bigint
			FROM public."Gis_HCV_Lands" WHERE `+reportIdentifier(hcvPermitColumn)+`::bigint=ANY($1::bigint[])`, permitIDs)
		if err != nil {
			return result, err
		}
		hcvCompany := map[int64]int64{}
		hcvIDs := []int64{}
		for hcvRows.Next() {
			var id, permitID int64
			if err = hcvRows.Scan(&id, &permitID); err != nil {
				hcvRows.Close()
				return result, err
			}
			hcvCompany[id] = permitCompany[permitID]
			hcvIDs = append(hcvIDs, id)
		}
		err = hcvRows.Err()
		hcvRows.Close()
		if err != nil {
			return result, err
		}
		if len(hcvIDs) > 0 {
			areas, err := m.allEstateSpatialAreas(ctx, spatialFKs["hcv"], hcvIDs)
			if err != nil {
				return result, err
			}
			for id, area := range areas {
				result.Companies[companyIndex[hcvCompany[id]]].HCVArea += area
			}
		}
	}
	if len(estateIDs) == 0 {
		return result, nil
	}

	landRows, err := m.DB.Query(ctx, `SELECT l."id_Gis_Compensated_Lands",d."Gis_Potential_Lands_id"::bigint,
		COALESCE(l.status,0),`+reportPrice+`,COALESCE(l.dry_land_wide,0),COALESCE(l.wet_land_wide,0),
		COALESCE(l.dry_land_perha_price,0),COALESCE(l.wet_land_perha_price,0),
		COALESCE(to_jsonb(l)->>'grtt_year',''),l.grtt_date,s.mapdata
		FROM public."Gis_Compensated_Lands" l
		JOIN public."Gis_Aquistion_Divisions" d ON d."id_Gis_Aqcuistion_Divisions"=l."Gis_Aquistion_Divisions_id"
		LEFT JOIN public."Gis_Data_Spatials" s ON s.`+reportIdentifier(spatialFKs["land"])+`::bigint=l."id_Gis_Compensated_Lands"
		WHERE d."Gis_Potential_Lands_id"::bigint=ANY($1::bigint[])
		AND ($2::date IS NULL OR (l.grtt_date >= $2::date AND l.grtt_date < $3::date + INTERVAL '1 day'))
		ORDER BY l."id_Gis_Compensated_Lands"`, estateIDs, start, end)
	if err != nil {
		return result, err
	}
	lands := map[int64]*allEstateLand{}
	for landRows.Next() {
		var id, estateID, status int64
		var price, dryWide, wetWide, dryPrice, wetPrice float64
		var yearText string
		var grttDate *time.Time
		var raw *string
		if err = landRows.Scan(&id, &estateID, &status, &price, &dryWide, &wetWide, &dryPrice, &wetPrice, &yearText, &grttDate, &raw); err != nil {
			landRows.Close()
			return result, err
		}
		land := lands[id]
		if land == nil {
			year, _ := strconv.Atoi(strings.TrimSpace(yearText))
			if year <= 0 && grttDate != nil {
				year = grttDate.Year()
			}
			land = &allEstateLand{
				estateID: estateID, status: status, price: price, grttYear: year,
				wide: dryWide + wetWide, landPrice: dryWide*dryPrice + wetWide*wetPrice,
			}
			lands[id] = land
		}
		if raw != nil {
			if area, valid := SpatialAreaHectares(*raw); valid {
				land.area += allEstateLegacyArea(area)
				land.spatial = true
			}
		}
	}
	err = landRows.Err()
	landRows.Close()
	if err != nil {
		return result, err
	}

	years := map[int]bool{}
	averages := map[int64]map[int]*allEstateAverage{}
	for _, land := range lands {
		position := estateIndex[land.estateID]
		estate := &result.Companies[position[0]].Estates[position[1]]
		if land.grttYear > 0 {
			years[land.grttYear] = true
			if land.status == 10 || land.status == 8 || land.status == 7 || land.status == 6 {
				if averages[land.estateID] == nil {
					averages[land.estateID] = map[int]*allEstateAverage{}
				}
				if averages[land.estateID][land.grttYear] == nil {
					averages[land.estateID][land.grttYear] = &allEstateAverage{}
				}
				average := averages[land.estateID][land.grttYear]
				average.wide += land.wide
				average.price += land.landPrice
			}
		}
		if land.status > 8 || land.status < 0 {
			if land.status != 10 {
				continue
			}
		}
		if !land.spatial {
			estate.MissingSpatialCount++
			continue
		}
		switch {
		case land.status == 10 || land.status == 8:
			addAllEstateMetric(&estate.Compensated, land)
		case land.status == 7:
			addAllEstateMetric(&estate.PaymentApproval, land)
		case land.status >= 0 && land.status <= 6:
			addAllEstateMetric(&estate.OnProgress, land)
		}
	}
	for year := range years {
		result.Years = append(result.Years, year)
	}
	sort.Ints(result.Years)
	for companyIndex := range result.Companies {
		for estateIndex := range result.Companies[companyIndex].Estates {
			estate := &result.Companies[companyIndex].Estates[estateIndex]
			finishAllEstateMetric(&estate.Compensated, estate.PotentialArea)
			finishAllEstateMetric(&estate.PaymentApproval, estate.PotentialArea)
			finishAllEstateMetric(&estate.OnProgress, estate.PotentialArea)
			for year, average := range averages[estate.ID] {
				if average.wide > 0 {
					estate.AveragePriceByYear[year] = average.price / average.wide
				}
			}
		}
	}
	return result, nil
}

func addAllEstateMetric(metric *AllEstateMetric, land *allEstateLand) {
	metric.Area += land.area
	metric.TotalPrice += land.price
	metric.LandCount++
}

func finishAllEstateMetric(metric *AllEstateMetric, potentialArea float64) {
	if potentialArea > 0 {
		metric.Percent = metric.Area / potentialArea * 100
	}
}

func allEstateLegacyArea(area float64) float64 {
	return area * (110.93 * 110.93) / (110.57 * 110.57)
}

func reportIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func (m *ReportModel) allEstateColumn(ctx context.Context, table string, aliases ...string) (string, error) {
	rows, err := m.DB.Query(ctx, `SELECT column_name FROM information_schema.columns
		WHERE table_schema='public' AND table_name=$1`, table)
	if err != nil {
		return "", err
	}
	columns := map[string]string{}
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			rows.Close()
			return "", err
		}
		columns[strings.ToLower(name)] = name
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return "", err
	}
	for _, alias := range aliases {
		if actual := columns[strings.ToLower(alias)]; actual != "" {
			return actual, nil
		}
	}
	return "", fmt.Errorf("kolom spasial %s tidak ditemukan pada %s", strings.Join(aliases, "/"), table)
}

func (m *ReportModel) allEstateSpatialFKs(ctx context.Context) (map[string]string, error) {
	aliases := map[string][]string{
		"permit": {"Gis_Land_Permits_id", "Gis_Land_Permits_idGis_Land_Permits", "Gis_Land_Permits_id_Gis_Land_Permits", "id_Gis_Land_Permits", "land_permit_id"},
		"estate": {"Gis_Potential_Lands_id", "Gis_Potential_Lands_idGis_Potential_Lands", "Gis_Potential_Lands_id_Gis_Potential_Lands", "id_Gis_Potential_Lands", "potential_land_id"},
		"hcv":    {"Gis_HCV_Lands_id", "Gis_HCV_Lands_idGis_HCV_Lands", "Gis_HCV_Lands_id_Gis_HCV_Lands", "id_Gis_HCV_Lands", "hcv_land_id"},
		"land":   {"Gis_Compensated_Lands_id", "Gis_Compensated_Lands_idGis_Compensated_Lands", "id_Gis_Compensated_Lands", "compensated_land_id"},
	}
	result := map[string]string{}
	for kind, names := range aliases {
		column, err := m.allEstateColumn(ctx, "Gis_Data_Spatials", names...)
		if err != nil {
			return nil, err
		}
		result[kind] = column
	}
	return result, nil
}

func (m *ReportModel) allEstateSpatialAreas(ctx context.Context, fk string, ids []int64) (map[int64]float64, error) {
	areas := map[int64]float64{}
	if len(ids) == 0 {
		return areas, nil
	}
	column := reportIdentifier(fk)
	rows, err := m.DB.Query(ctx, `SELECT `+column+`::bigint,mapdata FROM public."Gis_Data_Spatials"
		WHERE `+column+`::bigint=ANY($1::bigint[]) AND NULLIF(BTRIM(mapdata),'') IS NOT NULL`, ids)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id int64
		var raw string
		if err = rows.Scan(&id, &raw); err != nil {
			rows.Close()
			return nil, err
		}
		if area, valid := SpatialAreaHectares(raw); valid {
			areas[id] += allEstateLegacyArea(area)
		}
	}
	err = rows.Err()
	rows.Close()
	return areas, err
}
