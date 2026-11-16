package lands

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	companySpatial "gis-app/backend/internal/model/companies"
)

const hcvSpatialTable = `public."Gis_Data_Spatials"`

type hcvSpatialSchema struct {
	hcvFK, permitFK, mapColumn, idColumn, dataType string
	hcvFKType, permitFKType, idType                string
}

func (m *HCVLandModel) spatialSchema(ctx context.Context) (hcvSpatialSchema, error) {
	var schema hcvSpatialSchema
	rows, err := m.DB.Query(ctx, `SELECT column_name, data_type, udt_name, column_default, is_identity FROM information_schema.columns WHERE table_schema='public' AND table_name='Gis_Data_Spatials'`)
	if err != nil {
		return schema, err
	}
	defer rows.Close()
	columns := map[string]string{}
	generated := map[string]bool{}
	dataTypes := map[string]string{}
	for rows.Next() {
		var name, dataType, udt, identity string
		var defaultValue *string
		if err := rows.Scan(&name, &dataType, &udt, &defaultValue, &identity); err != nil {
			return schema, err
		}
		key := strings.ToLower(name)
		columns[key], dataTypes[key] = name, strings.ToLower(dataType+" "+udt)
		generated[key] = defaultValue != nil || identity == "YES"
	}
	if err := rows.Err(); err != nil {
		return schema, err
	}
	find := func(names ...string) string {
		for _, name := range names {
			if actual := columns[strings.ToLower(name)]; actual != "" {
				return actual
			}
		}
		return ""
	}
	schema.hcvFK = find("Gis_HCV_Lands_id", "Gis_HCV_Lands_idGis_HCV_Lands", "Gis_HCV_Lands_id_Gis_HCV_Lands", "id_Gis_HCV_Lands", "hcv_land_id")
	schema.permitFK = find("Gis_Land_Permits_id", "Gis_Land_Permits_idGis_Land_Permits", "Gis_Land_Permits_id_Gis_Land_Permits", "id_Gis_Land_Permits", "land_permit_id")
	schema.mapColumn = find("map", "mapdata", "map_data", "geojson", "geometry", "geom")
	schema.idColumn = find("id_Gis_Data_Spatials", "idGis_Data_Spatials", "id")
	if generated[strings.ToLower(schema.idColumn)] {
		schema.idColumn = ""
	}
	schema.dataType = dataTypes[strings.ToLower(schema.mapColumn)]
	// Legacy installations use different FK names; database constraints take precedence.
	fkRows, err := m.DB.Query(ctx, `SELECT kcu.column_name, ccu.table_name FROM information_schema.key_column_usage kcu JOIN information_schema.constraint_column_usage ccu ON ccu.constraint_name=kcu.constraint_name AND ccu.constraint_schema=kcu.constraint_schema WHERE kcu.table_schema='public' AND kcu.table_name='Gis_Data_Spatials' AND ccu.table_schema='public' AND ccu.table_name IN ('Gis_HCV_Lands','Gis_Land_Permits')`)
	if err != nil {
		return schema, err
	}
	for fkRows.Next() {
		var name, table string
		if err := fkRows.Scan(&name, &table); err != nil {
			fkRows.Close()
			return schema, err
		}
		if table == "Gis_HCV_Lands" && schema.hcvFK == "" {
			schema.hcvFK = name
		} else if table == "Gis_Land_Permits" && schema.permitFK == "" {
			schema.permitFK = name
		}
	}
	err = fkRows.Err()
	fkRows.Close()
	if err != nil {
		return schema, err
	}
	if schema.hcvFK == "" || schema.mapColumn == "" {
		return schema, fmt.Errorf("Gis_Data_Spatials harus memiliki relasi HCV Land dan kolom mapdata")
	}
	schema.hcvFKType = dataTypes[strings.ToLower(schema.hcvFK)]
	schema.permitFKType = dataTypes[strings.ToLower(schema.permitFK)]
	schema.idType = dataTypes[strings.ToLower(schema.idColumn)]
	return schema, nil
}

func hcvIDArgument(id int64, columnType string) any {
	if strings.Contains(columnType, "text") || strings.Contains(columnType, "char") {
		return strconv.FormatInt(id, 10)
	}
	return id
}

func (m *HCVLandModel) hcvPermitID(ctx context.Context, companyID, id int64) (int64, error) {
	h, p, err := m.schema(ctx)
	if err != nil {
		return 0, err
	}
	var permitText string
	err = m.DB.QueryRow(ctx, `SELECT h.`+quoteIdent(h["land_permit_id"])+`::text FROM `+hcvLandTable+` h JOIN `+hcvPermitTable+` p ON p.`+quoteIdent(p["id"])+`=h.`+quoteIdent(h["land_permit_id"])+` WHERE h.`+quoteIdent(h["id"])+`::text=$1 AND ($2='0' OR p.`+quoteIdent(p["company_id"])+`::text=$2)`, strconv.FormatInt(id, 10), strconv.FormatInt(companyID, 10)).Scan(&permitText)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(strings.TrimSpace(permitText), 10, 64)
}

