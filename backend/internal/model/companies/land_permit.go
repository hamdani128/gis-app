package companies

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Land permits are stored in a legacy table whose exact field spellings differ
// between deployments. Resolve known legacy aliases from PostgreSQL metadata.
type LandPermit struct {
	ID                int64  `json:"id"`
	CompanyID         int64  `json:"company_id"`
	CompanyName       string `json:"company_name"`
	PermitNumber      string `json:"permit_number"`
	PermitDate        string `json:"permit_date"`
	ProvinceID        *int64 `json:"province_id"`
	Province          string `json:"province"`
	Address           string `json:"address"`
	PermitAgency      string `json:"permit_agency"`
	AgencyPhone       string `json:"agency_phone"`
	AgencyAddress     string `json:"agency_address"`
	SignatoryName     string `json:"signatory_name"`
	SignatoryPosition string `json:"signatory_position"`
	SignatoryPlace    string `json:"signatory_place"`
	PermitDocument    string `json:"permit_document"`
	MapData           string `json:"map_data"`
	MapFile           string `json:"map_file"`
}

type LandPermitInput struct {
	CompanyID         int64  `json:"company_id"`
	PermitNumber      string `json:"permit_number"`
	PermitDate        string `json:"permit_date"`
	ProvinceID        *int64 `json:"province_id"`
	Address           string `json:"address"`
	PermitAgency      string `json:"permit_agency"`
	AgencyPhone       string `json:"agency_phone"`
	AgencyAddress     string `json:"agency_address"`
	SignatoryName     string `json:"signatory_name"`
	SignatoryPosition string `json:"signatory_position"`
	SignatoryPlace    string `json:"signatory_place"`
	PermitDocument    string `json:"permit_document"`
	MapData           string `json:"map_data"`
	MapFile           string `json:"map_file"`
}

type LandPermitOption struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
type LandPermitOptions struct {
	Companies []LandPermitOption `json:"companies"`
	Provinces []LandPermitOption `json:"provinces"`
}
type LandPermitModel struct{ DB *pgxpool.Pool }

const landPermitTable = `public."Gis_Land_Permits"`

var landPermitAliases = map[string][]string{
	"id":                 {"id_Gis_Land_Permits", "idGis_Land_Permits", "id_land_permits", "id"},
	"company_id":         {"Gis_Companies_id", "id_Gis_Companies", "company_id"},
	"permit_number":      {"permit_number", "permit_no", "permit_num", "nomor_izin", "no_permit"},
	"permit_date":        {"permit_date", "date_permit", "tanggal_izin"},
	"province_id":        {"province", "province_id", "cg_ref_codes_id"},
	"address":            {"address", "permit_address", "alamat"},
	"permit_agency":      {"permit_agency", "agency", "issuing_agency"},
	"agency_phone":       {"agency_phone", "phone_agency", "agency_phone_number"},
	"agency_address":     {"agency_address", "address_agency"},
	"signatory_name":     {"signatory_name", "signature_officer_name", "officer_name"},
	"signatory_position": {"signatory_position", "signature_officer_position", "officer_position"},
	"signatory_place":    {"signatory_place", "signature_place", "permit_place"},
	"permit_document":    {"permit_doc", "permit_document", "document_path", "file_permit"},
	"map_file":           {"map_file", "map_doc", "map_path", "mapinfo_file", "mapinfo", "coordinate_file", "geojson_file", "map_file_path"},
}

func quoteIdent(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }

