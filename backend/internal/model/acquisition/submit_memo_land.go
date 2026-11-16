package acquisition

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	authModel "gis-app/backend/internal/model/auth"
	landModel "gis-app/backend/internal/model/lands"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const submitSequenceID = 1

var ErrNotOfficer = errors.New("pengguna bukan petugas Submit Memo Land untuk company ini")
var ErrAlreadySubmitted = errors.New("Submit Memo Land untuk lahan ini sudah tercatat")

type SubmitMemoLandModel struct{ DB *pgxpool.Pool }
type SubmitMemoLandInput struct {
	MemoID int64  `json:"memo_id"`
	Note   string `json:"note"`
}
type SubmitMemoLandOfficer struct {
	EmployeeID int64  `json:"employee_id"`
	Name       string `json:"name"`
	CompanyID  int64  `json:"company_id"`
}
type SubmitMemoLandPage struct {
	Officer *SubmitMemoLandOfficer `json:"officer"`
	Lands   []map[string]any       `json:"lands"`
}
type SubmitMemoLandDetail struct {
	Land    map[string]any   `json:"land"`
	Memos   []Memo           `json:"memos"`
	History []map[string]any `json:"history"`
}

func (m *SubmitMemoLandModel) Officer(ctx context.Context, username string, companyID int64) (SubmitMemoLandOfficer, error) {
	return m.officerForSequence(ctx, username, companyID, submitSequenceID)
}

func (m *SubmitMemoLandModel) officerForSequence(ctx context.Context, username string, companyID int64, sequenceID int) (SubmitMemoLandOfficer, error) {
	var officer SubmitMemoLandOfficer
	err := m.DB.QueryRow(ctx, `SELECT e."id_Gis_Employees",COALESCE(e.name,''),
		CASE WHEN e."Gis_Companies_id"=0 THEN $2::bigint ELSE e."Gis_Companies_id" END
		FROM public."Gis_Employees" e
		WHERE LOWER(BTRIM(e.username))=LOWER(BTRIM($1))
		  AND e."Gis_Companies_id" IS NOT NULL
		  AND (e."Gis_Companies_id"=$2 OR (e."Gis_Companies_id"=0 AND ($2::bigint=0 OR EXISTS (
		    SELECT 1 FROM public."Gis_Employee_Companies" ec
		    WHERE ec."Gis_Employees_id"=e."id_Gis_Employees"
		      AND ec."Gis_Companies_id"=$2))))
		  AND EXISTS (SELECT 1 FROM public."Gis_Process_Officers" po
		    WHERE po."Gis_Employees_id_Gis_Employees"=e."id_Gis_Employees"
		      AND po."Gis_Aquisition_Process_Sequences_id"=$3
		      AND (po.valid_from IS NULL OR po.valid_from<=NOW())
		      AND (NULLIF(BTRIM(po.valid_until),'') IS NULL OR CASE WHEN BTRIM(po.valid_until) ~ '^\d{4}-\d{2}-\d{2}' THEN LEFT(BTRIM(po.valid_until),10)::date>=CURRENT_DATE ELSE TRUE END))
		ORDER BY e."id_Gis_Employees" DESC LIMIT 1`, username, companyID, sequenceID).Scan(&officer.EmployeeID, &officer.Name, &officer.CompanyID)
	if errors.Is(err, pgx.ErrNoRows) {
		return officer, ErrNotOfficer
	}
	return officer, err
}

const submitLandSelect = `WITH submit_summary AS (
	SELECT ap."Gis_Compensated_Lands_id" AS land_id, COUNT(*) AS submit_count
	FROM public."Gis_Aquisition_Processes" ap
	WHERE ap."Gis_Aquisition_Process_Sequences_id"=1
	GROUP BY ap."Gis_Compensated_Lands_id"
)
SELECT DISTINCT ON (l."id_Gis_Compensated_Lands") to_jsonb(l) || jsonb_build_object(
	'id',l."id_Gis_Compensated_Lands",'company_id',COALESCE(lp."Gis_Companies_id",o."Gis_Companies_id"),
	'company_name',COALESCE(NULLIF(BTRIM(co.name),''),NULLIF(BTRIM(co.com_name),''),''),
	'block_plan',COALESCE(d.division_name,''),'owner',COALESCE(o.name,''),
	'owner_ktp',COALESCE(o.ktp_number,''),'estate_name',COALESCE(p.estate_name,''),
	'permit_number',COALESCE(lp.permit_number,''),
	'submit_count',COALESCE(submit.submit_count,0))
	FROM public."Gis_Compensated_Lands" l
	LEFT JOIN public."Gis_Aquistion_Divisions" d ON d."id_Gis_Aqcuistion_Divisions"=l."Gis_Aquistion_Divisions_id"
	LEFT JOIN public."Gis_Potential_Lands" p ON p."id_Gis_Potential_Lands"=d."Gis_Potential_Lands_id"
	LEFT JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id"
	LEFT JOIN public."Gis_Land_Owners" o ON o."id_Gis_Land_Owners"=l."Gis_Land_Owners_id"
	JOIN public."Gis_Companies" co ON co."id_Gis_Companies"=COALESCE(lp."Gis_Companies_id",o."Gis_Companies_id")
	LEFT JOIN submit_summary submit ON submit.land_id=l."id_Gis_Compensated_Lands"
	WHERE ($1::bigint=0 OR COALESCE(lp."Gis_Companies_id",o."Gis_Companies_id")=$1)
	  AND l.status=$2`

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

