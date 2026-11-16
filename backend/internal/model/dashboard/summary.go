package dashboard

import (
	"context"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Model struct{ DB *pgxpool.Pool }

type TrendMonth struct {
	Month   int   `json:"month"`
	Created int64 `json:"created"`
	GRTT    int64 `json:"grtt"`
}

type StatusGroup struct {
	Key   string  `json:"key"`
	Count int64   `json:"count"`
	Area  float64 `json:"area"`
}

type Breakdown struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	LandCount int64   `json:"land_count"`
	Area      float64 `json:"area"`
}

type StageCount struct {
	Status int   `json:"status"`
	Count  int64 `json:"count"`
}

type Summary struct {
	GeneratedAt    string        `json:"generated_at"`
	Year           int           `json:"year"`
	Scope          string        `json:"scope"`
	CompanyCount   int64         `json:"company_count"`
	EstateCount    int64         `json:"estate_count"`
	OwnerCount     int64         `json:"owner_count"`
	LandCount      int64         `json:"land_count"`
	ActiveArea     float64       `json:"active_area"`
	EstimatedPrice float64       `json:"estimated_price"`
	CompletedCount int64         `json:"completed_count"`
	CompletedArea  float64       `json:"completed_area"`
	Trend          []TrendMonth  `json:"trend"`
	Statuses       []StatusGroup `json:"statuses"`
	Stages         []StageCount  `json:"stages"`
	Breakdown      []Breakdown   `json:"breakdown"`
	BreakdownKind  string        `json:"breakdown_kind"`
}

const landPrice = `(COALESCE(l.wet_land_wide,0)*COALESCE(l.wet_land_perha_price,0)
	+COALESCE(l.dry_land_wide,0)*COALESCE(l.dry_land_perha_price,0)
	+COALESCE(l.admin_price,0)+COALESCE(l.plant_price,0))`

var statusKeys = []string{"proses", "payment", "payment_owner", "selesai", "batal", "lainnya"}

func statusKey(status int64) string {
	switch {
	case status >= 0 && status <= 5:
		return "proses"
	case status == 6:
		return "payment"
	case status == 7:
		return "payment_owner"
	case status == 8 || status == 10:
		return "selesai"
	case status == 99:
		return "batal"
	default:
		return "lainnya"
	}
}

