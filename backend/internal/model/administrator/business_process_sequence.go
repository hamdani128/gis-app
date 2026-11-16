package administrator

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BusinessProcessSequence struct {
	ID                  *int32     `json:"id"`
	ProcessRoleID       *int32     `json:"process_role_id"`
	ProcessRole         string     `json:"process_role"`
	Name                string     `json:"name"`
	Sequence            *int32     `json:"sequence"`
	DocumentRequirement *string    `json:"document_requirement"`
	CreatedBy           *string    `json:"created_by"`
	DateCreated         *time.Time `json:"date_created"`
	ModifiedBy          *string    `json:"modified_by"`
	DateModified        *string    `json:"date_modified"`
}

type BusinessProcessSequenceInput struct {
	ProcessRoleID       *int32  `json:"process_role_id"`
	Name                string  `json:"name"`
	Sequence            *int32  `json:"sequence"`
	DocumentRequirement *string `json:"document_requirement"`
}

type BusinessProcessSequenceModel struct {
	DB *pgxpool.Pool
}

const businessProcessSequenceTable = `public."Gis_Aquisition_Process_Sequences"`
const businessProcessSequenceIDColumn = `"id_Gis_Aquisition_Process_Sequences"`
const businessProcessSequenceRoleIDColumn = `"Gis_Aquisition_Roles_id"`

const businessProcessSequenceSelect = `
	SELECT s.` + businessProcessSequenceIDColumn + `, s.` + businessProcessSequenceRoleIDColumn + `,
	       COALESCE(r.role_name, ''), COALESCE(s.name, ''), s.sequence,
	       s.seq_doc_grup, s.created_by, s.date_created, s.modified_by, s.date_modified
	FROM ` + businessProcessSequenceTable + ` s
	LEFT JOIN public."Gis_Process_Roles" r
	       ON r."id_Gis_Aquisition_Roles" = s.` + businessProcessSequenceRoleIDColumn

func scanBusinessProcessSequence(row pgx.Row) (BusinessProcessSequence, error) {
	var item BusinessProcessSequence
	err := row.Scan(
		&item.ID,
		&item.ProcessRoleID,
		&item.ProcessRole,
		&item.Name,
		&item.Sequence,
		&item.DocumentRequirement,
		&item.CreatedBy,
		&item.DateCreated,
		&item.ModifiedBy,
		&item.DateModified,
	)
	return item, err
}

func (m *BusinessProcessSequenceModel) GetAll(ctx context.Context) ([]BusinessProcessSequence, error) {
	rows, err := m.DB.Query(ctx, businessProcessSequenceSelect+`
		ORDER BY s.sequence NULLS LAST, s.name NULLS LAST,
		         s.`+businessProcessSequenceIDColumn+` DESC NULLS LAST
		LIMIT 2000
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]BusinessProcessSequence, 0)
	for rows.Next() {
		item, err := scanBusinessProcessSequence(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *BusinessProcessSequenceModel) GetByID(ctx context.Context, id int64) (BusinessProcessSequence, error) {
	return scanBusinessProcessSequence(m.DB.QueryRow(ctx, businessProcessSequenceSelect+`
		WHERE s.`+businessProcessSequenceIDColumn+` = $1
	`, id))
}

func (m *BusinessProcessSequenceModel) RoleExists(ctx context.Context, id int32) (bool, error) {
	var exists bool
	err := m.DB.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM public."Gis_Process_Roles" WHERE "id_Gis_Aquisition_Roles"=$1
		)
	`, id).Scan(&exists)
	return exists, err
}

// Create allocates an ID because this legacy table has no identity/default sequence.
func (m *BusinessProcessSequenceModel) Create(ctx context.Context, input BusinessProcessSequenceInput) (BusinessProcessSequence, error) {
	tx, err := m.DB.Begin(ctx)
	if err != nil {
		return BusinessProcessSequence{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `LOCK TABLE `+businessProcessSequenceTable+` IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return BusinessProcessSequence{}, err
	}
	var nextID int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(`+businessProcessSequenceIDColumn+`), 0) + 1 FROM `+businessProcessSequenceTable).Scan(&nextID); err != nil {
		return BusinessProcessSequence{}, err
	}
	var id int32
	err = tx.QueryRow(ctx, `
		INSERT INTO `+businessProcessSequenceTable+` (
			`+businessProcessSequenceIDColumn+`, `+businessProcessSequenceRoleIDColumn+`, name,
			sequence, seq_doc_grup, created_by, date_created
		)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING `+businessProcessSequenceIDColumn+`
	`, nextID, input.ProcessRoleID, input.Name, input.Sequence, input.DocumentRequirement, "local-dev").Scan(&id)
	if err != nil {
		return BusinessProcessSequence{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return BusinessProcessSequence{}, err
	}
	return m.GetByID(ctx, int64(id))
}

func (m *BusinessProcessSequenceModel) Update(ctx context.Context, id int64, input BusinessProcessSequenceInput) (BusinessProcessSequence, error) {
	result, err := m.DB.Exec(ctx, `
		UPDATE `+businessProcessSequenceTable+` s
		SET `+businessProcessSequenceRoleIDColumn+`=$1, name=$2, sequence=$3,
		    seq_doc_grup=$4, modified_by=$5, date_modified=NOW()::text
		WHERE s.`+businessProcessSequenceIDColumn+`=$6
	`, input.ProcessRoleID, input.Name, input.Sequence, input.DocumentRequirement, "local-dev", id)
	if err != nil {
		return BusinessProcessSequence{}, err
	}
	if result.RowsAffected() == 0 {
		return BusinessProcessSequence{}, pgx.ErrNoRows
	}
	return m.GetByID(ctx, id)
}

func (m *BusinessProcessSequenceModel) Delete(ctx context.Context, id int64) error {
	result, err := m.DB.Exec(ctx, `DELETE FROM `+businessProcessSequenceTable+` WHERE `+businessProcessSequenceIDColumn+`=$1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func NormalizeBusinessProcessSequenceInput(input *BusinessProcessSequenceInput) {
	input.Name = strings.TrimSpace(input.Name)
	if input.DocumentRequirement != nil {
		value := strings.TrimSpace(*input.DocumentRequirement)
		if value == "" {
			input.DocumentRequirement = nil
		} else {
			input.DocumentRequirement = &value
		}
	}
}
