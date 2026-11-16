package administrator

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Module struct {
	ID           *int32  `json:"id"`
	ModuleName   string  `json:"module_name"`
	ModuleCode   string  `json:"module_code"`
	ModuleFile   *string `json:"module_file"`
	MenuParent   *string `json:"menu_parent"`
	Sequence     *int32  `json:"sequence"`
	PrivType     *string `json:"privtype"`
	PrivNote     *string `json:"privnote"`
	DateCreated  *string `json:"date_created"`
	CreatedBy    *string `json:"created_by"`
	DateModified *string `json:"date_modified"`
	ModifiedBy   *string `json:"modified_by"`
}

type ModuleInput struct {
	ModuleName string  `json:"module_name"`
	ModuleCode string  `json:"module_code"`
	ModuleFile *string `json:"module_file"`
	MenuParent *string `json:"menu_parent"`
	Sequence   *int32  `json:"sequence"`
	PrivType   *string `json:"privtype"`
	PrivNote   *string `json:"privnote"`
}

type ModuleModel struct {
	DB *pgxpool.Pool
}

type ModuleOptions struct {
	MenuParents []string `json:"menu_parents"`
}

const moduleTable = `public.modul`

const moduleSelect = `
	SELECT idmodul, COALESCE(modulname, ''), COALESCE(modulcode, ''), modulfile,
	       menu_parent, seq, privtype, privnote, date_created, created_by, date_modified, modified_by
	FROM ` + moduleTable

func scanModule(row pgx.Row) (Module, error) {
	var item Module
	err := row.Scan(
		&item.ID,
		&item.ModuleName,
		&item.ModuleCode,
		&item.ModuleFile,
		&item.MenuParent,
		&item.Sequence,
		&item.PrivType,
		&item.PrivNote,
		&item.DateCreated,
		&item.CreatedBy,
		&item.DateModified,
		&item.ModifiedBy,
	)
	return item, err
}

func (m *ModuleModel) GetAll(ctx context.Context) ([]Module, error) {
	rows, err := m.DB.Query(ctx, moduleSelect+`
		ORDER BY modulname NULLS LAST, idmodul DESC NULLS LAST
		LIMIT 5000
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Module, 0)
	for rows.Next() {
		item, err := scanModule(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *ModuleModel) GetByID(ctx context.Context, id int64) (Module, error) {
	return scanModule(m.DB.QueryRow(ctx, moduleSelect+` WHERE idmodul=$1`, id))
}

func (m *ModuleModel) GetOptions(ctx context.Context) (ModuleOptions, error) {
	options := ModuleOptions{MenuParents: make([]string, 0)}
	rows, err := m.DB.Query(ctx, `
		SELECT DISTINCT BTRIM(menu_parent)
		FROM `+moduleTable+`
		WHERE menu_parent IS NOT NULL AND BTRIM(menu_parent) <> ''
		ORDER BY 1
	`)
	if err != nil {
		return options, err
	}
	defer rows.Close()
	for rows.Next() {
		var parent string
		if err := rows.Scan(&parent); err != nil {
			return options, err
		}
		options.MenuParents = append(options.MenuParents, parent)
	}
	return options, rows.Err()
}

// Create assigns the legacy integer ID because modul has no identity/default.
func (m *ModuleModel) Create(ctx context.Context, input ModuleInput) (Module, error) {
	tx, err := m.DB.Begin(ctx)
	if err != nil {
		return Module{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `LOCK TABLE `+moduleTable+` IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return Module{}, err
	}
	var id int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(idmodul), 0) + 1 FROM `+moduleTable).Scan(&id); err != nil {
		return Module{}, err
	}
	item, err := scanModule(tx.QueryRow(ctx, `
		INSERT INTO `+moduleTable+` (
			idmodul, modulname, modulcode, modulfile, menu_parent, seq, privtype, privnote, date_created, created_by
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW()::text,$9)
		RETURNING idmodul, COALESCE(modulname, ''), COALESCE(modulcode, ''), modulfile,
		          menu_parent, seq, privtype, privnote, date_created, created_by, date_modified, modified_by
	`, id, input.ModuleName, input.ModuleCode, input.ModuleFile, input.MenuParent, input.Sequence, input.PrivType, input.PrivNote, "local-dev"))
	if err != nil {
		return Module{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Module{}, err
	}
	return item, nil
}

func (m *ModuleModel) Update(ctx context.Context, id int64, input ModuleInput) (Module, error) {
	return scanModule(m.DB.QueryRow(ctx, `
		UPDATE `+moduleTable+`
		SET modulname=$1, modulcode=$2, modulfile=$3, menu_parent=$4, seq=$5,
		    privtype=$6, privnote=$7, modified_by=$8, date_modified=NOW()::text
		WHERE idmodul=$9
		RETURNING idmodul, COALESCE(modulname, ''), COALESCE(modulcode, ''), modulfile,
		          menu_parent, seq, privtype, privnote, date_created, created_by, date_modified, modified_by
	`, input.ModuleName, input.ModuleCode, input.ModuleFile, input.MenuParent, input.Sequence, input.PrivType, input.PrivNote, "local-dev", id))
}

func (m *ModuleModel) Delete(ctx context.Context, id int64) error {
	result, err := m.DB.Exec(ctx, `DELETE FROM `+moduleTable+` WHERE idmodul=$1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func NormalizeModuleInput(input *ModuleInput) {
	input.ModuleName = strings.TrimSpace(input.ModuleName)
	input.ModuleCode = strings.TrimSpace(input.ModuleCode)
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
	normalize(&input.ModuleFile)
	normalize(&input.MenuParent)
	normalize(&input.PrivType)
	normalize(&input.PrivNote)
}
