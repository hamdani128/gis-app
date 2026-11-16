package lands

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const divisionLandTable = `public."Gis_Aquistion_Divisions"`

type DivisionLand struct {
	ID              int64  `json:"id"`
	DivisionName    string `json:"division_name"`
	PotentialLandID int64  `json:"potential_land_id"`
	PotentialLand   string `json:"potential_land"`
	RequestedByID   *int64 `json:"requested_by_id"`
	RequestedBy     string `json:"requested_by"`
	ApprovedByID    *int64 `json:"approved_by_id"`
	ApprovedBy      string `json:"approved_by"`
	StartPlan       string `json:"start_plan"`
	EndPlan         string `json:"end_plan"`
	Note            string `json:"note"`
}

type DivisionLandInput struct {
	DivisionName    string `json:"division_name"`
	PotentialLandID int64  `json:"potential_land_id"`
	RequestedByID   *int64 `json:"requested_by_id"`
	ApprovedByID    *int64 `json:"approved_by_id"`
	StartPlan       string `json:"start_plan"`
	EndPlan         string `json:"end_plan"`
	Note            string `json:"note"`
}

type DivisionLandOptions struct {
	PotentialLands []HCVLandOption `json:"potential_lands"`
	Employees      []HCVLandOption `json:"employees"`
}

type DivisionLandModel struct{ DB *pgxpool.Pool }

var divisionLandAliases = map[string][]string{
	"id":                {"id_Gis_Aqcuistion_Divisions", "id_Gis_Aquistion_Divisions", "id"},
	"division_name":     {"division_name", "block_plan", "name"},
	"potential_land_id": {"Gis_Potential_Lands_id", "potential_land_id"},
	"requested_by_id":   {"Gis_Employees_id_Requestor", "requested_by_id", "requestor_id"},
	"approved_by_id":    {"Gis_Employees_id_Approval", "approved_by_id", "approval_id"},
	"start_plan":        {"start_plan", "start_date"},
	"end_plan":          {"end_plan", "end_date"},
	"note":              {"note", "notes"},
	"map_upload":        {"map_upload"},
}

func (m *DivisionLandModel) schema(ctx context.Context) (map[string]string, map[string]string, map[string]string, error) {
	helper := &HCVLandModel{DB: m.DB}
	d, err := helper.columns(ctx, "Gis_Aquistion_Divisions", divisionLandAliases)
	if err != nil {
		return nil, nil, nil, err
	}
	p, err := helper.columns(ctx, "Gis_Potential_Lands", potentialLandAliases)
	if err != nil {
		return nil, nil, nil, err
	}
	l, err := helper.columns(ctx, "Gis_Land_Permits", hcvPermitAliases)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, key := range []string{"id", "division_name", "potential_land_id"} {
		if d[key] == "" {
			return nil, nil, nil, fmt.Errorf("Gis_Aquistion_Divisions harus memiliki kolom %s", key)
		}
	}
	for _, key := range []string{"id", "land_permit_id", "potential_name"} {
		if p[key] == "" {
			return nil, nil, nil, fmt.Errorf("Gis_Potential_Lands harus memiliki kolom %s", key)
		}
	}
	for _, key := range []string{"id", "company_id", "permit_number"} {
		if l[key] == "" {
			return nil, nil, nil, fmt.Errorf("Gis_Land_Permits harus memiliki kolom %s", key)
		}
	}
	return d, p, l, nil
}

func (m *DivisionLandModel) GetOptions(ctx context.Context, companyID int64) (DivisionLandOptions, error) {
	out := DivisionLandOptions{PotentialLands: []HCVLandOption{}, Employees: []HCVLandOption{}}
	_, p, l, err := m.schema(ctx)
	if err != nil {
		return out, err
	}
	query := `SELECT p.` + quoteIdent(p["id"]) + `::text, COALESCE(NULLIF(p.` + quoteIdent(p["estate_name"]) + `::text,''),p.` + quoteIdent(p["potential_name"]) + `::text,'') FROM ` + potentialLandTable + ` p JOIN ` + hcvPermitTable + ` l ON l.` + quoteIdent(l["id"]) + `::text=p.` + quoteIdent(p["land_permit_id"]) + `::text WHERE ($1='0' OR l.` + quoteIdent(l["company_id"]) + `::text=$1) ORDER BY 2,1`
	rows, err := m.DB.Query(ctx, query, strconv.FormatInt(companyID, 10))
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
			out.PotentialLands = append(out.PotentialLands, HCVLandOption{ID: number, Name: name})
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	rows, err = m.DB.Query(ctx, `SELECT "id_Gis_Employees"::text, COALESCE(name,'') FROM public."Gis_Employees" WHERE ($1='0' OR "Gis_Companies_id"::text=$1) ORDER BY 2,1`, strconv.FormatInt(companyID, 10))
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
			out.Employees = append(out.Employees, HCVLandOption{ID: number, Name: name})
		}
	}
	return out, rows.Err()
}

