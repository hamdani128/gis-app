package acquisition

import (
	"context"
	"errors"
	"strings"

	landModel "gis-app/backend/internal/model/lands"

	"github.com/jackc/pgx/v5"
)

const rmSequenceID = 9

var ErrRMAlreadyProcessed = errors.New("RM Approval untuk lahan ini sudah tercatat")
var ErrRMNotReady = errors.New("lahan belum siap untuk RM Approval")

type RMApprovalInput struct {
	Approved *bool  `json:"approved"`
	Note     string `json:"note"`
}

func (m *SubmitMemoLandModel) RMList(ctx context.Context, username string, companyID int64) (SubmitMemoLandPage, error) {
	officer, err := m.officerForSequence(ctx, username, companyID, rmSequenceID)
	if err != nil && !errors.Is(err, ErrNotOfficer) {
		return SubmitMemoLandPage{}, err
	}
	var assigned *SubmitMemoLandOfficer
	if err == nil {
		assigned = &officer
	}
	lands, err := m.listLandsForStatus(ctx, username, companyID, 1, ` AND COALESCE(submit.submit_count,0)>0`)
	return SubmitMemoLandPage{Officer: assigned, Lands: lands}, err
}

func (m *SubmitMemoLandModel) RMDetail(ctx context.Context, username string, companyID, landID int64) (SubmitMemoLandDetail, error) {
	detail, err := m.detailForStatus(ctx, companyID, landID, 1)
	if err != nil {
		return detail, err
	}
	if count, ok := detail.Land["submit_count"].(float64); !ok || count == 0 {
		return SubmitMemoLandDetail{}, pgx.ErrNoRows
	}
	return detail, nil
}

func (m *SubmitMemoLandModel) RMMap(ctx context.Context, username string, companyID, landID int64) (string, error) {
	detail, err := m.RMDetail(ctx, username, companyID, landID)
	if err != nil {
		return "", err
	}
	return (&landModel.CompensatedLandModel{DB: m.DB, Status: 1}).GetMap(ctx, int64(detail.Land["company_id"].(float64)), landID)
}

func (m *SubmitMemoLandModel) RMApprove(ctx context.Context, username string, companyID, landID int64, in RMApprovalInput) error {
	officer, err := m.officerForSequence(ctx, username, companyID, rmSequenceID)
	if err != nil {
		return err
	}
	tx, err := m.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
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
	if status != 1 {
		return ErrRMNotReady
	}
	var requestNumber string
	var requestDate any
	err = tx.QueryRow(ctx, `SELECT request_number,request_date FROM public."Gis_Aquisition_Processes"
		WHERE "Gis_Compensated_Lands_id"=$1 AND "Gis_Aquisition_Process_Sequences_id"=$2 AND approval=1
		ORDER BY "id_Gis_Aquisition_Processes" DESC LIMIT 1`, landID, submitSequenceID).Scan(&requestNumber, &requestDate)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRMNotReady
	}
	if err != nil {
		return err
	}
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public."Gis_Aquisition_Processes" WHERE "Gis_Compensated_Lands_id"=$1 AND "Gis_Aquisition_Process_Sequences_id"=$2)`, landID, rmSequenceID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return ErrRMAlreadyProcessed
	}
	// Legacy process IDs have no database sequence. Serialize MAX+1 allocations.
	if _, err = tx.Exec(ctx, `LOCK TABLE public."Gis_Aquisition_Processes" IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return err
	}
	var id int64
	if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX("id_Gis_Aquisition_Processes"),0)+1 FROM public."Gis_Aquisition_Processes"`).Scan(&id); err != nil {
		return err
	}
	approval, nextStatus := 0, 99
	if *in.Approved {
		approval, nextStatus = 1, 2
	}
	_, err = tx.Exec(ctx, `INSERT INTO public."Gis_Aquisition_Processes"
		("id_Gis_Aquisition_Processes","Gis_Aquisition_Process_Sequences_id","Gis_Compensated_Lands_id",
		request_number,request_date,approval,note,date_created,created_by,username,revision)
		VALUES ($1,$2,$3,$4,$5, $6,$7,NOW(),$8,$8,1)`, id, rmSequenceID, landID, requestNumber, requestDate, approval, strings.TrimSpace(in.Note), username)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE public."Gis_Compensated_Lands" SET status=$2,date_modified=date_trunc('second',NOW()),modified_by=$3 WHERE "id_Gis_Compensated_Lands"=$1`, landID, nextStatus, username); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
