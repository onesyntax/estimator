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

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-09T12:56:52+05:30","module_hash":"0845964df485af987f2fe293992e1707d2be731f89a2f70391ded8857ce2757e","functions":[{"id":"func/Session.ProposalView","name":"Session.ProposalView","line":29,"end_line":49,"hash":"b6ca86bcc5970d5d2f968fc97d05077f6cea990b2d0d511932c100fa3cc02015"},{"id":"func/scopeDescriptions","name":"scopeDescriptions","line":51,"end_line":57,"hash":"f1ddff43cc8f6b73eab9c8abeac8033927d7839168c4862dd525abc6188ce944"}]}
// mutate4go-manifest-end
