package httpapi

import (
	"net/http"
	"testing"
)

// estimateTriples primes the three tasks with the given O/M/P triples.
func estimateTriples(tris ...[3]int) []map[string]any {
	out := make([]map[string]any, len(tris))
	for i, tri := range tris {
		out[i] = map[string]any{"taskNumber": i + 1, "optimistic": tri[0], "mostLikely": tri[1], "pessimistic": tri[2], "reasoning": "r"}
	}
	return out
}

// wbsWithEstimates sets up an approved WBS and generates the given estimate set
// (leaving the estimates unapproved).
func (c *client) wbsWithEstimates(tris ...[3]int) string {
	c.t.Helper()
	id := c.setupWBS()
	c.approve(id)
	c.primeEstimates(estimateTriples(tris...))
	if status, _ := c.generateEstimates(id); status != http.StatusOK {
		c.t.Fatalf("generate estimates status = %d, want 200", status)
	}
	return id
}

func pricingOf(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	return bodyFieldOf(t, body, "pricingStrategy")
}

func TestPricingStrategyBandsLiveOnUnapprovedSet(t *testing.T) {
	cases := []struct {
		name     string
		tris     [][3]int
		flag     string
		risk     string
		contract string
		basis    any // float64 or nil
	}{
		{"green", [][3]int{{2, 3, 5}, {2, 3, 5}, {2, 3, 5}}, "green", "low", "fixed-price", float64(10)},
		{"yellow", [][3]int{{2, 5, 13}, {1, 2, 3}, {3, 8, 20}}, "yellow", "medium", "fixed-price-with-buffer", float64(20)},
		{"red", [][3]int{{1, 8, 40}, {1, 8, 40}, {1, 8, 40}}, "red", "high", "time-and-materials", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newClient(t, true)
			id := c.wbsWithEstimates(tc.tris...)

			_, body := c.get(id)
			if body["estimatesState"] != "unapproved" {
				t.Fatalf("estimatesState = %v, want unapproved (live)", body["estimatesState"])
			}
			ps := pricingOf(t, body)
			if ps == nil {
				t.Fatal("pricingStrategy = null, want a strategy")
			}
			if ps["flag"] != tc.flag || ps["riskLevel"] != tc.risk || ps["contract"] != tc.contract {
				t.Fatalf("pricingStrategy = %v, want %s/%s/%s", ps, tc.flag, tc.risk, tc.contract)
			}
			if ps["recommendedBasis"] != tc.basis {
				t.Fatalf("recommendedBasis = %v, want %v", ps["recommendedBasis"], tc.basis)
			}
		})
	}
}

func TestPricingStrategyNullWithoutEstimates(t *testing.T) {
	c := newClient(t, true)
	id := c.setupWBS()
	c.approve(id)

	_, body := c.get(id)
	if ps := pricingOf(t, body); ps != nil {
		t.Fatalf("pricingStrategy = %v, want null without estimates", ps)
	}
}
