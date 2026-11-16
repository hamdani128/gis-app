package administrator

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RefValueSetting struct {
	ID           int64      `json:"id"`
	Domain       string     `json:"domain"`
	LowValue     string     `json:"low_value"`
	Meaning      *string    `json:"meaning"`
	DateCreated  *time.Time `json:"date_created"`
	CreatedBy    *string    `json:"created_by"`
	DateModified *string    `json:"date_modified"`
	ModifiedBy   *string    `json:"modified_by"`
}

type RefValueSettingInput struct {
	Domain   string  `json:"domain"`
	LowValue string  `json:"low_value"`
	Meaning  *string `json:"meaning"`
}

type RefValueSettingModel struct {
	DB *pgxpool.Pool
}

const refColumns = `
	id, domain, low_value, meaning,
	date_created, created_by, date_modified, modified_by
`

// Digunakan untuk membaca satu baris hasil query.
func scanRef(row pgx.Row) (RefValueSetting, error) {
	var item RefValueSetting

	err := row.Scan(
		&item.ID,
		&item.Domain,
		&item.LowValue,
		&item.Meaning,
		&item.DateCreated,
		&item.CreatedBy,
		&item.DateModified,
		&item.ModifiedBy,
	)

	return item, err
}

func (m *RefValueSettingModel) GetAll(
	ctx context.Context,
) ([]RefValueSetting, error) {
	rows, err := m.DB.Query(ctx, `
		SELECT `+refColumns+`
		FROM cg_ref_codes
		ORDER BY id DESC
		LIMIT 100
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]RefValueSetting, 0)

	for rows.Next() {
		item, err := scanRef(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (m *RefValueSettingModel) GetByID(
	ctx context.Context,
	id int64,
) (RefValueSetting, error) {
	return scanRef(m.DB.QueryRow(ctx, `
		SELECT `+refColumns+`
		FROM cg_ref_codes
		WHERE id = $1
	`, id))
}

func (m *RefValueSettingModel) Create(
	ctx context.Context,
	input RefValueSettingInput,
) (RefValueSetting, error) {
	return scanRef(m.DB.QueryRow(ctx, `
		INSERT INTO cg_ref_codes (
			domain, low_value, meaning,
			date_created, created_by
		)
		VALUES ($1, $2, $3, NOW(), $4)
		RETURNING `+refColumns,
		input.Domain,
		input.LowValue,
		input.Meaning,
		"local-dev",
	))
}

func (m *RefValueSettingModel) Update(
	ctx context.Context,
	id int64,
	input RefValueSettingInput,
) (RefValueSetting, error) {
	return scanRef(m.DB.QueryRow(ctx, `
		UPDATE cg_ref_codes
		SET domain = $1,
		    low_value = $2,
		    meaning = $3,
		    date_modified = NOW(),
		    modified_by = $4
		WHERE id = $5
		RETURNING `+refColumns,
		input.Domain,
		input.LowValue,
		input.Meaning,
		"local-dev",
		id,
	))
}

func (m *RefValueSettingModel) Delete(
	ctx context.Context,
	id int64,
) error {
	result, err := m.DB.Exec(ctx, `
		DELETE FROM cg_ref_codes WHERE id = $1
	`, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}
