package httpapi

import (
	"net/http"
	"strings"
	"testing"
)

func (c *client) proposal(id string, body map[string]any) (int, map[string]any) {
	c.t.Helper()
	return c.json(http.MethodPost, "/wbs/"+id+"/proposal", body)
}

// approvedEstimates sets up an approved WBS, generates the given estimate set,
// and approves the estimates.
func (c *client) approvedEstimates(tris ...[3]int) string {
	c.t.Helper()
	id := c.wbsWithEstimates(tris...)
	if status, _ := c.approveEstimates(id); status != http.StatusOK {
		c.t.Fatalf("approve estimates status = %d, want 200", status)
	}
	return id
}

var teamInputs = map[string]any{"teamVelocity": 3, "teamCapacity": 30, "hourlyRate": 120}

func rangeOf(t *testing.T, obj map[string]any, key string) (float64, float64) {
	t.Helper()
	r, ok := obj[key].(map[string]any)
	if !ok {
		t.Fatalf("%s is not an object: %v", key, obj[key])
	}
	return r["low"].(float64), r["high"].(float64)
}

func TestProposalMediumRisk(t *testing.T) {
	c := newClient(t, true)
	id := c.approvedEstimates([3]int{2, 5, 13}, [3]int{1, 2, 3}, [3]int{3, 8, 20})

	status, p := c.proposal(id, teamInputs)
	if status != http.StatusOK {
		t.Fatalf("proposal status = %d, want 200 (%v)", status, p)
	}
	if p["confidence"] != "medium" || p["contract"] != "fixed-price-with-buffer" || p["invitesRenegotiation"] != false {
		t.Fatalf("head = %v/%v/%v, want medium/fixed-price-with-buffer/false", p["confidence"], p["contract"], p["invitesRenegotiation"])
	}
	if lo, hi := rangeOf(t, p, "cost"); lo != 24469 || hi != 28539 {
		t.Fatalf("cost = %v..%v, want 24469..28539", lo, hi)
	}
	if lo, hi := rangeOf(t, p, "timelineWeeks"); lo != 7 || hi != 8 {
		t.Fatalf("weeks = %v..%v, want 7..8", lo, hi)
	}
	sp := p["successProbability"].(map[string]any)
	if sp["atLow"] != float64(84) || sp["atHigh"] != float64(98) {
		t.Fatalf("probability = %v/%v, want 84/98", sp["atLow"], sp["atHigh"])
	}
	reasoning, _ := sp["reasoning"].(string)
	if !strings.Contains(reasoning, "84%") || !strings.Contains(reasoning, "3 tasks") {
		t.Fatalf("reasoning = %q, want it to mention 84%% and 3 tasks", reasoning)
	}
}

func TestProposalLowRiskMeanAt50(t *testing.T) {
	c := newClient(t, true)
	id := c.approvedEstimates([3]int{2, 3, 5}, [3]int{2, 3, 5}, [3]int{2, 3, 5})

	status, p := c.proposal(id, teamInputs)
	if status != http.StatusOK {
		t.Fatalf("proposal status = %d, want 200 (%v)", status, p)
	}
	if p["confidence"] != "high" || p["contract"] != "fixed-price" {
		t.Fatalf("head = %v/%v, want high/fixed-price", p["confidence"], p["contract"])
	}
	if lo, hi := rangeOf(t, p, "cost"); lo != 11400 || hi != 13478 {
		t.Fatalf("cost = %v..%v, want 11400..13478", lo, hi)
	}
	sp := p["successProbability"].(map[string]any)
	if sp["atLow"] != float64(50) || sp["atHigh"] != float64(98) {
		t.Fatalf("probability = %v/%v, want 50/98", sp["atLow"], sp["atHigh"])
	}
}

func TestProposalHighRiskTimeAndMaterials(t *testing.T) {
	c := newClient(t, true)
	id := c.approvedEstimates([3]int{1, 8, 40}, [3]int{1, 8, 40}, [3]int{1, 8, 40})

	status, p := c.proposal(id, teamInputs)
	if status != http.StatusOK {
		t.Fatalf("proposal status = %d, want 200 (%v)", status, p)
	}
	if p["confidence"] != "lower" || p["contract"] != "time-and-materials" || p["invitesRenegotiation"] != true {
		t.Fatalf("head = %v/%v/%v, want lower/time-and-materials/true", p["confidence"], p["contract"], p["invitesRenegotiation"])
	}
	if lo, hi := rangeOf(t, p, "cost"); lo != 57310 || hi != 70820 {
		t.Fatalf("cost = %v..%v, want 57310..70820", lo, hi)
	}
	if lo, hi := rangeOf(t, p, "timelineWeeks"); lo != 16 || hi != 20 {
		t.Fatalf("weeks = %v..%v, want 16..20", lo, hi)
	}
}