func hcvFeatures(raw string) ([]any, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(strings.ToUpper(raw), "SRID=") {
		if separator := strings.IndexByte(raw, ';'); separator >= 0 {
			raw = raw[separator+1:]
		}
	}
	features, err := companySpatial.ParseSpatialFeatures(raw)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(features)
	if err != nil {
		return nil, err
	}
	var normalized []any
	if err := json.Unmarshal(encoded, &normalized); err != nil {
		return nil, err
	}
	return normalized, nil
}

func hcvGeometryWKT(geometry map[string]any) (string, error) {
	point := func(value any) (string, error) {
		pair, ok := value.([]any)
		if !ok || len(pair) < 2 {
			return "", fmt.Errorf("koordinat WKT tidak valid")
		}
		lng, lngOK := pair[0].(float64)
		lat, latOK := pair[1].(float64)
		if !lngOK || !latOK {
			return "", fmt.Errorf("koordinat WKT tidak valid")
		}
		return strconv.FormatFloat(lng, 'f', -1, 64) + " " + strconv.FormatFloat(lat, 'f', -1, 64), nil
	}
	var line func(any) (string, error)
	line = func(value any) (string, error) {
		items, ok := value.([]any)
		if !ok || len(items) == 0 {
			return "", fmt.Errorf("baris WKT kosong")
		}
		parts := make([]string, 0, len(items))
		for _, item := range items {
			text, err := point(item)
			if err != nil {
				return "", err
			}
			parts = append(parts, text)
		}
		return strings.Join(parts, ", "), nil
	}
	groups := func(value any, wrap bool) (string, error) {
		items, ok := value.([]any)
		if !ok || len(items) == 0 {
			return "", fmt.Errorf("geometri WKT kosong")
		}
		parts := make([]string, 0, len(items))
		for _, item := range items {
			text, err := line(item)
			if err != nil {
				return "", err
			}
			if wrap {
				text = "(" + text + ")"
			}
			parts = append(parts, text)
		}
		return strings.Join(parts, ", "), nil
	}
	coordinates := geometry["coordinates"]
	switch geometry["type"] {
	case "Point":
		text, err := point(coordinates)
		return "POINT(" + text + ")", err
	case "LineString":
		text, err := line(coordinates)
		return "LINESTRING(" + text + ")", err
	case "Polygon":
		text, err := groups(coordinates, true)
		return "POLYGON(" + text + ")", err
	case "MultiPoint":
		text, err := line(coordinates)
		return "MULTIPOINT(" + text + ")", err
	case "MultiLineString":
		text, err := groups(coordinates, true)
		return "MULTILINESTRING(" + text + ")", err
	case "MultiPolygon":
		polygons, ok := coordinates.([]any)
		if !ok || len(polygons) == 0 {
			return "", fmt.Errorf("MultiPolygon WKT kosong")
		}
		parts := make([]string, 0, len(polygons))
		for _, polygon := range polygons {
			rings, err := groups(polygon, true)
			if err != nil {
				return "", err
			}
			parts = append(parts, "("+rings+")")
		}
		return "MULTIPOLYGON(" + strings.Join(parts, ", ") + ")", nil
	}
	return "", fmt.Errorf("geometri %v tidak didukung untuk kolom map WKT", geometry["type"])
}

