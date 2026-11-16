package acquisition

import "testing"

func TestParallelApprovalStatusTransitions(t *testing.T) {
	cases := []struct {
		name          string
		stage         ParallelApprovalStage
		current       int
		approved      bool
		otherApproved bool
		want          int
	}{
		{"GIS waits for Legal", GISApprovalStage, 3, true, false, 3},
		{"Legal completes parallel approvals", LegalApprovalStage, 3, true, true, 4},
		{"Operation Control hands off to PD", OperationControlStage, 4, true, false, 5},
		{"PD hands off to Payment", PDApprovalStage, 5, true, false, 6},
		{"Operation Control rejection", OperationControlStage, 4, false, false, 99},
		{"PD rejection", PDApprovalStage, 5, false, false, 99},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.stage.status(); got != tc.current {
				t.Fatalf("current status = %d, want %d", got, tc.current)
			}
			if got := tc.stage.nextStatus(tc.approved, tc.otherApproved); got != tc.want {
				t.Fatalf("next status = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestPDUsesConfiguredProcessSequence(t *testing.T) {
	if PDApprovalStage.SequenceID != 5 {
		t.Fatalf("PD process sequence ID = %d, want 5", PDApprovalStage.SequenceID)
	}
}
