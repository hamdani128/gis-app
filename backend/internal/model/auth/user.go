package auth

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Department  string `json:"department"`
	ProcessRole string `json:"process_role"`
}

type AccessibleCompany struct {
	ID        int32  `json:"id"`
	Name      string `json:"name"`
	IsPrimary bool   `json:"is_primary"`
}

type ModulePrivileges struct {
	Add  bool `json:"p_add"`
	Del  bool `json:"p_del"`
	View bool `json:"p_view"`
	Edit bool `json:"p_edit"`
}

func (m *UserModel) GetModulePrivileges(ctx context.Context, username, moduleCode string) (ModulePrivileges, error) {
	var add, del, view, edit int
	err := m.DB.QueryRow(ctx, `SELECT COALESCE(MAX(ur.p_add),0), COALESCE(MAX(ur.p_del),0), COALESCE(MAX(ur.p_view),0), COALESCE(MAX(ur.p_edit),0) FROM public.userrole ur JOIN public.modul mo ON mo.idmodul=ur.modul_id WHERE LOWER(BTRIM(ur.username))=LOWER(BTRIM($1)) AND LOWER(BTRIM(mo.modulcode))=LOWER(BTRIM($2))`, username, moduleCode).Scan(&add, &del, &view, &edit)
	return ModulePrivileges{Add: add == 1, Del: del == 1, View: view == 1, Edit: edit == 1}, err
}

func (m *UserModel) GetVisibleModuleCodes(ctx context.Context, username string) ([]string, error) {
	rows, err := m.DB.Query(ctx, `SELECT DISTINCT LOWER(BTRIM(mo.modulcode))
		FROM public.modul mo
		WHERE (LOWER(BTRIM($1))='admin' OR EXISTS (
			SELECT 1 FROM public.userrole ur
			WHERE ur.modul_id=mo.idmodul
			  AND LOWER(BTRIM(ur.username))=LOWER(BTRIM($1))
			  AND COALESCE(ur.p_view,0)=1
		))
		  AND NULLIF(BTRIM(mo.modulcode),'') IS NOT NULL
		ORDER BY 1`, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	codes := []string{}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, rows.Err()
}

func (m *UserModel) CanManageAllCompanies(ctx context.Context, username string) (bool, error) {
	var allowed bool
	err := m.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public.userrole ur JOIN public.modul mo ON mo.idmodul=ur.modul_id WHERE LOWER(BTRIM(ur.username))=LOWER(BTRIM($1)) AND LOWER(BTRIM(mo.modulcode))='company' AND COALESCE(ur.p_add,0)=1 AND COALESCE(ur.p_del,0)=1 AND COALESCE(ur.p_view,0)=1 AND COALESCE(ur.p_edit,0)=1)`, username).Scan(&allowed)
	return allowed, err
}

// HasGlobalAcquisitionOfficer grants company-wide acquisition access only to
// an active process officer whose current employee record belongs to company 0.
func (m *UserModel) HasGlobalAcquisitionOfficer(ctx context.Context, username string) (bool, error) {
	var allowed bool
	err := m.DB.QueryRow(ctx, `WITH employee AS (
		SELECT e."id_Gis_Employees", e."Gis_Companies_id"
		FROM public."Gis_Employees" e
		WHERE LOWER(BTRIM(e.username))=LOWER(BTRIM($1))
		ORDER BY e."id_Gis_Employees" DESC LIMIT 1
	)
	SELECT EXISTS (
		SELECT 1 FROM employee e
		JOIN public."Gis_Process_Officers" po ON po."Gis_Employees_id_Gis_Employees"=e."id_Gis_Employees"
		WHERE e."Gis_Companies_id"=0
		  AND (po.valid_from IS NULL OR po.valid_from<=NOW())
		  AND (NULLIF(BTRIM(po.valid_until),'') IS NULL OR CASE WHEN BTRIM(po.valid_until) ~ '^\d{4}-\d{2}-\d{2}' THEN LEFT(BTRIM(po.valid_until),10)::date>=CURRENT_DATE ELSE TRUE END)
	)`, username).Scan(&allowed)
	return allowed, err
}

// HasSubmitMemoOfficer checks an active Submit Memo Land assignment for the
// selected company. A company-0 officer must also be assigned to that company.
func (m *UserModel) HasSubmitMemoOfficer(ctx context.Context, username string, companyID int64) (bool, error) {
	if companyID <= 0 {
		return false, nil
	}
	var allowed bool
	err := m.DB.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM public."Gis_Employees" e
		JOIN public."Gis_Process_Officers" po
		  ON po."Gis_Employees_id_Gis_Employees"=e."id_Gis_Employees"
		WHERE LOWER(BTRIM(e.username))=LOWER(BTRIM($1))
		  AND po."Gis_Aquisition_Process_Sequences_id"=1
		  AND (e."Gis_Companies_id"=$2 OR (e."Gis_Companies_id"=0 AND EXISTS (
		    SELECT 1 FROM public."Gis_Employee_Companies" ec
		    WHERE ec."Gis_Employees_id"=e."id_Gis_Employees"
		      AND ec."Gis_Companies_id"=$2)))
		  AND (po.valid_from IS NULL OR po.valid_from<=NOW())
		  AND (NULLIF(BTRIM(po.valid_until),'') IS NULL OR CASE WHEN BTRIM(po.valid_until) ~ '^\d{4}-\d{2}-\d{2}' THEN LEFT(BTRIM(po.valid_until),10)::date>=CURRENT_DATE ELSE TRUE END)
	)`, username, companyID).Scan(&allowed)
	return allowed, err
}

