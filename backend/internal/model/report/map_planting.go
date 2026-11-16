package report

import (
	"context"
	"strings"
)

type PlantingEstate struct {
	ID            int64  `json:"id"`
	PotentialName string `json:"potential_name"`
	EstateName    string `json:"estate_name"`
	EstateAddress string `json:"estate_address"`
	EstateManager string `json:"estate_manager"`
	PermitNumber  string `json:"permit_number"`
	Note          string `json:"note"`
	CompanyName   string `json:"company_name"`
}

type PlantingEstatePage struct {
	Rows     []PlantingEstate `json:"rows"`
	Total    int64            `json:"total"`
	Page     int64            `json:"page"`
	PageSize int64            `json:"page_size"`
}

type PlantingMapLayer struct {
	Kind    string  `json:"kind"`
	ID      int64   `json:"id"`
	Name    string  `json:"name"`
	MapData string  `json:"map_data"`
	Area    float64 `json:"area"`
}

type PlantingMapReport struct {
	EstateID      int64              `json:"estate_id"`
	CompanyName   string             `json:"company_name"`
	PermitNumber  string             `json:"permit_number"`
	PotentialName string             `json:"potential_name"`
	EstateName    string             `json:"estate_name"`
	Layers        []PlantingMapLayer `json:"layers"`
}

const plantingEstateFrom = ` FROM public."Gis_Potential_Lands" p
	JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id"
	JOIN public."Gis_Companies" co ON co."id_Gis_Companies"=lp."Gis_Companies_id"
	LEFT JOIN public."Gis_Employees" e ON e."id_Gis_Employees"=p."Gis_Employees_id"`

const plantingEstateWhere = ` WHERE ($1::bigint=0 OR lp."Gis_Companies_id"=$1)
	AND ($2::text='' OR COALESCE(p.potential_name,'') ILIKE '%'||$2||'%'
	OR COALESCE(p.estate_name,'') ILIKE '%'||$2||'%'
	OR COALESCE(p.estate_address,'') ILIKE '%'||$2||'%'
	OR COALESCE(e.name,'') ILIKE '%'||$2||'%'
	OR COALESCE(lp.permit_number,'') ILIKE '%'||$2||'%'
	OR COALESCE(p.note,'') ILIKE '%'||$2||'%')`

func (m *ReportModel) MapPlanting(ctx context.Context, companyID int64, search string, page, pageSize int64) (PlantingEstatePage, error) {
	result := PlantingEstatePage{Rows: []PlantingEstate{}, Page: page, PageSize: pageSize}
	search = strings.TrimSpace(search)
	if err := m.DB.QueryRow(ctx, `SELECT COUNT(*)`+plantingEstateFrom+plantingEstateWhere, companyID, search).Scan(&result.Total); err != nil {
		return result, err
	}
	rows, err := m.DB.Query(ctx, `SELECT p."id_Gis_Potential_Lands",COALESCE(p.potential_name,''),
		COALESCE(p.estate_name,''),COALESCE(p.estate_address,''),COALESCE(e.name,''),
		COALESCE(lp.permit_number,''),COALESCE(p.note,''),
		COALESCE(NULLIF(BTRIM(co.name),''),NULLIF(BTRIM(co.com_name),''),'')`+
		plantingEstateFrom+plantingEstateWhere+
		` ORDER BY p."id_Gis_Potential_Lands" DESC LIMIT $3 OFFSET $4`, companyID, search, pageSize, (page-1)*pageSize)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var item PlantingEstate
		if err = rows.Scan(&item.ID, &item.PotentialName, &item.EstateName, &item.EstateAddress, &item.EstateManager, &item.PermitNumber, &item.Note, &item.CompanyName); err != nil {
			return result, err
		}
		result.Rows = append(result.Rows, item)
	}
	return result, rows.Err()
}