func (m *LandPermitModel) columns(ctx context.Context) (map[string]string, error) {
	rows, err := m.DB.Query(ctx, `SELECT column_name FROM information_schema.columns WHERE table_schema='public' AND table_name='Gis_Land_Permits'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	actual := map[string]string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		actual[strings.ToLower(name)] = name
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	result := map[string]string{}
	for field, aliases := range landPermitAliases {
		for _, alias := range aliases {
			if name, ok := actual[strings.ToLower(alias)]; ok {
				result[field] = name
				break
			}
		}
	}
	var primaryKey string
	if err := m.DB.QueryRow(ctx, `
		SELECT kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name=kcu.constraint_name AND tc.table_schema=kcu.table_schema AND tc.table_name=kcu.table_name
		WHERE tc.table_schema='public' AND tc.table_name='Gis_Land_Permits' AND tc.constraint_type='PRIMARY KEY'
		ORDER BY kcu.ordinal_position LIMIT 1
	`).Scan(&primaryKey); err == nil && primaryKey != "" {
		result["id"] = primaryKey
	}
	if result["id"] == "" || result["company_id"] == "" {
		return nil, fmt.Errorf("Gis_Land_Permits harus memiliki kolom ID dan relasi company")
	}
	return result, nil
}

func rowToPermit(raw []byte, cols map[string]string, companyName, province string) (LandPermit, error) {
	var row map[string]any
	if err := json.Unmarshal(raw, &row); err != nil {
		return LandPermit{}, err
	}
	p := LandPermit{CompanyName: companyName, Province: province}
	for field, aliases := range landPermitAliases {
		keys := append([]string{}, aliases...)
		if actual := cols[field]; actual != "" {
			keys = append([]string{actual}, keys...)
		}
		for _, alias := range keys {
			if v, ok := row[alias]; ok && v != nil {
				switch field {
				case "id":
					fmt.Sscan(fmt.Sprint(v), &p.ID)
				case "company_id":
					fmt.Sscan(fmt.Sprint(v), &p.CompanyID)
				case "permit_number":
					p.PermitNumber = fmt.Sprint(v)
				case "permit_date":
					p.PermitDate = fmt.Sprint(v)
				case "province_id":
					var id int64
					if _, err := fmt.Sscan(fmt.Sprint(v), &id); err == nil {
						p.ProvinceID = &id
					}
				case "address":
					p.Address = fmt.Sprint(v)
				case "permit_agency":
					p.PermitAgency = fmt.Sprint(v)
				case "agency_phone":
					p.AgencyPhone = fmt.Sprint(v)
				case "agency_address":
					p.AgencyAddress = fmt.Sprint(v)
				case "signatory_name":
					p.SignatoryName = fmt.Sprint(v)
				case "signatory_position":
					p.SignatoryPosition = fmt.Sprint(v)
				case "signatory_place":
					p.SignatoryPlace = fmt.Sprint(v)
				case "permit_document":
					p.PermitDocument = fmt.Sprint(v)
				case "map_data":
					if text, ok := v.(string); ok {
						p.MapData = text
					} else if encoded, err := json.Marshal(v); err == nil {
						p.MapData = string(encoded)
					}
				case "map_file":
					p.MapFile = fmt.Sprint(v)
				}
				break
			}
		}
	}
	return p, nil
}

func (m *LandPermitModel) GetOptions(ctx context.Context) (LandPermitOptions, error) {
	out := LandPermitOptions{Companies: []LandPermitOption{}, Provinces: []LandPermitOption{}}
	rows, err := m.DB.Query(ctx, `SELECT "id_Gis_Companies", COALESCE(NULLIF(BTRIM(name),''), NULLIF(BTRIM(com_name),''), '') FROM public."Gis_Companies" ORDER BY 2`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var o LandPermitOption
		if err := rows.Scan(&o.ID, &o.Name); err != nil {
			rows.Close()
			return out, err
		}
		out.Companies = append(out.Companies, o)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()
	rows, err = m.DB.Query(ctx, `SELECT id, COALESCE(NULLIF(BTRIM(meaning),''),NULLIF(BTRIM(low_value),''),'') FROM public.cg_ref_codes WHERE UPPER(BTRIM(domain))='PROVINCE' ORDER BY 2`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var o LandPermitOption
		if err := rows.Scan(&o.ID, &o.Name); err != nil {
			return out, err
		}
		out.Provinces = append(out.Provinces, o)
	}
	return out, rows.Err()
}

func (m *LandPermitModel) get(ctx context.Context, cols map[string]string, companyID int64, id *int64) ([]LandPermit, error) {
	idCol, companyCol := quoteIdent(cols["id"]), quoteIdent(cols["company_id"])
	where := `p.` + companyCol + `=$1`
	args := []any{companyID}
	if id != nil {
		where += ` AND p.` + idCol + `=$2`
		args = append(args, *id)
	}
	provinceExpr := `NULL`
	if c := cols["province_id"]; c != "" {
		provinceExpr = `p.` + quoteIdent(c)
	}
	query := `SELECT to_jsonb(p), COALESCE(c.name,c.com_name,''), COALESCE(r.meaning,r.low_value,'') FROM ` + landPermitTable + ` p LEFT JOIN public."Gis_Companies" c ON c."id_Gis_Companies"=p.` + companyCol + ` LEFT JOIN public.cg_ref_codes r ON r.id=` + provinceExpr + ` AND UPPER(BTRIM(r.domain))='PROVINCE' WHERE ` + where + ` ORDER BY p.` + idCol + ` DESC LIMIT 5000`
	rows, err := m.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []LandPermit{}
	for rows.Next() {
		var raw []byte
		var company, province string
		if err := rows.Scan(&raw, &company, &province); err != nil {
			return nil, err
		}
		item, err := rowToPermit(raw, cols, company, province)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	permitIDs := make([]int64, 0, len(items))
	for _, item := range items {
		permitIDs = append(permitIDs, item.ID)
	}
	spatialData, err := m.GetSpatialMapData(ctx, permitIDs)
	if err != nil {
		return nil, err
	}
	for index := range items {
		if spatialMap, ok := spatialData[items[index].ID]; ok && strings.TrimSpace(spatialMap) != "" {
			items[index].MapData = spatialMap
		}
	}
	return items, nil
}
func (m *LandPermitModel) GetAll(ctx context.Context, companyID int64) ([]LandPermit, error) {
	cols, e := m.columns(ctx)
	if e != nil {
		return nil, e
	}
	return m.get(ctx, cols, companyID, nil)
}
func (m *LandPermitModel) GetByID(ctx context.Context, companyID, id int64) (LandPermit, error) {
	cols, e := m.columns(ctx)
	if e != nil {
		return LandPermit{}, e
	}
	items, e := m.get(ctx, cols, companyID, &id)
	if e != nil {
		return LandPermit{}, e
	}
	if len(items) == 0 {
		return LandPermit{}, pgx.ErrNoRows
	}
	return items[0], nil
}

func (m *LandPermitModel) Save(ctx context.Context, companyID int64, id *int64, in LandPermitInput) (LandPermit, error) {
	cols, err := m.columns(ctx)
	if err != nil {
		return LandPermit{}, err
	}
	var permitDate any = in.PermitDate
	if in.PermitDate == "" {
		permitDate = nil
	}
	values := map[string]any{"company_id": companyID, "permit_number": in.PermitNumber, "permit_date": permitDate, "province_id": in.ProvinceID, "address": in.Address, "permit_agency": in.PermitAgency, "agency_phone": in.AgencyPhone, "agency_address": in.AgencyAddress, "signatory_name": in.SignatoryName, "signatory_position": in.SignatoryPosition, "signatory_place": in.SignatoryPlace, "permit_document": in.PermitDocument, "map_data": in.MapData, "map_file": in.MapFile}
	fields := []string{}
	args := []any{}
	placeholders := []string{}
	for _, field := range []string{"company_id", "permit_number", "permit_date", "province_id", "address", "permit_agency", "agency_phone", "agency_address", "signatory_name", "signatory_position", "signatory_place", "permit_document", "map_data", "map_file"} {
		if col := cols[field]; col != "" {
			fields = append(fields, quoteIdent(col))
			args = append(args, values[field])
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
		}
	}
	if id == nil {
		var newID int64
		if err := m.DB.QueryRow(ctx, `SELECT COALESCE(MAX(`+quoteIdent(cols["id"])+`),0)+1 FROM `+landPermitTable).Scan(&newID); err != nil {
			return LandPermit{}, err
		}
		fields = append([]string{quoteIdent(cols["id"])}, fields...)
		args = append([]any{newID}, args...)
		placeholders = make([]string, len(args))
		for i := range args {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
		}
		_, err = m.DB.Exec(ctx, `INSERT INTO `+landPermitTable+` (`+strings.Join(fields, ",")+`) VALUES (`+strings.Join(placeholders, ",")+`)`, args...)
		if err != nil {
			return LandPermit{}, err
		}
		id = &newID
	} else {
		sets := []string{}
		for i, f := range fields {
			sets = append(sets, f+fmt.Sprintf("=$%d", i+1))
		}
		args = append(args, *id, companyID)
		tag, err := m.DB.Exec(ctx, `UPDATE `+landPermitTable+` SET `+strings.Join(sets, ",")+` WHERE `+quoteIdent(cols["id"])+fmt.Sprintf("=$%d", len(args)-1)+` AND `+quoteIdent(cols["company_id"])+fmt.Sprintf("=$%d", len(args)), args...)
		if err != nil {
			return LandPermit{}, err
		}
		if tag.RowsAffected() == 0 {
			return LandPermit{}, pgx.ErrNoRows
		}
	}
	if strings.TrimSpace(in.MapData) != "" || id != nil {
		if err := m.SaveSpatialMapData(ctx, *id, in.MapData); err != nil {
			return LandPermit{}, err
		}
	}
	return m.GetByID(ctx, companyID, *id)
}
func (m *LandPermitModel) Delete(ctx context.Context, companyID, id int64) error {
	cols, e := m.columns(ctx)
	if e != nil {
		return e
	}
	tag, e := m.DB.Exec(ctx, `DELETE FROM `+landPermitTable+` WHERE `+quoteIdent(cols["id"])+`=$1 AND `+quoteIdent(cols["company_id"])+`=$2`, id, companyID)
	if e == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return e
}
