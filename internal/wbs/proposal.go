package wbs

import (
	"fmt"
	"math"
)

// TeamInputs are the three delivery-team parameters a proposal translates points
// into money and time with: Velocity in points per week, Capacity in hours per
// week, and Rate in dollars per hour. All three must be positive.
type TeamInputs struct {
	Velocity int
	Capacity int
	Rate     int
}

// Range is an inclusive low..high band of a whole-number quantity (dollars or
// weeks).
type Range struct {
	Low  int
	High int
}

// SuccessProbability is the whole-percent chance of completing within each end
// of the range on a normal curve, with a plain-language reasoning.
type SuccessProbability struct {
	AtLow     int
	AtHigh    int
	Reasoning string
}

// ScopeItem is a client-facing task line: its stable id and description, with no
// estimate mechanics attached.
type ScopeItem struct {
	ID          string
	Description string
}

// Proposal is the single client-facing deliverable: a confidence label, a cost
// and timeline range with a success probability at each end, the detailed scope,
// and the risk-note-derived assumptions. It deliberately carries none of the raw
// estimate mechanics (O/M/P, SD, RSD, or the risk-band flag).
type Proposal struct {
	Confidence               string
	Contract                 string
	InvitesRenegotiation     bool
	Cost                     Range
	TimelineWeeks            Range
	SuccessProbability       SuccessProbability
	Scope                    []ScopeItem
	AssumptionsAndExclusions []string
}

// Proposal builds the client proposal from the approved estimate set and the
// team inputs. It fails with ErrEstimatesNotApprovedForProposal until the
// estimates are approved, and ErrNonPositiveTeamInputs when any input is not
// positive. The cost/timeline range runs from the pricing basis (the mean for
// green, the mean + 1 SD for yellow and red) up to the mean + 2 SD; costs round
// half-up to whole dollars and weeks round up.
func (w *WBS) Proposal(inputs TeamInputs) (Proposal, error) {
	if !w.estimatesApproved {
		return Proposal{}, ErrEstimatesNotApprovedForProposal
	}
	if inputs.Velocity <= 0 || inputs.Capacity <= 0 || inputs.Rate <= 0 {
		return Proposal{}, ErrNonPositiveTeamInputs
	}

	v, _ := w.projectPert()
	b := bandForRSD(v.metrics().RelativeStandardDeviation)
	expected, sd := v.expected(), v.stdDev
	lowPoints := expected + float64(b.lowSDMultiplier)*sd
	highPoints := expected + 2*sd

	hoursPerPoint := float64(inputs.Capacity) / float64(inputs.Velocity)
	cost := Range{
		Low:  roundHalfUp(lowPoints * hoursPerPoint * float64(inputs.Rate)),
		High: roundHalfUp(highPoints * hoursPerPoint * float64(inputs.Rate)),
	}
	weeks := Range{
		Low:  weeksFor(lowPoints, inputs.Velocity),
		High: weeksFor(highPoints, inputs.Velocity),
	}
	atLow, atHigh := normalCDFPercent(b.lowSDMultiplier), normalCDFPercent(2)

	return Proposal{
		Confidence:           b.confidence,
		Contract:             b.contract,
		InvitesRenegotiation: b.invitesRenegotiation,
		Cost:                 cost,
		TimelineWeeks:        weeks,
		SuccessProbability: SuccessProbability{
			AtLow:     atLow,
			AtHigh:    atHigh,
			Reasoning: proposalReasoning(atLow, atHigh, len(w.tasks)),
		},
		Scope:                    w.scope(),
		AssumptionsAndExclusions: w.assumptions(),
	}, nil
}

// scope renders the task list as client-facing scope items.
func (w *WBS) scope() []ScopeItem {
	out := make([]ScopeItem, len(w.tasks))
	for i, t := range w.tasks {
		out[i] = ScopeItem{ID: t.ID, Description: t.Description}
	}
	return out
}

// assumptions aggregates every task's risk-note text, in task and note order,
// into the proposal's Assumptions & Exclusions. The slice is non-nil so it
// serializes as an empty list when there are no risk notes.
func (w *WBS) assumptions() []string {
	out := []string{}
	for _, t := range w.tasks {
		for _, n := range t.RiskNotes {
			out = append(out, n.Description)
		}
	}
	return out
}

// weeksFor converts a points figure to whole weeks, rounding up so the timeline
// is never quoted short.
func weeksFor(points float64, velocity int) int {
	return int(math.Ceil(points / float64(velocity)))
}

// normalCDFPercent is the whole-percent probability of finishing within mean +
// k·SD on a normal curve: Phi(k) = 0.5·(1 + erf(k/√2)), rounded half-up. k = 0
// gives 50, 1 gives 84, and 2 gives 98.
func normalCDFPercent(k int) int {
	phi := 0.5 * (1 + math.Erf(float64(k)/math.Sqrt2))
	return roundHalfUp(phi * 100)
}

// proposalReasoning renders the deterministic templated explanation woven from
// the two success percentages and the task count.
func proposalReasoning(atLow, atHigh, taskCount int) string {
	return fmt.Sprintf("On a normal distribution of the estimates, there is a %d%% chance of delivering within the lower figure and a %d%% chance within the upper figure, across the %d tasks in scope.", atLow, atHigh, taskCount)
}

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-08T23:58:57+05:30","module_hash":"80126e28d8fd7c633fefbe68a1e93333ef411279458bbe964d9f98dc0e6b5e67","functions":[{"id":"func/WBS.Proposal","name":"WBS.Proposal","line":60,"end_line":99,"hash":"83565d9290fbf26f8a2a20c4468a73e5b9fa6f6a8d4ae0c991aa1ab140a13973"},{"id":"func/WBS.scope","name":"WBS.scope","line":102,"end_line":108,"hash":"46ef17f35bc15558d09345f8a06ccef95517ac960ca31b55be2cc85c84c65d15"},{"id":"func/WBS.assumptions","name":"WBS.assumptions","line":113,"end_line":121,"hash":"346543edfc3cf34a2d2dc8268d09933bcf2a505a51d0627237eaa91093803b86"},{"id":"func/weeksFor","name":"weeksFor","line":125,"end_line":127,"hash":"f73c7e6cc8a849c4f1527b0664dd6b82f81cf4a2cb0dce69fff1ead447faeb83"},{"id":"func/normalCDFPercent","name":"normalCDFPercent","line":132,"end_line":135,"hash":"0828cc334e0b0b811ec0521c13e6719a471b8605496957e4e65091eb5c3e924d"},{"id":"func/proposalReasoning","name":"proposalReasoning","line":139,"end_line":141,"hash":"5c26fcd7ee129a14dc0f03502358ab49cb8f64469ca8ed3e72c5552faf8837c6"}]}
// mutate4go-manifest-end
