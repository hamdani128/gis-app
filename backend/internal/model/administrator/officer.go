package administrator

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Officer struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Username        string     `json:"username"`
	CompanyID       *int32     `json:"company_id"`
	Company         string     `json:"company"`
	PotentialLandID *int32     `json:"potential_land_id"`
	Estate          string     `json:"estate"`
	DeptID          *int32     `json:"dept_id"`
	Department      string     `json:"department"`
	Email           string     `json:"email"`
	Note            string     `json:"note"`
	CreatedBy       *string    `json:"created_by"`
	DateCreated     *time.Time `json:"date_created"`
}

type OfficerInput struct {
	Name            string `json:"name"`
	Username        string `json:"username"`
	CompanyID       *int32 `json:"company_id"`
	PotentialLandID *int32 `json:"potential_land_id"`
	DeptID          *int32 `json:"dept_id"`
	Email           string `json:"email"`
	Note            string `json:"note"`
}

type EmployeeCompany struct {
	ID           int64  `json:"id"`
	EmployeeID   *int32 `json:"employee_id"`
	EmployeeName string `json:"employee_name"`
	Username     string `json:"username"`
	CompanyID    *int32 `json:"company_id"`
	CompanyName  string `json:"company_name"`
	Function     string `json:"function"`
}

type EmployeeCompanyInput struct {
	EmployeeID *int32 `json:"employee_id"`
	CompanyID  *int32 `json:"company_id"`
	Function   string `json:"function"`
}

type OfficerCompanyOption struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

type OfficerLandOption struct {
	ID            int32  `json:"id"`
	PotentialName string `json:"potential_name"`
	EstateName    string `json:"estate_name"`
}

type OfficerDepartmentOption struct {
	ID      int32  `json:"id"`
	Meaning string `json:"meaning"`
}

type OfficerOptions struct {
	Companies   []OfficerCompanyOption    `json:"companies"`
	Lands       []OfficerLandOption       `json:"lands"`
	Departments []OfficerDepartmentOption `json:"departments"`
}

type OfficerPrivilege struct {
	ModuleID   int32  `json:"module_id"`
	ModuleName string `json:"module_name"`
	ModuleCode string `json:"module_code"`
	ModuleFile string `json:"module_file"`
	PAdd       int32  `json:"p_add"`
	PDel       int32  `json:"p_del"`
	PView      int32  `json:"p_view"`
	PEdit      int32  `json:"p_edit"`
}

type OfficerPrivilegeInput struct {
	ModuleID int32 `json:"module_id"`
	PAdd     int32 `json:"p_add"`
	PDel     int32 `json:"p_del"`
	PView    int32 `json:"p_view"`
	PEdit    int32 `json:"p_edit"`
}

type OfficerLog struct {
	ID          *int32     `json:"id"`
	Type        string     `json:"type"`
	Object      string     `json:"object"`
	Log1        string     `json:"log1"`
	Log2        string     `json:"log2"`
	Log3        string     `json:"log3"`
	DateCreated *time.Time `json:"date_created"`
	CreatedBy   string     `json:"created_by"`
	Username    string     `json:"username"`
	LogCode     *int32     `json:"log_code"`
	IPRemote    string     `json:"ip_remote"`
}

type OfficerModel struct {
	DB *pgxpool.Pool
}

const (
	officerTable            = `public."Gis_Employees"`
	officerIDColumn         = `"id_Gis_Employees"`
	employeeCompanyTable    = `public."Gis_Employee_Companies"`
	employeeCompanyIDColumn = `"idGis_Employee_Companies"`
)

const officerSelect = `
	SELECT e."id_Gis_Employees", COALESCE(e.name, ''), COALESCE(e.username, ''),
	       e."Gis_Companies_id",
	       COALESCE(NULLIF(BTRIM(c.com_name), ''), NULLIF(BTRIM(c.name), ''), ''),
	       e."Gis_Potential_Lands_id", COALESCE(l.estate_name, ''), e.dept,
	       COALESCE(d.meaning, ''), COALESCE(e.email, ''), COALESCE(e.note, ''),
	       e.created_by, e.date_created
	FROM ` + officerTable + ` e
	LEFT JOIN public."Gis_Companies" c ON c."id_Gis_Companies" = e."Gis_Companies_id"
	LEFT JOIN public."Gis_Potential_Lands" l ON l."id_Gis_Potential_Lands" = e."Gis_Potential_Lands_id"
	LEFT JOIN public.cg_ref_codes d ON d.id = e.dept
`

func scanOfficer(row pgx.Row) (Officer, error) {
	var item Officer
	err := row.Scan(
		&item.ID,
		&item.Name,
		&item.Username,
		&item.CompanyID,
		&item.Company,
		&item.PotentialLandID,
		&item.Estate,
		&item.DeptID,
		&item.Department,
		&item.Email,
		&item.Note,
		&item.CreatedBy,
		&item.DateCreated,
	)
	return item, err
}

