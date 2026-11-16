package acquisition

import (
	"context"
	"errors"
	"strings"

	landModel "gis-app/backend/internal/model/lands"

	"github.com/jackc/pgx/v5"
)

const gmSequenceID = 8

var ErrGMAlreadyProcessed = errors.New("GM Approval untuk lahan ini sudah tercatat")
var ErrGMNotReady = errors.New("lahan belum siap untuk GM Approval")

type GMApprovalInput = RMApprovalInput

func (m *SubmitMemoLandModel) GMList(ctx context.Context, username string, companyID int64) (SubmitMemoLandPage, error) {
	officer, err := m.officerForSequence(ctx, username, companyID, gmSequenceID)
	if err != nil && !errors.Is(err, ErrNotOfficer) {
		return SubmitMemoLandPage{}, err
	}
	var assigned *SubmitMemoLandOfficer
	if err == nil {
		assigned = &officer
	}
	lands, err := m.listLandsForStatus(ctx, username, companyID, 2, ` AND EXISTS (
		SELECT 1 FROM public."Gis_Aquisition_Processes" previous
		WHERE previous."Gis_Compensated_Lands_id"=l."id_Gis_Compensated_Lands"
		AND previous."Gis_Aquisition_Process_Sequences_id"=$3 AND previous.approval=1
	)`, rmSequenceID)
	return SubmitMemoLandPage{Officer: assigned, Lands: lands}, err
}

func (m *SubmitMemoLandModel) GMDetail(ctx context.Context, username string, companyID, landID int64) (SubmitMemoLandDetail, error) {
	detail, err := m.detailForStatus(ctx, companyID, landID, 2)
	if err != nil {
		return detail, err
	}
	approvedByRM := false
	for _, record := range detail.History {
		sequence, sequenceOK := record["sequence_id"].(float64)
		approval, approvalOK := record["approval"].(float64)
		if sequenceOK && approvalOK && int(sequence) == rmSequenceID && approval == 1 {
			approvedByRM = true
			break
		}
	}
	if !approvedByRM {
		return SubmitMemoLandDetail{}, pgx.ErrNoRows
	}
	return detail, nil
}

func (m *SubmitMemoLandModel) GMMap(ctx context.Context, username string, companyID, landID int64) (string, error) {
	detail, err := m.GMDetail(ctx, username, companyID, landID)
	if err != nil {
		return "", err
	}
	return (&landModel.CompensatedLandModel{DB: m.DB, Status: 2}).GetMap(ctx, int64(detail.Land["company_id"].(float64)), landID)
}

func (m *SubmitMemoLandModel) GMApprove(ctx context.Context, username string, companyID, landID int64, in GMApprovalInput) error {
	officer, err := m.officerForSequence(ctx, username, companyID, gmSequenceID)
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
	if status != 2 {
		return ErrGMNotReady
	}
	var requestNumber string
	var requestDate any
	err = tx.QueryRow(ctx, `SELECT request_number,request_date FROM public."Gis_Aquisition_Processes"
		WHERE "Gis_Compensated_Lands_id"=$1 AND "Gis_Aquisition_Process_Sequences_id"=$2 AND approval=1
		ORDER BY "id_Gis_Aquisition_Processes" DESC LIMIT 1`, landID, rmSequenceID).Scan(&requestNumber, &requestDate)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrGMNotReady
	}
	if err != nil {
		return err
	}
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public."Gis_Aquisition_Processes" WHERE "Gis_Compensated_Lands_id"=$1 AND "Gis_Aquisition_Process_Sequences_id"=$2)`, landID, gmSequenceID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return ErrGMAlreadyProcessed
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
		approval, nextStatus = 1, 3
	}
	_, err = tx.Exec(ctx, `INSERT INTO public."Gis_Aquisition_Processes"
		("id_Gis_Aquisition_Processes","Gis_Aquisition_Process_Sequences_id","Gis_Compensated_Lands_id",
		request_number,request_date,approval,note,date_created,created_by,username,revision)
		VALUES ($1,$2,$3,$4,$5, $6,$7,NOW(),$8,$8,1)`, id, gmSequenceID, landID, requestNumber, requestDate, approval, strings.TrimSpace(in.Note), username)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE public."Gis_Compensated_Lands" SET status=$2,date_modified=date_trunc('second',NOW()),modified_by=$3 WHERE "id_Gis_Compensated_Lands"=$1`, landID, nextStatus, username); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
