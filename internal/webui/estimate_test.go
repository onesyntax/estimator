package webui

import "testing"

// estimatedWBS approves a WBS, primes the given estimates, and generates them.
func estimatedWBS(t *testing.T, seeds []EstimateSeed) *Session {
	t.Helper()
	s := approvedWBS(t)
	s.PrimeEstimates(seeds)
	s.GenerateEstimates()
	if v := s.BuildView(); v.Error != "" {
		t.Fatalf("generate estimates: unexpected error %q", v.Error)
	}
	return s
}

func uniformSeeds(o, m, p int) []EstimateSeed {
	return []EstimateSeed{
		{TaskNumber: 1, Optimistic: o, MostLikely: m, Pessimistic: p, Reasoning: "AI estimate"},
		{TaskNumber: 2, Optimistic: o, MostLikely: m, Pessimistic: p, Reasoning: "AI estimate"},
		{TaskNumber: 3, Optimistic: o, MostLikely: m, Pessimistic: p, Reasoning: "AI estimate"},
	}
}

func TestGenerateEstimatesRevealsMechanicsAndPricing(t *testing.T) {
	s := estimatedWBS(t, uniformSeeds(2, 3, 5))

	v := s.BuildView()
	e := v.Tasks[0].Estimate
	if e == nil || e.Optimistic != 2 || e.MostLikely != 3 || e.Pessimistic != 5 {
		t.Fatalf("task 1 estimate = %+v, want 2/3/5", e)
	}
	if v.Metrics == nil || v.Metrics.Expected != 10 || v.Metrics.StandardDeviation != 1 || v.Metrics.RelativeStandardDeviation != 9 {
		t.Fatalf("metrics = %+v, want E10 SD1 RSD9", v.Metrics)
	}
	if v.Pricing == nil || v.Pricing.Flag != "green" || v.Pricing.Contract != "fixed-price" || v.Pricing.Basis != "10" {
		t.Fatalf("pricing = %+v, want green fixed-price basis 10", v.Pricing)
	}
}

func TestRedBandShowsNoBasis(t *testing.T) {
	s := estimatedWBS(t, uniformSeeds(1, 8, 40))

	v := s.BuildView()
	if v.Pricing == nil || v.Pricing.Flag != "red" || v.Pricing.Contract != "time-and-materials" || v.Pricing.Basis != "none" {
		t.Fatalf("pricing = %+v, want red time-and-materials basis none", v.Pricing)
	}
}

func mixedSeeds() []EstimateSeed {
	return []EstimateSeed{
		{TaskNumber: 1, Optimistic: 2, MostLikely: 5, Pessimistic: 13, Reasoning: "AI estimate"},
		{TaskNumber: 2, Optimistic: 1, MostLikely: 2, Pessimistic: 3, Reasoning: "AI estimate"},
		{TaskNumber: 3, Optimistic: 3, MostLikely: 8, Pessimistic: 20, Reasoning: "AI estimate"},
	}
}

func TestValidOverrideAppliedOnScreen(t *testing.T) {
	s := estimatedWBS(t, mixedSeeds())
	s.OverrideEstimate(1, EstimateInput{Optimistic: 3, MostLikely: 8, Pessimistic: 20, Reasoning: "Team set them"})

	v := s.BuildView()
	e := v.Tasks[0].Estimate
	if e == nil || e.Optimistic != 3 || e.MostLikely != 8 || e.Pessimistic != 20 {
		t.Fatalf("task 1 estimate = %+v, want 3/8/20", e)
	}
}

func TestInvalidOverrideRefusedInlineLeavesEstimate(t *testing.T) {
	cases := []struct {
		name  string
		input EstimateInput
		msg   string
	}{
		{"not increasing", EstimateInput{Optimistic: 8, MostLikely: 5, Pessimistic: 13, Reasoning: "Adjusted"}, "estimate values must be strictly increasing"},
		{"off fibonacci", EstimateInput{Optimistic: 4, MostLikely: 5, Pessimistic: 13, Reasoning: "Adjusted"}, "estimate value is off the Fibonacci scale"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := estimatedWBS(t, mixedSeeds())
			s.OverrideEstimate(1, tc.input)

			v := s.BuildView()
			if v.Error != tc.msg {
				t.Fatalf("error = %q, want %q", v.Error, tc.msg)
			}
			e := v.Tasks[0].Estimate
			if e == nil || e.Optimistic != 2 || e.MostLikely != 5 || e.Pessimistic != 13 {
				t.Fatalf("task 1 estimate = %+v, want unchanged 2/5/13", e)
			}
		})
	}
}

func TestApproveEstimatesUnlocksProposalStage(t *testing.T) {
	s := estimatedWBS(t, mixedSeeds())
	if s.BuildView().ProposalReachable {
		t.Fatal("proposal should be unreachable before estimates are approved")
	}

	s.ApproveEstimates()

	if !s.BuildView().ProposalReachable {
		t.Fatal("proposal should be reachable after estimates are approved")
	}
}
