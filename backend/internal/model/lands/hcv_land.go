package lands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	hcvLandTable   = `public."Gis_HCV_Lands"`
	hcvPermitTable = `public."Gis_Land_Permits"`
)

func quoteIdent(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }

type HCVLand struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	LandPermitID  int64  `json:"land_permit_id"`
	LandPermit    string `json:"land_permit"`
	SpatialTypeID *int64 `json:"spatial_type_id"`
	SpatialType   string `json:"spatial_type"`
	AreaLength    string `json:"area_length"`
	Note          string `json:"note"`
}

type HCVLandInput struct {
	Name          string  `json:"name"`
	LandPermitID  int64   `json:"land_permit_id"`
	SpatialTypeID *int64  `json:"spatial_type_id"`
	AreaLength    *string `json:"area_length"`
	Note          *string `json:"note"`
}

type HCVLandOption struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
type HCVLandOptions struct {
	LandPermits  []HCVLandOption `json:"land_permits"`
	SpatialTypes []HCVLandOption `json:"spatial_types"`
}
type HCVLandModel struct{ DB *pgxpool.Pool }

var hcvLandAliases = map[string][]string{
	"id":              {"id_Gis_HCV_Lands", "idGis_HCV_Lands", "id_hcv_lands", "id"},
	"name":            {"name", "land_name", "hcv_name"},
	"land_permit_id":  {"Gis_Land_Permits_id", "Gis_Land_Permits_id_Gis_Land_Permits", "id_Gis_Land_Permits", "land_permit_id", "gis_land_permits_id"},
	"spatial_type_id": {"cg_ref_codes_id", "spatial_type_id", "Spatial_Type_id", "spatialtype_id", "spatial_type"},
	"area_length":     {"area_length", "area", "length", "area_or_length"},
	"note":            {"note", "notes", "description", "remark"},
	"map_upload":      {"map_upload"},
}