func (m *OfficerModel) GetOfficers(ctx context.Context) ([]Officer, error) {
	rows, err := m.DB.Query(ctx, officerSelect+`
		ORDER BY e.name NULLS LAST, e."id_Gis_Employees" DESC
		LIMIT 2000
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Officer, 0)
	for rows.Next() {
		item, err := scanOfficer(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *OfficerModel) GetOfficer(ctx context.Context, id int64) (Officer, error) {
	return scanOfficer(m.DB.QueryRow(ctx, officerSelect+`
		WHERE e."id_Gis_Employees" = $1
	`, id))
}

func (m *OfficerModel) CreateOfficer(ctx context.Context, input OfficerInput) (Officer, error) {
	tx, err := m.DB.Begin(ctx)
	if err != nil {
		return Officer{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `LOCK TABLE `+officerTable+` IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return Officer{}, err
	}
	var id int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(`+officerIDColumn+`), 0) + 1 FROM `+officerTable).Scan(&id); err != nil {
		return Officer{}, err
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO `+officerTable+` (
			`+officerIDColumn+`, name, username, "Gis_Companies_id", "Gis_Potential_Lands_id",
			dept, email, note, created_by, date_created
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW())
		RETURNING `+officerIDColumn+`
	`, id, input.Name, input.Username, input.CompanyID, input.PotentialLandID, input.DeptID, input.Email, input.Note, "local-dev").Scan(&id)
	if err != nil {
		return Officer{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Officer{}, err
	}
	return m.GetOfficer(ctx, id)
}

func (m *OfficerModel) UpdateOfficer(ctx context.Context, id int64, input OfficerInput) (Officer, error) {
	result, err := m.DB.Exec(ctx, `
		UPDATE `+officerTable+`
		SET name=$1, username=$2, "Gis_Companies_id"=$3, "Gis_Potential_Lands_id"=$4,
		    dept=$5, email=$6, note=$7, modified_by=$8, date_modified=NOW()::text
		WHERE `+officerIDColumn+`=$9
	`, input.Name, input.Username, input.CompanyID, input.PotentialLandID, input.DeptID, input.Email, input.Note, "local-dev", id)
	if err != nil {
		return Officer{}, err
	}
	if result.RowsAffected() == 0 {
		return Officer{}, pgx.ErrNoRows
	}
	return m.GetOfficer(ctx, id)
}

func (m *OfficerModel) DeleteOfficer(ctx context.Context, id int64) error {
	result, err := m.DB.Exec(ctx, `DELETE FROM `+officerTable+` WHERE `+officerIDColumn+`=$1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (m *OfficerModel) GetOptions(ctx context.Context) (OfficerOptions, error) {
	options := OfficerOptions{
		Companies:   make([]OfficerCompanyOption, 0),
		Lands:       make([]OfficerLandOption, 0),
		Departments: make([]OfficerDepartmentOption, 0),
	}
	rows, err := m.DB.Query(ctx, `
		SELECT "id_Gis_Companies", COALESCE(NULLIF(BTRIM(com_name), ''), NULLIF(BTRIM(name), ''), '')
		FROM public."Gis_Companies" ORDER BY 2
	`)
	if err != nil {
		return options, err
	}
	for rows.Next() {
		var option OfficerCompanyOption
		if err := rows.Scan(&option.ID, &option.Name); err != nil {
			rows.Close()
			return options, err
		}
		options.Companies = append(options.Companies, option)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return options, err
	}

	rows, err = m.DB.Query(ctx, `
		SELECT "id_Gis_Potential_Lands", COALESCE(potential_name, ''), COALESCE(estate_name, '')
		FROM public."Gis_Potential_Lands" ORDER BY 3, 2
	`)
	if err != nil {
		return options, err
	}
	for rows.Next() {
		var option OfficerLandOption
		if err := rows.Scan(&option.ID, &option.PotentialName, &option.EstateName); err != nil {
			rows.Close()
			return options, err
		}
		options.Lands = append(options.Lands, option)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return options, err
	}

	rows, err = m.DB.Query(ctx, `
		SELECT DISTINCT c.id, COALESCE(c.meaning, '')
		FROM public.cg_ref_codes c
		JOIN `+officerTable+` e ON e.dept = c.id
		WHERE c.id IS NOT NULL
		ORDER BY 2
	`)
	if err != nil {
		return options, err
	}
	for rows.Next() {
		var option OfficerDepartmentOption
		if err := rows.Scan(&option.ID, &option.Meaning); err != nil {
			rows.Close()
			return options, err
		}
		options.Departments = append(options.Departments, option)
	}
	err = rows.Err()
	rows.Close()
	return options, err
}

func (m *OfficerModel) GetEmployeeCompanies(ctx context.Context) ([]EmployeeCompany, error) {
	rows, err := m.DB.Query(ctx, `
		SELECT ec."idGis_Employee_Companies", ec."Gis_Employees_id", COALESCE(e.name, ''),
		       COALESCE(e.username, ''), ec."Gis_Companies_id",
		       COALESCE(NULLIF(BTRIM(c.com_name), ''), NULLIF(BTRIM(c.name), ''), ''),
	       COALESCE(ec.function, '')
		FROM `+employeeCompanyTable+` ec
		LEFT JOIN `+officerTable+` e ON e."id_Gis_Employees" = ec."Gis_Employees_id"
		LEFT JOIN public."Gis_Companies" c ON c."id_Gis_Companies" = ec."Gis_Companies_id"
		ORDER BY e.name NULLS LAST, c.name NULLS LAST
		LIMIT 2000
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]EmployeeCompany, 0)
	for rows.Next() {
		var item EmployeeCompany
		if err := rows.Scan(&item.ID, &item.EmployeeID, &item.EmployeeName, &item.Username, &item.CompanyID, &item.CompanyName, &item.Function); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *OfficerModel) GetEmployeeCompany(ctx context.Context, id int64) (EmployeeCompany, error) {
	var item EmployeeCompany
	err := m.DB.QueryRow(ctx, `
		SELECT ec."idGis_Employee_Companies", ec."Gis_Employees_id", COALESCE(e.name, ''),
		       COALESCE(e.username, ''), ec."Gis_Companies_id",
		       COALESCE(NULLIF(BTRIM(c.com_name), ''), NULLIF(BTRIM(c.name), ''), ''),
	       COALESCE(ec.function, '')
		FROM `+employeeCompanyTable+` ec
		LEFT JOIN `+officerTable+` e ON e."id_Gis_Employees" = ec."Gis_Employees_id"
		LEFT JOIN public."Gis_Companies" c ON c."id_Gis_Companies" = ec."Gis_Companies_id"
		WHERE ec."idGis_Employee_Companies"=$1
	`, id).Scan(&item.ID, &item.EmployeeID, &item.EmployeeName, &item.Username, &item.CompanyID, &item.CompanyName, &item.Function)
	return item, err
}

