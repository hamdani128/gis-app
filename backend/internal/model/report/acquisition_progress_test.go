package report

import (
	"math"
	"testing"
)

func TestAcquisitionProgressLegacyAreaFactors(t *testing.T) {
	land := &acquisitionProgressLand{area: legacyAreaFactor, price: 250000}
	var stage, compensated AcquisitionProgressMetric
	addAcquisitionProgressMetric(&stage, land, true)
	addAcquisitionProgressMetric(&compensated, land, false)
	finishAcquisitionProgressMetric(&stage, legacyAreaFactor)
	finishAcquisitionProgressMetric(&compensated, legacyAreaFactor)

	if math.Abs(stage.Area-110.93*110.93*100) > 0.001 {
		t.Fatalf("stage area = %f, want legacy 110.93 factor", stage.Area)
	}
	if compensated.Area != legacyAreaFactor || compensated.Percent != 100 {
		t.Fatalf("compensated area/percent = %f/%f, want %f/100", compensated.Area, compensated.Percent, legacyAreaFactor)
	}
	if stage.TotalPrice != 250000 || compensated.TotalPrice != 250000 {
		t.Fatalf("price must be counted once per status: stage=%f compensated=%f", stage.TotalPrice, compensated.TotalPrice)
	}
}
