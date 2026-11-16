package acquisition

import (
	"context"
	"errors"
	"strings"

	landModel "gis-app/backend/internal/model/lands"

	"github.com/jackc/pgx/v5"
)

const (
	gisSequenceID       = 2
	legalSequenceID     = 3
	pdSequenceID        = 5
	operationSequenceID = 10
)

var ErrParallelAlreadyProcessed = errors.New("approval untuk lahan ini sudah tercatat")
var ErrParallelNotReady = errors.New("lahan belum siap untuk approval ini")

type ParallelApprovalStage struct {
	Name          string
	SequenceID    int
	CompanyScoped bool
}

var GISApprovalStage = ParallelApprovalStage{Name: "GIS Approval", SequenceID: gisSequenceID}
var LegalApprovalStage = ParallelApprovalStage{Name: "Legal Approval", SequenceID: legalSequenceID}
var OperationControlStage = ParallelApprovalStage{Name: "Operation Control Approval", SequenceID: operationSequenceID, CompanyScoped: true}
var PDApprovalStage = ParallelApprovalStage{Name: "PD Approval", SequenceID: pdSequenceID, CompanyScoped: true}

func (s ParallelApprovalStage) valid() bool {
	return s.SequenceID == gisSequenceID || s.SequenceID == legalSequenceID || s.SequenceID == operationSequenceID || s.SequenceID == pdSequenceID
}

func (s ParallelApprovalStage) status() int {
	if s.SequenceID == pdSequenceID {
		return 5
	}
	if s.SequenceID == operationSequenceID {
		return 4
	}
	return 3
}

func (s ParallelApprovalStage) nextStatus(approved, otherApproved bool) int {
	if !approved {
		return 99
	}
	if s.CompanyScoped {
		return s.status() + 1
	}
	if otherApproved {
		return 4
	}
	return 3
}

func (m *SubmitMemoLandModel) ParallelList(ctx context.Context, username string, companyID int64, stage ParallelApprovalStage) (SubmitMemoLandPage, error) {
	if !stage.valid() {
		return SubmitMemoLandPage{}, ErrParallelNotReady
	}
	var assigned *SubmitMemoLandOfficer
	officer, err := m.officerForSequence(ctx, username, companyID, stage.SequenceID)
	if err != nil && !errors.Is(err, ErrNotOfficer) {
		return SubmitMemoLandPage{}, err
	}
	if err == nil {
		assigned = &officer
	}
	extraWhere := ""
	// Company-scoped stages follow the land status after the preceding approval.
	if !stage.CompanyScoped {
		extraWhere = ` AND EXISTS (
			SELECT 1 FROM public."Gis_Aquisition_Processes" gm
			WHERE gm."Gis_Compensated_Lands_id"=l."id_Gis_Compensated_Lands"
			AND gm."Gis_Aquisition_Process_Sequences_id"=$3 AND gm.approval=1)`
	}
	var lands []map[string]any
	if stage.CompanyScoped {
		lands, err = m.listLandsForStatus(ctx, username, companyID, stage.status(), extraWhere)
	} else {
		lands, err = m.listLandsForStatus(ctx, username, companyID, stage.status(), extraWhere, gmSequenceID)
	}
	return SubmitMemoLandPage{Officer: assigned, Lands: lands}, err
}

func (m *SubmitMemoLandModel) ParallelDetail(ctx context.Context, username string, companyID, landID int64, stage ParallelApprovalStage) (SubmitMemoLandDetail, error) {
	if !stage.valid() {
		return SubmitMemoLandDetail{}, ErrParallelNotReady
	}
	detail, err := m.detailForStatus(ctx, companyID, landID, stage.status())
	if err != nil {
		return detail, err
	}
	if stage.CompanyScoped {
		return detail, nil
	}
	approved := map[int]bool{}
	for _, record := range detail.History {
		sequence, sequenceOK := record["sequence_id"].(float64)
		approval, approvalOK := record["approval"].(float64)
		if sequenceOK {
			if approvalOK && approval == 1 {
				approved[int(sequence)] = true
			}
		}
	}
	if !approved[gmSequenceID] {
		return SubmitMemoLandDetail{}, pgx.ErrNoRows
	}
	return detail, nil
}

