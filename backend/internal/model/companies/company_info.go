package companies

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CompanyInfo struct {
	ID          *int32     `json:"id"`
	Name        string     `json:"name"`
	ProvinceID  *int32     `json:"province_id"`
	Province    string     `json:"province"`
	RegionID    *int32     `json:"region_id"`
	Region      string     `json:"region"`
	City        *string    `json:"city"`
	Address     *string    `json:"address"`
	DateCreated *time.Time `json:"date_created"`
	CreatedBy   *string    `json:"created_by"`
}

type CompanyInfoInput struct {
	Name       string  `json:"name"`
	ProvinceID *int32  `json:"province_id"`
	RegionID   *int32  `json:"region_id"`
	City       *string `json:"city"`
	Address    *string `json:"address"`
}

type CompanyInfoProvinceOption struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

type CompanyInfoRegionOption struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

type CompanyInfoOptions struct {
	Provinces []CompanyInfoProvinceOption `json:"provinces"`
	Regions   []CompanyInfoRegionOption   `json:"regions"`
}

type CompanyInfoModel struct {
	DB *pgxpool.Pool
}

const companyInfoTable = `public."Gis_Companies"`
const companyInfoIDColumn = `"id_Gis_Companies"`
const companyInfoRegionIDColumn = `"Gis_regions_id"`

const companyInfoSelect = `
	SELECT c.` + companyInfoIDColumn + `, COALESCE(c.name, ''), c.province, COALESCE(p.meaning, p.low_value, ''),
	       c.` + companyInfoRegionIDColumn + `, COALESCE(r.region_name, ''), c.city, c.address, c.date_created, c.created_by
	FROM ` + companyInfoTable + ` c
	LEFT JOIN public.cg_ref_codes p ON p.id = c.province AND UPPER(BTRIM(p.domain)) = 'PROVINCE'
	LEFT JOIN public."Gis_regions" r ON r."idGis_regions" = c.` + companyInfoRegionIDColumn

func scanCompanyInfo(row pgx.Row) (CompanyInfo, error) {
	var item CompanyInfo
	err := row.Scan(
		&item.ID,
		&item.Name,
		&item.ProvinceID,
		&item.Province,
		&item.RegionID,
		&item.Region,
		&item.City,
		&item.Address,
		&item.DateCreated,
		&item.CreatedBy,
	)
	return item, err
}

func (m *CompanyInfoModel) GetAll(ctx context.Context) ([]CompanyInfo, error) {
	rows, err := m.DB.Query(ctx, companyInfoSelect+`
		ORDER BY c.name NULLS LAST, c.`+companyInfoIDColumn+` DESC NULLS LAST
		LIMIT 5000
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]CompanyInfo, 0)
	for rows.Next() {
		item, err := scanCompanyInfo(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *CompanyInfoModel) GetByID(ctx context.Context, id int64) (CompanyInfo, error) {
	return scanCompanyInfo(m.DB.QueryRow(ctx, companyInfoSelect+` WHERE c.`+companyInfoIDColumn+`=$1`, id))
}

func (m *CompanyInfoModel) GetOptions(ctx context.Context) (CompanyInfoOptions, error) {
	options := CompanyInfoOptions{
		Provinces: make([]CompanyInfoProvinceOption, 0),
		Regions:   make([]CompanyInfoRegionOption, 0),
	}
	rows, err := m.DB.Query(ctx, `
		SELECT id, COALESCE(NULLIF(BTRIM(meaning), ''), NULLIF(BTRIM(low_value), ''), '')
		FROM public.cg_ref_codes
		WHERE UPPER(BTRIM(domain)) = 'PROVINCE'
		ORDER BY 2
	`)
	if err != nil {
		return options, err
	}
	for rows.Next() {
		var item CompanyInfoProvinceOption
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			rows.Close()
			return options, err
		}
		options.Provinces = append(options.Provinces, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return options, err
	}

	rows, err = m.DB.Query(ctx, `
		SELECT "idGis_regions", COALESCE(region_name, '')
		FROM public."Gis_regions"
		WHERE "idGis_regions" IS NOT NULL
		ORDER BY region_name NULLS LAST, "idGis_regions"
		LIMIT 5000
	`)
	if err != nil {
		return options, err
	}
	defer rows.Close()
	for rows.Next() {
		var item CompanyInfoRegionOption
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return options, err
		}
		options.Regions = append(options.Regions, item)
	}
	return options, rows.Err()
}

func (m *CompanyInfoModel) ProvinceExists(ctx context.Context, id int32) (bool, error) {
	var exists bool
	err := m.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public.cg_ref_codes WHERE id=$1 AND UPPER(BTRIM(domain))='PROVINCE')`, id).Scan(&exists)
	return exists, err
}

func (m *CompanyInfoModel) RegionExists(ctx context.Context, id int32) (bool, error) {
	var exists bool
	err := m.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public."Gis_regions" WHERE "idGis_regions"=$1)`, id).Scan(&exists)
	return exists, err
}

// Create assigns the legacy integer ID because this table has no identity/default.
func (m *CompanyInfoModel) Create(ctx context.Context, input CompanyInfoInput) (CompanyInfo, error) {
	tx, err := m.DB.Begin(ctx)
	if err != nil {
		return CompanyInfo{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `LOCK TABLE `+companyInfoTable+` IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return CompanyInfo{}, err
	}
	var id int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(`+companyInfoIDColumn+`), -1) + 1 FROM `+companyInfoTable).Scan(&id); err != nil {
		return CompanyInfo{}, err
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO `+companyInfoTable+` (
			`+companyInfoIDColumn+`, name, province, `+companyInfoRegionIDColumn+`, city, address, date_created, created_by
		)
		VALUES ($1,$2,$3,$4,$5,$6,NOW(),$7)
		RETURNING `+companyInfoIDColumn+`
	`, id, input.Name, input.ProvinceID, input.RegionID, input.City, input.Address, "local-dev").Scan(&id); err != nil {
		return CompanyInfo{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CompanyInfo{}, err
	}
	return m.GetByID(ctx, id)
}

func (m *CompanyInfoModel) Update(ctx context.Context, id int64, input CompanyInfoInput) (CompanyInfo, error) {
	_, err := m.DB.Exec(ctx, `
		UPDATE `+companyInfoTable+`
		SET name=$1, province=$2, `+companyInfoRegionIDColumn+`=$3, city=$4, address=$5,
		    modified_by=$6, date_modified=NOW()::text
		WHERE `+companyInfoIDColumn+`=$7
	`, input.Name, input.ProvinceID, input.RegionID, input.City, input.Address, "local-dev", id)
	if err != nil {
		return CompanyInfo{}, err
	}
	return m.GetByID(ctx, id)
}

func (m *CompanyInfoModel) Delete(ctx context.Context, id int64) error {
	result, err := m.DB.Exec(ctx, `DELETE FROM `+companyInfoTable+` WHERE `+companyInfoIDColumn+`=$1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func NormalizeCompanyInfoInput(input *CompanyInfoInput) {
	input.Name = strings.TrimSpace(input.Name)
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
	normalize(&input.City)
	normalize(&input.Address)
}