func TestProposalScopeAndAssumptionsHideInternals(t *testing.T) {
	c := newClient(t, true)
	id := c.setupWBS()
	c.approve(id)
	c.primeRisks([]map[string]any{
		{"taskNumber": 1, "description": "SQL injection"},
		{"taskNumber": 2, "description": "XSS"},
	})
	c.flag(id)
	c.primeEstimates(estimateTriples([3]int{2, 5, 13}, [3]int{1, 2, 3}, [3]int{3, 8, 20}))
	c.generateEstimates(id)
	c.approveEstimates(id)

	status, p := c.proposal(id, teamInputs)
	if status != http.StatusOK {
		t.Fatalf("proposal status = %d, want 200 (%v)", status, p)
	}
	scope, ok := p["scope"].([]any)
	if !ok || len(scope) != 3 {
		t.Fatalf("scope = %v, want 3 items", p["scope"])
	}
	if scope[0].(map[string]any)["description"] != "Login API" {
		t.Fatalf("scope[0] = %v, want Login API", scope[0])
	}
	assumptions := toStrings(t, p["assumptionsAndExclusions"])
	joined := strings.Join(assumptions, "|")
	if !strings.Contains(joined, "SQL injection") || !strings.Contains(joined, "XSS") {
		t.Fatalf("assumptions = %v, want SQL injection and XSS", assumptions)
	}
	// The client proposal must not leak any raw estimate mechanics.
	for _, forbidden := range []string{"optimistic", "mostLikely", "pessimistic", "standardDeviation", "relativeStandardDeviation", "flag"} {
		if _, present := p[forbidden]; present {
			t.Fatalf("proposal leaked %q: %v", forbidden, p)
		}
	}
}

func toStrings(t *testing.T, v any) []string {
	t.Helper()
	raw, ok := v.([]any)
	if !ok {
		t.Fatalf("not an array: %v", v)
	}
	out := make([]string, len(raw))
	for i, r := range raw {
		out[i] = r.(string)
	}
	return out
}

func TestProposalRejectedBeforeEstimatesApproved(t *testing.T) {
	c := newClient(t, true)
	id := c.wbsWithEstimates([3]int{2, 5, 13}, [3]int{1, 2, 3}, [3]int{3, 8, 20}) // not approved

	status, body := c.proposal(id, teamInputs)
	if status != http.StatusConflict || body["error"] != "estimates must be approved before a proposal" {
		t.Fatalf("proposal unapproved = %d %v, want 409 estimates must be approved before a proposal", status, body)
	}
}

func TestProposalRejectsNonPositiveInputs(t *testing.T) {
	cases := []map[string]any{
		{"teamVelocity": 0, "teamCapacity": 30, "hourlyRate": 120},
		{"teamVelocity": 3, "teamCapacity": 0, "hourlyRate": 120},
		{"teamVelocity": 3, "teamCapacity": 30, "hourlyRate": 0},
		{"teamVelocity": -3, "teamCapacity": 30, "hourlyRate": 120},
	}
	for _, in := range cases {
		c := newClient(t, true)
		id := c.approvedEstimates([3]int{2, 5, 13}, [3]int{1, 2, 3}, [3]int{3, 8, 20})
		status, body := c.proposal(id, in)
		if status != http.StatusUnprocessableEntity || body["error"] != "team inputs must be positive" {
			t.Fatalf("proposal %v = %d %v, want 422 team inputs must be positive", in, status, body)
		}
	}
}

func TestProposalUnknownWBS(t *testing.T) {
	c := newClient(t, true)
	status, body := c.proposal("does-not-exist", teamInputs)
	if status != http.StatusNotFound || body["error"] != "WBS not found" {
		t.Fatalf("proposal unknown = %d %v, want 404 WBS not found", status, body)
	}
}
