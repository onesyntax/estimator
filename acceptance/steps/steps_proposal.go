package steps

import (
	"fmt"
	"strings"

	"estimation/acceptance/runtime"
	"estimation/internal/wbs"
)

// registerProposalActions registers the step that requests a client proposal.
func registerProposalActions(reg *runtime.Registry) {
	reg.Step(`^the Tech Lead requests a proposal with velocity (-?\d+) capacity (-?\d+) rate (-?\d+)$`,
		func(wd any, a []string) error {
			s := w(wd)
			velocity, capacity, rate, err := threeInts(a[0], a[1], a[2])
			if err != nil {
				return err
			}
			s.lastProposal, s.lastErr = s.svc.Proposal(s.wbsID, wbs.TeamInputs{Velocity: velocity, Capacity: capacity, Rate: rate})
			return nil
		})
}

// registerProposalAssertions registers the steps that assert on the generated
// client proposal.
func registerProposalAssertions(reg *runtime.Registry) {
	reg.Step(`^the proposal confidence is (\S+)$`,
		func(wd any, a []string) error {
			p, err := generatedProposal(w(wd))
			if err != nil {
				return err
			}
			if p.Confidence != a[0] {
				return fmt.Errorf("confidence = %q, want %q", p.Confidence, a[0])
			}
			return nil
		})

	reg.Step(`^the proposal contract is (\S+)$`,
		func(wd any, a []string) error {
			p, err := generatedProposal(w(wd))
			if err != nil {
				return err
			}
			if p.Contract != a[0] {
				return fmt.Errorf("contract = %q, want %q", p.Contract, a[0])
			}
			return nil
		})

	reg.Step(`^the proposal invites renegotiation$`,
		func(wd any, _ []string) error { return expectRenegotiation(w(wd), true) })
	reg.Step(`^the proposal does not invite renegotiation$`,
		func(wd any, _ []string) error { return expectRenegotiation(w(wd), false) })

	reg.Step(`^the proposal cost ranges from (\d+) to (\d+)$`,
		func(wd any, a []string) error {
			p, err := generatedProposal(w(wd))
			if err != nil {
				return err
			}
			return expectRange("cost", p.Cost, a[0], a[1])
		})

	reg.Step(`^the proposal timeline ranges from (\d+) to (\d+) weeks$`,
		func(wd any, a []string) error {
			p, err := generatedProposal(w(wd))
			if err != nil {
				return err
			}
			return expectRange("timeline", p.TimelineWeeks, a[0], a[1])
		})

	reg.Step(`^the proposal success probability is (\d+)% at the low figure and (\d+)% at the high figure$`,
		func(wd any, a []string) error {
			p, err := generatedProposal(w(wd))
			if err != nil {
				return err
			}
			atLow, atHigh, err := twoInts(a[0], a[1])
			if err != nil {
				return err
			}
			if p.SuccessProbability.AtLow != atLow || p.SuccessProbability.AtHigh != atHigh {
				return fmt.Errorf("probability = %d/%d, want %d/%d", p.SuccessProbability.AtLow, p.SuccessProbability.AtHigh, atLow, atHigh)
			}
			return nil
		})

	reg.Step(`^the proposal reasoning mentions (.+)$`,
		func(wd any, a []string) error {
			p, err := generatedProposal(w(wd))
			if err != nil {
				return err
			}
			if !strings.Contains(p.SuccessProbability.Reasoning, a[0]) {
				return fmt.Errorf("reasoning %q does not mention %q", p.SuccessProbability.Reasoning, a[0])
			}
			return nil
		})

	reg.Step(`^the proposal scope includes the task (.+)$`,
		func(wd any, a []string) error {
			p, err := generatedProposal(w(wd))
			if err != nil {
				return err
			}
			for _, item := range p.Scope {
				if item.Description == a[0] {
					return nil
				}
			}
			return fmt.Errorf("scope %+v does not include task %q", p.Scope, a[0])
		})

	reg.Step(`^the proposal assumptions include (.+)$`,
		func(wd any, a []string) error {
			p, err := generatedProposal(w(wd))
			if err != nil {
				return err
			}
			for _, assumption := range p.AssumptionsAndExclusions {
				if strings.Contains(assumption, a[0]) {
					return nil
				}
			}
			return fmt.Errorf("assumptions %v do not include %q", p.AssumptionsAndExclusions, a[0])
		})

	reg.Step(`^the proposal is rejected because estimates are not approved$`,
		expectErr(wbs.ErrEstimatesNotApprovedForProposal))
	reg.Step(`^the proposal is rejected because the inputs must be positive$`,
		expectErr(wbs.ErrNonPositiveTeamInputs))
}

// generatedProposal returns the last requested proposal, or an error when the
// request failed (so success assertions surface the failure).
func generatedProposal(s *world) (wbs.Proposal, error) {
	if s.lastErr != nil {
		return wbs.Proposal{}, fmt.Errorf("proposal was rejected: %w", s.lastErr)
	}
	return s.lastProposal, nil
}

func expectRenegotiation(s *world, want bool) error {
	p, err := generatedProposal(s)
	if err != nil {
		return err
	}
	if p.InvitesRenegotiation != want {
		return fmt.Errorf("invitesRenegotiation = %v, want %v", p.InvitesRenegotiation, want)
	}
	return nil
}

// expectRange checks a proposal range against its captured low and high bounds.
func expectRange(label string, got wbs.Range, lowStr, highStr string) error {
	low, high, err := twoInts(lowStr, highStr)
	if err != nil {
		return err
	}
	if got.Low != low || got.High != high {
		return fmt.Errorf("%s range = %d..%d, want %d..%d", label, got.Low, got.High, low, high)
	}
	return nil
}

// twoInts parses two integers from their string captures.
func twoInts(a, b string) (int, int, error) {
	x, err := atoi(a)
	if err != nil {
		return 0, 0, err
	}
	y, err := atoi(b)
	if err != nil {
		return 0, 0, err
	}
	return x, y, nil
}
