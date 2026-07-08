package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// registerProposal registers the Module 6 cases (qa/proposal_qa_suite.md,
// mirroring features/proposal.feature). Module 6 generates a client-friendly
// proposal from the approved estimates plus three team inputs, translating points
// into time and money deterministically (no AI). It is the single view shared with
// both the Tech Lead and the client, so it shows a confidence label, a cost range,
// a timeline range, a success probability at each end with plain-language
// reasoning, the detailed scope, and Assumptions & Exclusions — and never the raw
// O/M/P, standard deviation, RSD, or the risk-band flag. A proposal requires the
// estimates to be approved (Module 3) and positive team inputs. QA exercises only
// POST /wbs/{id}/proposal.

// proposalRange is the {low, high} pair used for both cost (whole dollars) and
// timelineWeeks (whole weeks).
type proposalRange struct {
	Low  int `json:"low"`
	High int `json:"high"`
}

// successProbabilityView is the pair of normal-curve percentages plus the
// deterministic templated reasoning.
type successProbabilityView struct {
	AtLow     int    `json:"atLow"`
	AtHigh    int    `json:"atHigh"`
	Reasoning string `json:"reasoning"`
}

type scopeItemView struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

// proposalView is the client-facing proposal as POST /wbs/{id}/proposal renders
// it. It deliberately carries none of the raw estimate mechanics.
type proposalView struct {
	Confidence               string                 `json:"confidence"`
	Contract                 string                 `json:"contract"`
	InvitesRenegotiation     bool                   `json:"invitesRenegotiation"`
	Cost                     proposalRange          `json:"cost"`
	TimelineWeeks            proposalRange          `json:"timelineWeeks"`
	SuccessProbability       successProbabilityView `json:"successProbability"`
	Scope                    []scopeItemView        `json:"scope"`
	AssumptionsAndExclusions []string               `json:"assumptionsAndExclusions"`
}

func (r response) proposal() proposalView {
	var p proposalView
	_ = json.Unmarshal(r.body, &p)
	return p
}

// standardInputs is the shared team-input body: 3 points/week, 30 hours/week,
// $120/hour, matching the suite's worked cost/timeline figures.
func standardInputs() map[string]any {
	return map[string]any{"teamVelocity": 3, "teamCapacity": 30, "hourlyRate": 120}
}

// approvedEstimatesWBS is the shared proposal setup: an approved WBS with the given
// estimates primed, generated, and approved. It returns the WBS id.
func approvedEstimatesWBS(t *T, estimates []estimateAssign) string {
	id := generatedEstimates(t, estimates)
	t.eq("approve estimates status", postJSON("/wbs/"+id+"/estimates/approve", nil).status, http.StatusOK)
	return id
}

// scopeHas reports whether the proposal scope contains a task with the given
// description.
func scopeHas(p proposalView, description string) bool {
	for _, s := range p.Scope {
		if s.Description == description {
			return true
		}
	}
	return false
}

// contains reports whether any string in the slice equals want.
func contains(items []string, want string) bool {
	for _, it := range items {
		if it == want {
			return true
		}
	}
	return false
}