type UserModel struct {
	DB *pgxpool.Pool
}

func scanUser(row pgx.Row) (User, error) {
	var user User
	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Name,
		&user.Email,
		&user.Department,
		&user.ProcessRole,
	)
	user.Username = strings.TrimSpace(user.Username)
	user.Name = strings.TrimSpace(user.Name)
	user.Email = strings.TrimSpace(user.Email)
	user.Department = strings.TrimSpace(user.Department)
	user.ProcessRole = strings.TrimSpace(user.ProcessRole)
	return user, err
}

const userFields = `
	u."ID_USER",
	COALESCE(u."USERNAME", ''),
	COALESCE(
		NULLIF(BTRIM(e.name), ''),
		NULLIF(BTRIM(CONCAT_WS(' ', u."FIRSTNAME", u."MIDDLENAME", u."LASTNAME")), ''),
		NULLIF(BTRIM(u."USERNAME"), ''),
		''
	),
	COALESCE(NULLIF(BTRIM(e.email), ''), NULLIF(BTRIM(u."EMAIL"), ''), ''),
	COALESCE(NULLIF(BTRIM(dept.meaning), ''), ''),
	COALESCE(officer.process_role, '')
`

const employeeAndDepartmentJoins = `
	LEFT JOIN LATERAL (
		SELECT employee."id_Gis_Employees", employee.name, employee.email, employee.dept
		FROM public."Gis_Employees" employee
		WHERE LOWER(BTRIM(employee.username)) = LOWER(BTRIM(u."USERNAME"))
		ORDER BY employee."id_Gis_Employees" DESC
		LIMIT 1
	) e ON TRUE
	LEFT JOIN public.cg_ref_codes dept ON dept.id = e.dept
	LEFT JOIN LATERAL (
		SELECT STRING_AGG(DISTINCT seq.name, ', ' ORDER BY seq.name) AS process_role
		FROM public."Gis_Process_Officers" po
		JOIN public."Gis_Aquisition_Process_Sequences" seq
		  ON seq."id_Gis_Aquisition_Process_Sequences"=po."Gis_Aquisition_Process_Sequences_id"
		WHERE po."Gis_Employees_id_Gis_Employees"=e."id_Gis_Employees"
		  AND (po.valid_from IS NULL OR po.valid_from<=NOW())
		  AND (NULLIF(BTRIM(po.valid_until),'') IS NULL OR CASE WHEN BTRIM(po.valid_until) ~ '^\d{4}-\d{2}-\d{2}' THEN LEFT(BTRIM(po.valid_until),10)::date>=CURRENT_DATE ELSE TRUE END)
	) officer ON TRUE
`

func (m *UserModel) Authenticate(
	ctx context.Context,
	identifier string,
	password string,
) (User, error) {
	return scanUser(m.DB.QueryRow(ctx, `
		SELECT `+userFields+`
		FROM public.user_app u
		`+employeeAndDepartmentJoins+`
		WHERE (
			LOWER(u."USERNAME") = LOWER($1)
			OR LOWER(u."EMAIL") = LOWER($1)
		)
		AND u."PASSWORD" = $2
		AND COALESCE(u."ACCOUNTSTATUS", 0) = 0
		LIMIT 1
	`, identifier, password))
}

