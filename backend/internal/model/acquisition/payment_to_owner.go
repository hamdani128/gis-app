package acquisition

import (
	"context"
	"errors"
	"strings"
	"time"

	landModel "gis-app/backend/internal/model/lands"
	"github.com/jackc/pgx/v5"
)

const paymentToOwnerSequenceID = 7

var ErrPaymentToOwnerNotReady = errors.New("lahan belum siap untuk Payment to Owner")
var ErrPaymentToOwnerAlreadyProcessed = errors.New("Payment to Owner untuk lahan ini sudah tercatat")
var ErrInvalidPaymentToOwnerChoice = errors.New("pilihan Payment to Owner tidak valid")
var ErrPaymentToOwnerCancelReasonRequired = errors.New("alasan Cancel wajib diisi pada Note")

func hasCompletedPayment(history []map[string]any) bool {
	for _, record := range history {
		sequence, sequenceOK := record["sequence_id"].(float64)
		approval, approvalOK := record["approval"].(float64)
		if sequenceOK && approvalOK && int(sequence) == paymentSequenceID && approval == 1 {
			return true
		}
	}
	return false
}

func (m *SubmitMemoLandModel) PaymentToOwnerList(ctx context.Context, username string, companyID int64) (SubmitMemoLandPage, error) {
	var assigned *SubmitMemoLandOfficer
	officer, err := m.officerForSequence(ctx, username, companyID, paymentToOwnerSequenceID)
	if err != nil && !errors.Is(err, ErrNotOfficer) {
		return SubmitMemoLandPage{}, err
	}
	if err == nil {
		assigned = &officer
	}
	lands, err := m.listLandsForStatus(ctx, username, companyID, 7, ` AND EXISTS (
		SELECT 1 FROM public."Gis_Aquisition_Processes" payment
		WHERE payment."Gis_Compensated_Lands_id"=l."id_Gis_Compensated_Lands"
		AND payment."Gis_Aquisition_Process_Sequences_id"=6 AND payment.approval=1)`)
	return SubmitMemoLandPage{Officer: assigned, Lands: lands}, err
}

func (m *SubmitMemoLandModel) PaymentToOwnerDetail(ctx context.Context, companyID, landID int64) (SubmitMemoLandDetail, error) {
	detail, err := m.detailForStatus(ctx, companyID, landID, 7)
	if err != nil {
		return detail, err
	}
	if !hasCompletedPayment(detail.History) {
		return SubmitMemoLandDetail{}, pgx.ErrNoRows
	}
	return detail, nil
}

func (m *SubmitMemoLandModel) PaymentToOwnerMap(ctx context.Context, companyID, landID int64) (string, error) {
	detail, err := m.PaymentToOwnerDetail(ctx, companyID, landID)
	if err != nil {
		return "", err
	}
	return (&landModel.CompensatedLandModel{DB: m.DB, Status: 7}).GetMap(ctx, int64(detail.Land["company_id"].(float64)), landID)
}

func (m *SubmitMemoLandModel) SavePaymentToOwner(ctx context.Context, username string, companyID, landID int64, transferDate time.Time, note, choice string, documents []PaymentDocumentInput) error {
	choice = strings.ToLower(strings.TrimSpace(choice))
	if choice != "pay_to_owner" && choice != "cancel" {
		return ErrInvalidPaymentToOwnerChoice
	}
	if choice == "pay_to_owner" && len(documents) == 0 {
		return ErrPaymentDocumentsRequired
	}
	if choice == "cancel" && strings.TrimSpace(note) == "" {
		return ErrPaymentToOwnerCancelReasonRequired
	}
	approval, nextStatus := 1, 8
	if choice == "cancel" {
		approval, nextStatus = 0, 99
	}
	officer, err := m.officerForSequence(ctx, username, companyID, paymentToOwnerSequenceID)
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
	if status != 7 {
		return ErrPaymentToOwnerNotReady
	}
	var requestNumber string
	err = tx.QueryRow(ctx, `SELECT COALESCE(request_number,'') FROM public."Gis_Aquisition_Processes"
		WHERE "Gis_Compensated_Lands_id"=$1 AND "Gis_Aquisition_Process_Sequences_id"=6 AND approval=1
		ORDER BY "id_Gis_Aquisition_Processes" DESC LIMIT 1`, landID).Scan(&requestNumber)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPaymentToOwnerNotReady
	}
	if err != nil {
		return err
	}
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public."Gis_Aquisition_Processes"
		WHERE "Gis_Compensated_Lands_id"=$1 AND "Gis_Aquisition_Process_Sequences_id"=7)`, landID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return ErrPaymentToOwnerAlreadyProcessed
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
		VALUES ($1,7,$2,$3,$4,$5,$6,NOW(),$7,$7,1)`, processID, landID, requestNumber, transferDate, approval, strings.TrimSpace(note), username)
	if err != nil {
		return err
	}
	if len(documents) > 0 {
		if err = saveAcquisitionPaymentDocuments(ctx, tx, landID, username, "Payment", true, documents); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE public."Gis_Compensated_Lands" SET status=$2,date_modified=date_trunc('second',NOW()),modified_by=$3
		WHERE "id_Gis_Compensated_Lands"=$1`, landID, nextStatus, username); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