func (m *SubmitMemoLandModel) ParallelMap(ctx context.Context, username string, companyID, landID int64, stage ParallelApprovalStage) (string, error) {
	detail, err := m.ParallelDetail(ctx, username, companyID, landID, stage)
	if err != nil {
		return "", err
	}
	return (&landModel.CompensatedLandModel{DB: m.DB, Status: int64(stage.status())}).GetMap(ctx, int64(detail.Land["company_id"].(float64)), landID)
}

func (m *SubmitMemoLandModel) ParallelApprove(ctx context.Context, username string, companyID, landID int64, stage ParallelApprovalStage, in RMApprovalInput) error {
	if !stage.valid() || in.Approved == nil {
		return ErrParallelNotReady
	}
	officer, err := m.officerForSequence(ctx, username, companyID, stage.SequenceID)
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
	if status != stage.status() {
		return ErrParallelNotReady
	}
	previousSequenceID := gmSequenceID
	if stage.SequenceID == operationSequenceID {
		previousSequenceID = legalSequenceID
	} else if stage.SequenceID == pdSequenceID {
		previousSequenceID = operationSequenceID
	}
	var requestNumber string
	var requestDate any
	err = tx.QueryRow(ctx, `SELECT COALESCE(request_number,''),request_date FROM public."Gis_Aquisition_Processes"
		WHERE "Gis_Compensated_Lands_id"=$1 AND "Gis_Aquisition_Process_Sequences_id"=$2 AND approval=1
		ORDER BY "id_Gis_Aquisition_Processes" DESC LIMIT 1`, landID, previousSequenceID).Scan(&requestNumber, &requestDate)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrParallelNotReady
	}
	if err != nil {
		return err
	}
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public."Gis_Aquisition_Processes" WHERE "Gis_Compensated_Lands_id"=$1 AND "Gis_Aquisition_Process_Sequences_id"=$2)`, landID, stage.SequenceID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return ErrParallelAlreadyProcessed
	}
	// The process table uses legacy IDs without a database sequence.
	if _, err = tx.Exec(ctx, `LOCK TABLE public."Gis_Aquisition_Processes" IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return err
	}
	var id int64
	if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX("id_Gis_Aquisition_Processes"),0)+1 FROM public."Gis_Aquisition_Processes"`).Scan(&id); err != nil {
		return err
	}
	approval := 0
	if *in.Approved {
		approval = 1
	}
	_, err = tx.Exec(ctx, `INSERT INTO public."Gis_Aquisition_Processes"
		("id_Gis_Aquisition_Processes","Gis_Aquisition_Process_Sequences_id","Gis_Compensated_Lands_id",
		request_number,request_date,approval,note,date_created,created_by,username,revision)
		VALUES ($1,$2,$3,$4,$5,$6,$7,NOW(),$8,$8,1)`, id, stage.SequenceID, landID, requestNumber, requestDate, approval, strings.TrimSpace(in.Note), username)
	if err != nil {
		return err
	}
	var otherApproved bool
	if *in.Approved && !stage.CompanyScoped {
		otherSequenceID := gisSequenceID
		if stage.SequenceID == gisSequenceID {
			otherSequenceID = legalSequenceID
		}
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public."Gis_Aquisition_Processes"
			WHERE "Gis_Compensated_Lands_id"=$1 AND "Gis_Aquisition_Process_Sequences_id"=$2 AND approval=1)`, landID, otherSequenceID).Scan(&otherApproved); err != nil {
			return err
		}
	}
	nextStatus := stage.nextStatus(*in.Approved, otherApproved)
	if _, err = tx.Exec(ctx, `UPDATE public."Gis_Compensated_Lands" SET status=$2,date_modified=date_trunc('second',NOW()),modified_by=$3 WHERE "id_Gis_Compensated_Lands"=$1`, landID, nextStatus, username); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
