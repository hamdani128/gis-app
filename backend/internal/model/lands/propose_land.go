package lands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const proposeLandTable = `public."Gis_Compensated_Lands"`

type ProposeLand struct {
	ID                      int64   `json:"id"`
	GisAquistionDivisionsID int64   `json:"Gis_Aquistion_Divisions_id"`
	GisLandOwnersID         int64   `json:"Gis_Land_Owners_id"`
	BlockPlan               string  `json:"block_plan"`
	Owner                   string  `json:"owner"`
	PlantDivision           string  `json:"plant_division"`
	PlantBlock              string  `json:"plant_block"`
	AlasHakJenis            string  `json:"alas_hak_jenis"`
	AlasHakNo               string  `json:"alas_hak_no"`
	LandSurface             string  `json:"land_surface"`
	SlopePersentage         string  `json:"slope_persentage"`
	SoilType                string  `json:"soil_type"`
	NorthernBoundary        string  `json:"northern_boundary"`
	EasternBoundary         string  `json:"eastern_boundary"`
	WesternBoundary         string  `json:"western_boundary"`
	SouthernBoundary        string  `json:"southern_boundary"`
	Village                 string  `json:"village"`
	DryLandWide             float64 `json:"dry_land_wide"`
	WetLandWide             float64 `json:"wet_land_wide"`
	LandWide                float64 `json:"land_wide"`
	DryLandPerhaPrice       int64   `json:"dry_land_perha_price"`
	WetLandPerhaPrice       int64   `json:"wet_land_perha_price"`
	AdminPrice              int64   `json:"admin_price"`
	PlantPrice              int64   `json:"plant_price"`
	VegetationNote          string  `json:"vegetation_note"`
	MeasurementDate         string  `json:"measurement_date"`
	MeasuredBy              string  `json:"measured_by"`
}

type ProposeLandInput = ProposeLand

type ProposeLandOption struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type ProposeLandCodeOption struct {
	Value string `json:"value"`
	Name  string `json:"name"`
}

type ProposeLandOptions struct {
	BlockPlans       []ProposeLandOption     `json:"block_plans"`
	Owners           []ProposeLandOption     `json:"owners"`
	OriginalStatuses []ProposeLandCodeOption `json:"original_statuses"`
	LandSurfaces     []ProposeLandCodeOption `json:"land_surfaces"`
	SoilTypes        []ProposeLandCodeOption `json:"soil_types"`
}

type ProposeLandModel struct{ DB *pgxpool.Pool }