func (m *HCVLandModel) GetMap(ctx context.Context, companyID, id int64) (string, error) {
	if _, err := m.hcvPermitID(ctx, companyID, id); err != nil {
		return "", err
	}
	schema, err := m.spatialSchema(ctx)
	if err != nil {
		return "", err
	}
	selectMap := `s.` + quoteIdent(schema.mapColumn) + `::text`
	if strings.Contains(schema.dataType, "geometry") || strings.Contains(schema.dataType, "geography") {
		selectMap = `ST_AsGeoJSON(s.` + quoteIdent(schema.mapColumn) + `)`
	}
	rows, err := m.DB.Query(ctx, `SELECT `+selectMap+` FROM `+hcvSpatialTable+` s WHERE s.`+quoteIdent(schema.hcvFK)+`=$1`, hcvIDArgument(id, schema.hcvFKType))
	if err != nil {
		return "", err
	}
	defer rows.Close()
	features := []any{}
	for rows.Next() {
		var raw *string
		if err := rows.Scan(&raw); err != nil {
			return "", err
		}
		if raw == nil || strings.TrimSpace(*raw) == "" {
			continue
		}
		parts, err := hcvFeatures(*raw)
		if err != nil {
			return "", err
		}
		features = append(features, parts...)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	data, err := json.Marshal(map[string]any{"type": "FeatureCollection", "features": features})
	return string(data), err
}

func (m *HCVLandModel) GetMapUpload(ctx context.Context, companyID, id int64) (string, error) {
	if _, err := m.hcvPermitID(ctx, companyID, id); err != nil {
		return "", err
	}
	h, _, err := m.schema(ctx)
	if err != nil {
		return "", err
	}
	if h["map_upload"] == "" {
		return "", nil
	}
	var path *string
	err = m.DB.QueryRow(ctx, `SELECT `+quoteIdent(h["map_upload"])+`::text FROM `+hcvLandTable+` WHERE `+quoteIdent(h["id"])+`::text=$1`, strconv.FormatInt(id, 10)).Scan(&path)
	if err != nil {
		return "", err
	}
	if path == nil {
		return "", nil
	}
	return strings.TrimSpace(*path), nil
}

func (m *HCVLandModel) SaveMap(ctx context.Context, companyID, id int64, raw string, uploadPath *string) (string, error) {
	permitID, err := m.hcvPermitID(ctx, companyID, id)
	if err != nil {
		return "", err
	}
	schema, err := m.spatialSchema(ctx)
	if err != nil {
		return "", err
	}
	var uploadColumn, hcvIDColumn string
	if uploadPath != nil {
		h, _, schemaErr := m.schema(ctx)
		if schemaErr != nil {
			return "", schemaErr
		}
		uploadColumn = h["map_upload"]
		hcvIDColumn = h["id"]
		if uploadColumn == "" {
			return "", fmt.Errorf("Gis_HCV_Lands harus memiliki kolom map_upload untuk menyimpan file koordinat")
		}
	}
	features := []any{}
	if strings.TrimSpace(raw) != "" {
		features, err = hcvFeatures(raw)
		if err != nil {
			return "", err
		}
	}
	canonical, err := json.Marshal(map[string]any{"type": "FeatureCollection", "features": features})
	if err != nil {
		return "", err
	}
	tx, err := m.DB.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `DELETE FROM `+hcvSpatialTable+` WHERE `+quoteIdent(schema.hcvFK)+`=$1`, hcvIDArgument(id, schema.hcvFKType)); err != nil {
		return "", err
	}
	if len(features) > 0 {
		var nextID int64
		if schema.idColumn != "" {
			if _, err = tx.Exec(ctx, `LOCK TABLE `+hcvSpatialTable+` IN SHARE ROW EXCLUSIVE MODE`); err != nil {
				return "", err
			}
			if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(`+quoteIdent(schema.idColumn)+`::text::bigint),0)+1 FROM `+hcvSpatialTable).Scan(&nextID); err != nil {
				return "", err
			}
		}
		isGeometry := strings.Contains(schema.dataType, "geometry") || strings.Contains(schema.dataType, "geography")
		isWKTText := strings.EqualFold(schema.mapColumn, "map") && !isGeometry && !strings.Contains(schema.dataType, "json")
		for _, feature := range features {
			object, ok := feature.(map[string]any)
			if !ok {
				return "", fmt.Errorf("Feature GeoJSON tidak valid")
			}
			geometry, ok := object["geometry"].(map[string]any)
			if !ok || geometry["type"] == nil {
				return "", fmt.Errorf("Geometry GeoJSON tidak valid")
			}
			parts := []any{geometry}
			if isGeometry && geometry["type"] == "MultiPolygon" {
				polygons, ok := geometry["coordinates"].([]any)
				if !ok {
					return "", fmt.Errorf("Koordinat MultiPolygon tidak valid")
				}
				parts = []any{}
				for _, polygon := range polygons {
					parts = append(parts, map[string]any{"type": "Polygon", "coordinates": polygon})
				}
			}
			for _, part := range parts {
				columns, values := []string{quoteIdent(schema.hcvFK)}, []any{hcvIDArgument(id, schema.hcvFKType)}
				if schema.permitFK != "" {
					columns = append(columns, quoteIdent(schema.permitFK))
					values = append(values, hcvIDArgument(permitID, schema.permitFKType))
				}
				if schema.idColumn != "" {
					columns = append(columns, quoteIdent(schema.idColumn))
					values = append(values, hcvIDArgument(nextID, schema.idType))
					nextID++
				}
				var mapValue any = part
				if !isGeometry {
					mapValue = feature
				}
				var storedMap string
				if isWKTText {
					storedMap, err = hcvGeometryWKT(geometry)
				} else {
					var encoded []byte
					encoded, err = json.Marshal(mapValue)
					storedMap = string(encoded)
				}
				if err != nil {
					return "", err
				}
				columns = append(columns, quoteIdent(schema.mapColumn))
				values = append(values, storedMap)
				placeholders := make([]string, len(values))
				for i := range values {
					placeholders[i] = fmt.Sprintf("$%d", i+1)
				}
				if isGeometry {
					placeholders[len(placeholders)-1] = `ST_Force2D(ST_SetSRID(ST_GeomFromGeoJSON(` + placeholders[len(placeholders)-1] + `),4326))`
				}
				_, err = tx.Exec(ctx, `INSERT INTO `+hcvSpatialTable+` (`+strings.Join(columns, ",")+`) VALUES (`+strings.Join(placeholders, ",")+`)`, values...)
				if err != nil {
					return "", err
				}
			}
		}
	}
	if uploadPath != nil {
		if _, err = tx.Exec(ctx, `UPDATE `+hcvLandTable+` SET `+quoteIdent(uploadColumn)+`=$1 WHERE `+quoteIdent(hcvIDColumn)+`::text=$2`, *uploadPath, strconv.FormatInt(id, 10)); err != nil {
			return "", err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return "", err
	}
	return string(canonical), nil
}
