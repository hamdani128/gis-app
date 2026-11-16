package lands

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const potentialSpatialTable = `public."Gis_Data_Spatials"`

type potentialSpatialSchema struct {
	potentialFK, permitFK, mapColumn, idColumn, dataType string
	potentialFKType, permitFKType, idType                string
}

func (m *PotentialLandModel) spatialSchema(ctx context.Context) (potentialSpatialSchema, error) {
	var schema potentialSpatialSchema
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
	schema.potentialFK = find("Gis_Potential_Lands_id", "Gis_Potential_Lands_idGis_Potential_Lands", "Gis_Potential_Lands_id_Gis_Potential_Lands", "id_Gis_Potential_Lands", "potential_land_id")
	schema.permitFK = find("Gis_Land_Permits_id", "Gis_Land_Permits_idGis_Land_Permits", "Gis_Land_Permits_id_Gis_Land_Permits", "id_Gis_Land_Permits", "land_permit_id")
	schema.mapColumn = find("map", "mapdata", "map_data", "geojson", "geometry", "geom")
	schema.idColumn = find("id_Gis_Data_Spatials", "idGis_Data_Spatials", "id")
	if generated[strings.ToLower(schema.idColumn)] {
		schema.idColumn = ""
	}
	schema.dataType = dataTypes[strings.ToLower(schema.mapColumn)]
	// Legacy installations use different FK names; database constraints take precedence.
	fkRows, err := m.DB.Query(ctx, `SELECT kcu.column_name, ccu.table_name FROM information_schema.key_column_usage kcu JOIN information_schema.constraint_column_usage ccu ON ccu.constraint_name=kcu.constraint_name AND ccu.constraint_schema=kcu.constraint_schema WHERE kcu.table_schema='public' AND kcu.table_name='Gis_Data_Spatials' AND ccu.table_schema='public' AND ccu.table_name IN ('Gis_Potential_Lands','Gis_Land_Permits')`)
	if err != nil {
		return schema, err
	}
	for fkRows.Next() {
		var name, table string
		if err := fkRows.Scan(&name, &table); err != nil {
			fkRows.Close()
			return schema, err
		}
		if table == "Gis_Potential_Lands" && schema.potentialFK == "" {
			schema.potentialFK = name
		} else if table == "Gis_Land_Permits" && schema.permitFK == "" {
			schema.permitFK = name
		}
	}
	err = fkRows.Err()
	fkRows.Close()
	if err != nil {
		return schema, err
	}
	if schema.potentialFK == "" || schema.mapColumn == "" {
		return schema, fmt.Errorf("Gis_Data_Spatials harus memiliki relasi Potential Land dan kolom mapdata")
	}
	schema.potentialFKType = dataTypes[strings.ToLower(schema.potentialFK)]
	schema.permitFKType = dataTypes[strings.ToLower(schema.permitFK)]
	schema.idType = dataTypes[strings.ToLower(schema.idColumn)]
	return schema, nil
}

func potentialIDArgument(id int64, columnType string) any {
	if strings.Contains(columnType, "text") || strings.Contains(columnType, "char") {
		return strconv.FormatInt(id, 10)
	}
	return id
}

func (m *PotentialLandModel) potentialPermitID(ctx context.Context, companyID, id int64) (int64, error) {
	h, p, err := m.schema(ctx)
	if err != nil {
		return 0, err
	}
	var permitText string
	err = m.DB.QueryRow(ctx, `SELECT h.`+quoteIdent(h["land_permit_id"])+`::text FROM `+potentialLandTable+` h JOIN `+potentialPermitTable+` p ON p.`+quoteIdent(p["id"])+`::text=h.`+quoteIdent(h["land_permit_id"])+`::text WHERE h.`+quoteIdent(h["id"])+`::text=$1 AND ($2='0' OR p.`+quoteIdent(p["company_id"])+`::text=$2)`, strconv.FormatInt(id, 10), strconv.FormatInt(companyID, 10)).Scan(&permitText)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(strings.TrimSpace(permitText), 10, 64)
}

