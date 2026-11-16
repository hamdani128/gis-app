package lands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	potentialLandTable   = `public."Gis_Potential_Lands"`
	potentialPermitTable = `public."Gis_Land_Permits"`
)

type PotentialLand struct {
	ID            int64  `json:"id"`
	PotentialName string `json:"potential_name"`
	EstateName    string `json:"estate_name"`
	EstateAddress string `json:"estate_address"`
	EstateManager string `json:"estate_manager"`
	LandPermitID  int64  `json:"land_permit_id"`
	LandPermit    string `json:"land_permit"`
	Note          string `json:"note"`
}

type PotentialLandInput struct {
	PotentialName string `json:"potential_name"`
	EstateName    string `json:"estate_name"`
	EstateAddress string `json:"estate_address"`
	EstateManager string `json:"estate_manager"`
	LandPermitID  int64  `json:"land_permit_id"`
	Note          string `json:"note"`
}

type PotentialLandOption struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
type PotentialLandOptions struct {
	LandPermits    []PotentialLandOption `json:"land_permits"`
	EstateManagers []PotentialLandOption `json:"estate_managers"`
}
type PotentialLandModel struct{ DB *pgxpool.Pool }

var potentialLandAliases = map[string][]string{
	"id":             {"id_Gis_Potential_Lands", "idGis_Potential_Lands", "id_potential_lands", "id"},
	"potential_name": {"potential_name", "name"},
	"estate_name":    {"estate_name"},
	"estate_address": {"estate_address", "address"},
	"estate_manager": {"estate_manager", "est_manager", "estate_manager_id", "Gis_Employees_id", "Gis_Employees_id_Gis_Employees"},
	"land_permit_id": {"Gis_Land_Permits_id", "Gis_Land_Permits_id_Gis_Land_Permits", "id_Gis_Land_Permits", "land_permit_id", "gis_land_permits_id"},
	"note":           {"note", "notes", "description", "remark"},
	"map_upload":     {"map_upload"},
}

