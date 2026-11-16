package administrator

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BusinessProcessRole struct {
	ID           *int32     `json:"id"`
	RoleName     string     `json:"role_name"`
	ValidFrom    *time.Time `json:"valid_from"`
	ValidUntil   *string    `json:"valid_until"`
	Note         *string    `json:"note"`
	CreatedBy    *string    `json:"created_by"`
	DateCreated  *time.Time `json:"date_created"`
	ModifiedBy   *string    `json:"modified_by"`
	DateModified *time.Time `json:"date_modified"`
}

type BusinessProcessRoleInput struct {
	RoleName   string  `json:"role_name"`
	ValidFrom  *string `json:"valid_from"`
	ValidUntil *string `json:"valid_until"`
	Note       *string `json:"note"`
}

type BusinessProcessRoleModel struct {
	DB *pgxpool.Pool
}

const businessProcessRoleTable = `public."Gis_Process_Roles"`
const businessProcessRoleIDColumn = `"id_Gis_Aquisition_Roles"`

const businessProcessRoleSelect = `
	SELECT ` + businessProcessRoleIDColumn + `, COALESCE(role_name, ''), valid_from,
	       valid_until, note, created_by, date_created, modified_by, date_modified
	FROM ` + businessProcessRoleTable

func scanBusinessProcessRole(row pgx.Row) (BusinessProcessRole, error) {
	var item BusinessProcessRole
	err := row.Scan(
		&item.ID,
		&item.RoleName,
		&item.ValidFrom,
		&item.ValidUntil,
		&item.Note,
		&item.CreatedBy,
		&item.DateCreated,
		&item.ModifiedBy,
		&item.DateModified,
	)
	return item, err
}

func (m *BusinessProcessRoleModel) GetAll(ctx context.Context) ([]BusinessProcessRole, error) {
	rows, err := m.DB.Query(ctx, businessProcessRoleSelect+`
		ORDER BY role_name NULLS LAST, `+businessProcessRoleIDColumn+` DESC NULLS LAST
		LIMIT 2000
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]BusinessProcessRole, 0)
	for rows.Next() {
		item, err := scanBusinessProcessRole(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *BusinessProcessRoleModel) GetByID(ctx context.Context, id int64) (BusinessProcessRole, error) {
	return scanBusinessProcessRole(m.DB.QueryRow(ctx, businessProcessRoleSelect+`
		WHERE `+businessProcessRoleIDColumn+` = $1
	`, id))
}

// Create assigns the legacy integer ID because the table does not define an identity/default.
func (m *BusinessProcessRoleModel) Create(ctx context.Context, input BusinessProcessRoleInput) (BusinessProcessRole, error) {
	tx, err := m.DB.Begin(ctx)
	if err != nil {
		return BusinessProcessRole{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `LOCK TABLE `+businessProcessRoleTable+` IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return BusinessProcessRole{}, err
	}
	var nextID int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(`+businessProcessRoleIDColumn+`), 0) + 1 FROM `+businessProcessRoleTable).Scan(&nextID); err != nil {
		return BusinessProcessRole{}, err
	}

	item, err := scanBusinessProcessRole(tx.QueryRow(ctx, `
		INSERT INTO `+businessProcessRoleTable+` (
			`+businessProcessRoleIDColumn+`, role_name, valid_from, valid_until, note, created_by, date_created
		)
		VALUES ($1, $2, $3::timestamp, $4, $5, $6, NOW())
		RETURNING `+businessProcessRoleIDColumn+`, COALESCE(role_name, ''), valid_from,
		          valid_until, note, created_by, date_created, modified_by, date_modified
	`, nextID, input.RoleName, input.ValidFrom, input.ValidUntil, input.Note, "local-dev"))
	if err != nil {
		return BusinessProcessRole{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return BusinessProcessRole{}, err
	}
	return item, nil
}

func (m *BusinessProcessRoleModel) Update(ctx context.Context, id int64, input BusinessProcessRoleInput) (BusinessProcessRole, error) {
	return scanBusinessProcessRole(m.DB.QueryRow(ctx, `
		UPDATE `+businessProcessRoleTable+`
		SET role_name = $1, valid_from = $2::timestamp, valid_until = $3, note = $4,
		    modified_by = $5, date_modified = NOW()
		WHERE `+businessProcessRoleIDColumn+` = $6
		RETURNING `+businessProcessRoleIDColumn+`, COALESCE(role_name, ''), valid_from,
		          valid_until, note, created_by, date_created, modified_by, date_modified
	`, input.RoleName, input.ValidFrom, input.ValidUntil, input.Note, "local-dev", id))
}

func (m *BusinessProcessRoleModel) Delete(ctx context.Context, id int64) error {
	result, err := m.DB.Exec(ctx, `DELETE FROM `+businessProcessRoleTable+` WHERE `+businessProcessRoleIDColumn+` = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func NormalizeBusinessProcessRoleInput(input *BusinessProcessRoleInput) {
	input.RoleName = strings.TrimSpace(input.RoleName)
	if input.ValidFrom != nil {
		value := strings.TrimSpace(*input.ValidFrom)
		if value == "" {
			input.ValidFrom = nil
		} else {
			input.ValidFrom = &value
		}
	}
	if input.ValidUntil != nil {
		value := strings.TrimSpace(*input.ValidUntil)
		if value == "" {
			input.ValidUntil = nil
		} else {
			input.ValidUntil = &value
		}
	}
	if input.Note != nil {
		value := strings.TrimSpace(*input.Note)
		if value == "" {
			input.Note = nil
		} else {
			input.Note = &value
		}
	}
}
