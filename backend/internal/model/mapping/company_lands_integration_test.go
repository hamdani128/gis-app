package mapping

import (
	"context"
	"errors"
	"gis-app/backend/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"os"
	"strconv"
	"testing"
	"time"
)

// Opt-in, read-only verification against a configured GIS database.
func TestCompanyLayerDatabase(t *testing.T) {
	selected, err := strconv.ParseInt(os.Getenv("MAPPING_TEST_LAND_ID"), 10, 64)
	if err != nil || selected <= 0 {
		t.Skip("set MAPPING_TEST_LAND_ID to run read-only database verification")
	}
	if err := godotenv.Load("../../../.env"); err != nil {
		t.Fatal("cannot load backend environment")
	}
	db, err := database.Connect()
	if err != nil {
		t.Fatal("cannot connect to GIS database")
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	m := &CompanyLandModel{DB: db}
	first, err := m.List(ctx, 0, selected, 0)
	if err != nil {
		t.Fatal(err)
	}
	if first.SelectedLand["land_id"] != float64(selected) {
		t.Fatal("selected popup metadata does not match selected land")
	}
	var expectedOwner string
	var expectedArea *float64
	err = db.QueryRow(ctx, `SELECT COALESCE(o.name,''),l.land_wide`+companyLandFrom+` WHERE l."id_Gis_Compensated_Lands"=$1`, selected).Scan(&expectedOwner, &expectedArea)
	if err != nil {
		t.Fatal(err)
	}
	if first.SelectedLand["owner"] != expectedOwner {
		t.Fatal("selected popup owner does not match database")
	}
	if expectedArea != nil && first.SelectedLand["land_wide"] != *expectedArea {
		t.Fatal("selected popup hectares do not match database")
	}
	if _, err := m.List(ctx, first.CompanyID+1000000, selected, 0); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("selected land escaped company restriction")
	}
	page := first
	total := 0
	previous := int64(0)
	for {
		ids := []int64{}
		for _, feature := range page.Features {
			id := feature.(map[string]any)["properties"].(map[string]any)["land_id"].(int64)
			if id == selected || id <= previous {
				t.Fatal("selected or previously paginated land leaked into background")
			}
			ids = append(ids, id)
		}
		if len(ids) > 0 {
			var outside int
			err = db.QueryRow(ctx, `SELECT COUNT(*)`+companyLandFrom+` WHERE l."id_Gis_Compensated_Lands"=ANY($1) AND COALESCE(lp."Gis_Companies_id",o."Gis_Companies_id") IS DISTINCT FROM $2`, ids, first.CompanyID).Scan(&outside)
			if err != nil {
				t.Fatal(err)
			}
			if outside != 0 {
				t.Fatal("another company's land appeared in map")
			}
		}
		total += len(page.Features)
		if page.NextCursor == 0 {
			break
		}
		if page.NextCursor <= previous {
			t.Fatal("pagination failed to advance")
		}
		previous = page.NextCursor
		page, err = m.List(ctx, first.CompanyID, selected, previous)
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("company=%d background features=%d; all pages stayed within company", first.CompanyID, total)
}
