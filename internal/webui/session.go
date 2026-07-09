// Package webui is the presentation adapter for the two-stage estimation UI. It
// wraps the in-process WBS service in a single-session controller that turns UI
// actions into service calls, maps domain errors to friendly inline messages,
// and exposes the observable screen state as view models the HTML templates and
// acceptance steps both render. It holds one active estimate per session,
// matching the in-memory backend; it adds no estimation logic of its own.
package webui

import "estimation/internal/wbs"

// Session is one browser session's UI state: the backing service, the single
// active WBS, the last inline error to surface, and the last generated
// proposal. A zero Session is not usable; construct one with NewSession.
type Session struct {
	svc         *wbs.Service
	wbsID       string
	err         string
	proposal    *wbs.Proposal
	proposalErr string
	hasProposal bool
}

// NewSession builds a session backed by a fresh deterministic service.
func NewSession() *Session {
	return NewSessionWithService(wbs.NewService())
}

// NewSessionWithService builds a session backed by the given service, letting a
// caller inject a real AI provider.
func NewSessionWithService(svc *wbs.Service) *Session {
	return &Session{svc: svc}
}

// PrimeWBS seeds the tasks the next Generate returns (QA priming affordance).
func (s *Session) PrimeWBS(tasks []string) { s.svc.Prime(tasks) }

// GenerateWBS generates a WBS from the requirement document and makes it the
// session's active WBS. A domain error is surfaced as an inline message and no
// WBS becomes active.
func (s *Session) GenerateWBS(doc wbs.RequirementDocument) {
	s.err = ""
	id, err := s.svc.Generate(doc)
	if err != nil {
		s.err = message(err)
		return
	}
	s.wbsID = id
}

// AddTask appends a task with the given description to the active WBS.
func (s *Session) AddTask(description string) {
	s.apply(func(id string) error { return s.svc.AddTask(id, description) })
}

// EditTask replaces task number's description on the active WBS.
func (s *Session) EditTask(number int, description string) {
	s.apply(func(id string) error { return s.svc.EditTask(id, number, description) })
}

// DeleteTask removes task number from the active WBS.
func (s *Session) DeleteTask(number int) {
	s.apply(func(id string) error { return s.svc.DeleteTask(id, number) })
}

// DeleteAllTasks removes every task from the active WBS, emptying it.
func (s *Session) DeleteAllTasks() {
	s.err = ""
	cur, ok := s.current()
	if !ok {
		return
	}
	for cur.TaskCount() > 0 {
		if err := s.svc.DeleteTask(s.wbsID, 1); err != nil {
			s.err = message(err)
			return
		}
		cur, _ = s.current()
	}
}

// ApproveWBS approves the active WBS, opening the risk and estimate gates. A
// domain error (an empty WBS) is surfaced as an inline message.
func (s *Session) ApproveWBS() { s.apply(s.svc.Approve) }

// RiskSeed is one task-number → risk description pair QA primes the next Flag
// risks with.
type RiskSeed struct {
	TaskNumber  int
	Description string
}

// PrimeRisks seeds the risk notes the next Flag risks returns (QA priming
// affordance).
func (s *Session) PrimeRisks(seeds []RiskSeed) {
	assignments := make([]wbs.RiskAssignment, len(seeds))
	for i, seed := range seeds {
		assignments[i] = wbs.RiskAssignment{TaskNumber: seed.TaskNumber, Description: seed.Description}
	}
	s.svc.PrimeRisks(assignments)
}

// FlagRisks invokes the AI to attach risk notes to the active WBS's tasks.
func (s *Session) FlagRisks() { s.apply(s.svc.FlagRisks) }

// AddRiskNote adds a manual risk note to task number on the active WBS.
func (s *Session) AddRiskNote(number int, description string) {
	s.apply(func(id string) error { return s.svc.AddRiskNote(id, number, description) })
}

// EstimateSeed is one task-number → 3-point estimate QA primes the next
// Generate estimates with.
type EstimateSeed struct {
	TaskNumber  int
	Optimistic  int
	MostLikely  int
	Pessimistic int
	Reasoning   string
}

// PrimeEstimates seeds the 3-point estimates the next Generate estimates returns
// (QA priming affordance).
func (s *Session) PrimeEstimates(seeds []EstimateSeed) {
	assignments := make([]wbs.EstimateAssignment, len(seeds))
	for i, seed := range seeds {
		assignments[i] = wbs.EstimateAssignment{
			TaskNumber:  seed.TaskNumber,
			Optimistic:  seed.Optimistic,
			MostLikely:  seed.MostLikely,
			Pessimistic: seed.Pessimistic,
			Reasoning:   seed.Reasoning,
		}
	}
	s.svc.PrimeEstimates(assignments)
}

// GenerateEstimates invokes the AI to produce 3-point estimates for the active
// WBS's tasks.
func (s *Session) GenerateEstimates() { s.apply(s.svc.GenerateEstimates) }

// EstimateInput is a manual estimate override entered on the estimates section.
type EstimateInput struct {
	Optimistic  int
	MostLikely  int
	Pessimistic int
	Reasoning   string
}

// OverrideEstimate replaces task number's estimate on the active WBS. A domain
// validation error is surfaced as an inline message and the estimate is left
// unchanged.
func (s *Session) OverrideEstimate(number int, in EstimateInput) {
	estimate := wbs.Estimate{
		Optimistic:  in.Optimistic,
		MostLikely:  in.MostLikely,
		Pessimistic: in.Pessimistic,
		Reasoning:   in.Reasoning,
	}
	s.apply(func(id string) error { return s.svc.OverrideEstimate(id, number, estimate) })
}

// ApproveEstimates approves the active WBS's estimates, the gate into Stage 2.
func (s *Session) ApproveEstimates() { s.apply(s.svc.ApproveEstimates) }

// TeamInput carries the Stage-2 team inputs entered on the Proposal screen.
type TeamInput struct {
	Velocity int
	Capacity int
	Rate     int
}

// RequestProposal builds the client proposal from the team inputs and makes it
// the session's active proposal. A domain error (non-positive inputs) is
// surfaced as an inline message and no proposal is shown.
func (s *Session) RequestProposal(in TeamInput) {
	s.proposalErr = ""
	s.hasProposal = false
	s.proposal = nil
	if s.wbsID == "" {
		return
	}
	proposal, err := s.svc.Proposal(s.wbsID, wbs.TeamInputs{Velocity: in.Velocity, Capacity: in.Capacity, Rate: in.Rate})
	if err != nil {
		s.proposalErr = message(err)
		return
	}
	s.proposal = &proposal
	s.hasProposal = true
}

// apply runs an id-only service mutation on the active WBS, recording any
// domain error as the inline message.
func (s *Session) apply(action func(id string) error) {
	s.err = ""
	if s.wbsID == "" {
		return
	}
	if err := action(s.wbsID); err != nil {
		s.err = message(err)
	}
}

func (s *Session) current() (*wbs.WBS, bool) {
	if s.wbsID == "" {
		return nil, false
	}
	cur, err := s.svc.Get(s.wbsID)
	if err != nil {
		return nil, false
	}
	return cur, true
}