func (m *ProposeLandModel) GetAll(ctx context.Context, companyID int64) ([]ProposeLand, error) {
	rows, err := m.DB.Query(ctx, `SELECT DISTINCT ON (c."id_Gis_Compensated_Lands") to_jsonb(c) || jsonb_build_object('id',c."id_Gis_Compensated_Lands",'block_plan',COALESCE(d.division_name,''),'owner',TRIM(CONCAT(COALESCE(o.name,''),CASE WHEN NULLIF(o.ktp_number,'') IS NULL THEN '' ELSE ' - ' || o.ktp_number END)))
		FROM `+proposeLandTable+` c
		JOIN public."Gis_Aquistion_Divisions" d ON d."id_Gis_Aqcuistion_Divisions"=c."Gis_Aquistion_Divisions_id"
		JOIN public."Gis_Potential_Lands" p ON p."id_Gis_Potential_Lands"=d."Gis_Potential_Lands_id"
		JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id"
		LEFT JOIN public."Gis_Land_Owners" o ON o."id_Gis_Land_Owners"=c."Gis_Land_Owners_id"
		WHERE c.status=0 AND ($1::bigint=0 OR lp."Gis_Companies_id"=$1)
		ORDER BY c."id_Gis_Compensated_Lands" DESC LIMIT 5000`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ProposeLand{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var item ProposeLand
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *ProposeLandModel) GetOptions(ctx context.Context, companyID int64) (ProposeLandOptions, error) {
	out := ProposeLandOptions{BlockPlans: []ProposeLandOption{}, Owners: []ProposeLandOption{}, OriginalStatuses: []ProposeLandCodeOption{}, LandSurfaces: []ProposeLandCodeOption{}, SoilTypes: []ProposeLandCodeOption{}}
	rows, err := m.DB.Query(ctx, `SELECT d."id_Gis_Aqcuistion_Divisions", COALESCE(d.division_name,'') FROM public."Gis_Aquistion_Divisions" d JOIN public."Gis_Potential_Lands" p ON p."id_Gis_Potential_Lands"=d."Gis_Potential_Lands_id" JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id" WHERE ($1::bigint=0 OR lp."Gis_Companies_id"=$1) ORDER BY 2,1`, companyID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item ProposeLandOption
		if err = rows.Scan(&item.ID, &item.Name); err != nil {
			rows.Close()
			return out, err
		}
		out.BlockPlans = append(out.BlockPlans, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	rows, err = m.DB.Query(ctx, `SELECT o."id_Gis_Land_Owners", TRIM(CONCAT(COALESCE(o.name,''),CASE WHEN NULLIF(o.ktp_number,'') IS NULL THEN '' ELSE ' - ' || o.ktp_number END)) FROM public."Gis_Land_Owners" o WHERE ($1::bigint=0 OR o."Gis_Companies_id"=$1) ORDER BY 2,1`, companyID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var item ProposeLandOption
		if err = rows.Scan(&item.ID, &item.Name); err != nil {
			rows.Close()
			return out, err
		}
		out.Owners = append(out.Owners, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	rows, err = m.DB.Query(ctx, `SELECT UPPER(BTRIM(domain)), COALESCE(low_value,''), COALESCE(meaning,'') FROM public.cg_ref_codes WHERE UPPER(BTRIM(domain)) IN ('ORIGIN_LAND_STATUS','LAND_SURFACE','SOIL_TYPE') ORDER BY domain, meaning, low_value`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var domain string
		var item ProposeLandCodeOption
		if err = rows.Scan(&domain, &item.Value, &item.Name); err != nil {
			return out, err
		}
		switch domain {
		case "ORIGIN_LAND_STATUS":
			out.OriginalStatuses = append(out.OriginalStatuses, item)
		case "LAND_SURFACE":
			out.LandSurfaces = append(out.LandSurfaces, item)
		case "SOIL_TYPE":
			out.SoilTypes = append(out.SoilTypes, item)
		}
	}
	return out, rows.Err()
}

func (m *ProposeLandModel) checkRelations(ctx context.Context, companyID int64, in ProposeLandInput) error {
	var divisionCompany, ownerCompany int64
	if err := m.DB.QueryRow(ctx, `SELECT lp."Gis_Companies_id" FROM public."Gis_Aquistion_Divisions" d JOIN public."Gis_Potential_Lands" p ON p."id_Gis_Potential_Lands"=d."Gis_Potential_Lands_id" JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id" WHERE d."id_Gis_Aqcuistion_Divisions"=$1`, in.GisAquistionDivisionsID).Scan(&divisionCompany); err != nil {
		return fmt.Errorf("Block Plan tidak ditemukan: %w", err)
	}
	if err := m.DB.QueryRow(ctx, `SELECT "Gis_Companies_id" FROM public."Gis_Land_Owners" WHERE "id_Gis_Land_Owners"=$1`, in.GisLandOwnersID).Scan(&ownerCompany); err != nil {
		return fmt.Errorf("Owner tidak ditemukan: %w", err)
	}
	if (companyID > 0 && divisionCompany != companyID) || divisionCompany != ownerCompany {
		return fmt.Errorf("Block Plan dan Owner harus berasal dari company aktif yang sama")
	}
	for _, ref := range []struct{ domain, value string }{{"ORIGIN_LAND_STATUS", in.AlasHakJenis}, {"LAND_SURFACE", in.LandSurface}, {"SOIL_TYPE", in.SoilType}} {
		if ref.value == "" {
			continue
		}
		var exists bool
		if err := m.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public.cg_ref_codes WHERE UPPER(BTRIM(domain))=$1 AND low_value=$2)`, ref.domain, ref.value).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("pilihan %s tidak valid", ref.domain)
		}
	}
	return nil
}

func (m *ProposeLandModel) Save(ctx context.Context, companyID int64, id *int64, in ProposeLandInput) (ProposeLand, error) {
	if err := m.checkRelations(ctx, companyID, in); err != nil {
		return ProposeLand{}, err
	}
	columns := []string{"Gis_Aquistion_Divisions_id", "Gis_Land_Owners_id", "plant_division", "plant_block", "alas_hak_jenis", "alas_hak_no", "land_surface", "slope_persentage", "soil_type", "northern_boundary", "eastern_boundary", "western_boundary", "southern_boundary", "village", "dry_land_wide", "wet_land_wide", "land_wide", "dry_land_perha_price", "wet_land_perha_price", "admin_price", "plant_price", "vegetation_note", "measurement_date", "measured_by"}
	values := []any{in.GisAquistionDivisionsID, in.GisLandOwnersID, strings.TrimSpace(in.PlantDivision), strings.TrimSpace(in.PlantBlock), in.AlasHakJenis, strings.TrimSpace(in.AlasHakNo), in.LandSurface, strings.TrimSpace(in.SlopePersentage), in.SoilType, strings.TrimSpace(in.NorthernBoundary), strings.TrimSpace(in.EasternBoundary), strings.TrimSpace(in.WesternBoundary), strings.TrimSpace(in.SouthernBoundary), strings.TrimSpace(in.Village), in.DryLandWide, in.WetLandWide, in.LandWide, in.DryLandPerhaPrice, in.WetLandPerhaPrice, in.AdminPrice, in.PlantPrice, strings.TrimSpace(in.VegetationNote), strings.TrimSpace(in.MeasurementDate), strings.TrimSpace(in.MeasuredBy)}
	if id == nil {
		tx, err := m.DB.Begin(ctx)
		if err != nil {
			return ProposeLand{}, err
		}
		defer tx.Rollback(ctx)
		// This legacy table has no sequence/default for its identifier.
		if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(73210481)`); err != nil {
			return ProposeLand{}, err
		}
		var next int64
		if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX("id_Gis_Compensated_Lands"),0)+1 FROM `+proposeLandTable).Scan(&next); err != nil {
			return ProposeLand{}, err
		}
		columns = append([]string{"id_Gis_Compensated_Lands", "status"}, columns...)
		values = append([]any{next, 0}, values...)
		quoted := make([]string, len(columns))
		placeholders := make([]string, len(columns))
		for i, col := range columns {
			quoted[i] = quoteIdent(col)
			placeholders[i] = fmt.Sprintf("$%d", i+1)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO `+proposeLandTable+` (`+strings.Join(quoted, ",")+`) VALUES (`+strings.Join(placeholders, ",")+`)`, values...); err != nil {
			return ProposeLand{}, err
		}
		if err = tx.Commit(ctx); err != nil {
			return ProposeLand{}, err
		}
		id = &next
	} else {
		sets := make([]string, len(columns))
		for i, col := range columns {
			sets[i] = fmt.Sprintf("%s=$%d", quoteIdent(col), i+1)
		}
		values = append(values, *id, companyID)
		tag, err := m.DB.Exec(ctx, `UPDATE `+proposeLandTable+` c SET `+strings.Join(sets, ",")+` WHERE c."id_Gis_Compensated_Lands"=$25 AND c.status=0 AND EXISTS(SELECT 1 FROM public."Gis_Aquistion_Divisions" d JOIN public."Gis_Potential_Lands" p ON p."id_Gis_Potential_Lands"=d."Gis_Potential_Lands_id" JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id" WHERE d."id_Gis_Aqcuistion_Divisions"=c."Gis_Aquistion_Divisions_id" AND ($26::bigint=0 OR lp."Gis_Companies_id"=$26))`, values...)
		if err != nil {
			return ProposeLand{}, err
		}
		if tag.RowsAffected() == 0 {
			return ProposeLand{}, pgx.ErrNoRows
		}
	}
	items, err := m.GetAll(ctx, companyID)
	if err != nil {
		return ProposeLand{}, err
	}
	for _, item := range items {
		if item.ID == *id {
			return item, nil
		}
	}
	return ProposeLand{}, pgx.ErrNoRows
}

func (m *ProposeLandModel) Delete(ctx context.Context, companyID, id int64) error {
	tag, err := m.DB.Exec(ctx, `DELETE FROM `+proposeLandTable+` c WHERE c."id_Gis_Compensated_Lands"=$1 AND c.status=0 AND EXISTS(SELECT 1 FROM public."Gis_Aquistion_Divisions" d JOIN public."Gis_Potential_Lands" p ON p."id_Gis_Potential_Lands"=d."Gis_Potential_Lands_id" JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id" WHERE d."id_Gis_Aqcuistion_Divisions"=c."Gis_Aquistion_Divisions_id" AND ($2::bigint=0 OR lp."Gis_Companies_id"=$2))`, id, companyID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