func (m *PotentialLandModel) GetMap(ctx context.Context, companyID, id int64) (string, error) {
	if _, err := m.potentialPermitID(ctx, companyID, id); err != nil {
		return "", err
	}
	schema, err := m.spatialSchema(ctx)
	if err != nil {
		return "", err
	}
	selectMap := `s.` + quoteIdent(schema.mapColumn) + `::text`
	if strings.Contains(schema.dataType, "geometry") || strings.Contains(schema.dataType, "geography") {
		selectMap = `ST_AsGeoJSON(ST_MakeValid(s.` + quoteIdent(schema.mapColumn) + `::geometry))`
	}
	rows, err := m.DB.Query(ctx, `SELECT `+selectMap+` FROM `+potentialSpatialTable+` s WHERE s.`+quoteIdent(schema.potentialFK)+`::text=$1`, strconv.FormatInt(id, 10))
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

func (m *PotentialLandModel) GetMapUpload(ctx context.Context, companyID, id int64) (string, error) {
	if _, err := m.potentialPermitID(ctx, companyID, id); err != nil {
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
	err = m.DB.QueryRow(ctx, `SELECT `+quoteIdent(h["map_upload"])+`::text FROM `+potentialLandTable+` WHERE `+quoteIdent(h["id"])+`::text=$1`, strconv.FormatInt(id, 10)).Scan(&path)
	if err != nil {
		return "", err
	}
	if path == nil {
		return "", nil
	}
	return strings.TrimSpace(*path), nil
}

func (m *PotentialLandModel) SaveMap(ctx context.Context, companyID, id int64, raw string, uploadPath *string) (string, error) {
	permitID, err := m.potentialPermitID(ctx, companyID, id)
	if err != nil {
		return "", err
	}
	schema, err := m.spatialSchema(ctx)
	if err != nil {
		return "", err
	}
	var uploadColumn, potentialIDColumn string
	if uploadPath != nil {
		h, _, schemaErr := m.schema(ctx)
		if schemaErr != nil {
			return "", schemaErr
		}
		uploadColumn = h["map_upload"]
		potentialIDColumn = h["id"]
		if uploadColumn == "" {
			return "", fmt.Errorf("Gis_Potential_Lands harus memiliki kolom map_upload untuk menyimpan file koordinat")
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
	if _, err = tx.Exec(ctx, `DELETE FROM `+potentialSpatialTable+` WHERE `+quoteIdent(schema.potentialFK)+`=$1`, potentialIDArgument(id, schema.potentialFKType)); err != nil {
		return "", err
	}
	if len(features) > 0 {
		var nextID int64
		if schema.idColumn != "" {
			if _, err = tx.Exec(ctx, `LOCK TABLE `+potentialSpatialTable+` IN SHARE ROW EXCLUSIVE MODE`); err != nil {
				return "", err
			}
			if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(`+quoteIdent(schema.idColumn)+`::text::bigint),0)+1 FROM `+potentialSpatialTable).Scan(&nextID); err != nil {
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
				columns, values := []string{quoteIdent(schema.potentialFK)}, []any{potentialIDArgument(id, schema.potentialFKType)}
				if schema.permitFK != "" {
					columns = append(columns, quoteIdent(schema.permitFK))
					values = append(values, potentialIDArgument(permitID, schema.permitFKType))
				}
				if schema.idColumn != "" {
					columns = append(columns, quoteIdent(schema.idColumn))
					values = append(values, potentialIDArgument(nextID, schema.idType))
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
				_, err = tx.Exec(ctx, `INSERT INTO `+potentialSpatialTable+` (`+strings.Join(columns, ",")+`) VALUES (`+strings.Join(placeholders, ",")+`)`, values...)
				if err != nil {
					return "", err
				}
			}
		}
	}
	if uploadPath != nil {
		if _, err = tx.Exec(ctx, `UPDATE `+potentialLandTable+` SET `+quoteIdent(uploadColumn)+`=$1 WHERE `+quoteIdent(potentialIDColumn)+`::text=$2`, *uploadPath, strconv.FormatInt(id, 10)); err != nil {
			return "", err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return "", err
	}
	return string(canonical), nil
}
