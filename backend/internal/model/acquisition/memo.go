package acquisition

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const memoTable = `public."Gis_Aquisition_memo"`
const memoIDColumn = `"idGis_Aquisition_memo"`

var ErrMemoHasLands = errors.New("memo masih memiliki Propose Land")

type Memo struct {
	ID          int64  `json:"id"`
	CompanyID   int64  `json:"company_id"`
	CompanyName string `json:"company_name"`
	MemoNumber  string `json:"memo_number"`
	MemoDate    string `json:"memo_date"`
	MemoStatus  int32  `json:"memo_status"`
	StatusName  string `json:"status_name"`
	MemoDoc     string `json:"memo_doc"`
	MemoNote    string `json:"memo_note"`
}

type MemoInput struct {
	CompanyID  int64  `json:"company_id"`
	MemoNumber string `json:"memo_number"`
	MemoDate   string `json:"memo_date"`
	MemoNote   string `json:"memo_note"`
	ClearDoc   bool   `json:"clear_doc"`
}

type ProposedLand struct {
	ID                int64   `json:"id"`
	Division          string  `json:"division"`
	Owner             string  `json:"owner"`
	Village           string  `json:"village"`
	LandWide          float64 `json:"land_wide"`
	WetLandWide       float64 `json:"wet_land_wide"`
	WetLandPerhaPrice int64   `json:"wet_land_perha_price"`
	DryLandWide       float64 `json:"dry_land_wide"`
	DryLandPerhaPrice int64   `json:"dry_land_perha_price"`
	AdminPrice        int64   `json:"admin_price"`
	PlantPrice        int64   `json:"plant_price"`
	Status            int32   `json:"status"`
	StatusName        string  `json:"status_name"`
}

type MemoModel struct{ DB *pgxpool.Pool }

const memoSelect = `SELECT m."idGis_Aquisition_memo", COALESCE(m."Gis_Companies_id",0),
	COALESCE(NULLIF(BTRIM(c.name),''), NULLIF(BTRIM(c.com_name),''),''),
	COALESCE(m.memo_number,''), COALESCE(to_char(m.memo_date,'YYYY-MM-DD"T"HH24:MI:SS'),''),
	COALESCE(m.memo_status,0), COALESCE(s.meaning,''), COALESCE(m.memo_doc,''), COALESCE(m.memo_note,'')
	FROM public."Gis_Aquisition_memo" m
	LEFT JOIN public."Gis_Companies" c ON c."id_Gis_Companies"=m."Gis_Companies_id"
	LEFT JOIN public.view_memo_status s ON s.id=m.memo_status`

func scanMemo(row pgx.Row) (Memo, error) {
	var item Memo
	err := row.Scan(&item.ID, &item.CompanyID, &item.CompanyName, &item.MemoNumber,
		&item.MemoDate, &item.MemoStatus, &item.StatusName, &item.MemoDoc, &item.MemoNote)
	return item, err
}

func (m *MemoModel) GetAll(ctx context.Context, companyID int64) ([]Memo, error) {
	rows, err := m.DB.Query(ctx, memoSelect+`
		WHERE ($1::bigint=0 OR m."Gis_Companies_id"=$1)
		ORDER BY m."idGis_Aquisition_memo" DESC LIMIT 10000`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Memo{}
	for rows.Next() {
		item, err := scanMemo(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *MemoModel) GetByID(ctx context.Context, companyID, id int64) (Memo, error) {
	return scanMemo(m.DB.QueryRow(ctx, memoSelect+`
		WHERE m."idGis_Aquisition_memo"=$1 AND ($2::bigint=0 OR m."Gis_Companies_id"=$2)`, id, companyID))
}

func (m *MemoModel) Save(ctx context.Context, companyID int64, id *int64, input MemoInput, memoDate time.Time, docPath *string) (Memo, error) {
	input.MemoNumber = strings.TrimSpace(input.MemoNumber)
	input.MemoNote = strings.TrimSpace(input.MemoNote)
	if input.CompanyID <= 0 || input.MemoNumber == "" {
		return Memo{}, fmt.Errorf("company dan nomor memo wajib diisi")
	}
	if companyID != 0 && companyID != input.CompanyID {
		return Memo{}, fmt.Errorf("company memo harus sama dengan company aktif")
	}
	var exists bool
	if err := m.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public."Gis_Companies" WHERE "id_Gis_Companies"=$1)`, input.CompanyID).Scan(&exists); err != nil {
		return Memo{}, err
	}
	if !exists {
		return Memo{}, fmt.Errorf("company tidak ditemukan")
	}
	if id == nil {
		var newID int64
		err := m.DB.QueryRow(ctx, `INSERT INTO `+memoTable+` ("Gis_Companies_id",memo_number,memo_date,memo_note,memo_status,memo_doc,date_created,created_by)
			VALUES ($1,$2,$3,$4,2,$5,NOW(),'local-dev') RETURNING `+memoIDColumn,
			input.CompanyID, input.MemoNumber, memoDate, input.MemoNote, docPath).Scan(&newID)
		if err != nil {
			return Memo{}, err
		}
		return m.GetByID(ctx, companyID, newID)
	}
	existing, err := m.GetByID(ctx, companyID, *id)
	if err != nil {
		return Memo{}, err
	}
	if existing.CompanyID != input.CompanyID {
		var linked bool
		if err := m.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public."Gis_Compensated_Lands" WHERE BTRIM("Gis_Aquisition_memo_id")=$1)`, fmt.Sprint(*id)).Scan(&linked); err != nil {
			return Memo{}, err
		}
		if linked {
			return Memo{}, fmt.Errorf("company memo tidak dapat diubah karena masih memiliki Propose Land")
		}
	}
	var updatedID int64
	err = m.DB.QueryRow(ctx, `UPDATE `+memoTable+` SET "Gis_Companies_id"=$1,memo_number=$2,memo_date=$3,memo_note=$4,
		memo_doc=COALESCE($5,memo_doc),date_modified=NOW(),modified_by='local-dev'
		WHERE `+memoIDColumn+`=$6 AND ($7::bigint=0 OR "Gis_Companies_id"=$7) RETURNING `+memoIDColumn,
		input.CompanyID, input.MemoNumber, memoDate, input.MemoNote, docPath, *id, companyID).Scan(&updatedID)
	if err != nil {
		return Memo{}, err
	}
	return m.GetByID(ctx, companyID, updatedID)
}

