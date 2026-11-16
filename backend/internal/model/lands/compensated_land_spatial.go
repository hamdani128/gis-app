package lands

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type compensatedSpatialSchema struct {
	landFK, permitFK, mapColumn, idColumn, mapType, landType, permitType, idType string
}

// ParseCompensatedSpatialFeatures preserves legacy normalization used by land maps.
func ParseCompensatedSpatialFeatures(raw string) ([]any, error) { return hcvFeatures(raw) }

// CompensatedSpatialSource shares the legacy schema detection with read-only map layers.
// Both expressions are built exclusively from database column names, never user input.
func CompensatedSpatialSource(ctx context.Context, m *CompensatedLandModel) (landColumn, mapExpression string, err error) {
	schema, err := m.spatialSchema(ctx)
	if err != nil {
		return "", "", err
	}
	landColumn = `s.` + quoteIdent(schema.landFK)
	mapExpression = `s.` + quoteIdent(schema.mapColumn) + `::text`
	if strings.Contains(schema.mapType, "geometry") || strings.Contains(schema.mapType, "geography") {
		mapExpression = `ST_AsGeoJSON(ST_MakeValid(s.` + quoteIdent(schema.mapColumn) + `::geometry))`
	}
	return landColumn, mapExpression, nil
}

func (m *CompensatedLandModel) spatialSchema(ctx context.Context) (compensatedSpatialSchema, error) {
	var schema compensatedSpatialSchema
	rows, err := m.DB.Query(ctx, `SELECT column_name, data_type, udt_name, column_default, is_identity FROM information_schema.columns WHERE table_schema='public' AND table_name='Gis_Data_Spatials'`)
	if err != nil {
		return schema, err
	}
	defer rows.Close()
	actual, types, generated := map[string]string{}, map[string]string{}, map[string]bool{}
	for rows.Next() {
		var name, dataType, udt, identity string
		var defaultValue *string
		if err := rows.Scan(&name, &dataType, &udt, &defaultValue, &identity); err != nil {
			return schema, err
		}
		key := strings.ToLower(name)
		actual[key], types[key], generated[key] = name, strings.ToLower(dataType+" "+udt), defaultValue != nil || identity == "YES"
	}
	if err := rows.Err(); err != nil {
		return schema, err
	}
	find := func(names ...string) string {
		for _, name := range names {
			if col := actual[strings.ToLower(name)]; col != "" {
				return col
			}
		}
		return ""
	}
	schema.landFK = find("Gis_Compensated_Lands_id", "Gis_Compensated_Lands_idGis_Compensated_Lands", "Gis_Compensated_Lands_id_Gis_Compensated_Lands", "compensated_land_id")
	schema.permitFK = find("Gis_Land_Permits_id", "Gis_Land_Permits_idGis_Land_Permits", "Gis_Land_Permits_id_Gis_Land_Permits", "land_permit_id")
	schema.mapColumn = find("map", "mapdata", "map_data", "geojson", "geometry", "geom")
	schema.idColumn = find("id_Gis_Data_Spatial", "id_Gis_Data_Spatials", "idGis_Data_Spatials", "id")
	fkRows, err := m.DB.Query(ctx, `SELECT kcu.column_name, ccu.table_name FROM information_schema.key_column_usage kcu JOIN information_schema.constraint_column_usage ccu ON ccu.constraint_name=kcu.constraint_name AND ccu.constraint_schema=kcu.constraint_schema WHERE kcu.table_schema='public' AND kcu.table_name='Gis_Data_Spatials' AND ccu.table_schema='public' AND ccu.table_name IN ('Gis_Compensated_Lands','Gis_Land_Permits')`)
	if err != nil {
		return schema, err
	}
	for fkRows.Next() {
		var name, table string
		if err := fkRows.Scan(&name, &table); err != nil {
			fkRows.Close()
			return schema, err
		}
		if table == "Gis_Compensated_Lands" {
			schema.landFK = name
		} else if table == "Gis_Land_Permits" {
			schema.permitFK = name
		}
	}
	err = fkRows.Err()
	fkRows.Close()
	if err != nil {
		return schema, err
	}
	if generated[strings.ToLower(schema.idColumn)] {
		schema.idColumn = ""
	}
	if schema.landFK == "" || schema.mapColumn == "" {
		return schema, fmt.Errorf("Gis_Data_Spatials harus memiliki relasi Compensated Land dan kolom mapdata")
	}
	schema.mapType = types[strings.ToLower(schema.mapColumn)]
	schema.landType = types[strings.ToLower(schema.landFK)]
	schema.permitType = types[strings.ToLower(schema.permitFK)]
	schema.idType = types[strings.ToLower(schema.idColumn)]
	return schema, nil
}

func (m *CompensatedLandModel) permitID(ctx context.Context, companyID, id int64) (int64, error) {
	var permitID int64
	err := m.DB.QueryRow(ctx, `SELECT lp."id_Land_Permits" FROM `+proposeLandTable+` c JOIN public."Gis_Aquistion_Divisions" d ON d."id_Gis_Aqcuistion_Divisions"=c."Gis_Aquistion_Divisions_id" JOIN public."Gis_Potential_Lands" p ON p."id_Gis_Potential_Lands"=d."Gis_Potential_Lands_id" JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id" WHERE c."id_Gis_Compensated_Lands"=$1 AND c.status=$3 AND ($2::bigint=0 OR lp."Gis_Companies_id"=$2)`, id, companyID, m.Status).Scan(&permitID)
	return permitID, err
}