func (m *ReportModel) PlantingMap(ctx context.Context, companyID, potentialID int64) (PlantingMapReport, error) {
	result := PlantingMapReport{Layers: []PlantingMapLayer{}}
	var permitID int64
	err := m.DB.QueryRow(ctx, `SELECT p."id_Gis_Potential_Lands",COALESCE(p.potential_name,''),
		COALESCE(p.estate_name,''),COALESCE(lp.permit_number,''),
		COALESCE(NULLIF(BTRIM(co.name),''),NULLIF(BTRIM(co.com_name),''),''),lp."id_Land_Permits"
		FROM public."Gis_Potential_Lands" p
		JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id"
		JOIN public."Gis_Companies" co ON co."id_Gis_Companies"=lp."Gis_Companies_id"
		WHERE p."id_Gis_Potential_Lands"=$1 AND ($2::bigint=0 OR lp."Gis_Companies_id"=$2)`, potentialID, companyID).
		Scan(&result.EstateID, &result.PotentialName, &result.EstateName, &result.PermitNumber, &result.CompanyName, &permitID)
	if err != nil {
		return result, err
	}
	queries := []struct {
		kind, sql string
		id        int64
	}{
		{"permit", `SELECT s."id_Gis_Data_Spatial",COALESCE(lp.permit_number,''),s.mapdata
			FROM public."Gis_Data_Spatials" s JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=s."Gis_Land_Permits_id"
			WHERE lp."id_Land_Permits"=$1 AND NULLIF(BTRIM(s.mapdata),'') IS NOT NULL`, permitID},
		{"potential", `SELECT s."id_Gis_Data_Spatial",COALESCE(p.potential_name,''),s.mapdata
			FROM public."Gis_Data_Spatials" s JOIN public."Gis_Potential_Lands" p ON p."id_Gis_Potential_Lands"=s."Gis_Potential_Lands_id"
			WHERE p."id_Gis_Potential_Lands"=$1 AND NULLIF(BTRIM(s.mapdata),'') IS NOT NULL`, potentialID},
		{"hcv", `SELECT s."id_Gis_Data_Spatial",COALESCE(h.name,''),s.mapdata
			FROM public."Gis_Data_Spatials" s JOIN public."Gis_HCV_Lands" h ON h."id_Gis_HCV_Lands"::text=s."Gis_HCV_Lands_id"
			WHERE h."Gis_Land_Permits_id"=$1 AND NULLIF(BTRIM(s.mapdata),'') IS NOT NULL`, permitID},
		{"division", `SELECT d.idgis_plant_division,COALESCE(d.gis_plant_division_name,''),d.mapdata
			FROM public.gis_plant_division d WHERE d.gis_plant_division_id_potential=$1
			AND NULLIF(BTRIM(d.mapdata),'') IS NOT NULL`, potentialID},
		{"block", `SELECT b.idgis_plant_div_block,COALESCE(b.gis_plant_div_block_name,''),b.mapdata
			FROM public.gis_plant_div_block b JOIN public.gis_plant_division d
			ON d.idgis_plant_division=b.gis_plant_div_block_id_division
			WHERE d.gis_plant_division_id_potential=$1 AND NULLIF(BTRIM(b.mapdata),'') IS NOT NULL`, potentialID},
		{"acquisition", `SELECT u."id_Gis_Potential_Compensated_Union_Lands",COALESCE(seq.name,'Acquisition'),u.mapdata
			FROM public."Gis_Potential_Compensated_Union_Lands" u
			LEFT JOIN public."Gis_Aquisition_Process_Sequences" seq ON seq."id_Gis_Aquisition_Process_Sequences"=u."Gis_Aquisition_Process_Sequences_Id"
			WHERE u."Gis_Potential_Lands_Id"=$1 AND NULLIF(BTRIM(u.mapdata),'') IS NOT NULL`, potentialID},
	}
	for _, query := range queries {
		rows, err := m.DB.Query(ctx, query.sql, query.id)
		if err != nil {
			return result, err
		}
		for rows.Next() {
			layer := PlantingMapLayer{Kind: query.kind}
			if err = rows.Scan(&layer.ID, &layer.Name, &layer.MapData); err != nil {
				rows.Close()
				return result, err
			}
			if area, valid := SpatialAreaHectares(layer.MapData); valid {
				layer.Area = area * (110.93 * 110.93) / (110.57 * 110.57)
			}
			result.Layers = append(result.Layers, layer)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return result, err
		}
	}
	return result, nil
}
