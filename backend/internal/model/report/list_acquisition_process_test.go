package report

import (
	"math"
	"testing"
)

func TestSpatialAreaHectares(t *testing.T) {
	items := []struct {
		name    string
		wkt     string
		degrees float64
		valid   bool
	}{
		{"polygon", "POLYGON((100 1,100.01 1,100.01 1.01,100 1.01,100 1))", 0.0001, true},
		{"hole", "POLYGON((100 1,100.02 1,100.02 1.02,100 1.02,100 1),(100.005 1.005,100.015 1.005,100.015 1.015,100.005 1.015,100.005 1.005))", 0.0003, true},
		{"multipolygon", "MULTIPOLYGON(((100 1,100.01 1,100.01 1.01,100 1.01,100 1)),((101 1,101.01 1,101.01 1.01,101 1.01,101 1)))", 0.0002, true},
		{"invalid coordinate", "POLYGON((102 6849,103 6849,103 6850,102 6849))", 0, false},
	}
	for _, item := range items {
		t.Run(item.name, func(t *testing.T) {
			got, valid := SpatialAreaHectares(item.wkt)
			if valid != item.valid || (valid && math.Abs(got-item.degrees*legacyAreaFactor) > 0.01) {
				t.Fatalf("area=%v valid=%v", got, valid)
			}
		})
	}
}