func (m *UserModel) GetActiveByID(ctx context.Context, id int64) (User, error) {
	return scanUser(m.DB.QueryRow(ctx, `
		SELECT `+userFields+`
		FROM public.user_app u
		`+employeeAndDepartmentJoins+`
		WHERE u."ID_USER" = $1
		AND COALESCE(u."ACCOUNTSTATUS", 0) = 0
	`, id))
}

func (m *UserModel) GetAccessibleCompanies(ctx context.Context, username string) ([]AccessibleCompany, error) {
	var canManageAllCompanies bool
	err := m.DB.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM public.userrole ur
			JOIN public.modul module ON module.idmodul = ur.modul_id
			WHERE LOWER(BTRIM(ur.username)) = LOWER(BTRIM($1))
			  AND LOWER(BTRIM(module.modulcode)) = 'company'
			  AND COALESCE(ur.p_add, 0) = 1
			  AND COALESCE(ur.p_del, 0) = 1
			  AND COALESCE(ur.p_view, 0) = 1
			  AND COALESCE(ur.p_edit, 0) = 1
		)
	`, username).Scan(&canManageAllCompanies)
	if err != nil {
		return nil, err
	}
	if canManageAllCompanies {
		return m.getAllCompanies(ctx, username)
	}

	rows, err := m.DB.Query(ctx, `
		WITH employee AS (
			SELECT e."id_Gis_Employees", e."Gis_Companies_id"
			FROM public."Gis_Employees" e
			WHERE LOWER(BTRIM(e.username)) = LOWER(BTRIM($1))
			ORDER BY e."id_Gis_Employees" DESC NULLS LAST
			LIMIT 1
		), accessible_company AS (
			SELECT ec."Gis_Companies_id"
			FROM employee
			JOIN public."Gis_Employee_Companies" ec
			  ON ec."Gis_Employees_id" = employee."id_Gis_Employees"
			UNION
			SELECT employee."Gis_Companies_id"
			FROM employee
			WHERE employee."Gis_Companies_id" IS NOT NULL
		)
		SELECT DISTINCT c."id_Gis_Companies",
		       COALESCE(NULLIF(BTRIM(c.name), ''), NULLIF(BTRIM(c.com_name), ''), ''),
		       COALESCE(c."id_Gis_Companies" = employee."Gis_Companies_id", FALSE) AS is_primary
		FROM accessible_company access
		CROSS JOIN employee
		JOIN public."Gis_Companies" c
		  ON c."id_Gis_Companies" = access."Gis_Companies_id"
		WHERE c."id_Gis_Companies" IS NOT NULL
		ORDER BY is_primary DESC, c."id_Gis_Companies"
		LIMIT 5000
	`, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AccessibleCompany, 0)
	for rows.Next() {
		var item AccessibleCompany
		if err := rows.Scan(&item.ID, &item.Name, &item.IsPrimary); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *UserModel) getAllCompanies(ctx context.Context, username string) ([]AccessibleCompany, error) {
	rows, err := m.DB.Query(ctx, `
			WITH employee AS (
				SELECT e."Gis_Companies_id"
				FROM public."Gis_Employees" e
				WHERE LOWER(BTRIM(e.username)) = LOWER(BTRIM($1))
				ORDER BY e."id_Gis_Employees" DESC NULLS LAST
				LIMIT 1
			)
			SELECT c."id_Gis_Companies",
			       COALESCE(NULLIF(BTRIM(c.name), ''), NULLIF(BTRIM(c.com_name), ''), ''),
			       COALESCE(c."id_Gis_Companies" = employee."Gis_Companies_id", FALSE)
			FROM public."Gis_Companies" c
			LEFT JOIN employee ON TRUE
			WHERE c."id_Gis_Companies" IS NOT NULL
			ORDER BY 3 DESC, c."id_Gis_Companies"
			LIMIT 5000
		`, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AccessibleCompany, 0)
	for rows.Next() {
		var item AccessibleCompany
		if err := rows.Scan(&item.ID, &item.Name, &item.IsPrimary); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
