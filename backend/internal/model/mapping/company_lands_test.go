package mapping

import "testing"

func TestCompanyFeaturesLegacyAndPrivateProperties(t *testing.T) {
	for _, raw := range []string{
		`POLYGON ((98 1,99 1,99 2,98 1))`,
		`{"type":"FeatureCollection","features":[{"type":"Feature","properties":{"secret":"hidden","land_id":999},"geometry":{"type":"Point","coordinates":[98,1]}},{"type":"Feature","properties":{},"geometry":{"type":"Point","coordinates":[99,2]}}]}`,
	} {
		features, err := companyFeatures(raw, 42, 8)
		if err != nil || len(features) == 0 {
			t.Fatalf("geometry lost: %v", err)
		}
		for _, feature := range features {
			properties := feature.(map[string]any)["properties"].(map[string]any)
			if len(properties) != 2 || properties["land_id"] != int64(42) || properties["status"] != int64(8) {
				t.Fatal("unsafe legacy properties or wrong land identity")
			}
		}
	}
	if _, err := companyFeatures("invalid geometry", 42, 8); err == nil {
		t.Fatal("invalid geometry must be reported")
	}
	for _, raw := range []string{`{"type":"FeatureCollection","features":[42]}`, `{"type":"Point","coordinates":[999,1]}`, `{"type":"Polygon","coordinates":[1,2]}`} {
		if _, err := companyFeatures(raw, 42, 8); err == nil {
			t.Fatal("malformed geometry must not break a company layer")
		}
	}
}

func TestCompanyPopupUsesDatabaseDetails(t *testing.T) {
	features, err := companyFeatures(`{"type":"Feature","properties":{"owner":"wrong owner","land_wide":999},"geometry":{"type":"Point","coordinates":[98,1]}}`, 42, 8, map[string]any{
		"owner": "Pemilik sebenarnya", "land_wide": 4.85, "wet_land_wide": 0.0, "dry_land_wide": nil, "company_name": "Company A", "estate_name": "Estate A", "village": "Desa A", "block_plan": "Block II", "permit_number": "10/2013", "owner_ktp": "private",
	})
	if err != nil {
		t.Fatal(err)
	}
	properties := features[0].(map[string]any)["properties"].(map[string]any)
	if properties["owner"] != "Pemilik sebenarnya" || properties["land_wide"] != 4.85 || properties["wet_land_wide"] != 0.0 || properties["dry_land_wide"] != nil {
		t.Fatal("popup metadata was not preserved")
	}
	if _, exists := properties["owner_ktp"]; exists {
		t.Fatal("unrequested identity field exposed")
	}
}
