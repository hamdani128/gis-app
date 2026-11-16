package acquisition

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrInvalidPaymentDocumentType = errors.New("jenis dokumen Payment tidak valid")
var ErrPaymentDocumentsRequired = errors.New("minimal satu dokumen Payment wajib ditambahkan")

type PaymentDocumentType struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FileType string `json:"file_type"`
}

type PaymentDocumentInput struct {
	TypeID   int64
	Title    string
	Note     string
	FileName string
}

type PaymentDocumentRecord struct {
	ID       int64      `json:"id"`
	TypeID   int64      `json:"type_id"`
	TypeName string     `json:"type_name"`
	Title    string     `json:"title"`
	Note     string     `json:"note"`
	FileName string     `json:"file_name"`
	Created  *time.Time `json:"date_created"`
}

type PaymentSupportedDocument struct {
	ID         int64      `json:"id"`
	TypeName   string     `json:"type_name"`
	Title      string     `json:"title"`
	Note       string     `json:"note"`
	FileName   string     `json:"file_name"`
	Group      string     `json:"doc_group"`
	Created    *time.Time `json:"date_created"`
	Available  bool       `json:"available"`
	LegacyURL  string     `json:"legacy_url,omitempty"`
	PreviewURL string     `json:"preview_url,omitempty"`
}

type PaymentMemo struct {
	ID         string `json:"id"`
	CompanyID  int64  `json:"company_id"`
	Number     string `json:"memo_number"`
	Date       string `json:"memo_date"`
	Note       string `json:"memo_note"`
	FileName   string `json:"memo_doc"`
	Found      bool   `json:"found"`
	Available  bool   `json:"available"`
	LegacyURL  string `json:"legacy_url,omitempty"`
	PreviewURL string `json:"preview_url,omitempty"`
}

func (m *SubmitMemoLandModel) PaymentDocumentTypes(ctx context.Context) ([]PaymentDocumentType, error) {
	return m.paymentDocumentTypes(ctx, false)
}

func (m *SubmitMemoLandModel) PaymentToOwnerDocumentTypes(ctx context.Context) ([]PaymentDocumentType, error) {
	return m.paymentDocumentTypes(ctx, true)
}

