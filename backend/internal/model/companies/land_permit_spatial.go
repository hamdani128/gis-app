package companies

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type spatialTableInfo struct {
	Columns  map[string]string
	DataType string
	IDColumn string
	PermitFK string
}

func (m *LandPermitModel) spatialInfo(ctx context.Context) (spatialTableInfo, error) {
	info := spatialTableInfo{Columns: map[string]string{}}
	rows, err := m.DB.Query(ctx, `SELECT column_name, data_type, udt_name, column_default, is_identity FROM information_schema.columns WHERE table_schema='public' AND table_name='Gis_Data_Spatials'`)
	if err != nil {
		return info, err
	}
	defer rows.Close()
	types := map[string]string{}
	generated := map[string]bool{}
	for rows.Next() {
		var name, dataType, udt string
		var defaultValue *string
		var identity string
		if err := rows.Scan(&name, &dataType, &udt, &defaultValue, &identity); err != nil {
			return info, err
		}
		info.Columns[strings.ToLower(name)] = name
		types[strings.ToLower(name)] = strings.ToLower(dataType + " " + udt)
		generated[strings.ToLower(name)] = defaultValue != nil || identity == "YES"
	}
	if err := rows.Err(); err != nil {
		return info, err
	}
	rows.Close()
	for _, name := range []string{"mapdata", "map_data", "geojson", "geometry", "geom"} {
		if actual := info.Columns[name]; actual != "" {
			info.DataType = types[name]
			break
		}
	}
	info.IDColumn = spatialColumn(info, "id_Gis_Data_Spatials", "idGis_Data_Spatials", "id")
	if generated[strings.ToLower(info.IDColumn)] {
		info.IDColumn = ""
	}
	// Prefer the database's actual FK metadata. GIS tables created by older
	// versions use several different generated names for this same relation.
	_ = m.DB.QueryRow(ctx, `
		SELECT kcu.column_name
		FROM information_schema.key_column_usage kcu
		JOIN information_schema.constraint_column_usage ccu
		  ON ccu.constraint_name=kcu.constraint_name
		 AND ccu.constraint_schema=kcu.constraint_schema
		WHERE kcu.table_schema='public' AND kcu.table_name='Gis_Data_Spatials'
		  AND ccu.table_schema='public' AND ccu.table_name='Gis_Land_Permits'
		ORDER BY kcu.ordinal_position LIMIT 1
	`).Scan(&info.PermitFK)
	return info, nil
}

func spatialColumn(info spatialTableInfo, candidates ...string) string {
	for _, candidate := range candidates {
		if name := info.Columns[strings.ToLower(candidate)]; name != "" {
			return name
		}
	}
	return ""
}

func geoJSONFeatures(value any) []any {
	if object, ok := value.(map[string]any); ok {
		switch object["type"] {
		case "FeatureCollection":
			features, _ := object["features"].([]any)
			return features
		case "Feature":
			return []any{object}
		case "Point", "MultiPoint", "LineString", "MultiLineString", "Polygon", "MultiPolygon", "GeometryCollection":
			return []any{map[string]any{"type": "Feature", "properties": map[string]any{}, "geometry": object}}
		}
		if coordinates, ok := object["coordinates"]; ok {
			return geoJSONFeatures(coordinates)
		}
		return nil
	}
	pairs := make([][]float64, 0)
	var collect func(any)
	collect = func(item any) {
		array, ok := item.([]any)
		if !ok {
			return
		}
		if len(array) >= 2 {
			lng, lngOK := array[0].(float64)
			lat, latOK := array[1].(float64)
			if lngOK && latOK {
				pairs = append(pairs, []float64{lng, lat})
				return
			}
		}
		for _, child := range array {
			collect(child)
		}
	}
	if flat, ok := value.([]any); ok && len(flat) > 2 {
		allNumbers := true
		for _, part := range flat {
			if _, ok := part.(float64); !ok {
				allNumbers = false
				break
			}
		}
		if allNumbers {
			for index := 0; index+1 < len(flat); index += 2 {
				pairs = append(pairs, []float64{flat[index].(float64), flat[index+1].(float64)})
			}
		} else {
			collect(flat)
		}
	} else {
		collect(value)
	}
	if len(pairs) == 0 {
		return nil
	}
	geometryType := "LineString"
	var coordinates any = pairs
	if len(pairs) == 1 {
		geometryType = "Point"
	}
	if len(pairs) > 3 && pairs[0][0] == pairs[len(pairs)-1][0] && pairs[0][1] == pairs[len(pairs)-1][1] {
		geometryType = "Polygon"
		coordinates = [][][]float64{pairs}
	}
	if geometryType == "Point" {
		coordinates = pairs[0]
	}
	return []any{map[string]any{"type": "Feature", "properties": map[string]any{}, "geometry": map[string]any{"type": geometryType, "coordinates": coordinates}}}
}

