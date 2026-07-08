package httpapi

import (
	"net/http"
	"testing"
)

func (c *client) primeEstimates(estimates []map[string]any) {
	c.t.Helper()
	status, _ := c.json(http.MethodPost, "/qa/ai/next-estimates", map[string]any{"estimates": estimates})
	if status != http.StatusNoContent {
		c.t.Fatalf("prime estimates status = %d, want 204", status)
	}
}

func (c *client) generateEstimates(id string) (int, map[string]any) {
	c.t.Helper()
	return c.do(http.MethodPost, "/wbs/"+id+"/estimates/generate", "", nil)
}

func (c *client) approveEstimates(id string) (int, map[string]any) {
	c.t.Helper()
	return c.do(http.MethodPost, "/wbs/"+id+"/estimates/approve", "", nil)
}

func (c *client) overrideEstimate(id, taskID string, body map[string]any) (int, map[string]any) {
	c.t.Helper()
	return c.json(http.MethodPut, "/wbs/"+id+"/tasks/"+taskID+"/estimate", body)
}

// loginEstimatesPrime is the standard estimate prime: 2/5/13, 1/2/3, 3/8/20.
func loginEstimatesPrime() []map[string]any {
	return []map[string]any{
		{"taskNumber": 1, "optimistic": 2, "mostLikely": 5, "pessimistic": 13, "reasoning": "clean"},
		{"taskNumber": 2, "optimistic": 1, "mostLikely": 2, "pessimistic": 3, "reasoning": "trivial"},
		{"taskNumber": 3, "optimistic": 3, "mostLikely": 8, "pessimistic": 20, "reasoning": "legacy"},
	}
}

// estimatedWBS generates the standard approved WBS with estimates generated.
func (c *client) estimatedWBS() string {
	c.t.Helper()
	id := c.setupWBS()
	c.approve(id)
	c.primeEstimates(loginEstimatesPrime())
	if status, _ := c.generateEstimates(id); status != http.StatusOK {
		c.t.Fatalf("generate estimates status = %d, want 200", status)
	}
	return id
}

func estimateOf(t *testing.T, body map[string]any, taskNumber int) map[string]any {
	t.Helper()
	return taskFieldOf(t, body, taskNumber, "estimate")
}

func TestGenerateEstimatesOnApprovedWBS(t *testing.T) {
	c := newClient(t, true)
	id := c.setupWBS()
	c.approve(id)
	c.primeEstimates(loginEstimatesPrime())

	status, body := c.generateEstimates(id)
	if status != http.StatusOK {
		t.Fatalf("generate status = %d, want 200", status)
	}
	if body["estimatesState"] != "unapproved" {
		t.Fatalf("estimatesState = %v, want unapproved", body["estimatesState"])
	}
	_, got := c.get(id)
	e := estimateOf(t, got, 1)
	if e["optimistic"].(float64) != 2 || e["mostLikely"].(float64) != 5 || e["pessimistic"].(float64) != 13 || e["reasoning"] != "clean" {
		t.Fatalf("task 1 estimate = %v, want 2/5/13 clean", e)
	}
}

func TestGenerateEstimatesRejectsUnapprovedWBS(t *testing.T) {
	c := newClient(t, true)
	id := c.setupWBS()
	c.primeEstimates(loginEstimatesPrime())

	status, body := c.generateEstimates(id)
	if status != http.StatusConflict || body["error"] != "WBS must be approved before estimation" {
		t.Fatalf("generate unapproved = %d %v, want 409 WBS must be approved before estimation", status, body)
	}
	_, got := c.get(id)
	if e := estimateOf(t, got, 1); e != nil {
		t.Fatalf("task 1 estimate = %v, want null", e)
	}
}

func TestGenerateEstimatesUnknownWBS(t *testing.T) {
	c := newClient(t, true)
	status, body := c.generateEstimates("does-not-exist")
	if status != http.StatusNotFound || body["error"] != "WBS not found" {
		t.Fatalf("generate unknown = %d %v, want 404 WBS not found", status, body)
	}
}

func TestReGenerateEstimatesReplacesOverride(t *testing.T) {
	c := newClient(t, true)
	id := c.estimatedWBS()
	_, body := c.get(id)
	t1 := taskID(t, body, 1)
	c.overrideEstimate(id, t1, map[string]any{"optimistic": 3, "mostLikely": 8, "pessimistic": 20, "reasoning": "manual"})

	c.primeEstimates([]map[string]any{
		{"taskNumber": 1, "optimistic": 5, "mostLikely": 13, "pessimistic": 40, "reasoning": "rewrite"},
		{"taskNumber": 2, "optimistic": 3, "mostLikely": 5, "pessimistic": 8, "reasoning": "validation"},
		{"taskNumber": 3, "optimistic": 8, "mostLikely": 20, "pessimistic": 40, "reasoning": "migration"},
	})
	c.generateEstimates(id)

	_, got := c.get(id)
	e := estimateOf(t, got, 1)
	if e["mostLikely"].(float64) != 13 || e["reasoning"] != "rewrite" {
		t.Fatalf("task 1 estimate = %v, want regenerated 5/13/40 rewrite", e)
	}
}

