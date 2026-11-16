package lands

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LandOwnerModel struct{ DB *pgxpool.Pool }

type LandOwner struct {
	ID          int64      `json:"id"`
	CompanyID   int64      `json:"Gis_Companies_id"`
	Company     string     `json:"company"`
	Name        string     `json:"name"`
	KTPNumber   string     `json:"ktp_number"`
	Address     string     `json:"address"`
	Note        string     `json:"note"`
	DateCreated *time.Time `json:"date_created"`
}

type LandOwnerInput struct {
	CompanyID   int64      `json:"Gis_Companies_id"`
	Name        string     `json:"name"`
	KTPNumber   string     `json:"ktp_number"`
	Address     string     `json:"address"`
	Note        string     `json:"note"`
	DateCreated *time.Time `json:"date_created"`
}

type LandOwnerCompany struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type OwnerLand struct {
	ID                int64   `json:"id"`
	Owner             string  `json:"owner"`
	KTPNumber         string  `json:"ktp_number"`
	Village           string  `json:"village"`
	Status            int64   `json:"status"`
	LandWide          float64 `json:"land_wide"`
	WetLandWide       float64 `json:"wet_land_wide"`
	WetLandPerhaPrice float64 `json:"wet_land_perha_price"`
	DryLandWide       float64 `json:"dry_land_wide"`
	DryLandPerhaPrice float64 `json:"dry_land_perha_price"`
	AdminPrice        float64 `json:"admin_price"`
	PlantPrice        float64 `json:"plant_price"`
}

const landOwnerSelect = `SELECT o."id_Gis_Land_Owners", o."Gis_Companies_id", COALESCE(NULLIF(BTRIM(c.name),''),NULLIF(BTRIM(c.com_name),''),''), COALESCE(o.name,''), COALESCE(o.ktp_number,''), COALESCE(o.address,''), COALESCE(o.note,''), o.date_created FROM public."Gis_Land_Owners" o JOIN public."Gis_Companies" c ON c."id_Gis_Companies"=o."Gis_Companies_id"`

func scanLandOwner(row pgx.Row) (LandOwner, error) {
	var item LandOwner
	err := row.Scan(&item.ID, &item.CompanyID, &item.Company, &item.Name, &item.KTPNumber, &item.Address, &item.Note, &item.DateCreated)
	return item, err
}