func parseGeoJSONFeatures(raw string) ([]any, error) {
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		numbers := regexp.MustCompile(`[-+]?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d+)?`).FindAllString(raw, -1)
		if len(numbers) < 2 {
			return nil, err
		}
		coords := make([]float64, 0, len(numbers))
		for _, number := range numbers {
			parsed, parseErr := strconv.ParseFloat(number, 64)
			if parseErr == nil {
				coords = append(coords, parsed)
			}
		}
		if len(coords) < 2 {
			return nil, err
		}
		flatCoordinates := make([]any, 0, len(coords))
		for _, coordinate := range coords {
			flatCoordinates = append(flatCoordinates, coordinate)
		}
		value = flatCoordinates
	} else if text, ok := value.(string); ok {
		return parseGeoJSONFeatures(text)
	}
	features := geoJSONFeatures(value)
	if len(features) == 0 {
		return nil, fmt.Errorf("mapdata bukan GeoJSON Feature, FeatureCollection, atau Geometry")
	}
	return features, nil
}

// ParseSpatialFeatures also accepts legacy coordinate arrays and coordinate text.
func ParseSpatialFeatures(raw string) ([]any, error) { return parseGeoJSONFeatures(raw) }

func (m *LandPermitModel) GetSpatialMapData(ctx context.Context, permitIDs []int64) (map[int64]string, error) {
	result := map[int64]string{}
	if len(permitIDs) == 0 {
		return result, nil
	}
	info, err := m.spatialInfo(ctx)
	if err != nil {
		return nil, err
	}
	fk := info.PermitFK
	if fk == "" {
		fk = spatialColumn(info, "Gis_Land_Permits_id", "Gis_Land_Permits_idGis_Land_Permits", "Gis_Land_Permits_id_Gis_Land_Permits", "id_Gis_Land_Permits", "land_permit_id")
	}
	if fk == "" {
		for lowerName, actualName := range info.Columns {
			if strings.Contains(lowerName, "land") && strings.Contains(lowerName, "permit") && strings.Contains(lowerName, "id") {
				fk = actualName
				break
			}
		}
	}
	mapColumn := spatialColumn(info, "mapdata", "map_data", "geojson", "geometry", "geom")
	if fk == "" {
		return nil, fmt.Errorf(`kolom relasi ke "Gis_Land_Permits" tidak ditemukan di "Gis_Data_Spatials"`)
	}
	if mapColumn == "" {
		return nil, fmt.Errorf(`kolom mapdata tidak ditemukan di "Gis_Data_Spatials"`)
	}
	selectMap := `s.` + quoteIdent(mapColumn) + `::text`
	if strings.Contains(info.DataType, "geometry") || strings.Contains(info.DataType, "geography") {
		selectMap = `ST_AsGeoJSON(s.` + quoteIdent(mapColumn) + `)`
	}
	rows, err := m.DB.Query(ctx, `SELECT s.`+quoteIdent(fk)+`, `+selectMap+` FROM public."Gis_Data_Spatials" s WHERE s.`+quoteIdent(fk)+` = ANY($1) ORDER BY s.`+quoteIdent(fk), permitIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	featuresByPermit := map[int64][]any{}
	for rows.Next() {
		var permitID int64
		var raw *string
		if err := rows.Scan(&permitID, &raw); err != nil {
			return nil, err
		}
		if raw == nil || strings.TrimSpace(*raw) == "" {
			continue
		}
		features, err := parseGeoJSONFeatures(*raw)
		if err != nil {
			return nil, fmt.Errorf("mapdata untuk land permit %d tidak dapat dibaca: %w", permitID, err)
		}
		featuresByPermit[permitID] = append(featuresByPermit[permitID], features...)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for id, features := range featuresByPermit {
		encoded, err := json.Marshal(map[string]any{"type": "FeatureCollection", "features": features})
		if err != nil {
			return nil, err
		}
		result[id] = string(encoded)
	}
	return result, nil
}

func (m *LandPermitModel) SaveSpatialMapData(ctx context.Context, permitID int64, raw string) error {
	info, err := m.spatialInfo(ctx)
	if err != nil {
		return err
	}
	fk := info.PermitFK
	if fk == "" {
		fk = spatialColumn(info, "Gis_Land_Permits_id", "Gis_Land_Permits_idGis_Land_Permits", "Gis_Land_Permits_id_Gis_Land_Permits", "id_Gis_Land_Permits", "land_permit_id")
	}
	if fk == "" {
		for lowerName, actualName := range info.Columns {
			if strings.Contains(lowerName, "land") && strings.Contains(lowerName, "permit") && strings.Contains(lowerName, "id") {
				fk = actualName
				break
			}
		}
	}
	mapColumn := spatialColumn(info, "mapdata", "map_data", "geojson", "geometry", "geom")
	if fk == "" || mapColumn == "" {
		return fmt.Errorf("Gis_Data_Spatials harus memiliki Gis_Land_Permits_id dan kolom mapdata")
	}
	features := []any{}
	if strings.TrimSpace(raw) != "" {
		features, err = parseGeoJSONFeatures(raw)
		if err != nil {
			return err
		}
	}
	tx, err := m.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `DELETE FROM public."Gis_Data_Spatials" WHERE `+quoteIdent(fk)+`=$1`, permitID); err != nil {
		return err
	}
	if strings.TrimSpace(raw) == "" {
		return tx.Commit(ctx)
	}
	isGeometry := strings.Contains(info.DataType, "geometry") || strings.Contains(info.DataType, "geography")
	var nextID int64
	if info.IDColumn != "" {
		if _, err = tx.Exec(ctx, `LOCK TABLE public."Gis_Data_Spatials" IN SHARE ROW EXCLUSIVE MODE`); err != nil {
			return err
		}
		if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(`+quoteIdent(info.IDColumn)+`),0)+1 FROM public."Gis_Data_Spatials"`).Scan(&nextID); err != nil {
			return err
		}
	}
	if isGeometry {
		for _, feature := range features {
			object, ok := feature.(map[string]any)
			if !ok {
				continue
			}
			geometry, ok := object["geometry"]
			if !ok || geometry == nil {
				continue
			}
			geometryParts := []any{geometry}
			if geo, ok := geometry.(map[string]any); ok && geo["type"] == "MultiPolygon" {
				if polygons, ok := geo["coordinates"].([]any); ok {
					geometryParts = make([]any, 0, len(polygons))
					for _, polygon := range polygons {
						geometryParts = append(geometryParts, map[string]any{"type": "Polygon", "coordinates": polygon})
					}
				}
			}
			for _, part := range geometryParts {
				encoded, err := json.Marshal(part)
				if err != nil {
					return err
				}
				if info.IDColumn == "" {
					_, err = tx.Exec(ctx, `INSERT INTO public."Gis_Data_Spatials" (`+quoteIdent(fk)+`, `+quoteIdent(mapColumn)+`) VALUES ($1, ST_Force2D(ST_SetSRID(ST_GeomFromGeoJSON($2),4326)))`, permitID, string(encoded))
				} else {
					_, err = tx.Exec(ctx, `INSERT INTO public."Gis_Data_Spatials" (`+quoteIdent(info.IDColumn)+`, `+quoteIdent(fk)+`, `+quoteIdent(mapColumn)+`) VALUES ($1,$2,ST_Force2D(ST_SetSRID(ST_GeomFromGeoJSON($3),4326)))`, nextID, permitID, string(encoded))
					nextID++
				}
				if err != nil {
					return err
				}
			}
		}
	} else {
		var value any = raw
		if strings.Contains(info.DataType, "json") {
			value = json.RawMessage(raw)
		}
		if info.IDColumn == "" {
			_, err = tx.Exec(ctx, `INSERT INTO public."Gis_Data_Spatials" (`+quoteIdent(fk)+`, `+quoteIdent(mapColumn)+`) VALUES ($1,$2)`, permitID, value)
		} else {
			_, err = tx.Exec(ctx, `INSERT INTO public."Gis_Data_Spatials" (`+quoteIdent(info.IDColumn)+`, `+quoteIdent(fk)+`, `+quoteIdent(mapColumn)+`) VALUES ($1,$2,$3)`, nextID, permitID, value)
		}
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
