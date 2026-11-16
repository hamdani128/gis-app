package administrator

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DocType struct {
	ID           int64      `json:"id"`
	TypeName     string     `json:"type_name"`
	Mandatory    int32      `json:"mandatory"`
	FileType     string     `json:"file_type"`
	SeqRelated   *int32     `json:"seq_related"`
	TypeDocGroup *string    `json:"type_doc_group"`
	CreatedBy    *string    `json:"created_by"`
	ModifiedBy   *string    `json:"modified_by"`
	DateModified *string    `json:"date_modified"`
	DateCreated  *time.Time `json:"date_created"`
}

type DocTypeInput struct {
	TypeName     string  `json:"type_name"`
	Mandatory    int32   `json:"mandatory"`
	FileType     string  `json:"file_type"`
	SeqRelated   *int32  `json:"seq_related"`
	TypeDocGroup *string `json:"type_doc_group"`
}

type DocTypeModel struct {
	DB *pgxpool.Pool
}

const docTypeTable = `public."Gis_Supported_Doc_Types"`

const docTypeColumns = `
	"idGis_Supported_Doc_Types", type_name, mandatory_, file_type,
	seq_related, type_doc_group, created_by, modified_by, date_modified, date_created
`

func scanDocType(row pgx.Row) (DocType, error) {
	var item DocType
	err := row.Scan(
		&item.ID,
		&item.TypeName,
		&item.Mandatory,
		&item.FileType,
		&item.SeqRelated,
		&item.TypeDocGroup,
		&item.CreatedBy,
		&item.ModifiedBy,
		&item.DateModified,
		&item.DateCreated,
	)
	return item, err
}

func (m *DocTypeModel) GetAll(ctx context.Context) ([]DocType, error) {
	rows, err := m.DB.Query(ctx, `
		SELECT "idGis_Supported_Doc_Types", COALESCE(type_name, ''),
		       COALESCE(mandatory_, 0), COALESCE(file_type, ''), seq_related,
		       type_doc_group, created_by, modified_by, date_modified, date_created
		FROM `+docTypeTable+`
		ORDER BY "idGis_Supported_Doc_Types" DESC
		LIMIT 500
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]DocType, 0)
	for rows.Next() {
		item, err := scanDocType(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *DocTypeModel) GetByID(ctx context.Context, id int64) (DocType, error) {
	return scanDocType(m.DB.QueryRow(ctx, `
		SELECT "idGis_Supported_Doc_Types", COALESCE(type_name, ''),
		       COALESCE(mandatory_, 0), COALESCE(file_type, ''), seq_related,
		       type_doc_group, created_by, modified_by, date_modified, date_created
		FROM `+docTypeTable+`
		WHERE "idGis_Supported_Doc_Types" = $1
	`, id))
}

// Create generates the legacy integer ID safely because this table has no
// identity or default sequence configured.
func (m *DocTypeModel) Create(ctx context.Context, input DocTypeInput) (DocType, error) {
	tx, err := m.DB.Begin(ctx)
	if err != nil {
		return DocType{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `LOCK TABLE `+docTypeTable+` IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return DocType{}, err
	}

	var nextID int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX("idGis_Supported_Doc_Types"), 0) + 1
		FROM `+docTypeTable+`
	`).Scan(&nextID); err != nil {
		return DocType{}, err
	}

	item, err := scanDocType(tx.QueryRow(ctx, `
		INSERT INTO `+docTypeTable+` (
			"idGis_Supported_Doc_Types", type_name, mandatory_, file_type,
			seq_related, type_doc_group, created_by, date_created
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING `+docTypeColumns,
		nextID,
		input.TypeName,
		input.Mandatory,
		input.FileType,
		input.SeqRelated,
		input.TypeDocGroup,
		"local-dev",
	))
	if err != nil {
		return DocType{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DocType{}, err
	}
	return item, nil
}

func (m *DocTypeModel) Update(ctx context.Context, id int64, input DocTypeInput) (DocType, error) {
	return scanDocType(m.DB.QueryRow(ctx, `
		UPDATE `+docTypeTable+`
		SET type_name = $1,
		    mandatory_ = $2,
		    file_type = $3,
		    seq_related = $4,
		    type_doc_group = $5,
		    modified_by = $6,
		    date_modified = NOW()::text
		WHERE "idGis_Supported_Doc_Types" = $7
		RETURNING `+docTypeColumns,
		input.TypeName,
		input.Mandatory,
		input.FileType,
		input.SeqRelated,
		input.TypeDocGroup,
		"local-dev",
		id,
	))
}

func (m *DocTypeModel) Delete(ctx context.Context, id int64) error {
	result, err := m.DB.Exec(ctx, `
		DELETE FROM `+docTypeTable+` WHERE "idGis_Supported_Doc_Types" = $1
	`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
