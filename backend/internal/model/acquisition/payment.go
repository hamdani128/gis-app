package acquisition

import (
	"context"
	"errors"
	"strings"
	"time"

	landModel "gis-app/backend/internal/model/lands"

	"github.com/jackc/pgx/v5"
)

const paymentSequenceID = 6

var ErrPaymentNotReady = errors.New("lahan belum siap untuk Payment")
var ErrPaymentAlreadyProcessed = errors.New("Payment untuk lahan ini sudah tercatat")

func hasApprovedPD(history []map[string]any) bool {
	for _, record := range history {
		sequence, sequenceOK := record["sequence_id"].(float64)
		approval, approvalOK := record["approval"].(float64)
		if sequenceOK && approvalOK && int(sequence) == pdSequenceID && approval == 1 {
			return true
		}
	}
	return false
}

func (m *SubmitMemoLandModel) PaymentList(ctx context.Context, username string, companyID int64) (SubmitMemoLandPage, error) {
	var assigned *SubmitMemoLandOfficer
	officer, err := m.officerForSequence(ctx, username, companyID, paymentSequenceID)
	if err != nil && !errors.Is(err, ErrNotOfficer) {
		return SubmitMemoLandPage{}, err
	}
	if err == nil {
		assigned = &officer
	}
	lands, err := m.listLandsForStatus(ctx, username, companyID, 6, ` AND EXISTS (
		SELECT 1 FROM public."Gis_Aquisition_Processes" pd
		WHERE pd."Gis_Compensated_Lands_id"=l."id_Gis_Compensated_Lands"
		AND pd."Gis_Aquisition_Process_Sequences_id"=5 AND pd.approval=1)`)
	return SubmitMemoLandPage{Officer: assigned, Lands: lands}, err
}

func (m *SubmitMemoLandModel) PaymentDetail(ctx context.Context, companyID, landID int64) (SubmitMemoLandDetail, error) {
	detail, err := m.detailForStatus(ctx, companyID, landID, 6)
	if err != nil {
		return detail, err
	}
	if !hasApprovedPD(detail.History) {
		return SubmitMemoLandDetail{}, pgx.ErrNoRows
	}
	return detail, nil
}

func (m *SubmitMemoLandModel) PaymentDocumentDetail(ctx context.Context, companyID, landID int64) (SubmitMemoLandDetail, error) {
	for _, status := range []int{6, 7, 10} {
		detail, err := m.detailForStatus(ctx, companyID, landID, status)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return SubmitMemoLandDetail{}, err
		}
		if !hasApprovedPD(detail.History) {
			return SubmitMemoLandDetail{}, pgx.ErrNoRows
		}
		return detail, nil
	}
	return SubmitMemoLandDetail{}, pgx.ErrNoRows
}

func (m *SubmitMemoLandModel) PaymentMap(ctx context.Context, companyID, landID int64) (string, error) {
	detail, err := m.PaymentDetail(ctx, companyID, landID)
	if err != nil {
		return "", err
	}
	return (&landModel.CompensatedLandModel{DB: m.DB, Status: 6}).GetMap(ctx, int64(detail.Land["company_id"].(float64)), landID)
}

func (m *SubmitMemoLandModel) SavePayment(ctx context.Context, username string, companyID, landID int64, transferDate time.Time, note string, documents []PaymentDocumentInput) error {
	if len(documents) == 0 {
		return ErrPaymentDocumentsRequired
	}
	officer, err := m.officerForSequence(ctx, username, companyID, paymentSequenceID)
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
	if status != 6 {
		return ErrPaymentNotReady
	}
	var requestNumber string
	err = tx.QueryRow(ctx, `SELECT COALESCE(request_number,'') FROM public."Gis_Aquisition_Processes"
		WHERE "Gis_Compensated_Lands_id"=$1 AND "Gis_Aquisition_Process_Sequences_id"=$2 AND approval=1
		ORDER BY "id_Gis_Aquisition_Processes" DESC LIMIT 1`, landID, pdSequenceID).Scan(&requestNumber)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPaymentNotReady
	}
	if err != nil {
		return err
	}
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public."Gis_Aquisition_Processes"
		WHERE "Gis_Compensated_Lands_id"=$1 AND "Gis_Aquisition_Process_Sequences_id"=$2)`, landID, paymentSequenceID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return ErrPaymentAlreadyProcessed
	}
	if _, err = tx.Exec(ctx, `LOCK TABLE public."Gis_Aquisition_Processes" IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return err
	}
	var processID int64
	if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX("id_Gis_Aquisition_Processes"),0)+1 FROM public."Gis_Aquisition_Processes"`).Scan(&processID); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO public."Gis_Aquisition_Processes"
		("id_Gis_Aquisition_Processes","Gis_Aquisition_Process_Sequences_id","Gis_Compensated_Lands_id",
		request_number,request_date,approval,note,date_created,created_by,username,revision)
		VALUES ($1,$2,$3,$4,$5,1,$6,NOW(),$7,$7,1)`, processID, paymentSequenceID, landID, requestNumber, transferDate, strings.TrimSpace(note), username)
	if err != nil {
		return err
	}
	if err = savePaymentDocuments(ctx, tx, landID, username, documents); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE public."Gis_Compensated_Lands" SET status=7,date_modified=date_trunc('second',NOW()),modified_by=$2
		WHERE "id_Gis_Compensated_Lands"=$1`, landID, username); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