func (m *HCVLandModel) columns(ctx context.Context, table string, aliases map[string][]string) (map[string]string, error) {
	rows, err := m.DB.Query(ctx, `SELECT column_name FROM information_schema.columns WHERE table_schema='public' AND table_name=$1`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	actual := map[string]string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		actual[strings.ToLower(name)] = name
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := map[string]string{}
	for field, candidates := range aliases {
		for _, candidate := range candidates {
			if name, ok := actual[strings.ToLower(candidate)]; ok {
				result[field] = name
				break
			}
		}
	}
	var primaryKey string
	_ = m.DB.QueryRow(ctx, `SELECT kcu.column_name FROM information_schema.table_constraints tc JOIN information_schema.key_column_usage kcu ON tc.constraint_name=kcu.constraint_name AND tc.table_schema=kcu.table_schema AND tc.table_name=kcu.table_name WHERE tc.table_schema='public' AND tc.table_name=$1 AND tc.constraint_type='PRIMARY KEY' ORDER BY kcu.ordinal_position LIMIT 1`, table).Scan(&primaryKey)
	if primaryKey != "" {
		result["id"] = primaryKey
	}
	return result, nil
}

var hcvPermitAliases = map[string][]string{"id": {"id_Gis_Land_Permits", "idGis_Land_Permits", "id_land_permits", "id"}, "company_id": {"Gis_Companies_id", "id_Gis_Companies", "company_id"}, "permit_number": {"permit_number", "permit_no", "permit_num", "nomor_izin"}}

func (m *HCVLandModel) tableColumns(ctx context.Context, table string, aliases map[string][]string) (map[string]string, error) {
	return m.columns(ctx, table, aliases)
}

func (m *HCVLandModel) columnTypes(ctx context.Context, table string) (map[string]string, error) {
	rows, err := m.DB.Query(ctx, `SELECT column_name, data_type, udt_name FROM information_schema.columns WHERE table_schema='public' AND table_name=$1`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	types := map[string]string{}
	for rows.Next() {
		var name, dataType, udt string
		if err := rows.Scan(&name, &dataType, &udt); err != nil {
			return nil, err
		}
		types[strings.ToLower(name)] = strings.ToLower(dataType + " " + udt)
	}
	return types, rows.Err()
}

func (m *HCVLandModel) schema(ctx context.Context) (map[string]string, map[string]string, error) {
	h, err := m.tableColumns(ctx, "Gis_HCV_Lands", hcvLandAliases)
	if err != nil {
		return nil, nil, err
	}
	p, err := m.tableColumns(ctx, "Gis_Land_Permits", hcvPermitAliases)
	if err != nil {
		return nil, nil, err
	}
	for _, key := range []string{"id", "land_permit_id"} {
		if h[key] == "" {
			return nil, nil, fmt.Errorf("Gis_HCV_Lands harus memiliki kolom %s", key)
		}
	}
	for _, key := range []string{"id", "company_id", "permit_number"} {
		if p[key] == "" {
			return nil, nil, fmt.Errorf("Gis_Land_Permits harus memiliki kolom %s", key)
		}
	}
	return h, p, nil
}

func (m *HCVLandModel) GetOptions(ctx context.Context, companyID int64) (HCVLandOptions, error) {
	out := HCVLandOptions{LandPermits: []HCVLandOption{}, SpatialTypes: []HCVLandOption{}}
	_, p, err := m.schema(ctx)
	if err != nil {
		return out, err
	}
	query := `SELECT p.` + quoteIdent(p["id"]) + `, COALESCE(NULLIF(BTRIM(p.` + quoteIdent(p["permit_number"]) + `::text),''),'') FROM ` + hcvPermitTable + ` p`
	args := []any{}
	if companyID > 0 {
		query += ` WHERE p.` + quoteIdent(p["company_id"]) + `=$1`
		args = append(args, companyID)
	}
	query += ` ORDER BY 2,1`
	rows, err := m.DB.Query(ctx, query, args...)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var o HCVLandOption
		if err := rows.Scan(&o.ID, &o.Name); err != nil {
			rows.Close()
			return out, err
		}
		out.LandPermits = append(out.LandPermits, o)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()
	rows, err = m.DB.Query(ctx, `SELECT id, COALESCE(NULLIF(BTRIM(meaning),''),NULLIF(BTRIM(low_value),''),'') FROM public.cg_ref_codes WHERE UPPER(BTRIM(domain)) LIKE '%SPATIAL%' ORDER BY 2,1`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var o HCVLandOption
		if err := rows.Scan(&o.ID, &o.Name); err != nil {
			return out, err
		}
		out.SpatialTypes = append(out.SpatialTypes, o)
	}
	return out, rows.Err()
}

func (m *HCVLandModel) GetAll(ctx context.Context, companyID int64) ([]HCVLand, error) {
	h, p, err := m.schema(ctx)
	if err != nil {
		return nil, err
	}
	selectText := func(field string) string {
		if h[field] == "" {
			return `NULL`
		}
		return `h.` + quoteIdent(h[field])
	}
	spatial := selectText("spatial_type_id")
	name := selectText("name")
	area := selectText("area_length")
	note := selectText("note")
	query := `SELECT h.` + quoteIdent(h["id"]) + `, COALESCE(` + name + `::text,''), h.` + quoteIdent(h["land_permit_id"]) + `, COALESCE(p.` + quoteIdent(p["permit_number"]) + `::text,''), ` + spatial + `, COALESCE(r.meaning,r.low_value,''), COALESCE(` + area + `::text,''), COALESCE(` + note + `::text,'') FROM ` + hcvLandTable + ` h JOIN ` + hcvPermitTable + ` p ON p.` + quoteIdent(p["id"]) + `=h.` + quoteIdent(h["land_permit_id"]) + ` LEFT JOIN public.cg_ref_codes r ON r.id::text=(` + spatial + `)::text WHERE ($1=0 OR p.` + quoteIdent(p["company_id"]) + `=$1) ORDER BY h.` + quoteIdent(h["id"]) + ` DESC LIMIT 5000`
	rows, err := m.DB.Query(ctx, query, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []HCVLand{}
	for rows.Next() {
		var item HCVLand
		var spatialID pgtype.Int8
		if err := rows.Scan(&item.ID, &item.Name, &item.LandPermitID, &item.LandPermit, &spatialID, &item.SpatialType, &item.AreaLength, &item.Note); err != nil {
			return nil, err
		}
		if spatialID.Valid {
			item.SpatialTypeID = &spatialID.Int64
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *HCVLandModel) Save(ctx context.Context, companyID int64, id *int64, in HCVLandInput) (HCVLand, error) {
	h, p, err := m.schema(ctx)
	if err != nil {
		return HCVLand{}, err
	}
	types, err := m.columnTypes(ctx, "Gis_HCV_Lands")
	if err != nil {
		return HCVLand{}, err
	}
	var permitCompanyText string
	err = m.DB.QueryRow(ctx, `SELECT `+quoteIdent(p["company_id"])+`::text FROM `+hcvPermitTable+` WHERE `+quoteIdent(p["id"])+`::text=$1`, strconv.FormatInt(in.LandPermitID, 10)).Scan(&permitCompanyText)
	if err != nil {
		return HCVLand{}, err
	}
	permitCompany, err := strconv.ParseInt(strings.TrimSpace(permitCompanyText), 10, 64)
	if err != nil {
		return HCVLand{}, err
	}
	if companyID > 0 && permitCompany != companyID {
		return HCVLand{}, fmt.Errorf("land permit tidak termasuk company aktif")
	}
	var spatialType any
	if in.SpatialTypeID != nil {
		spatialType = hcvIDArgument(*in.SpatialTypeID, types[strings.ToLower(h["spatial_type_id"])])
	}
	values := map[string]any{"name": strings.TrimSpace(in.Name), "land_permit_id": hcvIDArgument(in.LandPermitID, types[strings.ToLower(h["land_permit_id"])]), "spatial_type_id": spatialType, "area_length": in.AreaLength, "note": in.Note}
	fields := []string{}
	args := []any{}
	for _, field := range []string{"name", "land_permit_id", "spatial_type_id", "area_length", "note"} {
		if col := h[field]; col != "" {
			fields = append(fields, quoteIdent(col))
			args = append(args, values[field])
		}
	}
	if id == nil {
		var newID int64
		if err := m.DB.QueryRow(ctx, `SELECT COALESCE(MAX(`+quoteIdent(h["id"])+`::text::bigint),0)+1 FROM `+hcvLandTable).Scan(&newID); err != nil {
			return HCVLand{}, err
		}
		fields = append([]string{quoteIdent(h["id"])}, fields...)
		args = append([]any{hcvIDArgument(newID, types[strings.ToLower(h["id"])])}, args...)
		ph := make([]string, len(args))
		for i := range args {
			ph[i] = fmt.Sprintf("$%d", i+1)
		}
		_, err = m.DB.Exec(ctx, `INSERT INTO `+hcvLandTable+` (`+strings.Join(fields, ",")+`) VALUES (`+strings.Join(ph, ",")+`)`, args...)
		if err != nil {
			return HCVLand{}, err
		}
		id = &newID
	} else {
		sets := []string{}
		for i, f := range fields {
			sets = append(sets, f+fmt.Sprintf("=$%d", i+1))
		}
		args = append(args, strconv.FormatInt(*id, 10))
		scope := ``
		if companyID > 0 {
			args = append(args, strconv.FormatInt(companyID, 10))
			scope = ` AND EXISTS (SELECT 1 FROM ` + hcvPermitTable + ` p WHERE p.` + quoteIdent(p["id"]) + `=` + hcvLandTable + `.` + quoteIdent(h["land_permit_id"]) + ` AND p.` + quoteIdent(p["company_id"]) + fmt.Sprintf("::text=$%d", len(args)) + `)`
		}
		tag, e := m.DB.Exec(ctx, `UPDATE `+hcvLandTable+` SET `+strings.Join(sets, ",")+` WHERE `+quoteIdent(h["id"])+fmt.Sprintf("::text=$%d", len(fields)+1)+scope, args...)
		if e != nil {
			return HCVLand{}, e
		}
		if tag.RowsAffected() == 0 {
			return HCVLand{}, pgx.ErrNoRows
		}
	}
	items, err := m.GetAll(ctx, companyID)
	if err != nil {
		return HCVLand{}, err
	}
	for _, item := range items {
		if item.ID == *id {
			return item, nil
		}
	}
	return HCVLand{}, pgx.ErrNoRows
}

func (m *HCVLandModel) Delete(ctx context.Context, companyID, id int64) error {
	h, p, err := m.schema(ctx)
	if err != nil {
		return err
	}
	query := `DELETE FROM ` + hcvLandTable + ` WHERE ` + quoteIdent(h["id"]) + `::text=$1`
	args := []any{strconv.FormatInt(id, 10)}
	if companyID > 0 {
		query += ` AND EXISTS (SELECT 1 FROM ` + hcvPermitTable + ` p WHERE p.` + quoteIdent(p["id"]) + `=` + hcvLandTable + `.` + quoteIdent(h["land_permit_id"]) + ` AND p.` + quoteIdent(p["company_id"]) + `::text=$2)`
		args = append(args, strconv.FormatInt(companyID, 10))
	}
	tag, err := m.DB.Exec(ctx, query, args...)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}