// Company 0 is an aggregate view. Process officers see their employee
// companies; company administrators retain their all-company view.
func (m *SubmitMemoLandModel) listLandsForStatus(ctx context.Context, username string, companyID int64, status int, extraWhere string, extraArgs ...any) ([]map[string]any, error) {
	args := []any{companyID, status}
	args = append(args, extraArgs...)
	query := submitLandSelect + extraWhere
	if companyID == 0 {
		users := &authModel.UserModel{DB: m.DB}
		manageAll, err := users.CanManageAllCompanies(ctx, username)
		if err != nil {
			return nil, err
		}
		if !manageAll {
			companies, err := users.GetAccessibleCompanies(ctx, username)
			if err != nil {
				return nil, err
			}
			ids := make([]int64, 0, len(companies))
			for _, company := range companies {
				if company.ID > 0 {
					ids = append(ids, int64(company.ID))
				}
			}
			query += fmt.Sprintf(` AND COALESCE(lp."Gis_Companies_id",o."Gis_Companies_id")=ANY($%d::bigint[])`, len(args)+1)
			args = append(args, ids)
		}
	}
	query += ` ORDER BY l."id_Gis_Compensated_Lands" DESC LIMIT 5000`
	rows, err := m.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return scanJSONRows(rows)
}

func (m *SubmitMemoLandModel) List(ctx context.Context, username string, companyID int64) (SubmitMemoLandPage, error) {
	officer, err := m.Officer(ctx, username, companyID)
	if err != nil && !errors.Is(err, ErrNotOfficer) {
		return SubmitMemoLandPage{}, err
	}
	var assigned *SubmitMemoLandOfficer
	if err == nil {
		assigned = &officer
	}
	lands, err := m.listLandsForStatus(ctx, username, companyID, 0, "")
	return SubmitMemoLandPage{Officer: assigned, Lands: lands}, err
}

func (m *SubmitMemoLandModel) Detail(ctx context.Context, username string, companyID, landID int64) (SubmitMemoLandDetail, error) {
	return m.detailForStatus(ctx, companyID, landID, 0)
}

func (m *SubmitMemoLandModel) detailForStatus(ctx context.Context, companyID, landID int64, status int) (SubmitMemoLandDetail, error) {
	var raw []byte
	err := m.DB.QueryRow(ctx, submitLandSelect+` AND l."id_Gis_Compensated_Lands"=$3`, companyID, status, landID).Scan(&raw)
	if err != nil {
		return SubmitMemoLandDetail{}, err
	}
	var land map[string]any
	if err = json.Unmarshal(raw, &land); err != nil {
		return SubmitMemoLandDetail{}, err
	}
	landCompanyID, ok := land["company_id"].(float64)
	if !ok {
		return SubmitMemoLandDetail{}, pgx.ErrNoRows
	}
	var memos []Memo
	if status == 0 {
		memos, err = (&MemoModel{DB: m.DB}).GetAll(ctx, int64(landCompanyID))
		if err != nil {
			return SubmitMemoLandDetail{}, err
		}
	}
	rows, err := m.DB.Query(ctx, `SELECT jsonb_build_object(
		'id',ap."id_Gis_Aquisition_Processes",'sequence_id',ap."Gis_Aquisition_Process_Sequences_id",
		'name',COALESCE(seq.name,''),'request_number',ap.request_number,'request_date',ap.request_date,
		'approval',ap.approval,'note',ap.note,'revision',ap.revision,'date_created',ap.date_created,
		'username',COALESCE(ap.username,''),'employee_name',COALESCE(employee.name,ap.username,''))
		FROM public."Gis_Aquisition_Processes" ap
		LEFT JOIN public."Gis_Aquisition_Process_Sequences" seq ON seq."id_Gis_Aquisition_Process_Sequences"=ap."Gis_Aquisition_Process_Sequences_id"
		LEFT JOIN LATERAL (SELECT e.name FROM public."Gis_Employees" e WHERE LOWER(BTRIM(e.username))=LOWER(BTRIM(ap.username)) ORDER BY e."id_Gis_Employees" DESC LIMIT 1) employee ON TRUE
		WHERE ap."Gis_Compensated_Lands_id"=$1
		ORDER BY seq.sequence NULLS LAST,ap."id_Gis_Aquisition_Processes" DESC`, landID)
	if err != nil {
		return SubmitMemoLandDetail{}, err
	}
	history, err := scanJSONRows(rows)
	return SubmitMemoLandDetail{Land: land, Memos: memos, History: history}, err
}