func (m *SubmitMemoLandModel) paymentDocumentTypes(ctx context.Context, includeMandatory bool) ([]PaymentDocumentType, error) {
	rows, err := m.DB.Query(ctx, `SELECT "idGis_Supported_Doc_Types", COALESCE(type_name,''), COALESCE(file_type,'')
		FROM public."Gis_Supported_Doc_Types"
		WHERE LOWER(TRIM(COALESCE(type_doc_group,'')))='payment'
			AND ($1::boolean OR COALESCE(mandatory_,0)=0)
		ORDER BY type_name, "idGis_Supported_Doc_Types"`, includeMandatory)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []PaymentDocumentType{}
	for rows.Next() {
		var item PaymentDocumentType
		if err := rows.Scan(&item.ID, &item.Name, &item.FileType); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func savePaymentDocuments(ctx context.Context, tx pgx.Tx, landID int64, username string, documents []PaymentDocumentInput) error {
	return saveAcquisitionPaymentDocuments(ctx, tx, landID, username, "Payment", false, documents)
}

func saveAcquisitionPaymentDocuments(ctx context.Context, tx pgx.Tx, landID int64, username, group string, includeMandatory bool, documents []PaymentDocumentInput) error {
	if len(documents) == 0 {
		return ErrPaymentDocumentsRequired
	}
	if _, err := tx.Exec(ctx, `LOCK TABLE public."Gis_Compensated_Land_Supported_Docs" IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return err
	}
	var nextID int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX("id_Gis_Supported_Docs"),0)+1 FROM public."Gis_Compensated_Land_Supported_Docs"`).Scan(&nextID); err != nil {
		return err
	}
	for _, document := range documents {
		var typeName string
		err := tx.QueryRow(ctx, `SELECT COALESCE(type_name,'') FROM public."Gis_Supported_Doc_Types"
			WHERE "idGis_Supported_Doc_Types"=$1 AND LOWER(TRIM(COALESCE(type_doc_group,'')))='payment'
			AND ($2::boolean OR COALESCE(mandatory_,0)=0)`, document.TypeID, includeMandatory).Scan(&typeName)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidPaymentDocumentType
		}
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO public."Gis_Compensated_Land_Supported_Docs"
			("id_Gis_Supported_Docs","Gis_Compensated_Lands_id","Gis_Supported_Doc_Types_id",
			name,doc_file,doc_note,date_created,created_by,doc_type,doc_group)
			VALUES ($1,$2,$3,$4,$5,$6,NOW(),$7,$8,$9)`, nextID, landID, document.TypeID,
			strings.TrimSpace(document.Title), document.FileName, strings.TrimSpace(document.Note), username, typeName, group)
		if err != nil {
			return err
		}
		nextID++
	}
	return nil
}

func (m *SubmitMemoLandModel) PaymentDocuments(ctx context.Context, landID int64) ([]PaymentDocumentRecord, error) {
	rows, err := m.DB.Query(ctx, `SELECT "id_Gis_Supported_Docs", COALESCE("Gis_Supported_Doc_Types_id",0),
		COALESCE(doc_type,''), COALESCE(name,''), COALESCE(doc_note,''), COALESCE(doc_file,''), date_created
		FROM public."Gis_Compensated_Land_Supported_Docs"
		WHERE "Gis_Compensated_Lands_id"=$1 AND doc_group='Payment'
		ORDER BY "id_Gis_Supported_Docs"`, landID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []PaymentDocumentRecord{}
	for rows.Next() {
		var item PaymentDocumentRecord
		if err := rows.Scan(&item.ID, &item.TypeID, &item.TypeName, &item.Title, &item.Note, &item.FileName, &item.Created); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *SubmitMemoLandModel) PaymentSupportedDocuments(ctx context.Context, landID int64) ([]PaymentSupportedDocument, error) {
	rows, err := m.DB.Query(ctx, `SELECT d."id_Gis_Supported_Docs",
		COALESCE(NULLIF(BTRIM(t.type_name),''), NULLIF(BTRIM(d.doc_type),''),''),
		COALESCE(d.name,''), COALESCE(d.doc_note,''), COALESCE(d.doc_file,''),
		COALESCE(d.doc_group,''), d.date_created
		FROM public."Gis_Compensated_Land_Supported_Docs" d
		LEFT JOIN public."Gis_Supported_Doc_Types" t
			ON t."idGis_Supported_Doc_Types"=d."Gis_Supported_Doc_Types_id"
		WHERE d."Gis_Compensated_Lands_id"=$1
		ORDER BY d."id_Gis_Supported_Docs"`, landID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []PaymentSupportedDocument{}
	for rows.Next() {
		var item PaymentSupportedDocument
		if err := rows.Scan(&item.ID, &item.TypeName, &item.Title, &item.Note, &item.FileName, &item.Group, &item.Created); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *SubmitMemoLandModel) PaymentMemo(ctx context.Context, landID int64) (PaymentMemo, error) {
	var item PaymentMemo
	err := m.DB.QueryRow(ctx, `SELECT COALESCE(BTRIM(l."Gis_Aquisition_memo_id"),''),
		COALESCE(m."Gis_Companies_id",0),
		COALESCE(m.memo_number,''), COALESCE(to_char(m.memo_date,'YYYY-MM-DD'),''),
		COALESCE(m.memo_note,''), COALESCE(m.memo_doc,''),
		m."idGis_Aquisition_memo" IS NOT NULL
		FROM public."Gis_Compensated_Lands" l
		LEFT JOIN public."Gis_Aquisition_memo" m ON m."idGis_Aquisition_memo"::text=BTRIM(l."Gis_Aquisition_memo_id")
		WHERE l."id_Gis_Compensated_Lands"=$1`, landID).
		Scan(&item.ID, &item.CompanyID, &item.Number, &item.Date, &item.Note, &item.FileName, &item.Found)
	return item, err
}
