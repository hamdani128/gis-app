package lands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CompensatedLandModel struct {
	DB     *pgxpool.Pool
	Status int64
}

type CompensatedLandInput struct {
	ProposeLand
	GritDate string `json:"grit_date"`
}

func (m *CompensatedLandModel) GetAll(ctx context.Context, companyID int64) ([]map[string]any, error) {
	return m.getRows(ctx, companyID, 0)
}

func (m *CompensatedLandModel) GetProcesses(ctx context.Context, companyID, landID int64) ([]map[string]any, error) {
	if _, err := m.permitID(ctx, companyID, landID); err != nil {
		return nil, err
	}
	rows, err := m.DB.Query(ctx, `SELECT jsonb_build_object(
		'id', process."id_Gis_Aquisition_Processes",
		'name', COALESCE(NULLIF(BTRIM(sequence.name),''), 'Process #' || COALESCE(process."Gis_Aquisition_Process_Sequences_id"::text, '?')),
		'sequence_id', process."Gis_Aquisition_Process_Sequences_id",
		'sequence', sequence.sequence,
		'approval', process.approval,
		'request_number', process.request_number,
		'request_date', process.request_date,
		'revision', process.revision,
		'note', process.note
	) FROM public."Gis_Aquisition_Processes" process
	LEFT JOIN public."Gis_Aquisition_Process_Sequences" sequence
		ON sequence."id_Gis_Aquisition_Process_Sequences"=process."Gis_Aquisition_Process_Sequences_id"
	WHERE process."Gis_Compensated_Lands_id"=$1
	ORDER BY sequence.sequence NULLS LAST, sequence."id_Gis_Aquisition_Process_Sequences" NULLS LAST,
		process.revision DESC NULLS LAST, process."id_Gis_Aquisition_Processes" DESC`, landID)
	if err != nil {
		return nil, err
	}
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

func (m *CompensatedLandModel) CheckAccess(ctx context.Context, companyID, landID int64) error {
	_, err := m.permitID(ctx, companyID, landID)
	return err
}

func (m *CompensatedLandModel) getRows(ctx context.Context, companyID, landID int64) ([]map[string]any, error) {
	rows, err := m.DB.Query(ctx, `SELECT DISTINCT ON (c."id_Gis_Compensated_Lands") to_jsonb(c) || jsonb_build_object(
		'id',c."id_Gis_Compensated_Lands", 'block_plan',COALESCE(d.division_name,''),
		'owner',COALESCE(o.name,''), 'owner_ktp',COALESCE(o.ktp_number,''),
		'potential_land',COALESCE(p.potential_name,''), 'estate_name',COALESCE(p.estate_name,''),
		'permit_number',COALESCE(lp.permit_number,''), 'company_name',COALESCE(NULLIF(BTRIM(co.name),''),NULLIF(BTRIM(co.com_name),''),''),
		'company_id',lp."Gis_Companies_id")
		FROM `+proposeLandTable+` c
		JOIN public."Gis_Aquistion_Divisions" d ON d."id_Gis_Aqcuistion_Divisions"=c."Gis_Aquistion_Divisions_id"
		JOIN public."Gis_Potential_Lands" p ON p."id_Gis_Potential_Lands"=d."Gis_Potential_Lands_id"
		JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id"
		JOIN public."Gis_Companies" co ON co."id_Gis_Companies"=lp."Gis_Companies_id"
		LEFT JOIN public."Gis_Land_Owners" o ON o."id_Gis_Land_Owners"=c."Gis_Land_Owners_id"
		WHERE c.status=$3 AND ($1::bigint=0 OR lp."Gis_Companies_id"=$1)
		AND ($2::bigint=0 OR c."id_Gis_Compensated_Lands"=$2)
		ORDER BY c."id_Gis_Compensated_Lands" DESC LIMIT 5000`, companyID, landID, m.Status)
	if err != nil {
		return nil, err
	}
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

func (m *CompensatedLandModel) GetOptions(ctx context.Context, companyID int64) (ProposeLandOptions, error) {
	return (&ProposeLandModel{DB: m.DB}).GetOptions(ctx, companyID)
}

func (m *CompensatedLandModel) Update(ctx context.Context, companyID, id int64, in CompensatedLandInput) (map[string]any, error) {
	if err := (&ProposeLandModel{DB: m.DB}).checkRelations(ctx, companyID, in.ProposeLand); err != nil {
		return nil, err
	}
	columns := []string{"Gis_Aquistion_Divisions_id", "Gis_Land_Owners_id", "alas_hak_jenis", "alas_hak_no", "land_surface", "slope_persentage", "soil_type", "northern_boundary", "eastern_boundary", "western_boundary", "southern_boundary", "village", "dry_land_wide", "wet_land_wide", "land_wide", "dry_land_perha_price", "wet_land_perha_price", "admin_price", "plant_price", "vegetation_note", "measurement_date", "measured_by"}
	values := []any{in.GisAquistionDivisionsID, in.GisLandOwnersID, in.AlasHakJenis, strings.TrimSpace(in.AlasHakNo), in.LandSurface, strings.TrimSpace(in.SlopePersentage), in.SoilType, strings.TrimSpace(in.NorthernBoundary), strings.TrimSpace(in.EasternBoundary), strings.TrimSpace(in.WesternBoundary), strings.TrimSpace(in.SouthernBoundary), strings.TrimSpace(in.Village), in.DryLandWide, in.WetLandWide, in.LandWide, in.DryLandPerhaPrice, in.WetLandPerhaPrice, in.AdminPrice, in.PlantPrice, strings.TrimSpace(in.VegetationNote), strings.TrimSpace(in.MeasurementDate), strings.TrimSpace(in.MeasuredBy)}
	var hasGrit bool
	if err := m.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='Gis_Compensated_Lands' AND column_name='grit_date')`).Scan(&hasGrit); err != nil {
		return nil, err
	}
	if hasGrit {
		columns = append(columns, "grit_date")
		if strings.TrimSpace(in.GritDate) == "" {
			values = append(values, nil)
		} else {
			values = append(values, strings.TrimSpace(in.GritDate))
		}
	}
	sets := make([]string, len(columns))
	for i, column := range columns {
		sets[i] = fmt.Sprintf("%s=$%d", quoteIdent(column), i+1)
	}
	values = append(values, id, companyID)
	tag, err := m.DB.Exec(ctx, `UPDATE `+proposeLandTable+` c SET `+strings.Join(sets, ",")+` WHERE c."id_Gis_Compensated_Lands"=$`+fmt.Sprint(len(columns)+1)+` AND c.status=10 AND EXISTS (SELECT 1 FROM public."Gis_Aquistion_Divisions" d JOIN public."Gis_Potential_Lands" p ON p."id_Gis_Potential_Lands"=d."Gis_Potential_Lands_id" JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id" WHERE d."id_Gis_Aqcuistion_Divisions"=c."Gis_Aquistion_Divisions_id" AND ($`+fmt.Sprint(len(columns)+2)+`::bigint=0 OR lp."Gis_Companies_id"=$`+fmt.Sprint(len(columns)+2)+`))`, values...)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	items, err := m.getRows(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item["id"] == float64(id) {
			return item, nil
		}
	}
	return nil, pgx.ErrNoRows
}
