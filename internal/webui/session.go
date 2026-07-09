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

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-09T12:56:50+05:30","module_hash":"9c8ee20ae2af5ce84a8c4dae4c9ffd491aa6aace1c72780b56267f05766bd2a8","functions":[{"id":"func/NewSession","name":"NewSession","line":24,"end_line":26,"hash":"d1695494c68e52f29c94fd30c67ae97fd409b20744c905b628f0c825e67515db"},{"id":"func/NewSessionWithService","name":"NewSessionWithService","line":30,"end_line":32,"hash":"0d4756d5bb62566579c2c00dd9cbbc30c4428fd04253e876ae9184618079c766"},{"id":"func/Session.PrimeWBS","name":"Session.PrimeWBS","line":35,"end_line":35,"hash":"ecd5019351107428751fe7386536bd5a9c52d541c99e26546f50e739a9186189"},{"id":"func/Session.GenerateWBS","name":"Session.GenerateWBS","line":40,"end_line":48,"hash":"ceb57085994cc25c253ce23d95d794aef89304208136a661d51dd6d5071e577c"},{"id":"func/Session.AddTask","name":"Session.AddTask","line":51,"end_line":53,"hash":"3a82a5ebaa850ed74c3d944508243df2195e9f2fef171d3d783631dea113607f"},{"id":"func/Session.EditTask","name":"Session.EditTask","line":56,"end_line":58,"hash":"4c57b02f44dd348b1f2e22e0eeb80bd1659fcddaa60edfb588dbdf64eaacc86d"},{"id":"func/Session.DeleteTask","name":"Session.DeleteTask","line":61,"end_line":63,"hash":"89713be268d985d8006e6477090b37409d06a6a3aa54e9b31b3c4732ffc76f4d"},{"id":"func/Session.DeleteAllTasks","name":"Session.DeleteAllTasks","line":66,"end_line":79,"hash":"6d4012f8f307d6a6e9de6ce73cf0d2ad4ba0a5565301523fd7e055bf4862d00a"},{"id":"func/Session.ApproveWBS","name":"Session.ApproveWBS","line":83,"end_line":83,"hash":"e4823210f73e57ef93b24df0c6fd3c1c3b534f4ad1e34ec31617b94ce9dc6d43"},{"id":"func/Session.PrimeRisks","name":"Session.PrimeRisks","line":94,"end_line":100,"hash":"1f8f953fe2e0f3ba497de7d6d3dc992b598a975d7040ccfbef345512a97e98e5"},{"id":"func/Session.FlagRisks","name":"Session.FlagRisks","line":103,"end_line":103,"hash":"005702f1890d075dbfc5d1b4c430277e6bcd48fb99413df41170ee4944bc2b30"},{"id":"func/Session.AddRiskNote","name":"Session.AddRiskNote","line":106,"end_line":108,"hash":"91523ba37095fee13caba17d469f948d8f8edf5be4b5e7fd330b6335e9df84e5"},{"id":"func/Session.PrimeEstimates","name":"Session.PrimeEstimates","line":122,"end_line":134,"hash":"7ae65890e7d0a1df66425d9e1e2ed7ff64262b12d356e89673ad7163823cea52"},{"id":"func/Session.GenerateEstimates","name":"Session.GenerateEstimates","line":138,"end_line":138,"hash":"065147f4397c716aacb7421c7730b41dfb04ddc2dfbae54e8155da32841b1128"},{"id":"func/Session.OverrideEstimate","name":"Session.OverrideEstimate","line":151,"end_line":159,"hash":"cd8a5bef7ec9767cba921e6f6fc08048ffe4bd65cd2d9910bf4ac0129e15c273"},{"id":"func/Session.ApproveEstimates","name":"Session.ApproveEstimates","line":162,"end_line":162,"hash":"a2004e72645c98a927203c1bebde257f8a507876870d94c612d9fa51d56af73e"},{"id":"func/Session.RequestProposal","name":"Session.RequestProposal","line":174,"end_line":188,"hash":"ec3806bb25d92db454f5baa3cc067f298774e783fa8997823070e0d2ea8eab2f"},{"id":"func/Session.apply","name":"Session.apply","line":192,"end_line":200,"hash":"a470968ec861510bd795e2c040b694ab477d943678437369f68be97767b6eaca"},{"id":"func/Session.current","name":"Session.current","line":202,"end_line":211,"hash":"3ed06ea2cd217be97afcebbd5133ed0caa692c42564ba412bba823dfd05e2f32"}]}
// mutate4go-manifest-end