func (m *Model) Summary(ctx context.Context, companyID int64, year int) (Summary, error) {
	result := Summary{
		GeneratedAt: time.Now().Format(time.RFC3339), Year: year,
		Scope: "company", Trend: []TrendMonth{}, Statuses: []StatusGroup{},
		Stages: []StageCount{}, Breakdown: []Breakdown{}, BreakdownKind: "estate",
	}
	if companyID == 0 {
		result.Scope, result.BreakdownKind = "all", "company"
	}
	for month := 1; month <= 12; month++ {
		result.Trend = append(result.Trend, TrendMonth{Month: month})
	}
	for status := 0; status <= 7; status++ {
		result.Stages = append(result.Stages, StageCount{Status: status})
	}
	statusIndex := map[string]int{}
	for _, key := range statusKeys {
		statusIndex[key] = len(result.Statuses)
		result.Statuses = append(result.Statuses, StatusGroup{Key: key})
	}

	companies, err := m.DB.Query(ctx, `SELECT c."id_Gis_Companies",
		COALESCE(NULLIF(BTRIM(c.name),''),NULLIF(BTRIM(c.com_name),''),''),
		(SELECT COUNT(*) FROM public."Gis_Land_Owners" o WHERE o."Gis_Companies_id"=c."id_Gis_Companies"),
		(SELECT COUNT(*) FROM public."Gis_Potential_Lands" p
		 JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id"
		 WHERE lp."Gis_Companies_id"=c."id_Gis_Companies")
		FROM public."Gis_Companies" c
		WHERE c."id_Gis_Companies"<>0 AND ($1::bigint=0 OR c."id_Gis_Companies"=$1)
		ORDER BY c.name,c."id_Gis_Companies"`, companyID)
	if err != nil {
		return result, err
	}
	companyIDs := []int64{}
	companyNames := map[int64]string{}
	for companies.Next() {
		var id, owners, estates int64
		var name string
		if err = companies.Scan(&id, &name, &owners, &estates); err != nil {
			companies.Close()
			return result, err
		}
		companyIDs = append(companyIDs, id)
		companyNames[id] = name
		result.CompanyCount++
		result.OwnerCount += owners
		result.EstateCount += estates
	}
	err = companies.Err()
	companies.Close()
	if err != nil || len(companyIDs) == 0 {
		return result, err
	}
	if companyID > 0 {
		result.Scope = companyNames[companyID]
	} else {
		result.Scope = "AEP Plantations"
	}

	lands, err := m.DB.Query(ctx, `SELECT l."id_Gis_Compensated_Lands",
		COALESCE(lp."Gis_Companies_id",o."Gis_Companies_id")::bigint,
		COALESCE(p."id_Gis_Potential_Lands",0)::bigint,
		COALESCE(p.estate_name,''),COALESCE(l.status,0),COALESCE(l.land_wide,0),
		`+landPrice+`,l.date_created,l.grtt_date
		FROM public."Gis_Compensated_Lands" l
		LEFT JOIN public."Gis_Aquistion_Divisions" d ON d."id_Gis_Aqcuistion_Divisions"=l."Gis_Aquistion_Divisions_id"
		LEFT JOIN public."Gis_Potential_Lands" p ON p."id_Gis_Potential_Lands"=d."Gis_Potential_Lands_id"
		LEFT JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id"
		LEFT JOIN public."Gis_Land_Owners" o ON o."id_Gis_Land_Owners"=l."Gis_Land_Owners_id"
		WHERE COALESCE(lp."Gis_Companies_id",o."Gis_Companies_id")::bigint=ANY($1::bigint[])
		ORDER BY l."id_Gis_Compensated_Lands"`, companyIDs)
	if err != nil {
		return result, err
	}
	groups := map[int64]*Breakdown{}
	if companyID == 0 {
		for id, name := range companyNames {
			groups[id] = &Breakdown{ID: id, Name: name}
		}
	}
	for lands.Next() {
		var id, ownerCompanyID, estateID, status int64
		var estateName string
		var area, price float64
		var created, grtt *time.Time
		if err = lands.Scan(&id, &ownerCompanyID, &estateID, &estateName, &status, &area, &price, &created, &grtt); err != nil {
			lands.Close()
			return result, err
		}
		result.LandCount++
		segment := &result.Statuses[statusIndex[statusKey(status)]]
		segment.Count++
		segment.Area += area
		if status >= 0 && status <= 7 {
			result.Stages[status].Count++
		}
		if status != 99 {
			result.ActiveArea += area
			result.EstimatedPrice += price
		}
		if status == 8 || status == 10 {
			result.CompletedCount++
			result.CompletedArea += area
		}
		if created != nil && created.Year() == year {
			result.Trend[int(created.Month())-1].Created++
		}
		if grtt != nil && grtt.Year() == year && status != 99 {
			result.Trend[int(grtt.Month())-1].GRTT++
		}
		groupID, name := estateID, estateName
		if companyID == 0 {
			groupID, name = ownerCompanyID, companyNames[ownerCompanyID]
		} else if estateID == 0 {
			name = "Tanpa estate"
		}
		group := groups[groupID]
		if group == nil {
			group = &Breakdown{ID: groupID, Name: name}
			groups[groupID] = group
		}
		group.LandCount++
		if status != 99 {
			group.Area += area
		}
	}
	err = lands.Err()
	lands.Close()
	if err != nil {
		return result, err
	}
	for _, group := range groups {
		result.Breakdown = append(result.Breakdown, *group)
	}
	sort.Slice(result.Breakdown, func(i, j int) bool {
		if result.Breakdown[i].Area != result.Breakdown[j].Area {
			return result.Breakdown[i].Area > result.Breakdown[j].Area
		}
		return result.Breakdown[i].Name < result.Breakdown[j].Name
	})
	return result, nil
}