func (m *SubmitMemoLandModel) Map(ctx context.Context, username string, companyID, landID int64) (string, error) {
	detail, err := m.Detail(ctx, username, companyID, landID)
	if err != nil {
		return "", err
	}
	status, ok := detail.Land["status"].(float64)
	if !ok {
		return "", pgx.ErrNoRows
	}
	return (&landModel.CompensatedLandModel{DB: m.DB, Status: int64(status)}).GetMap(ctx, int64(detail.Land["company_id"].(float64)), landID)
}

func (m *SubmitMemoLandModel) Submit(ctx context.Context, username string, companyID, landID int64, in SubmitMemoLandInput) error {
	officer, err := m.Officer(ctx, username, companyID)
	if err != nil {
		return err
	}
	if in.MemoID <= 0 {
		return fmt.Errorf("memo request wajib dipilih")
	}
	memo, err := (&MemoModel{DB: m.DB}).GetByID(ctx, officer.CompanyID, in.MemoID)
	if err != nil {
		return err
	}
	tx, err := m.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// Serialize submissions for the same land, including requests from two officers.
	var status int
	err = tx.QueryRow(ctx, `SELECT l.status FROM public."Gis_Compensated_Lands" l
		LEFT JOIN public."Gis_Aquistion_Divisions" d ON d."id_Gis_Aqcuistion_Divisions"=l."Gis_Aquistion_Divisions_id"
		LEFT JOIN public."Gis_Potential_Lands" p ON p."id_Gis_Potential_Lands"=d."Gis_Potential_Lands_id"
		LEFT JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id"
		LEFT JOIN public."Gis_Land_Owners" o ON o."id_Gis_Land_Owners"=l."Gis_Land_Owners_id"
		WHERE l."id_Gis_Compensated_Lands"=$1 AND COALESCE(lp."Gis_Companies_id",o."Gis_Companies_id")=$2 FOR UPDATE OF l`, landID, officer.CompanyID).Scan(&status)
	if err != nil {
		return err
	}
	if status != 0 {
		return ErrAlreadySubmitted
	}
	var exists bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public."Gis_Aquisition_Processes" WHERE "Gis_Compensated_Lands_id"=$1 AND "Gis_Aquisition_Process_Sequences_id"=1)`, landID).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return ErrAlreadySubmitted
	}
	// Legacy process IDs have no database sequence. The table lock protects MAX+1.
	if _, err = tx.Exec(ctx, `LOCK TABLE public."Gis_Aquisition_Processes" IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return err
	}
	var id int64
	err = tx.QueryRow(ctx, `SELECT COALESCE(MAX("id_Gis_Aquisition_Processes"),0)+1 FROM public."Gis_Aquisition_Processes"`).Scan(&id)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO public."Gis_Aquisition_Processes"
		("id_Gis_Aquisition_Processes","Gis_Aquisition_Process_Sequences_id","Gis_Compensated_Lands_id",
		request_number,request_date,approval,note,date_created,created_by,username,revision)
		VALUES ($1,1,$2,$3,$4::timestamp,1,$5,NOW(),$6,$6,1)`, id, landID, memo.MemoNumber, memo.MemoDate, strings.TrimSpace(in.Note), username)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE public."Gis_Compensated_Lands" SET status=1,"Gis_Aquisition_memo_id"=$2,date_modified=date_trunc('second',NOW()),modified_by=$3 WHERE "id_Gis_Compensated_Lands"=$1`, landID, fmt.Sprint(in.MemoID), username)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