func registerProposal() {
	// --- Client Proposal (proposal.feature) --------------------------------

	// QA-PROP-1 — Medium-risk (yellow) proposal: ranges, probabilities, reasoning.
	register("QA-PROP-1 medium-risk proposal ranges and probabilities", func(t *T) {
		id := approvedEstimatesWBS(t, standardEstimates())
		r := postJSON("/wbs/"+id+"/proposal", standardInputs())
		t.eq("proposal status", r.status, http.StatusOK)
		p := r.proposal()
		t.eq("confidence", p.Confidence, "medium")
		t.eq("contract", p.Contract, "fixed-price-with-buffer")
		t.eq("invitesRenegotiation", p.InvitesRenegotiation, false)
		t.eq("cost.low", p.Cost.Low, 24469)
		t.eq("cost.high", p.Cost.High, 28539)
		t.eq("timelineWeeks.low", p.TimelineWeeks.Low, 7)
		t.eq("timelineWeeks.high", p.TimelineWeeks.High, 8)
		t.eq("successProbability.atLow", p.SuccessProbability.AtLow, 84)
		t.eq("successProbability.atHigh", p.SuccessProbability.AtHigh, 98)
		if !bytes.Contains([]byte(p.SuccessProbability.Reasoning), []byte("84%")) {
			t.fail("reasoning %q does not contain %q", p.SuccessProbability.Reasoning, "84%")
		}
		if !bytes.Contains([]byte(p.SuccessProbability.Reasoning), []byte("3 tasks")) {
			t.fail("reasoning %q does not contain %q", p.SuccessProbability.Reasoning, "3 tasks")
		}
	})

	// QA-PROP-2 — Low-risk (green) proposal: 50% at the mean.
	register("QA-PROP-2 low-risk proposal 50% at the mean", func(t *T) {
		id := approvedEstimatesWBS(t, greenEstimates())
		r := postJSON("/wbs/"+id+"/proposal", standardInputs())
		t.eq("proposal status", r.status, http.StatusOK)
		p := r.proposal()
		t.eq("confidence", p.Confidence, "high")
		t.eq("contract", p.Contract, "fixed-price")
		t.eq("cost.low", p.Cost.Low, 11400)
		t.eq("cost.high", p.Cost.High, 13478)
		t.eq("successProbability.atLow", p.SuccessProbability.AtLow, 50)
		t.eq("successProbability.atHigh", p.SuccessProbability.AtHigh, 98)
		if !bytes.Contains([]byte(p.SuccessProbability.Reasoning), []byte("50%")) {
			t.fail("reasoning %q does not contain %q", p.SuccessProbability.Reasoning, "50%")
		}
	})

	// QA-PROP-3 — High-risk (red) proposal recommends T&M and invites renegotiation.
	register("QA-PROP-3 high-risk proposal recommends T&M", func(t *T) {
		id := approvedEstimatesWBS(t, redEstimates())
		r := postJSON("/wbs/"+id+"/proposal", standardInputs())
		t.eq("proposal status", r.status, http.StatusOK)
		p := r.proposal()
		t.eq("confidence", p.Confidence, "lower")
		t.eq("contract", p.Contract, "time-and-materials")
		t.eq("invitesRenegotiation", p.InvitesRenegotiation, true)
		t.eq("cost.low", p.Cost.Low, 57310)
		t.eq("cost.high", p.Cost.High, 70820)
		t.eq("timelineWeeks.low", p.TimelineWeeks.Low, 16)
		t.eq("timelineWeeks.high", p.TimelineWeeks.High, 20)
		t.eq("successProbability.atLow", p.SuccessProbability.AtLow, 84)
		t.eq("successProbability.atHigh", p.SuccessProbability.AtHigh, 98)
	})

	// QA-PROP-4 — Scope and Assumptions & Exclusions, without raw internals.
	register("QA-PROP-4 scope and assumptions without raw internals", func(t *T) {
		id := approvedWBS(t)
		primeRisks(t, []riskAssignment{
			{1, "SQL injection"},
			{2, "XSS"},
		})
		t.eq("flag risks status", postJSON("/wbs/"+id+"/risk-notes/flag", nil).status, http.StatusOK)
		primeEstimates(t, standardEstimates())
		t.eq("generate estimates status", postJSON("/wbs/"+id+"/estimates/generate", nil).status, http.StatusOK)
		t.eq("approve estimates status", postJSON("/wbs/"+id+"/estimates/approve", nil).status, http.StatusOK)

		r := postJSON("/wbs/"+id+"/proposal", standardInputs())
		t.eq("proposal status", r.status, http.StatusOK)
		p := r.proposal()
		if !scopeHas(p, "Login API") {
			t.fail("scope does not contain a task described %q", "Login API")
		}
		if !contains(p.AssumptionsAndExclusions, "SQL injection") {
			t.fail("assumptionsAndExclusions %v does not contain %q", p.AssumptionsAndExclusions, "SQL injection")
		}
		if !contains(p.AssumptionsAndExclusions, "XSS") {
			t.fail("assumptionsAndExclusions %v does not contain %q", p.AssumptionsAndExclusions, "XSS")
		}
		// The proposal must expose none of the raw estimate mechanics; assert the
		// forbidden field names appear nowhere in the response body.
		for _, forbidden := range []string{
			"optimistic", "mostLikely", "pessimistic",
			"standardDeviation", "relativeStandardDeviation", "flag",
		} {
			if bytes.Contains(r.body, []byte(forbidden)) {
				t.fail("proposal body leaks raw internal field %q", forbidden)
			}
		}
	})

	// QA-PROP-5 — Reject a proposal before estimates are approved.
	register("QA-PROP-5 reject proposal before estimates approved", func(t *T) {
		id := generatedEstimates(t, standardEstimates()) // generated, not approved
		t.estimatesStateIs(id, "unapproved")
		r := postJSON("/wbs/"+id+"/proposal", standardInputs())
		t.errorCase(r, http.StatusConflict, "estimates must be approved before a proposal")
	})

	// QA-PROP-6 — Reject non-positive team inputs.
	register("QA-PROP-6 reject non-positive team inputs", func(t *T) {
		id := approvedEstimatesWBS(t, standardEstimates())
		for i, inputs := range []map[string]any{
			{"teamVelocity": 0, "teamCapacity": 30, "hourlyRate": 120},
			{"teamVelocity": 3, "teamCapacity": 0, "hourlyRate": 120},
			{"teamVelocity": 3, "teamCapacity": 30, "hourlyRate": 0},
			{"teamVelocity": -3, "teamCapacity": 30, "hourlyRate": 120},
		} {
			r := postJSON("/wbs/"+id+"/proposal", inputs)
			t.eq(fmt.Sprintf("reject input set %d status", i+1), r.status, http.StatusUnprocessableEntity)
			if msg := r.errorMessage(); msg != "team inputs must be positive" {
				t.fail("input set %d error = %q, want %q", i+1, msg, "team inputs must be positive")
			}
		}
	})
}