func (m *OfficerModel) CreateEmployeeCompany(ctx context.Context, input EmployeeCompanyInput) (EmployeeCompany, error) {
	tx, err := m.DB.Begin(ctx)
	if err != nil {
		return EmployeeCompany{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `LOCK TABLE `+employeeCompanyTable+` IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return EmployeeCompany{}, err
	}
	var id int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(`+employeeCompanyIDColumn+`), 0) + 1 FROM `+employeeCompanyTable).Scan(&id); err != nil {
		return EmployeeCompany{}, err
	}
	var item EmployeeCompany
	err = tx.QueryRow(ctx, `
		INSERT INTO `+employeeCompanyTable+` (
			`+employeeCompanyIDColumn+`, "Gis_Employees_id", "Gis_Companies_id", function, created_by, date_created
		)
		VALUES ($1,$2,$3,$4,$5,NOW())
		RETURNING `+employeeCompanyIDColumn+`, "Gis_Employees_id", '', '', "Gis_Companies_id", '', COALESCE(function,'')
	`, id, input.EmployeeID, input.CompanyID, input.Function, "local-dev").Scan(
		&item.ID, &item.EmployeeID, &item.EmployeeName, &item.Username, &item.CompanyID, &item.CompanyName, &item.Function,
	)
	if err != nil {
		return EmployeeCompany{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return EmployeeCompany{}, err
	}
	return m.GetEmployeeCompany(ctx, item.ID)
}

func (m *OfficerModel) UpdateEmployeeCompany(ctx context.Context, id int64, input EmployeeCompanyInput) (EmployeeCompany, error) {
	_, err := m.DB.Exec(ctx, `
		UPDATE `+employeeCompanyTable+`
		SET "Gis_Employees_id"=$1, "Gis_Companies_id"=$2, function=$3,
		    modified_by=$4, date_modified=NOW()::text
		WHERE `+employeeCompanyIDColumn+`=$5
	`, input.EmployeeID, input.CompanyID, input.Function, "local-dev", id)
	if err != nil {
		return EmployeeCompany{}, err
	}
	return m.GetEmployeeCompany(ctx, id)
}

func (m *OfficerModel) DeleteEmployeeCompany(ctx context.Context, id int64) error {
	result, err := m.DB.Exec(ctx, `DELETE FROM `+employeeCompanyTable+` WHERE `+employeeCompanyIDColumn+`=$1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (m *OfficerModel) GetPrivileges(ctx context.Context, username string) ([]OfficerPrivilege, error) {
	rows, err := m.DB.Query(ctx, `
		SELECT m.idmodul, COALESCE(m.modulname, ''), COALESCE(m.modulcode, ''),
		       COALESCE(m.modulfile, ''), COALESCE(ur.p_add, 0), COALESCE(ur.p_del, 0),
		       COALESCE(ur.p_view, 0), COALESCE(ur.p_edit, 0)
		FROM public.modul m
		LEFT JOIN LATERAL (
			SELECT p_add, p_del, p_view, p_edit
			FROM public.userrole
			WHERE LOWER(BTRIM(username)) = LOWER(BTRIM($1))
			  AND modul_id = m.idmodul
			ORDER BY iduserrole DESC NULLS LAST
			LIMIT 1
		) ur ON TRUE
		WHERE m.idmodul IS NOT NULL
		ORDER BY COALESCE(m.seq, 0), m.modulname
	`, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]OfficerPrivilege, 0)
	for rows.Next() {
		var item OfficerPrivilege
		if err := rows.Scan(&item.ModuleID, &item.ModuleName, &item.ModuleCode, &item.ModuleFile, &item.PAdd, &item.PDel, &item.PView, &item.PEdit); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *OfficerModel) SetPrivileges(ctx context.Context, username string, privileges []OfficerPrivilegeInput) error {
	tx, err := m.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `LOCK TABLE public.userrole IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return err
	}
	var nextID int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(iduserrole), 0) + 1 FROM public.userrole`).Scan(&nextID); err != nil {
		return err
	}
	for _, permission := range privileges {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public.modul WHERE idmodul=$1)`, permission.ModuleID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return pgx.ErrNoRows
		}
		result, err := tx.Exec(ctx, `
			UPDATE public.userrole
			SET p_add=$1, p_del=$2, p_view=$3, p_edit=$4
			WHERE LOWER(BTRIM(username)) = LOWER(BTRIM($5)) AND modul_id=$6
		`, permission.PAdd, permission.PDel, permission.PView, permission.PEdit, username, permission.ModuleID)
		if err != nil {
			return err
		}
		if result.RowsAffected() > 0 || (permission.PAdd == 0 && permission.PDel == 0 && permission.PView == 0 && permission.PEdit == 0) {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO public.userrole (iduserrole, modul_id, username, p_add, p_del, p_view, p_edit)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
		`, nextID, permission.ModuleID, username, permission.PAdd, permission.PDel, permission.PView, permission.PEdit); err != nil {
			return err
		}
		nextID++
	}
	return tx.Commit(ctx)
}

func (m *OfficerModel) GetLogs(ctx context.Context, username string) ([]OfficerLog, error) {
	rows, err := m.DB.Query(ctx, `
		SELECT IDLOG, COALESCE(TYPE, ''), COALESCE(OBJECT, ''), COALESCE(LOG1, ''),
		       COALESCE(LOG2, ''), COALESCE(LOG3, ''), DATE_CREATED,
		       COALESCE(CREATED_BY, ''), COALESCE(USERNAME, ''), LOG_CODE,
		       COALESCE(ip_remote, '')
		FROM public.loguser
		WHERE LOWER(BTRIM(USERNAME)) = LOWER(BTRIM($1))
		ORDER BY DATE_CREATED DESC NULLS LAST, IDLOG DESC NULLS LAST
		LIMIT 250
	`, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]OfficerLog, 0)
	for rows.Next() {
		var item OfficerLog
		if err := rows.Scan(&item.ID, &item.Type, &item.Object, &item.Log1, &item.Log2, &item.Log3, &item.DateCreated, &item.CreatedBy, &item.Username, &item.LogCode, &item.IPRemote); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *OfficerModel) SetPassword(ctx context.Context, username, password string) (int64, error) {
	tx, err := m.DB.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `LOCK TABLE public.user_app IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return 0, err
	}
	var matches int64
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM public.user_app
		WHERE LOWER(BTRIM("USERNAME")) = LOWER(BTRIM($1))
	`, username).Scan(&matches); err != nil {
		return 0, err
	}
	if matches != 1 {
		return matches, nil
	}
	result, err := tx.Exec(ctx, `
		UPDATE public.user_app
		SET "PASSWORD"=$1, "MODIFIED_BY"=$2, "DATE_MODIFIED"=NOW()::text
		WHERE LOWER(BTRIM("USERNAME")) = LOWER(BTRIM($3))
	`, password, "local-dev", username)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

func NormalizeOfficerInput(input *OfficerInput) {
	input.Name = strings.TrimSpace(input.Name)
	input.Username = strings.TrimSpace(input.Username)
	input.Email = strings.TrimSpace(input.Email)
	input.Note = strings.TrimSpace(input.Note)
}

func NormalizeEmployeeCompanyInput(input *EmployeeCompanyInput) {
	input.Function = strings.TrimSpace(input.Function)
}
