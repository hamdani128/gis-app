package report

import (
	"context"
	"strings"
)

const compensatedDocumentFrom = ` FROM public."Gis_Compensated_Lands" l
	JOIN public."Gis_Aquistion_Divisions" d ON d."id_Gis_Aqcuistion_Divisions"=l."Gis_Aquistion_Divisions_id"
	JOIN public."Gis_Potential_Lands" p ON p."id_Gis_Potential_Lands"=d."Gis_Potential_Lands_id"
	JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id"
	LEFT JOIN public."Gis_Land_Owners" o ON o."id_Gis_Land_Owners"=l."Gis_Land_Owners_id"
	LEFT JOIN (
		SELECT "Gis_Compensated_Lands_id" land_id,COUNT(*) count_doc
		FROM public."Gis_Compensated_Land_Supported_Docs"
		GROUP BY "Gis_Compensated_Lands_id"
	) docs ON docs.land_id=l."id_Gis_Compensated_Lands"`

const compensatedDocumentWhere = ` WHERE l.status=10
	AND ($1::bigint=0 OR lp."Gis_Companies_id"=$1)
	AND ($2::bigint=-1 OR COALESCE(docs.count_doc,0)=$2)
	AND ($3::text='' OR l."id_Gis_Compensated_Lands"::text ILIKE '%'||$3||'%'
		OR COALESCE(o.name,'') ILIKE '%'||$3||'%'
		OR COALESCE(l.village,'') ILIKE '%'||$3||'%'
		OR COALESCE(d.division_name,'') ILIKE '%'||$3||'%'
		OR COALESCE(p.estate_name,'') ILIKE '%'||$3||'%')
	AND ($4::bigint=0 OR EXTRACT(YEAR FROM l.date_created)=$4)
	AND ($5::bigint=0 OR EXTRACT(MONTH FROM l.date_created)=$5)`

func (m *ReportModel) CompensatedLandDocument(ctx context.Context, companyID, documentCount, year, month int64, search string, page, pageSize int64) (ReportPage, error) {
	result := ReportPage{Rows: []map[string]any{}, Groups: []ReportGroup{}, Page: page, PageSize: pageSize}
	args := []any{companyID, documentCount, strings.TrimSpace(search), year, month}
	err := m.DB.QueryRow(ctx, `SELECT COUNT(*),COALESCE(SUM(COALESCE(l.land_wide,0)),0),
		COALESCE(SUM(`+reportPrice+`),0)`+compensatedDocumentFrom+compensatedDocumentWhere, args...).
		Scan(&result.Total, &result.TotalArea, &result.TotalPrice)
	if err != nil {
		return result, err
	}
	rows, err := m.DB.Query(ctx, `SELECT jsonb_build_object(
		'id',l."id_Gis_Compensated_Lands",
		'division',COALESCE(d.division_name,''),
		'estate',COALESCE(p.estate_name,''),
		'owner',COALESCE(o.name,''),
		'status',l.status,
		'date_created',COALESCE(to_char(l.date_created,'YYYY-MM-DD'),''),
		'village',COALESCE(l.village,''),
		'land_wide',COALESCE(l.land_wide,0),
		'wet_land_wide',COALESCE(l.wet_land_wide,0),
		'wet_land_perha_price',COALESCE(l.wet_land_perha_price,0),
		'dry_land_wide',COALESCE(l.dry_land_wide,0),
		'dry_land_perha_price',COALESCE(l.dry_land_perha_price,0),
		'cost_per_ha',CASE WHEN COALESCE(l.land_wide,0)>0 THEN (`+reportPrice+`)/l.land_wide ELSE 0 END,
		'admin_price',COALESCE(l.admin_price,0),
		'plant_price',COALESCE(l.plant_price,0),
		'total_price',`+reportPrice+`,
		'document_count',COALESCE(docs.count_doc,0)
	)`+compensatedDocumentFrom+compensatedDocumentWhere+
		` ORDER BY l."id_Gis_Compensated_Lands" DESC LIMIT $6 OFFSET $7`,
		companyID, documentCount, strings.TrimSpace(search), year, month, pageSize, (page-1)*pageSize)
	if err != nil {
		return result, err
	}
	result.Rows, err = scanJSONRows(rows)
	return result, err
}
