package administrator

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BusinessProcessOfficer struct {
	ID                  *int32     `json:"id"`
	EmployeeID          *int32     `json:"employee_id"`
	EmployeeName        string     `json:"employee_name"`
	ProcessSequenceID   *int32     `json:"process_sequence_id"`
	ProcessSequenceName string     `json:"process_sequence_name"`
	CreatedBy           *string    `json:"created_by"`
	DateCreated         *time.Time `json:"date_created"`
}

type BusinessProcessOfficerInput struct {
	EmployeeID        *int32 `json:"employee_id"`
	ProcessSequenceID *int32 `json:"process_sequence_id"`
}

type BusinessProcessOfficerModel struct {
	DB *pgxpool.Pool
}

const businessProcessOfficerTable = `public."Gis_Process_Officers"`
const businessProcessOfficerIDColumn = `"id_Gis_Process_Officers"`
const businessProcessOfficerEmployeeIDColumn = `"Gis_Employees_id_Gis_Employees"`
const businessProcessOfficerSequenceIDColumn = `"Gis_Aquisition_Process_Sequences_id"`

const businessProcessOfficerSelect = `
	SELECT p.` + businessProcessOfficerIDColumn + `, p.` + businessProcessOfficerEmployeeIDColumn + `,
	       COALESCE(e.name, ''), p.` + businessProcessOfficerSequenceIDColumn + `,
	       COALESCE(s.name, ''), p.created_by, p.date_created
	FROM ` + businessProcessOfficerTable + ` p
	LEFT JOIN public."Gis_Employees" e
	       ON e."id_Gis_Employees" = p.` + businessProcessOfficerEmployeeIDColumn + `
	LEFT JOIN public."Gis_Aquisition_Process_Sequences" s
	       ON s."id_Gis_Aquisition_Process_Sequences" = p.` + businessProcessOfficerSequenceIDColumn

func scanBusinessProcessOfficer(row pgx.Row) (BusinessProcessOfficer, error) {
	var item BusinessProcessOfficer
	err := row.Scan(
		&item.ID,
		&item.EmployeeID,
		&item.EmployeeName,
		&item.ProcessSequenceID,
		&item.ProcessSequenceName,
		&item.CreatedBy,
		&item.DateCreated,
	)
	return item, err
}

func (m *BusinessProcessOfficerModel) GetAll(ctx context.Context) ([]BusinessProcessOfficer, error) {
	rows, err := m.DB.Query(ctx, businessProcessOfficerSelect+`
		ORDER BY e.name NULLS LAST, s.name NULLS LAST, p.`+businessProcessOfficerIDColumn+` DESC NULLS LAST
		LIMIT 5000
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]BusinessProcessOfficer, 0)
	for rows.Next() {
		item, err := scanBusinessProcessOfficer(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *BusinessProcessOfficerModel) GetByID(ctx context.Context, id int64) (BusinessProcessOfficer, error) {
	return scanBusinessProcessOfficer(m.DB.QueryRow(ctx, businessProcessOfficerSelect+`
		WHERE p.`+businessProcessOfficerIDColumn+` = $1
	`, id))
}

func (m *BusinessProcessOfficerModel) EmployeeExists(ctx context.Context, id int32) (bool, error) {
	var exists bool
	err := m.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public."Gis_Employees" WHERE "id_Gis_Employees"=$1)`, id).Scan(&exists)
	return exists, err
}

func (m *BusinessProcessOfficerModel) SequenceExists(ctx context.Context, id int32) (bool, error) {
	var exists bool
	err := m.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public."Gis_Aquisition_Process_Sequences" WHERE "id_Gis_Aquisition_Process_Sequences"=$1)`, id).Scan(&exists)
	return exists, err
}

// Create allocates the legacy ID because this table has no identity/default.
func (m *BusinessProcessOfficerModel) Create(ctx context.Context, input BusinessProcessOfficerInput) (BusinessProcessOfficer, error) {
	tx, err := m.DB.Begin(ctx)
	if err != nil {
		return BusinessProcessOfficer{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `LOCK TABLE `+businessProcessOfficerTable+` IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return BusinessProcessOfficer{}, err
	}
	var id int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(`+businessProcessOfficerIDColumn+`), 0) + 1 FROM `+businessProcessOfficerTable).Scan(&id); err != nil {
		return BusinessProcessOfficer{}, err
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO `+businessProcessOfficerTable+` (
			`+businessProcessOfficerIDColumn+`, `+businessProcessOfficerEmployeeIDColumn+`,
			`+businessProcessOfficerSequenceIDColumn+`, created_by, date_created
		)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING `+businessProcessOfficerIDColumn+`
	`, id, input.EmployeeID, input.ProcessSequenceID, "local-dev").Scan(&id); err != nil {
		return BusinessProcessOfficer{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return BusinessProcessOfficer{}, err
	}
	return m.GetByID(ctx, id)
}

func (m *BusinessProcessOfficerModel) Update(ctx context.Context, id int64, input BusinessProcessOfficerInput) (BusinessProcessOfficer, error) {
	result, err := m.DB.Exec(ctx, `
		UPDATE `+businessProcessOfficerTable+`
		SET `+businessProcessOfficerEmployeeIDColumn+`=$1, `+businessProcessOfficerSequenceIDColumn+`=$2,
		    modified_by=$3, date_modified=NOW()::text
		WHERE `+businessProcessOfficerIDColumn+`=$4
	`, input.EmployeeID, input.ProcessSequenceID, "local-dev", id)
	if err != nil {
		return BusinessProcessOfficer{}, err
	}
	if result.RowsAffected() == 0 {
		return BusinessProcessOfficer{}, pgx.ErrNoRows
	}
	return m.GetByID(ctx, id)
}

func (m *BusinessProcessOfficerModel) Delete(ctx context.Context, id int64) error {
	result, err := m.DB.Exec(ctx, `DELETE FROM `+businessProcessOfficerTable+` WHERE `+businessProcessOfficerIDColumn+`=$1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