func (m *PotentialLandModel) columns(ctx context.Context, table string, aliases map[string][]string) (map[string]string, error) {
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

var potentialPermitAliases = map[string][]string{"id": {"id_Gis_Land_Permits", "idGis_Land_Permits", "id_land_permits", "id"}, "company_id": {"Gis_Companies_id", "id_Gis_Companies", "company_id"}, "permit_number": {"permit_number", "permit_no", "permit_num", "nomor_izin"}}

func (m *PotentialLandModel) tableColumns(ctx context.Context, table string, aliases map[string][]string) (map[string]string, error) {
	return m.columns(ctx, table, aliases)
}

func (m *PotentialLandModel) columnTypes(ctx context.Context, table string) (map[string]string, error) {
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

func (m *PotentialLandModel) schema(ctx context.Context) (map[string]string, map[string]string, error) {
	h, err := m.tableColumns(ctx, "Gis_Potential_Lands", potentialLandAliases)
	if err != nil {
		return nil, nil, err
	}
	p, err := m.tableColumns(ctx, "Gis_Land_Permits", potentialPermitAliases)
	if err != nil {
		return nil, nil, err
	}
	for _, key := range []string{"id", "potential_name", "land_permit_id"} {
		if h[key] == "" {
			return nil, nil, fmt.Errorf("Gis_Potential_Lands harus memiliki kolom %s", key)
		}
	}
	for _, key := range []string{"id", "company_id", "permit_number"} {
		if p[key] == "" {
			return nil, nil, fmt.Errorf("Gis_Land_Permits harus memiliki kolom %s", key)
		}
	}
	return h, p, nil
}

func (m *PotentialLandModel) GetOptions(ctx context.Context, companyID int64) (PotentialLandOptions, error) {
	out := PotentialLandOptions{LandPermits: []PotentialLandOption{}, EstateManagers: []PotentialLandOption{}}
	_, p, err := m.schema(ctx)
	if err != nil {
		return out, err
	}
	query := `SELECT p.` + quoteIdent(p["id"]) + `::text, COALESCE(p.` + quoteIdent(p["permit_number"]) + `::text,'') FROM ` + potentialPermitTable + ` p`
	args := []any{}
	if companyID > 0 {
		query += ` WHERE p.` + quoteIdent(p["company_id"]) + `::text=$1`
		args = append(args, strconv.FormatInt(companyID, 10))
	}
	rows, err := m.DB.Query(ctx, query+` ORDER BY 2,1`, args...)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var id, name string
		if err = rows.Scan(&id, &name); err != nil {
			rows.Close()
			return out, err
		}
		number, e := strconv.ParseInt(id, 10, 64)
		if e == nil {
			out.LandPermits = append(out.LandPermits, PotentialLandOption{number, name})
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	query = `SELECT e."id_Gis_Employees"::text, COALESCE(e.name,'') FROM public."Gis_Employees" e`
	args = []any{}
	if companyID > 0 {
		query += ` WHERE e."Gis_Companies_id"::text=$1`
		args = append(args, strconv.FormatInt(companyID, 10))
	}
	rows, err = m.DB.Query(ctx, query+` ORDER BY 2,1`, args...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		if err = rows.Scan(&id, &name); err != nil {
			return out, err
		}
		number, e := strconv.ParseInt(id, 10, 64)
		if e == nil {
			out.EstateManagers = append(out.EstateManagers, PotentialLandOption{number, name})
		}
	}
	return out, rows.Err()
}

func (m *PotentialLandModel) GetAll(ctx context.Context, companyID int64) ([]PotentialLand, error) {
	h, p, err := m.schema(ctx)
	if err != nil {
		return nil, err
	}
	field := func(key string) string {
		if h[key] == "" {
			return `NULL`
		}
		return `l.` + quoteIdent(h[key]) + `::text`
	}
	manager := field("estate_manager")
	query := `SELECT l.` + quoteIdent(h["id"]) + `::text, COALESCE(` + field("potential_name") + `,''), COALESCE(` + field("estate_name") + `,''), COALESCE(` + field("estate_address") + `,''), COALESCE(e.name,` + manager + `,''), l.` + quoteIdent(h["land_permit_id"]) + `::text, COALESCE(p.` + quoteIdent(p["permit_number"]) + `::text,''), COALESCE(` + field("note") + `,'') FROM ` + potentialLandTable + ` l JOIN ` + potentialPermitTable + ` p ON p.` + quoteIdent(p["id"]) + `::text=l.` + quoteIdent(h["land_permit_id"]) + `::text LEFT JOIN public."Gis_Employees" e ON e."id_Gis_Employees"::text=` + manager + ` WHERE ($1='0' OR p.` + quoteIdent(p["company_id"]) + `::text=$1) ORDER BY l.` + quoteIdent(h["id"]) + `::text::bigint DESC LIMIT 5000`
	rows, err := m.DB.Query(ctx, query, strconv.FormatInt(companyID, 10))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []PotentialLand{}
	for rows.Next() {
		var item PotentialLand
		var id, permitID string
		if err = rows.Scan(&id, &item.PotentialName, &item.EstateName, &item.EstateAddress, &item.EstateManager, &permitID, &item.LandPermit, &item.Note); err != nil {
			return nil, err
		}
		item.ID, _ = strconv.ParseInt(id, 10, 64)
		item.LandPermitID, _ = strconv.ParseInt(permitID, 10, 64)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *PotentialLandModel) Save(ctx context.Context, companyID int64, id *int64, in PotentialLandInput) (PotentialLand, error) {
	h, p, err := m.schema(ctx)
	if err != nil {
		return PotentialLand{}, err
	}
	types, err := m.columnTypes(ctx, "Gis_Potential_Lands")
	if err != nil {
		return PotentialLand{}, err
	}
	var permitCompany string
	err = m.DB.QueryRow(ctx, `SELECT `+quoteIdent(p["company_id"])+`::text FROM `+potentialPermitTable+` WHERE `+quoteIdent(p["id"])+`::text=$1`, strconv.FormatInt(in.LandPermitID, 10)).Scan(&permitCompany)
	if err != nil {
		return PotentialLand{}, err
	}
	if companyID > 0 && permitCompany != strconv.FormatInt(companyID, 10) {
		return PotentialLand{}, fmt.Errorf("land permit tidak termasuk company aktif")
	}
	var manager any
	if name := strings.TrimSpace(in.EstateManager); name != "" {
		manager = name
	}
	if col := h["estate_manager"]; col != "" && manager != nil {
		typ := types[strings.ToLower(col)]
		if strings.Contains(typ, "int") || strings.Contains(strings.ToLower(col), "gis_employees") || strings.HasSuffix(strings.ToLower(col), "_id") {
			var employeeID string
			err = m.DB.QueryRow(ctx, `SELECT "id_Gis_Employees"::text FROM public."Gis_Employees" WHERE name=$1 AND ($2='0' OR "Gis_Companies_id"::text=$2) ORDER BY "id_Gis_Employees" LIMIT 1`, strings.TrimSpace(in.EstateManager), strconv.FormatInt(companyID, 10)).Scan(&employeeID)
			if err != nil {
				return PotentialLand{}, fmt.Errorf("estate manager tidak ditemukan: %w", err)
			}
			if strings.Contains(typ, "int") {
				manager, err = strconv.ParseInt(employeeID, 10, 64)
				if err != nil {
					return PotentialLand{}, err
				}
			} else {
				manager = employeeID
			}
		}
	}
	values := map[string]any{"potential_name": strings.TrimSpace(in.PotentialName), "estate_name": strings.TrimSpace(in.EstateName), "estate_address": strings.TrimSpace(in.EstateAddress), "estate_manager": manager, "land_permit_id": potentialIDArgument(in.LandPermitID, types[strings.ToLower(h["land_permit_id"])]), "note": strings.TrimSpace(in.Note)}
	fields := []string{}
	args := []any{}
	for _, key := range []string{"potential_name", "estate_name", "estate_address", "estate_manager", "land_permit_id", "note"} {
		if col := h[key]; col != "" {
			fields = append(fields, quoteIdent(col))
			args = append(args, values[key])
		}
	}
	if id == nil {
		var next int64
		err = m.DB.QueryRow(ctx, `SELECT COALESCE(MAX(`+quoteIdent(h["id"])+`::text::bigint),0)+1 FROM `+potentialLandTable).Scan(&next)
		if err != nil {
			return PotentialLand{}, err
		}
		fields = append([]string{quoteIdent(h["id"])}, fields...)
		args = append([]any{potentialIDArgument(next, types[strings.ToLower(h["id"])])}, args...)
		ph := make([]string, len(args))
		for i := range ph {
			ph[i] = fmt.Sprintf("$%d", i+1)
		}
		_, err = m.DB.Exec(ctx, `INSERT INTO `+potentialLandTable+` (`+strings.Join(fields, ",")+`) VALUES (`+strings.Join(ph, ",")+`)`, args...)
		if err != nil {
			return PotentialLand{}, err
		}
		id = &next
	} else {
		sets := []string{}
		for i, f := range fields {
			sets = append(sets, f+fmt.Sprintf("=$%d", i+1))
		}
		args = append(args, strconv.FormatInt(*id, 10))
		where := ` WHERE ` + quoteIdent(h["id"]) + fmt.Sprintf("::text=$%d", len(args))
		if companyID > 0 {
			args = append(args, strconv.FormatInt(companyID, 10))
			where += ` AND EXISTS (SELECT 1 FROM ` + potentialPermitTable + ` p WHERE p.` + quoteIdent(p["id"]) + `::text=` + potentialLandTable + `.` + quoteIdent(h["land_permit_id"]) + fmt.Sprintf("::text AND p.%s::text=$%d)", quoteIdent(p["company_id"]), len(args))
		}
		tag, e := m.DB.Exec(ctx, `UPDATE `+potentialLandTable+` SET `+strings.Join(sets, ",")+where, args...)
		if e != nil {
			return PotentialLand{}, e
		}
		if tag.RowsAffected() == 0 {
			return PotentialLand{}, pgx.ErrNoRows
		}
	}
	items, err := m.GetAll(ctx, companyID)
	if err != nil {
		return PotentialLand{}, err
	}
	for _, item := range items {
		if item.ID == *id {
			return item, nil
		}
	}
	return PotentialLand{}, pgx.ErrNoRows
}

func (m *PotentialLandModel) Delete(ctx context.Context, companyID, id int64) error {
	h, p, err := m.schema(ctx)
	if err != nil {
		return err
	}
	if _, err = m.potentialPermitID(ctx, companyID, id); err != nil {
		return err
	}
	spatial, err := m.spatialSchema(ctx)
	if err != nil {
		return err
	}
	tx, err := m.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `DELETE FROM `+potentialSpatialTable+` WHERE `+quoteIdent(spatial.potentialFK)+`=$1`, potentialIDArgument(id, spatial.potentialFKType)); err != nil {
		return err
	}
	query := `DELETE FROM ` + potentialLandTable + ` WHERE ` + quoteIdent(h["id"]) + `::text=$1`
	args := []any{strconv.FormatInt(id, 10)}
	if companyID > 0 {
		query += ` AND EXISTS (SELECT 1 FROM ` + potentialPermitTable + ` p WHERE p.` + quoteIdent(p["id"]) + `::text=` + potentialLandTable + `.` + quoteIdent(h["land_permit_id"]) + `::text AND p.` + quoteIdent(p["company_id"]) + `::text=$2)`
		args = append(args, strconv.FormatInt(companyID, 10))
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return tx.Commit(ctx)
}
