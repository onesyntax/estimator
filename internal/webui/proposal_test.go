package webui

import "testing"

// approvedEstimates reaches an approved-estimates session ready for a proposal.
func approvedEstimates(t *testing.T, seeds []EstimateSeed) *Session {
	t.Helper()
	s := estimatedWBS(t, seeds)
	s.ApproveEstimates()
	return s
}

func TestProposalRevealPerBand(t *testing.T) {
	cases := []struct {
		name       string
		seeds      []EstimateSeed
		confidence string
		contract   string
		costLow    int
		costHigh   int
		weeksLow   int
		weeksHigh  int
		pLow       int
		pHigh      int
	}{
		{"green", uniformSeeds(2, 3, 5), "high", "fixed-price", 11400, 13478, 4, 4, 50, 98},
		{"yellow", mixedSeeds(), "medium", "fixed-price-with-buffer", 24469, 28539, 7, 8, 84, 98},
		{"red", uniformSeeds(1, 8, 40), "lower", "time-and-materials", 57310, 70820, 16, 20, 84, 98},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := approvedEstimates(t, tc.seeds)
			s.RequestProposal(TeamInput{Velocity: 3, Capacity: 30, Rate: 120})

			v := s.ProposalView()
			if !v.HasProposal {
				t.Fatalf("expected a proposal, got error %q", v.Error)
			}
			if v.Confidence != tc.confidence {
				t.Errorf("confidence = %q, want %q", v.Confidence, tc.confidence)
			}
			if v.Contract != tc.contract {
				t.Errorf("contract = %q, want %q", v.Contract, tc.contract)
			}
			if v.CostLow != tc.costLow || v.CostHigh != tc.costHigh {
				t.Errorf("cost = %d-%d, want %d-%d", v.CostLow, v.CostHigh, tc.costLow, tc.costHigh)
			}
			if v.WeeksLow != tc.weeksLow || v.WeeksHigh != tc.weeksHigh {
				t.Errorf("weeks = %d-%d, want %d-%d", v.WeeksLow, v.WeeksHigh, tc.weeksLow, tc.weeksHigh)
			}
			if v.SuccessLow != tc.pLow || v.SuccessHigh != tc.pHigh {
				t.Errorf("success = %d/%d, want %d/%d", v.SuccessLow, v.SuccessHigh, tc.pLow, tc.pHigh)
			}
		})
	}
}

func TestProposalShowsScopeAndAssumptions(t *testing.T) {
	s := approvedWBS(t)
	s.PrimeRisks([]RiskSeed{{TaskNumber: 1, Description: "SQL injection"}, {TaskNumber: 2, Description: "XSS"}})
	s.FlagRisks()
	s.PrimeEstimates(mixedSeeds())
	s.GenerateEstimates()
	s.ApproveEstimates()
	s.RequestProposal(TeamInput{Velocity: 3, Capacity: 30, Rate: 120})

	v := s.ProposalView()
	if !containsString(v.Scope, "Login API") {
		t.Errorf("scope = %v, want to include Login API", v.Scope)
	}
	if !containsString(v.Assumptions, "SQL injection") || !containsString(v.Assumptions, "XSS") {
		t.Errorf("assumptions = %v, want to include SQL injection and XSS", v.Assumptions)
	}
}

func TestNonPositiveTeamInputsRefusedInlineNoProposal(t *testing.T) {
	inputs := []TeamInput{
		{Velocity: 0, Capacity: 30, Rate: 120},
		{Velocity: 3, Capacity: 0, Rate: 120},
		{Velocity: 3, Capacity: 30, Rate: 0},
		{Velocity: -3, Capacity: 30, Rate: 120},
	}
	for _, in := range inputs {
		s := approvedEstimates(t, mixedSeeds())
		s.RequestProposal(in)

		v := s.ProposalView()
		if v.Error != "team inputs must be positive" {
			t.Errorf("input %+v: error = %q, want %q", in, v.Error, "team inputs must be positive")
		}
		if v.HasProposal {
			t.Errorf("input %+v: expected no proposal", in)
		}
	}
}

func containsString(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
