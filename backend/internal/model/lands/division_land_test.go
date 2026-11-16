package lands

import "testing"

func TestDivisionPlanDateAcceptsFormAndLegacyDate(t *testing.T) {
	for _, input := range []string{"2003-04-01", "01/04/2003"} {
		date, err := divisionPlanDate(input)
		if err != nil || date != "01/04/2003" {
			t.Fatalf("date %q: got %q, err=%v", input, date, err)
		}
	}
	if _, err := divisionPlanDate("2003-31-01"); err == nil {
		t.Fatal("invalid date accepted")
	}
}