func (m *LandOwnerModel) GetAll(ctx context.Context, companyID int64) ([]LandOwner, error) {
	rows, err := m.DB.Query(ctx, landOwnerSelect+` WHERE ($1::bigint=0 OR o."Gis_Companies_id"=$1) ORDER BY o."id_Gis_Land_Owners" DESC LIMIT 5000`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []LandOwner{}
	for rows.Next() {
		item, err := scanLandOwner(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *LandOwnerModel) GetByID(ctx context.Context, companyID, id int64) (LandOwner, error) {
	return scanLandOwner(m.DB.QueryRow(ctx, landOwnerSelect+` WHERE o."id_Gis_Land_Owners"=$1 AND ($2::bigint=0 OR o."Gis_Companies_id"=$2)`, id, companyID))
}

func (m *LandOwnerModel) Companies(ctx context.Context, companyID int64) ([]LandOwnerCompany, error) {
	rows, err := m.DB.Query(ctx, `SELECT "id_Gis_Companies", COALESCE(NULLIF(BTRIM(name),''),NULLIF(BTRIM(com_name),''),'') FROM public."Gis_Companies" WHERE ($1::bigint=0 OR "id_Gis_Companies"=$1) ORDER BY 2`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []LandOwnerCompany{}
	for rows.Next() {
		var item LandOwnerCompany
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *LandOwnerModel) Save(ctx context.Context, companyID int64, id *int64, in LandOwnerInput) (LandOwner, error) {
	if in.CompanyID <= 0 || (companyID != 0 && companyID != in.CompanyID) {
		return LandOwner{}, errors.New("company tidak valid untuk pilihan aktif")
	}
	in.Name = strings.TrimSpace(in.Name)
	in.KTPNumber = strings.TrimSpace(in.KTPNumber)
	in.Address = strings.TrimSpace(in.Address)
	in.Note = strings.TrimSpace(in.Note)
	if in.Name == "" {
		return LandOwner{}, errors.New("name wajib diisi")
	}
	var exists bool
	if err := m.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public."Gis_Companies" WHERE "id_Gis_Companies"=$1)`, in.CompanyID).Scan(&exists); err != nil {
		return LandOwner{}, err
	}
	if !exists {
		return LandOwner{}, errors.New("company tidak ditemukan")
	}
	if id == nil {
		tx, err := m.DB.Begin(ctx)
		if err != nil {
			return LandOwner{}, err
		}
		defer tx.Rollback(ctx)
		if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(73210482)`); err != nil {
			return LandOwner{}, err
		}
		var next int64
		if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX("id_Gis_Land_Owners"),0)+1 FROM public."Gis_Land_Owners"`).Scan(&next); err != nil {
			return LandOwner{}, err
		}
		date := in.DateCreated
		if date == nil {
			now := time.Now()
			date = &now
		}
		_, err = tx.Exec(ctx, `INSERT INTO public."Gis_Land_Owners" ("id_Gis_Land_Owners","Gis_Companies_id",name,ktp_number,address,note,date_created) VALUES ($1,$2,$3,$4,$5,$6,$7)`, next, in.CompanyID, in.Name, in.KTPNumber, in.Address, in.Note, date)
		if err != nil {
			return LandOwner{}, err
		}
		if err = tx.Commit(ctx); err != nil {
			return LandOwner{}, err
		}
		id = &next
	} else {
		tag, err := m.DB.Exec(ctx, `UPDATE public."Gis_Land_Owners" SET "Gis_Companies_id"=$1,name=$2,ktp_number=$3,address=$4,note=$5,date_created=$6 WHERE "id_Gis_Land_Owners"=$7 AND ($8::bigint=0 OR "Gis_Companies_id"=$8)`, in.CompanyID, in.Name, in.KTPNumber, in.Address, in.Note, in.DateCreated, *id, companyID)
		if err != nil {
			return LandOwner{}, err
		}
		if tag.RowsAffected() == 0 {
			return LandOwner{}, pgx.ErrNoRows
		}
	}
	return m.GetByID(ctx, companyID, *id)
}

func (m *LandOwnerModel) Delete(ctx context.Context, companyID, id int64) error {
	tag, err := m.DB.Exec(ctx, `DELETE FROM public."Gis_Land_Owners" WHERE "id_Gis_Land_Owners"=$1 AND ($2::bigint=0 OR "Gis_Companies_id"=$2)`, id, companyID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (m *LandOwnerModel) Lands(ctx context.Context, companyID, ownerID int64) ([]OwnerLand, error) {
	if _, err := m.GetByID(ctx, companyID, ownerID); err != nil {
		return nil, err
	}
	rows, err := m.DB.Query(ctx, `SELECT l."id_Gis_Compensated_Lands",COALESCE(o.name,''),COALESCE(o.ktp_number,''),COALESCE(l.village,''),COALESCE(l.status,0),COALESCE(l.land_wide,0),COALESCE(l.wet_land_wide,0),COALESCE(l.wet_land_perha_price,0),COALESCE(l.dry_land_wide,0),COALESCE(l.dry_land_perha_price,0),COALESCE(l.admin_price,0),COALESCE(l.plant_price,0) FROM public."Gis_Compensated_Lands" l JOIN public."Gis_Land_Owners" o ON o."id_Gis_Land_Owners"=l."Gis_Land_Owners_id" WHERE l."Gis_Land_Owners_id"=$1 AND ($2::bigint=0 OR o."Gis_Companies_id"=$2) ORDER BY l."id_Gis_Compensated_Lands" DESC`, ownerID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []OwnerLand{}
	for rows.Next() {
		var item OwnerLand
		if err := rows.Scan(&item.ID, &item.Owner, &item.KTPNumber, &item.Village, &item.Status, &item.LandWide, &item.WetLandWide, &item.WetLandPerhaPrice, &item.DryLandWide, &item.DryLandPerhaPrice, &item.AdminPrice, &item.PlantPrice); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (m *LandOwnerModel) LandMap(ctx context.Context, companyID, ownerID, landID int64) (string, error) {
	if _, err := m.GetByID(ctx, companyID, ownerID); err != nil {
		return "", err
	}
	var status int64
	err := m.DB.QueryRow(ctx, `SELECT COALESCE(status,0) FROM public."Gis_Compensated_Lands" WHERE "id_Gis_Compensated_Lands"=$1 AND "Gis_Land_Owners_id"=$2`, landID, ownerID).Scan(&status)
	if err != nil {
		return "", err
	}
	return (&CompensatedLandModel{DB: m.DB, Status: status}).GetMap(ctx, companyID, landID)
}

func (m *LandOwnerModel) LandDetail(ctx context.Context, companyID, ownerID, landID int64) (map[string]any, error) {
	var raw []byte
	err := m.DB.QueryRow(ctx, `SELECT to_jsonb(l) || jsonb_build_object(
		'id',l."id_Gis_Compensated_Lands",'owner',COALESCE(o.name,''),'owner_ktp',COALESCE(o.ktp_number,''),
		'block_plan',COALESCE(d.division_name,''),'potential_land',COALESCE(p.potential_name,''),
		'estate_name',COALESCE(p.estate_name,''),'permit_number',COALESCE(lp.permit_number,''),
		'company_name',COALESCE(NULLIF(BTRIM(c.name),''),NULLIF(BTRIM(c.com_name),''),''),
		'company_id',o."Gis_Companies_id")
		FROM public."Gis_Compensated_Lands" l
		JOIN public."Gis_Land_Owners" o ON o."id_Gis_Land_Owners"=l."Gis_Land_Owners_id"
		JOIN public."Gis_Companies" c ON c."id_Gis_Companies"=o."Gis_Companies_id"
		LEFT JOIN public."Gis_Aquistion_Divisions" d ON d."id_Gis_Aqcuistion_Divisions"=l."Gis_Aquistion_Divisions_id"
		LEFT JOIN public."Gis_Potential_Lands" p ON p."id_Gis_Potential_Lands"=d."Gis_Potential_Lands_id"
		LEFT JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id"
		WHERE l."id_Gis_Compensated_Lands"=$1 AND l."Gis_Land_Owners_id"=$2
		AND ($3::bigint=0 OR o."Gis_Companies_id"=$3)`, landID, ownerID, companyID).Scan(&raw)
	if err != nil {
		return nil, err
	}
	var item map[string]any
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, err
	}
	return item, nil
}

func (m *LandOwnerModel) LandProcesses(ctx context.Context, companyID, ownerID, landID int64) ([]map[string]any, error) {
	item, err := m.LandDetail(ctx, companyID, ownerID, landID)
	if err != nil {
		return nil, err
	}
	status, _ := item["status"].(float64)
	return (&CompensatedLandModel{DB: m.DB, Status: int64(status)}).GetProcesses(ctx, companyID, landID)
}
