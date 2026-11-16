package lands

import "testing"

func TestHCVFeaturesReadsLegacyCoordinates(t *testing.T) {
	for _, raw := range []string{
		`[[115.106903,-2.125969],[115.106758,-2.126405]]`,
		`LINESTRING(115.106903 -2.125969, 115.106758 -2.126405)`,
		`SRID=4326;LINESTRING(115.106903 -2.125969, 115.106758 -2.126405)`,
	} {
		features, err := hcvFeatures(raw)
		if err != nil || len(features) != 1 {
			t.Fatalf("map %q: features=%d, err=%v", raw, len(features), err)
		}
	}
}

func TestHCVFeaturesReadsGeoJSON(t *testing.T) {
	features, err := hcvFeatures(`{"type":"FeatureCollection","features":[{"type":"Feature","properties":{},"geometry":{"type":"Point","coordinates":[115.106903,-2.125969]}}]}`)
	if err != nil || len(features) != 1 {
		t.Fatalf("features=%d, err=%v", len(features), err)
	}
}

func TestHCVFeaturesReadsWKTPolygon(t *testing.T) {
	features, err := hcvFeatures(`POLYGON((103.04291 -3.938772, 103.042887 -3.938816, 103.04288 -3.938838, 103.04291 -3.938772))`)
	if err != nil || len(features) != 1 {
		t.Fatalf("features=%d, err=%v", len(features), err)
	}
	feature, ok := features[0].(map[string]any)
	if !ok {
		t.Fatal("feature bukan object")
	}
	geometry, ok := feature["geometry"].(map[string]any)
	if !ok || geometry["type"] != "Polygon" {
		t.Fatalf("geometry=%v", feature["geometry"])
	}
	wkt, err := hcvGeometryWKT(geometry)
	if err != nil || wkt != `POLYGON((103.04291 -3.938772, 103.042887 -3.938816, 103.04288 -3.938838, 103.04291 -3.938772))` {
		t.Fatalf("WKT=%q, err=%v", wkt, err)
	}
}

func TestHCVIDArgumentMatchesColumnType(t *testing.T) {
	if value, ok := hcvIDArgument(79, "character varying varchar").(string); !ok || value != "79" {
		t.Fatalf("text FK argument = %v", hcvIDArgument(79, "character varying varchar"))
	}
	if value, ok := hcvIDArgument(79, "bigint int8").(int64); !ok || value != 79 {
		t.Fatalf("integer FK argument = %v", hcvIDArgument(79, "bigint int8"))
	}
}
