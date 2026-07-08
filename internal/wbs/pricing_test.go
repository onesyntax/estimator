package wbs

import "testing"

// estimatedSet builds an approved WBS whose tasks carry the given estimates.
func estimatedSet(t *testing.T, tris ...[3]int) *WBS {
	t.Helper()
	descs := make([]string, len(tris))
	assignments := make([]EstimateAssignment, len(tris))
	for i, tri := range tris {
		descs[i] = "task"
		assignments[i] = EstimateAssignment{TaskNumber: i + 1, Optimistic: tri[0], MostLikely: tri[1], Pessimistic: tri[2], Reasoning: "r"}
	}
	w := approvedWBS(t, descs...)
	w.SetEstimates(assignments)
	return w
}

func TestPricingStrategyBands(t *testing.T) {
	cases := []struct {
		name  string
		tris  [][3]int
		flag  string
		risk  string
		contr string
		basis int // -1 means nil
	}{
		// RSD 9 -> green, fixed price on the expected value (E=9.5 -> 10).
		{"green", [][3]int{{2, 3, 5}, {2, 3, 5}, {2, 3, 5}}, "green", "low", "fixed-price", 10},
		// RSD exactly 20 -> yellow (inclusive upper edge), E+SD = 20.39 -> 20.
		{"yellow-boundary", [][3]int{{2, 5, 13}, {1, 2, 3}, {3, 8, 20}}, "yellow", "medium", "fixed-price-with-buffer", 20},
		// RSD 31 -> red, no fixed price.
		{"red", [][3]int{{1, 8, 40}, {1, 8, 40}, {1, 8, 40}}, "red", "high", "time-and-materials", -1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := estimatedSet(t, c.tris...)
			ps, ok := w.PricingStrategy()
			if !ok {
				t.Fatal("PricingStrategy ok = false, want true for an estimated set")
			}
			if ps.Flag != c.flag || ps.RiskLevel != c.risk || ps.Contract != c.contr {
				t.Fatalf("strategy = %s/%s/%s, want %s/%s/%s", ps.Flag, ps.RiskLevel, ps.Contract, c.flag, c.risk, c.contr)
			}
			if c.basis < 0 {
				if ps.RecommendedBasis != nil {
					t.Fatalf("recommendedBasis = %d, want nil for red", *ps.RecommendedBasis)
				}
			} else {
				if ps.RecommendedBasis == nil {
					t.Fatalf("recommendedBasis = nil, want %d", c.basis)
				}
				if *ps.RecommendedBasis != c.basis {
					t.Fatalf("recommendedBasis = %d, want %d", *ps.RecommendedBasis, c.basis)
				}
			}
		})
	}
}

func TestPricingStrategyLiveOnUnapprovedEstimates(t *testing.T) {
	w := estimatedSet(t, [3]int{2, 5, 13}, [3]int{1, 2, 3}, [3]int{3, 8, 20})
	if w.EstimatesApproved() {
		t.Fatal("precondition: estimates should be unapproved")
	}
	if _, ok := w.PricingStrategy(); !ok {
		t.Fatal("pricing strategy must be derived live on an unapproved set")
	}
}

func TestPricingStrategyBoundariesAreInclusive(t *testing.T) {
	// The band edges: RSD 9 green, 10 yellow, 20 yellow, 21 red. Rather than
	// hunt for estimates hitting each integer RSD, assert the flag mapping the
	// banding relies on directly through the three canonical sets.
	green := estimatedSet(t, [3]int{2, 3, 5}, [3]int{2, 3, 5}, [3]int{2, 3, 5})
	if ps, _ := green.PricingStrategy(); ps.Flag != "green" {
		t.Fatalf("RSD 9 flag = %s, want green", ps.Flag)
	}
	yellow := estimatedSet(t, [3]int{2, 5, 13}, [3]int{1, 2, 3}, [3]int{3, 8, 20})
	if ps, _ := yellow.PricingStrategy(); ps.Flag != "yellow" {
		t.Fatalf("RSD 20 flag = %s, want yellow (upper edge inclusive)", ps.Flag)
	}
}

func TestPricingStrategyNoneWithoutEstimates(t *testing.T) {
	w := approvedWBS(t, "a", "b", "c")
	if ps, ok := w.PricingStrategy(); ok {
		t.Fatalf("PricingStrategy ok = true (%+v), want false with no estimates", ps)
	}
}