func (m *CompensatedLandModel) GetMap(ctx context.Context, companyID, id int64) (string, error) {
	// A compensated land can belong to a company through its owner even when
	// its division/potential-land/permit chain has not been populated.
	var exists int
	err := m.DB.QueryRow(ctx, `SELECT 1 FROM `+proposeLandTable+` c
		LEFT JOIN public."Gis_Aquistion_Divisions" d ON d."id_Gis_Aqcuistion_Divisions"=c."Gis_Aquistion_Divisions_id"
		LEFT JOIN public."Gis_Potential_Lands" p ON p."id_Gis_Potential_Lands"=d."Gis_Potential_Lands_id"
		LEFT JOIN public."Gis_Land_Permits" lp ON lp."id_Land_Permits"=p."Gis_Land_Permits_id"
		LEFT JOIN public."Gis_Land_Owners" o ON o."id_Gis_Land_Owners"=c."Gis_Land_Owners_id"
		WHERE c."id_Gis_Compensated_Lands"=$1 AND c.status=$3
		AND ($2::bigint=0 OR COALESCE(lp."Gis_Companies_id",o."Gis_Companies_id")=$2)`, id, companyID, m.Status).Scan(&exists)
	if err != nil {
		return "", err
	}
	schema, err := m.spatialSchema(ctx)
	if err != nil {
		return "", err
	}
	selectMap := `s.` + quoteIdent(schema.mapColumn) + `::text`
	if strings.Contains(schema.mapType, "geometry") || strings.Contains(schema.mapType, "geography") {
		selectMap = `ST_AsGeoJSON(ST_MakeValid(s.` + quoteIdent(schema.mapColumn) + `::geometry))`
	}
	rows, err := m.DB.Query(ctx, `SELECT `+selectMap+` FROM public."Gis_Data_Spatials" s WHERE s.`+quoteIdent(schema.landFK)+`::text=$1`, strconv.FormatInt(id, 10))
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

func (m *CompensatedLandModel) SaveMap(ctx context.Context, companyID, id int64, raw string) (string, error) {
	permitID, err := m.permitID(ctx, companyID, id)
	if err != nil {
		return "", err
	}
	schema, err := m.spatialSchema(ctx)
	if err != nil {
		return "", err
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
	if _, err = tx.Exec(ctx, `DELETE FROM public."Gis_Data_Spatials" WHERE `+quoteIdent(schema.landFK)+`=$1`, hcvIDArgument(id, schema.landType)); err != nil {
		return "", err
	}
	var nextID int64
	if schema.idColumn != "" && len(features) > 0 {
		if _, err = tx.Exec(ctx, `LOCK TABLE public."Gis_Data_Spatials" IN SHARE ROW EXCLUSIVE MODE`); err != nil {
			return "", err
		}
		if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(`+quoteIdent(schema.idColumn)+`::text::bigint),0)+1 FROM public."Gis_Data_Spatials"`).Scan(&nextID); err != nil {
			return "", err
		}
	}
	isGeometry := strings.Contains(schema.mapType, "geometry") || strings.Contains(schema.mapType, "geography")
	isWKT := strings.EqualFold(schema.mapColumn, "map") && !isGeometry && !strings.Contains(schema.mapType, "json")
	for _, feature := range features {
		object, ok := feature.(map[string]any)
		if !ok {
			return "", fmt.Errorf("Feature GeoJSON tidak valid")
		}
		geometry, ok := object["geometry"].(map[string]any)
		if !ok {
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
			columns, values := []string{quoteIdent(schema.landFK)}, []any{hcvIDArgument(id, schema.landType)}
			if schema.permitFK != "" {
				columns = append(columns, quoteIdent(schema.permitFK))
				values = append(values, hcvIDArgument(permitID, schema.permitType))
			}
			if schema.idColumn != "" {
				columns = append(columns, quoteIdent(schema.idColumn))
				values = append(values, hcvIDArgument(nextID, schema.idType))
				nextID++
			}
			var mapValue any = feature
			if isGeometry {
				mapValue = part
			}
			var stored string
			if isWKT {
				stored, err = hcvGeometryWKT(geometry)
			} else {
				var data []byte
				data, err = json.Marshal(mapValue)
				stored = string(data)
			}
			if err != nil {
				return "", err
			}
			columns = append(columns, quoteIdent(schema.mapColumn))
			values = append(values, stored)
			placeholders := make([]string, len(values))
			for i := range values {
				placeholders[i] = fmt.Sprintf("$%d", i+1)
			}
			if isGeometry {
				placeholders[len(placeholders)-1] = `ST_Force2D(ST_SetSRID(ST_GeomFromGeoJSON(` + placeholders[len(placeholders)-1] + `),4326))`
			}
			if _, err = tx.Exec(ctx, `INSERT INTO public."Gis_Data_Spatials" (`+strings.Join(columns, ",")+`) VALUES (`+strings.Join(placeholders, ",")+`)`, values...); err != nil {
				return "", err
			}
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return "", err
	}
	return string(canonical), nil
}