func TestOverrideEstimateAccepted(t *testing.T) {
	c := newClient(t, true)
	id := c.estimatedWBS()
	c.approveEstimates(id)
	_, body := c.get(id)
	t1 := taskID(t, body, 1)

	status, resp := c.overrideEstimate(id, t1, map[string]any{"optimistic": 3, "mostLikely": 8, "pessimistic": 20, "reasoning": "team review"})
	if status != http.StatusOK {
		t.Fatalf("override status = %d, want 200", status)
	}
	if resp["estimatesState"] != "unapproved" {
		t.Fatalf("estimatesState = %v, want unapproved after override", resp["estimatesState"])
	}
	_, got := c.get(id)
	e := estimateOf(t, got, 1)
	if e["optimistic"].(float64) != 3 || e["mostLikely"].(float64) != 8 || e["pessimistic"].(float64) != 20 || e["reasoning"] != "team review" {
		t.Fatalf("task 1 estimate = %v, want 3/8/20 team review", e)
	}
}

func TestOverrideEstimateRejectsInvalid(t *testing.T) {
	cases := []struct {
		name    string
		body    map[string]any
		message string
	}{
		{"O>M", map[string]any{"optimistic": 8, "mostLikely": 5, "pessimistic": 13, "reasoning": "r"}, "estimate values must be strictly increasing"},
		{"M>P", map[string]any{"optimistic": 2, "mostLikely": 20, "pessimistic": 13, "reasoning": "r"}, "estimate values must be strictly increasing"},
		{"O=M", map[string]any{"optimistic": 5, "mostLikely": 5, "pessimistic": 8, "reasoning": "r"}, "estimate values must be strictly increasing"},
		{"M=P", map[string]any{"optimistic": 8, "mostLikely": 13, "pessimistic": 13, "reasoning": "r"}, "estimate values must be strictly increasing"},
		{"offscale-4", map[string]any{"optimistic": 4, "mostLikely": 5, "pessimistic": 13, "reasoning": "r"}, "estimate value is off the Fibonacci scale"},
		{"offscale-21", map[string]any{"optimistic": 21, "mostLikely": 40, "pessimistic": 100, "reasoning": "r"}, "estimate value is off the Fibonacci scale"},
		{"negative", map[string]any{"optimistic": -1, "mostLikely": 5, "pessimistic": 13, "reasoning": "r"}, "estimate value is off the Fibonacci scale"},
		{"empty-reasoning", map[string]any{"optimistic": 3, "mostLikely": 8, "pessimistic": 20, "reasoning": ""}, "reasoning is empty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newClient(t, true)
			id := c.estimatedWBS()
			_, body := c.get(id)
			t1 := taskID(t, body, 1)

			status, resp := c.overrideEstimate(id, t1, tc.body)
			if status != http.StatusUnprocessableEntity || resp["error"] != tc.message {
				t.Fatalf("override %s = %d %v, want 422 %q", tc.name, status, resp, tc.message)
			}
			_, got := c.get(id)
			e := estimateOf(t, got, 1)
			if e["optimistic"].(float64) != 2 || e["mostLikely"].(float64) != 5 || e["pessimistic"].(float64) != 13 {
				t.Fatalf("task 1 estimate changed after rejected override: %v", e)
			}
		})
	}
}

func TestOverrideEstimateUnknownTask(t *testing.T) {
	c := newClient(t, true)
	id := c.estimatedWBS()

	status, body := c.overrideEstimate(id, "nonexistent", map[string]any{"optimistic": 1, "mostLikely": 2, "pessimistic": 3, "reasoning": "x"})
	if status != http.StatusNotFound || body["error"] != "task not found" {
		t.Fatalf("override unknown task = %d %v, want 404 task not found", status, body)
	}
}

func TestOverrideEstimateBeforeGeneration(t *testing.T) {
	c := newClient(t, true)
	id := c.setupWBS()
	c.approve(id)
	_, body := c.get(id)
	t1 := taskID(t, body, 1)

	status, resp := c.overrideEstimate(id, t1, map[string]any{"optimistic": 1, "mostLikely": 2, "pessimistic": 3, "reasoning": "x"})
	if status != http.StatusConflict || resp["error"] != "task has no estimate to override" {
		t.Fatalf("override before generation = %d %v, want 409 task has no estimate to override", status, resp)
	}
	_, got := c.get(id)
	if e := estimateOf(t, got, 1); e != nil {
		t.Fatalf("task 1 estimate = %v, want null", e)
	}
}

func TestApproveEstimates(t *testing.T) {
	c := newClient(t, true)
	id := c.estimatedWBS()

	status, body := c.approveEstimates(id)
	if status != http.StatusOK || body["estimatesState"] != "approved" {
		t.Fatalf("approve = %d %v, want 200 approved", status, body)
	}
	_, got := c.get(id)
	if got["estimatesState"] != "approved" {
		t.Fatalf("GET estimatesState = %v, want approved", got["estimatesState"])
	}
}

func TestApproveEstimatesBeforeGeneration(t *testing.T) {
	c := newClient(t, true)
	id := c.setupWBS()
	c.approve(id)

	status, body := c.approveEstimates(id)
	if status != http.StatusConflict || body["error"] != "estimates have not been generated" {
		t.Fatalf("approve before generation = %d %v, want 409 estimates have not been generated", status, body)
	}
	_, got := c.get(id)
	if got["estimatesState"] != "unapproved" {
		t.Fatalf("GET estimatesState = %v, want unapproved", got["estimatesState"])
	}
}

func TestEstimatePrimingAffordanceDisabledWithoutMock(t *testing.T) {
	c := newClient(t, false)
	status, _ := c.json(http.MethodPost, "/qa/ai/next-estimates", map[string]any{"estimates": []any{}})
	if status != http.StatusNotFound {
		t.Fatalf("prime estimates without mock = %d, want 404", status)
	}
}