func (m *MemoModel) Delete(ctx context.Context, companyID, id int64) error {
	if _, err := m.GetByID(ctx, companyID, id); err != nil {
		return err
	}
	var linked bool
	if err := m.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public."Gis_Compensated_Lands" WHERE BTRIM("Gis_Aquisition_memo_id")=$1)`, fmt.Sprint(id)).Scan(&linked); err != nil {
		return err
	}
	if linked {
		return ErrMemoHasLands
	}
	result, err := m.DB.Exec(ctx, `DELETE FROM `+memoTable+` WHERE `+memoIDColumn+`=$1 AND ($2::bigint=0 OR "Gis_Companies_id"=$2)`, id, companyID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (m *MemoModel) GetLands(ctx context.Context, companyID, memoID int64) ([]ProposedLand, error) {
	memo, err := m.GetByID(ctx, companyID, memoID)
	if err != nil {
		return nil, err
	}
	rows, err := m.DB.Query(ctx, `SELECT l."id_Gis_Compensated_Lands",COALESCE(d.division_name,''),
		COALESCE(o.name,''),COALESCE(l.village,''),COALESCE(l.land_wide,0),
		COALESCE(l.wet_land_wide,0),COALESCE(l.wet_land_perha_price,0),
		COALESCE(l.dry_land_wide,0),COALESCE(l.dry_land_perha_price,0),
		COALESCE(l.admin_price,0),COALESCE(l.plant_price,0),COALESCE(l.status,0),
		CASE l.status WHEN 0 THEN 'Propose Land' WHEN 8 THEN 'New Land'
			WHEN 10 THEN 'Payment to Owner' WHEN 99 THEN 'Rejected'
			ELSE 'Status ' || COALESCE(l.status::text,'-') END
		FROM public."Gis_Compensated_Lands" l
		JOIN public."Gis_Aquistion_Divisions" d ON d."id_Gis_Aqcuistion_Divisions"=l."Gis_Aquistion_Divisions_id"
		JOIN public."Gis_Potential_Lands" p ON p."id_Gis_Potential_Lands"=d."Gis_Potential_Lands_id"
		JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id"
		LEFT JOIN public."Gis_Land_Owners" o ON o."id_Gis_Land_Owners"=l."Gis_Land_Owners_id"
		WHERE BTRIM(l."Gis_Aquisition_memo_id")=$1 AND lp."Gis_Companies_id"=$2
		ORDER BY l."id_Gis_Compensated_Lands" DESC`, fmt.Sprint(memoID), memo.CompanyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ProposedLand{}
	for rows.Next() {
		var item ProposedLand
		err := rows.Scan(&item.ID, &item.Division, &item.Owner, &item.Village, &item.LandWide,
			&item.WetLandWide, &item.WetLandPerhaPrice, &item.DryLandWide, &item.DryLandPerhaPrice,
			&item.AdminPrice, &item.PlantPrice, &item.Status, &item.StatusName)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// CheckLand checks the same memo/company relationship as GetLands before serving
// an individual land's document archive from Acquisition Request.
func (m *MemoModel) CheckLand(ctx context.Context, companyID, memoID, landID int64) error {
	memo, err := m.GetByID(ctx, companyID, memoID)
	if err != nil {
		return err
	}
	var found bool
	err = m.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public."Gis_Compensated_Lands" l
		JOIN public."Gis_Aquistion_Divisions" d ON d."id_Gis_Aqcuistion_Divisions"=l."Gis_Aquistion_Divisions_id"
		JOIN public."Gis_Potential_Lands" p ON p."id_Gis_Potential_Lands"=d."Gis_Potential_Lands_id"
		JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id"
		WHERE l."id_Gis_Compensated_Lands"=$1 AND BTRIM(l."Gis_Aquisition_memo_id")=$2
		AND lp."Gis_Companies_id"=$3)`, landID, fmt.Sprint(memoID), memo.CompanyID).Scan(&found)
	if err != nil {
		return err
	}
	if !found {
		return pgx.ErrNoRows
	}
	return nil
}
