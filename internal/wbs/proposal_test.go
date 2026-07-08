package wbs

import (
	"errors"
	"strings"
	"testing"
)

func approvedEstimatedSet(t *testing.T, tris ...[3]int) *WBS {
	t.Helper()
	w := estimatedSet(t, tris...)
	if err := w.ApproveEstimates(); err != nil {
		t.Fatalf("ApproveEstimates: %v", err)
	}
	return w
}

var standardInputs = TeamInputs{Velocity: 3, Capacity: 30, Rate: 120}

func TestProposalMediumRisk(t *testing.T) {
	w := approvedEstimatedSet(t, [3]int{2, 5, 13}, [3]int{1, 2, 3}, [3]int{3, 8, 20})

	p, err := w.Proposal(standardInputs)
	if err != nil {
		t.Fatalf("Proposal: %v", err)
	}
	if p.Confidence != "medium" || p.Contract != "fixed-price-with-buffer" || p.InvitesRenegotiation {
		t.Fatalf("proposal head = %s/%s/renegotiate=%v, want medium/fixed-price-with-buffer/false", p.Confidence, p.Contract, p.InvitesRenegotiation)
	}
	if p.Cost != (Range{Low: 24469, High: 28539}) {
		t.Fatalf("cost = %+v, want {24469 28539}", p.Cost)
	}
	if p.TimelineWeeks != (Range{Low: 7, High: 8}) {
		t.Fatalf("weeks = %+v, want {7 8}", p.TimelineWeeks)
	}
	if p.SuccessProbability.AtLow != 84 || p.SuccessProbability.AtHigh != 98 {
		t.Fatalf("probability = %d/%d, want 84/98", p.SuccessProbability.AtLow, p.SuccessProbability.AtHigh)
	}
	if !strings.Contains(p.SuccessProbability.Reasoning, "84%") || !strings.Contains(p.SuccessProbability.Reasoning, "3 tasks") {
		t.Fatalf("reasoning = %q, want it to mention 84%% and 3 tasks", p.SuccessProbability.Reasoning)
	}
}

func TestProposalLowRiskQuotesMeanAt50(t *testing.T) {
	w := approvedEstimatedSet(t, [3]int{2, 3, 5}, [3]int{2, 3, 5}, [3]int{2, 3, 5})

	p, err := w.Proposal(standardInputs)
	if err != nil {
		t.Fatalf("Proposal: %v", err)
	}
	if p.Confidence != "high" || p.Contract != "fixed-price" {
		t.Fatalf("proposal head = %s/%s, want high/fixed-price", p.Confidence, p.Contract)
	}
	if p.Cost != (Range{Low: 11400, High: 13478}) {
		t.Fatalf("cost = %+v, want {11400 13478}", p.Cost)
	}
	// Green's low bound is the mean, so 50%.
	if p.SuccessProbability.AtLow != 50 || p.SuccessProbability.AtHigh != 98 {
		t.Fatalf("probability = %d/%d, want 50/98", p.SuccessProbability.AtLow, p.SuccessProbability.AtHigh)
	}
	if !strings.Contains(p.SuccessProbability.Reasoning, "50%") {
		t.Fatalf("reasoning = %q, want it to mention 50%%", p.SuccessProbability.Reasoning)
	}
}

func TestProposalHighRiskRecommendsTimeAndMaterials(t *testing.T) {
	w := approvedEstimatedSet(t, [3]int{1, 8, 40}, [3]int{1, 8, 40}, [3]int{1, 8, 40})

	p, err := w.Proposal(standardInputs)
	if err != nil {
		t.Fatalf("Proposal: %v", err)
	}
	if p.Confidence != "lower" || p.Contract != "time-and-materials" || !p.InvitesRenegotiation {
		t.Fatalf("proposal head = %s/%s/renegotiate=%v, want lower/time-and-materials/true", p.Confidence, p.Contract, p.InvitesRenegotiation)
	}
	if p.Cost != (Range{Low: 57310, High: 70820}) {
		t.Fatalf("cost = %+v, want {57310 70820}", p.Cost)
	}
	if p.TimelineWeeks != (Range{Low: 16, High: 20}) {
		t.Fatalf("weeks = %+v, want {16 20}", p.TimelineWeeks)
	}
	if p.SuccessProbability.AtLow != 84 || p.SuccessProbability.AtHigh != 98 {
		t.Fatalf("probability = %d/%d, want 84/98", p.SuccessProbability.AtLow, p.SuccessProbability.AtHigh)
	}
}

func TestProposalScopeAndAssumptions(t *testing.T) {
	w := approvedWBS(t, "Login API", "Login UI", "Session store")
	w.ReplaceRiskNotes([]RiskAssignment{
		{TaskNumber: 1, Description: "SQL injection"},
		{TaskNumber: 2, Description: "XSS"},
	})
	w.SetEstimates([]EstimateAssignment{
		{TaskNumber: 1, Optimistic: 2, MostLikely: 5, Pessimistic: 13, Reasoning: "r"},
		{TaskNumber: 2, Optimistic: 1, MostLikely: 2, Pessimistic: 3, Reasoning: "r"},
		{TaskNumber: 3, Optimistic: 3, MostLikely: 8, Pessimistic: 20, Reasoning: "r"},
	})
	if err := w.ApproveEstimates(); err != nil {
		t.Fatalf("ApproveEstimates: %v", err)
	}

	p, err := w.Proposal(standardInputs)
	if err != nil {
		t.Fatalf("Proposal: %v", err)
	}
	if len(p.Scope) != 3 || p.Scope[0].Description != "Login API" {
		t.Fatalf("scope = %+v, want first task Login API of 3", p.Scope)
	}
	if got := strings.Join(p.AssumptionsAndExclusions, "|"); !strings.Contains(got, "SQL injection") || !strings.Contains(got, "XSS") {
		t.Fatalf("assumptions = %v, want SQL injection and XSS", p.AssumptionsAndExclusions)
	}
}

func TestProposalRejectedUntilEstimatesApproved(t *testing.T) {
	w := estimatedSet(t, [3]int{2, 5, 13}, [3]int{1, 2, 3}, [3]int{3, 8, 20}) // generated, not approved

	if _, err := w.Proposal(standardInputs); !errors.Is(err, ErrEstimatesNotApprovedForProposal) {
		t.Fatalf("Proposal on unapproved estimates err = %v, want ErrEstimatesNotApprovedForProposal", err)
	}
}

func TestProposalRejectsNonPositiveInputs(t *testing.T) {
	cases := []TeamInputs{
		{Velocity: 0, Capacity: 30, Rate: 120},
		{Velocity: 3, Capacity: 0, Rate: 120},
		{Velocity: 3, Capacity: 30, Rate: 0},
		{Velocity: -3, Capacity: 30, Rate: 120},
	}
	for _, in := range cases {
		w := approvedEstimatedSet(t, [3]int{2, 5, 13}, [3]int{1, 2, 3}, [3]int{3, 8, 20})
		if _, err := w.Proposal(in); !errors.Is(err, ErrNonPositiveTeamInputs) {
			t.Fatalf("Proposal(%+v) err = %v, want ErrNonPositiveTeamInputs", in, err)
		}
	}
}