func (m *DivisionLandModel) GetAll(ctx context.Context, companyID int64) ([]DivisionLand, error) {
	d, p, l, err := m.schema(ctx)
	if err != nil {
		return nil, err
	}
	field := func(key string) string {
		if d[key] == "" {
			return `NULL`
		}
		return `d.` + quoteIdent(d[key]) + `::text`
	}
	potentialName := `COALESCE(NULLIF(p.` + quoteIdent(p["estate_name"]) + `::text,''),p.` + quoteIdent(p["potential_name"]) + `::text,'')`
	query := `SELECT d.` + quoteIdent(d["id"]) + `::text, COALESCE(` + field("division_name") + `,''), d.` + quoteIdent(d["potential_land_id"]) + `::text, ` + potentialName + `, COALESCE(` + field("requested_by_id") + `,''), COALESCE(req.name,''), COALESCE(` + field("approved_by_id") + `,''), COALESCE(appr.name,''), COALESCE(` + field("start_plan") + `,''), COALESCE(` + field("end_plan") + `,''), COALESCE(` + field("note") + `,'') FROM ` + divisionLandTable + ` d JOIN ` + potentialLandTable + ` p ON p.` + quoteIdent(p["id"]) + `::text=d.` + quoteIdent(d["potential_land_id"]) + `::text JOIN ` + hcvPermitTable + ` l ON l.` + quoteIdent(l["id"]) + `::text=p.` + quoteIdent(p["land_permit_id"]) + `::text LEFT JOIN public."Gis_Employees" req ON req."id_Gis_Employees"::text=` + field("requested_by_id") + ` LEFT JOIN public."Gis_Employees" appr ON appr."id_Gis_Employees"::text=` + field("approved_by_id") + ` WHERE ($1='0' OR l.` + quoteIdent(l["company_id"]) + `::text=$1) ORDER BY d.` + quoteIdent(d["id"]) + `::text::bigint DESC LIMIT 5000`
	rows, err := m.DB.Query(ctx, query, strconv.FormatInt(companyID, 10))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []DivisionLand{}
	for rows.Next() {
		var item DivisionLand
		var id, potentialID, requestID, approvalID string
		if err = rows.Scan(&id, &item.DivisionName, &potentialID, &item.PotentialLand, &requestID, &item.RequestedBy, &approvalID, &item.ApprovedBy, &item.StartPlan, &item.EndPlan, &item.Note); err != nil {
			return nil, err
		}
		item.ID, _ = strconv.ParseInt(id, 10, 64)
		item.PotentialLandID, _ = strconv.ParseInt(potentialID, 10, 64)
		if requestID != "" {
			n, e := strconv.ParseInt(requestID, 10, 64)
			if e == nil {
				item.RequestedByID = &n
			}
		}
		if approvalID != "" {
			n, e := strconv.ParseInt(approvalID, 10, 64)
			if e == nil {
				item.ApprovedByID = &n
			}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func divisionPlanDate(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	for _, layout := range []string{"2006-01-02", "02/01/2006"} {
		if date, err := time.Parse(layout, value); err == nil {
			return date.Format("02/01/2006"), nil
		}
	}
	return "", fmt.Errorf("tanggal target harus berformat YYYY-MM-DD")
}

func (m *DivisionLandModel) Save(ctx context.Context, companyID int64, id *int64, in DivisionLandInput) (DivisionLand, error) {
	d, p, l, err := m.schema(ctx)
	if err != nil {
		return DivisionLand{}, err
	}
	types, err := (&HCVLandModel{DB: m.DB}).columnTypes(ctx, "Gis_Aquistion_Divisions")
	if err != nil {
		return DivisionLand{}, err
	}
	var ownerCompany string
	err = m.DB.QueryRow(ctx, `SELECT l.`+quoteIdent(l["company_id"])+`::text FROM `+potentialLandTable+` p JOIN `+hcvPermitTable+` l ON l.`+quoteIdent(l["id"])+`::text=p.`+quoteIdent(p["land_permit_id"])+`::text WHERE p.`+quoteIdent(p["id"])+`::text=$1`, strconv.FormatInt(in.PotentialLandID, 10)).Scan(&ownerCompany)
	if err != nil {
		return DivisionLand{}, err
	}
	if companyID > 0 && ownerCompany != strconv.FormatInt(companyID, 10) {
		return DivisionLand{}, fmt.Errorf("potential land tidak termasuk company aktif")
	}
	for _, employeeID := range []*int64{in.RequestedByID, in.ApprovedByID} {
		if employeeID == nil {
			continue
		}
		var exists bool
		err = m.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public."Gis_Employees" WHERE "id_Gis_Employees"::text=$1 AND ($2='0' OR "Gis_Companies_id"::text=$2))`, strconv.FormatInt(*employeeID, 10), strconv.FormatInt(companyID, 10)).Scan(&exists)
		if err != nil {
			return DivisionLand{}, err
		}
		if !exists {
			return DivisionLand{}, fmt.Errorf("employee tidak termasuk company aktif")
		}
	}
	start, err := divisionPlanDate(in.StartPlan)
	if err != nil {
		return DivisionLand{}, err
	}
	end, err := divisionPlanDate(in.EndPlan)
	if err != nil {
		return DivisionLand{}, err
	}
	var requested, approved any
	if in.RequestedByID != nil && d["requested_by_id"] != "" {
		requested = hcvIDArgument(*in.RequestedByID, types[strings.ToLower(d["requested_by_id"])])
	}
	if in.ApprovedByID != nil && d["approved_by_id"] != "" {
		approved = hcvIDArgument(*in.ApprovedByID, types[strings.ToLower(d["approved_by_id"])])
	}
	values := map[string]any{"division_name": strings.TrimSpace(in.DivisionName), "potential_land_id": hcvIDArgument(in.PotentialLandID, types[strings.ToLower(d["potential_land_id"])]), "requested_by_id": requested, "approved_by_id": approved, "start_plan": start, "end_plan": end, "note": strings.TrimSpace(in.Note)}
	fields := []string{}
	args := []any{}
	for _, key := range []string{"division_name", "potential_land_id", "requested_by_id", "approved_by_id", "start_plan", "end_plan", "note"} {
		if col := d[key]; col != "" {
			fields = append(fields, quoteIdent(col))
			args = append(args, values[key])
		}
	}
	if id == nil {
		var next int64
		err = m.DB.QueryRow(ctx, `SELECT COALESCE(MAX(`+quoteIdent(d["id"])+`::text::bigint),0)+1 FROM `+divisionLandTable).Scan(&next)
		if err != nil {
			return DivisionLand{}, err
		}
		fields = append([]string{quoteIdent(d["id"])}, fields...)
		args = append([]any{hcvIDArgument(next, types[strings.ToLower(d["id"])])}, args...)
		ph := make([]string, len(args))
		for i := range ph {
			ph[i] = fmt.Sprintf("$%d", i+1)
		}
		_, err = m.DB.Exec(ctx, `INSERT INTO `+divisionLandTable+` (`+strings.Join(fields, ",")+`) VALUES (`+strings.Join(ph, ",")+`)`, args...)
		if err != nil {
			return DivisionLand{}, err
		}
		id = &next
	} else {
		sets := []string{}
		for i, f := range fields {
			sets = append(sets, f+fmt.Sprintf("=$%d", i+1))
		}
		args = append(args, strconv.FormatInt(*id, 10))
		query := `UPDATE ` + divisionLandTable + ` SET ` + strings.Join(sets, ",") + ` WHERE ` + quoteIdent(d["id"]) + fmt.Sprintf("::text=$%d", len(args))
		if companyID > 0 {
			args = append(args, strconv.FormatInt(companyID, 10))
			query += ` AND EXISTS(SELECT 1 FROM ` + potentialLandTable + ` p JOIN ` + hcvPermitTable + ` l ON l.` + quoteIdent(l["id"]) + `::text=p.` + quoteIdent(p["land_permit_id"]) + `::text WHERE p.` + quoteIdent(p["id"]) + `::text=` + divisionLandTable + `.` + quoteIdent(d["potential_land_id"]) + fmt.Sprintf("::text AND l.%s::text=$%d)", quoteIdent(l["company_id"]), len(args))
		}
		tag, e := m.DB.Exec(ctx, query, args...)
		if e != nil {
			return DivisionLand{}, e
		}
		if tag.RowsAffected() == 0 {
			return DivisionLand{}, pgx.ErrNoRows
		}
	}
	items, err := m.GetAll(ctx, companyID)
	if err != nil {
		return DivisionLand{}, err
	}
	for _, item := range items {
		if item.ID == *id {
			return item, nil
		}
	}
	return DivisionLand{}, pgx.ErrNoRows
}

func (m *DivisionLandModel) Delete(ctx context.Context, companyID, id int64) error {
	d, p, l, err := m.schema(ctx)
	if err != nil {
		return err
	}
	if _, err = m.divisionPermitID(ctx, companyID, id); err != nil {
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
	if _, err = tx.Exec(ctx, `DELETE FROM `+hcvSpatialTable+` WHERE `+quoteIdent(spatial.divisionFK)+`::text=$1`, strconv.FormatInt(id, 10)); err != nil {
		return err
	}
	query := `DELETE FROM ` + divisionLandTable + ` WHERE ` + quoteIdent(d["id"]) + `::text=$1`
	args := []any{strconv.FormatInt(id, 10)}
	if companyID > 0 {
		query += ` AND EXISTS(SELECT 1 FROM ` + potentialLandTable + ` p JOIN ` + hcvPermitTable + ` l ON l.` + quoteIdent(l["id"]) + `::text=p.` + quoteIdent(p["land_permit_id"]) + `::text WHERE p.` + quoteIdent(p["id"]) + `::text=` + divisionLandTable + `.` + quoteIdent(d["potential_land_id"]) + `::text AND l.` + quoteIdent(l["company_id"]) + `::text=$2)`
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
