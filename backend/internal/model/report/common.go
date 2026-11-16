package report

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	landModel "gis-app/backend/internal/model/lands"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportModel struct{ DB *pgxpool.Pool }

type ReportFilter struct {
	Search       string
	Estate       string
	Status       int64
	PlantingOnly bool
	DocsOnly     bool
	Page         int64
	PageSize     int64
}

type ReportGroup struct {
	Estate     string  `json:"estate"`
	Status     int64   `json:"status"`
	Lands      int64   `json:"lands"`
	Area       float64 `json:"area"`
	TotalPrice float64 `json:"total_price"`
}

type ReportPage struct {
	Rows       []map[string]any `json:"rows"`
	Total      int64            `json:"total"`
	Page       int64            `json:"page"`
	PageSize   int64            `json:"page_size"`
	Groups     []ReportGroup    `json:"groups"`
	TotalArea  float64          `json:"total_area"`
	TotalPrice float64          `json:"total_price"`
}

const reportPrice = `(COALESCE(l.wet_land_wide,0)*COALESCE(l.wet_land_perha_price,0)
	+COALESCE(l.dry_land_wide,0)*COALESCE(l.dry_land_perha_price,0)
	+COALESCE(l.admin_price,0)+COALESCE(l.plant_price,0))`

const reportFrom = ` FROM public."Gis_Compensated_Lands" l
	LEFT JOIN public."Gis_Aquistion_Divisions" d ON d."id_Gis_Aqcuistion_Divisions"=l."Gis_Aquistion_Divisions_id"
	LEFT JOIN public."Gis_Potential_Lands" p ON p."id_Gis_Potential_Lands"=d."Gis_Potential_Lands_id"
	LEFT JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id"
	LEFT JOIN public."Gis_Land_Owners" o ON o."id_Gis_Land_Owners"=l."Gis_Land_Owners_id"
	LEFT JOIN public."Gis_Companies" co ON co."id_Gis_Companies"=COALESCE(lp."Gis_Companies_id",o."Gis_Companies_id")`

const reportWhere = ` WHERE COALESCE(lp."Gis_Companies_id",o."Gis_Companies_id") IS NOT NULL
	AND ($1::bigint=0 OR COALESCE(lp."Gis_Companies_id",o."Gis_Companies_id")=$1)
	AND ($2::text='' OR l."id_Gis_Compensated_Lands"::text ILIKE '%'||$2||'%'
		OR COALESCE(o.name,'') ILIKE '%'||$2||'%'
		OR COALESCE(l.village,'') ILIKE '%'||$2||'%'
		OR COALESCE(p.estate_name,'') ILIKE '%'||$2||'%')
	AND ($3::bigint=-1 OR COALESCE(l.status,0)=$3)
	AND ($4::text='' OR COALESCE(p.estate_name,'')=$4)
	AND (NOT $5::boolean OR NULLIF(BTRIM(COALESCE(l.plant_division,'')), '') IS NOT NULL
		OR NULLIF(BTRIM(COALESCE(l.plant_block,'')), '') IS NOT NULL)
	AND (NOT $6::boolean OR EXISTS (SELECT 1 FROM public."Gis_Compensated_Land_Supported_Docs" doc
		WHERE doc."Gis_Compensated_Lands_id"=l."id_Gis_Compensated_Lands"))`

const reportLandJSON = `jsonb_build_object(
	'id',l."id_Gis_Compensated_Lands",
	'company_id',COALESCE(lp."Gis_Companies_id",o."Gis_Companies_id"),
	'company_name',COALESCE(NULLIF(BTRIM(co.name),''),NULLIF(BTRIM(co.com_name),''),''),
	'estate_name',COALESCE(p.estate_name,''),
	'owner',COALESCE(o.name,''),
	'village',COALESCE(l.village,''),
	'block_plan',COALESCE(d.division_name,''),
	'permit_number',COALESCE(lp.permit_number,''),
	'status',COALESCE(l.status,0),
	'land_wide',COALESCE(l.land_wide,0),
	'wet_land_wide',COALESCE(l.wet_land_wide,0),
	'dry_land_wide',COALESCE(l.dry_land_wide,0),
	'admin_price',COALESCE(l.admin_price,0),
	'plant_price',COALESCE(l.plant_price,0),
	'total_price',` + reportPrice + `,
	'plant_division',COALESCE(l.plant_division,''),
	'plant_block',COALESCE(l.plant_block,''),
	'document_count',(SELECT COUNT(*) FROM public."Gis_Compensated_Land_Supported_Docs" doc
		WHERE doc."Gis_Compensated_Lands_id"=l."id_Gis_Compensated_Lands"),
	'process_count',(SELECT COUNT(*) FROM public."Gis_Aquisition_Processes" ap
		WHERE ap."Gis_Compensated_Lands_id"=l."id_Gis_Compensated_Lands"),
	'last_process',COALESCE((SELECT seq.name FROM public."Gis_Aquisition_Processes" ap
		LEFT JOIN public."Gis_Aquisition_Process_Sequences" seq
		ON seq."id_Gis_Aquisition_Process_Sequences"=ap."Gis_Aquisition_Process_Sequences_id"
		WHERE ap."Gis_Compensated_Lands_id"=l."id_Gis_Compensated_Lands"
		ORDER BY ap."id_Gis_Aquisition_Processes" DESC LIMIT 1),'')
)`

