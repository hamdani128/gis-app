package companies

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Region struct {
	ID            *int32     `json:"id"`
	EmployeeID    *int32     `json:"employee_id"`
	RegionName    string     `json:"region_name"`
	RegionGM      string     `json:"region_gm"`
	OfficeAddress *string    `json:"office_address"`
	OfficeCity    *string    `json:"office_city"`
	Note          *string    `json:"note"`
	DateCreated   *time.Time `json:"date_created"`
	CreatedBy     *string    `json:"created_by"`
}

type RegionInput struct {
	RegionName    string  `json:"region_name"`
	EmployeeID    *int32  `json:"employee_id"`
	OfficeAddress *string `json:"office_address"`
	OfficeCity    *string `json:"office_city"`
	Note          *string `json:"note"`
}

type RegionEmployeeOption struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

type RegionModel struct {
	DB *pgxpool.Pool
}

const regionTable = `public."Gis_regions"`
const regionIDColumn = `"idGis_regions"`
const regionEmployeeIDColumn = `"Gis_Employees_id"`

const regionSelect = `
	SELECT r.` + regionIDColumn + `, r.` + regionEmployeeIDColumn + `,
	       COALESCE(r.region_name, ''), COALESCE(e.name, ''),
	       r.office_address, r.office_city, r.note, r.date_created, r.created_by
	FROM ` + regionTable + ` r
	LEFT JOIN public."Gis_Employees" e ON e."id_Gis_Employees" = r.` + regionEmployeeIDColumn

func scanRegion(row pgx.Row) (Region, error) {
	var item Region
	err := row.Scan(
		&item.ID,
		&item.EmployeeID,
		&item.RegionName,
		&item.RegionGM,
		&item.OfficeAddress,
		&item.OfficeCity,
		&item.Note,
		&item.DateCreated,
		&item.CreatedBy,
	)
	return item, err
}

func (m *RegionModel) GetAll(ctx context.Context) ([]Region, error) {
	rows, err := m.DB.Query(ctx, regionSelect+`
		ORDER BY r.region_name NULLS LAST, r.`+regionIDColumn+` DESC NULLS LAST
		LIMIT 5000
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Region, 0)
	for rows.Next() {
		item, err := scanRegion(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *RegionModel) GetByID(ctx context.Context, id int64) (Region, error) {
	return scanRegion(m.DB.QueryRow(ctx, regionSelect+` WHERE r.`+regionIDColumn+`=$1`, id))
}

func (m *RegionModel) GetEmployees(ctx context.Context) ([]RegionEmployeeOption, error) {
	rows, err := m.DB.Query(ctx, `
		SELECT "id_Gis_Employees", COALESCE(name, '')
		FROM public."Gis_Employees"
		WHERE "id_Gis_Employees" IS NOT NULL
		ORDER BY name NULLS LAST, "id_Gis_Employees"
		LIMIT 5000
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]RegionEmployeeOption, 0)
	for rows.Next() {
		var item RegionEmployeeOption
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *RegionModel) EmployeeExists(ctx context.Context, id int32) (bool, error) {
	var exists bool
	err := m.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public."Gis_Employees" WHERE "id_Gis_Employees"=$1)`, id).Scan(&exists)
	return exists, err
}

// Create assigns an integer ID because this legacy table has no identity/default.
func (m *RegionModel) Create(ctx context.Context, input RegionInput) (Region, error) {
	tx, err := m.DB.Begin(ctx)
	if err != nil {
		return Region{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `LOCK TABLE `+regionTable+` IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return Region{}, err
	}
	var id int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(`+regionIDColumn+`), 0) + 1 FROM `+regionTable).Scan(&id); err != nil {
		return Region{}, err
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO `+regionTable+` (
			`+regionIDColumn+`, `+regionEmployeeIDColumn+`, region_name, office_address, office_city, note, date_created, created_by
		)
		VALUES ($1,$2,$3,$4,$5,$6,NOW(),$7)
		RETURNING `+regionIDColumn+`
	`, id, input.EmployeeID, input.RegionName, input.OfficeAddress, input.OfficeCity, input.Note, "local-dev").Scan(&id)
	if err != nil {
		return Region{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Region{}, err
	}
	return m.GetByID(ctx, id)
}

func (m *RegionModel) Update(ctx context.Context, id int64, input RegionInput) (Region, error) {
	_, err := m.DB.Exec(ctx, `
		UPDATE `+regionTable+`
		SET `+regionEmployeeIDColumn+`=$1, region_name=$2, office_address=$3, office_city=$4,
		    note=$5, modified_by=$6, date_modified=NOW()::text
		WHERE `+regionIDColumn+`=$7
	`, input.EmployeeID, input.RegionName, input.OfficeAddress, input.OfficeCity, input.Note, "local-dev", id)
	if err != nil {
		return Region{}, err
	}
	return m.GetByID(ctx, id)
}

func (m *RegionModel) Delete(ctx context.Context, id int64) error {
	result, err := m.DB.Exec(ctx, `DELETE FROM `+regionTable+` WHERE `+regionIDColumn+`=$1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func NormalizeRegionInput(input *RegionInput) {
	input.RegionName = strings.TrimSpace(input.RegionName)
	normalize := func(value **string) {
		if *value == nil {
			return
		}
		trimmed := strings.TrimSpace(**value)
		if trimmed == "" {
			*value = nil
		} else {
			*value = &trimmed
		}
	}
	normalize(&input.OfficeAddress)
	normalize(&input.OfficeCity)
	normalize(&input.Note)
}
