package webui

import "estimation/internal/wbs"

// ProposalView is the observable state of the Stage-2 Proposal screen: the
// single client-clean deliverable. It carries the confidence, contract, cost
// and timeline ranges, success probabilities, scope, and assumptions — and, by
// construction, none of the raw estimate mechanics (O/M/P, standard deviation,
// relative standard deviation, or the pricing-band flag).
type ProposalView struct {
	Error                string
	HasProposal          bool
	Confidence           string
	Contract             string
	InvitesRenegotiation bool
	CostLow              int
	CostHigh             int
	WeeksLow             int
	WeeksHigh            int
	SuccessLow           int
	SuccessHigh          int
	SuccessReasoning     string
	Scope                []string
	Assumptions          []string
}

// ProposalView computes the current Stage-2 screen state from the last proposal
// request.
func (s *Session) ProposalView() ProposalView {
	v := ProposalView{Error: s.proposalErr}
	if !s.hasProposal || s.proposal == nil {
		return v
	}
	p := s.proposal
	v.HasProposal = true
	v.Confidence = p.Confidence
	v.Contract = p.Contract
	v.InvitesRenegotiation = p.InvitesRenegotiation
	v.CostLow = p.Cost.Low
	v.CostHigh = p.Cost.High
	v.WeeksLow = p.TimelineWeeks.Low
	v.WeeksHigh = p.TimelineWeeks.High
	v.SuccessLow = p.SuccessProbability.AtLow
	v.SuccessHigh = p.SuccessProbability.AtHigh
	v.SuccessReasoning = p.SuccessProbability.Reasoning
	v.Scope = scopeDescriptions(p.Scope)
	v.Assumptions = p.AssumptionsAndExclusions
	return v
}

func scopeDescriptions(scope []wbs.ScopeItem) []string {
	out := make([]string, len(scope))
	for i, item := range scope {
		out[i] = item.Description
	}
	return out
}