func reportArgs(companyID int64, f ReportFilter) []any {
	return []any{companyID, strings.TrimSpace(f.Search), f.Status, strings.TrimSpace(f.Estate), f.PlantingOnly, f.DocsOnly}
}

func (m *ReportModel) list(ctx context.Context, companyID int64, f ReportFilter) (ReportPage, error) {
	result := ReportPage{Rows: []map[string]any{}, Groups: []ReportGroup{}, Page: f.Page, PageSize: f.PageSize}
	args := reportArgs(companyID, f)
	rows, err := m.DB.Query(ctx, `SELECT COALESCE(p.estate_name,''),COALESCE(l.status,0),COUNT(*),
		COALESCE(SUM(COALESCE(l.land_wide,0)),0),COALESCE(SUM(`+reportPrice+`),0)`+reportFrom+reportWhere+
		` GROUP BY COALESCE(p.estate_name,''),COALESCE(l.status,0)
		ORDER BY COALESCE(p.estate_name,''),COALESCE(l.status,0)`, args...)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var group ReportGroup
		if err = rows.Scan(&group.Estate, &group.Status, &group.Lands, &group.Area, &group.TotalPrice); err != nil {
			rows.Close()
			return result, err
		}
		result.Total += group.Lands
		result.TotalArea += group.Area
		result.TotalPrice += group.TotalPrice
		result.Groups = append(result.Groups, group)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	pageArgs := append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err = m.DB.Query(ctx, `SELECT `+reportLandJSON+reportFrom+reportWhere+
		` ORDER BY l."id_Gis_Compensated_Lands" DESC LIMIT $7 OFFSET $8`, pageArgs...)
	if err != nil {
		return result, err
	}
	result.Rows, err = scanJSONRows(rows)
	return result, err
}

func (m *ReportModel) Land(ctx context.Context, companyID, landID int64) (map[string]any, error) {
	var raw []byte
	err := m.DB.QueryRow(ctx, `SELECT `+reportLandJSON+reportFrom+
		` WHERE l."id_Gis_Compensated_Lands"=$1
		AND COALESCE(lp."Gis_Companies_id",o."Gis_Companies_id") IS NOT NULL
		AND ($2::bigint=0 OR COALESCE(lp."Gis_Companies_id",o."Gis_Companies_id")=$2)`, landID, companyID).Scan(&raw)
	if err != nil {
		return nil, err
	}
	var land map[string]any
	err = json.Unmarshal(raw, &land)
	return land, err
}

func (m *ReportModel) History(ctx context.Context, landID int64) ([]map[string]any, error) {
	rows, err := m.DB.Query(ctx, `SELECT jsonb_build_object(
		'id',ap."id_Gis_Aquisition_Processes",
		'name',COALESCE(seq.name,''),
		'sequence',seq.sequence,
		'approval',ap.approval,
		'request_number',COALESCE(ap.request_number,''),
		'request_date',ap.request_date,
		'date_created',ap.date_created,
		'note',COALESCE(ap.note,''),
		'username',COALESCE(ap.username,''))
		FROM public."Gis_Aquisition_Processes" ap
		LEFT JOIN public."Gis_Aquisition_Process_Sequences" seq
		ON seq."id_Gis_Aquisition_Process_Sequences"=ap."Gis_Aquisition_Process_Sequences_id"
		WHERE ap."Gis_Compensated_Lands_id"=$1
		ORDER BY seq.sequence NULLS LAST,ap."id_Gis_Aquisition_Processes" DESC`, landID)
	if err != nil {
		return nil, err
	}
	return scanJSONRows(rows)
}

func (m *ReportModel) Map(ctx context.Context, companyID, landID int64) (string, error) {
	land, err := m.Land(ctx, companyID, landID)
	if err != nil {
		return "", err
	}
	status, ok := land["status"].(float64)
	if !ok {
		return "", fmt.Errorf("status lahan %d tidak valid", landID)
	}
	return (&landModel.CompensatedLandModel{DB: m.DB, Status: int64(status)}).GetMap(ctx, companyID, landID)
}

func scanJSONRows(rows pgx.Rows) ([]map[string]any, error) {
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var item map[string]any
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
