package steps

import (
	"fmt"
	"strconv"
	"strings"

	"estimation/acceptance/runtime"
	"estimation/internal/wbs"
	"estimation/internal/webui"
)

// registerUI registers the two-stage estimation UI step handlers. They drive the
// webui.Session controller — the same view models the HTML screens render — so
// the assertions read the on-screen state, not the domain API. The session
// shares its backing service with the world's svc, so the existing "primed to …"
// steps seed the AI the UI session consumes.
func registerUI(reg *runtime.Registry) {
	registerUISetup(reg)
	registerUIBuildActions(reg)
	registerUIBuildAssertions(reg)
	registerUIProposal(reg)
}

func uiSession(wd any) *webui.Session { return w(wd).ui }

// registerUISetup registers the UI background and Given precondition steps.
func registerUISetup(reg *runtime.Registry) {
	reg.Step(`^the estimation app is running with a deterministic AI provider$`,
		func(wd any, _ []string) error {
			s := w(wd)
			svc := wbs.NewService()
			s.svc = svc
			s.ui = webui.NewSessionWithService(svc)
			return nil
		})

	reg.Step(`^a Tech Lead has generated a WBS on the build screen$`,
		func(wd any, _ []string) error {
			uiSession(wd).GenerateWBS(wbs.NewTextDocument("a valid requirement"))
			return nil
		})

	reg.Step(`^a Tech Lead has generated and approved a WBS on the build screen$`,
		func(wd any, _ []string) error {
			s := uiSession(wd)
			s.GenerateWBS(wbs.NewTextDocument("a valid requirement"))
			s.ApproveWBS()
			return nil
		})

	reg.Step(`^the Tech Lead has flagged risks on the build screen$`,
		func(wd any, _ []string) error {
			uiSession(wd).FlagRisks()
			return nil
		})

	reg.Step(`^the Tech Lead has generated estimates on the build screen$`,
		func(wd any, _ []string) error {
			uiSession(wd).GenerateEstimates()
			return nil
		})

	reg.Step(`^the Tech Lead has generated and approved estimates on the build screen$`,
		func(wd any, _ []string) error {
			s := uiSession(wd)
			s.GenerateEstimates()
			s.ApproveEstimates()
			return nil
		})
}

// registerUIBuildActions registers the Stage-1 Build action steps.
func registerUIBuildActions(reg *runtime.Registry) {
	reg.Step(`^a Tech Lead (?:enters|uploads) an? (valid|empty|corrupt) (text|PDF) requirement and generates on the build screen$`,
		func(wd any, a []string) error {
			uiSession(wd).GenerateWBS(requirementDoc(a[0], a[1]))
			return nil
		})

	reg.Step(`^the Tech Lead adds a task with the description (.+) on the build screen$`,
		func(wd any, a []string) error {
			uiSession(wd).AddTask(a[0])
			return nil
		})

	reg.Step(`^the Tech Lead edits task number (\d+) to the description (.+) on the build screen$`,
		func(wd any, a []string) error {
			n, err := taskNumber(a[0])
			if err != nil {
				return err
			}
			uiSession(wd).EditTask(n, a[1])
			return nil
		})

	reg.Step(`^the Tech Lead deletes task number (\d+) on the build screen$`,
		func(wd any, a []string) error {
			n, err := taskNumber(a[0])
			if err != nil {
				return err
			}
			uiSession(wd).DeleteTask(n)
			return nil
		})

	reg.Step(`^the Tech Lead deletes every task on the build screen$`,
		func(wd any, _ []string) error {
			uiSession(wd).DeleteAllTasks()
			return nil
		})

	reg.Step(`^the Tech Lead approves the WBS on the build screen$`,
		func(wd any, _ []string) error {
			uiSession(wd).ApproveWBS()
			return nil
		})

	reg.Step(`^the Tech Lead flags risks on the build screen$`,
		func(wd any, _ []string) error {
			uiSession(wd).FlagRisks()
			return nil
		})

	reg.Step(`^the Tech Lead adds a risk note to task number (\d+) with the description (.+) on the build screen$`,
		func(wd any, a []string) error {
			n, err := taskNumber(a[0])
			if err != nil {
				return err
			}
			uiSession(wd).AddRiskNote(n, a[1])
			return nil
		})

	reg.Step(`^the Tech Lead generates estimates on the build screen$`,
		func(wd any, _ []string) error {
			uiSession(wd).GenerateEstimates()
			return nil
		})

	reg.Step(`^the Tech Lead overrides task number (\d+) with optimistic (-?\d+) most likely (-?\d+) pessimistic (-?\d+) and reasoning (.+) on the build screen$`,
		func(wd any, a []string) error {
			n, err := taskNumber(a[0])
			if err != nil {
				return err
			}
			o, m, p, err := threeInts(a[1], a[2], a[3])
			if err != nil {
				return err
			}
			uiSession(wd).OverrideEstimate(n, webui.EstimateInput{Optimistic: o, MostLikely: m, Pessimistic: p, Reasoning: a[4]})
			return nil
		})

	reg.Step(`^the Tech Lead approves the estimates on the build screen$`,
		func(wd any, _ []string) error {
			uiSession(wd).ApproveEstimates()
			return nil
		})
}

// requirementDoc builds a requirement document of the given validity and format
// for a Stage-1 generate action.
func requirementDoc(validity, format string) wbs.RequirementDocument {
	pdf := strings.EqualFold(format, "pdf")
	switch validity {
	case "corrupt":
		return wbs.NewPDFDocument([]byte("not a valid pdf"))
	case "empty":
		if pdf {
			return wbs.NewPDFDocument(wbs.EncodeMinimalPDF(""))
		}
		return wbs.NewTextDocument("")
	default: // valid
		if pdf {
			return wbs.NewPDFDocument(wbs.EncodeMinimalPDF("a valid requirement"))
		}
		return wbs.NewTextDocument("a valid requirement")
	}
}

// registerUIBuildAssertions registers the Stage-1 Build on-screen assertions.
func registerUIBuildAssertions(reg *runtime.Registry) {
	reg.Step(`^the build screen shows a WBS with (\d+) tasks$`,
		func(wd any, a []string) error {
			return expectCount("task count", len(uiSession(wd).BuildView().Tasks), a[0])
		})

	reg.Step(`^the build screen shows task number (\d+) as (.+)$`,
		func(wd any, a []string) error {
			row, err := buildTask(wd, a[0])
			if err != nil {
				return err
			}
			if row.Description != a[1] {
				return fmt.Errorf("task %s = %q, want %q", a[0], row.Description, a[1])
			}
			return nil
		})

	reg.Step(`^the build screen shows the error (.+)$`,
		func(wd any, a []string) error {
			if got := uiSession(wd).BuildView().Error; got != a[0] {
				return fmt.Errorf("build error = %q, want %q", got, a[0])
			}
			return nil
		})

	reg.Step(`^the build screen shows no WBS$`,
		func(wd any, _ []string) error {
			if uiSession(wd).BuildView().HasWBS {
				return fmt.Errorf("expected no WBS on the build screen")
			}
			return nil
		})

	reg.Step(`^the risks section is (locked|available)$`,
		func(wd any, a []string) error {
			return expectLock("risks section", uiSession(wd).BuildView().RisksLocked, a[0])
		})

	reg.Step(`^the estimates section is (locked|available)$`,
		func(wd any, a []string) error {
			return expectLock("estimates section", uiSession(wd).BuildView().EstimatesLocked, a[0])
		})

	reg.Step(`^task number (\d+) shows the risk note (.+)$`,
		func(wd any, a []string) error {
			row, err := buildTask(wd, a[0])
			if err != nil {
				return err
			}
			for _, note := range row.RiskNotes {
				if note == a[1] {
					return nil
				}
			}
			return fmt.Errorf("task %s risk notes %v do not include %q", a[0], row.RiskNotes, a[1])
		})

	reg.Step(`^the build screen shows task number (\d+) with estimate optimistic (\d+) most likely (\d+) pessimistic (\d+)$`,
		func(wd any, a []string) error {
			row, err := buildTask(wd, a[0])
			if err != nil {
				return err
			}
			if row.Estimate == nil {
				return fmt.Errorf("task %s has no estimate on screen", a[0])
			}
			o, m, p, err := threeInts(a[1], a[2], a[3])
			if err != nil {
				return err
			}
			if row.Estimate.Optimistic != o || row.Estimate.MostLikely != m || row.Estimate.Pessimistic != p {
				return fmt.Errorf("task %s estimate = %d/%d/%d, want %d/%d/%d", a[0],
					row.Estimate.Optimistic, row.Estimate.MostLikely, row.Estimate.Pessimistic, o, m, p)
			}
			return nil
		})

	reg.Step(`^the metrics panel shows project expected (\d+) standard deviation (\d+) relative standard deviation (\d+)$`,
		func(wd any, a []string) error {
			metrics := uiSession(wd).BuildView().Metrics
			if metrics == nil {
				return fmt.Errorf("no metrics panel on screen")
			}
			e, sd, rsd, err := threeInts(a[0], a[1], a[2])
			if err != nil {
				return err
			}
			if metrics.Expected != e || metrics.StandardDeviation != sd || metrics.RelativeStandardDeviation != rsd {
				return fmt.Errorf("metrics = E%d SD%d RSD%d, want E%d SD%d RSD%d",
					metrics.Expected, metrics.StandardDeviation, metrics.RelativeStandardDeviation, e, sd, rsd)
			}
			return nil
		})

	reg.Step(`^the pricing panel shows flag (\S+) contract (\S+) basis (\S+)$`,
		func(wd any, a []string) error {
			pricing := uiSession(wd).BuildView().Pricing
			if pricing == nil {
				return fmt.Errorf("no pricing panel on screen")
			}
			if pricing.Flag != a[0] || pricing.Contract != a[1] || pricing.Basis != a[2] {
				return fmt.Errorf("pricing = flag %q contract %q basis %q, want flag %q contract %q basis %q",
					pricing.Flag, pricing.Contract, pricing.Basis, a[0], a[1], a[2])
			}
			return nil
		})

	reg.Step(`^the proposal stage is (not reachable|reachable)$`,
		func(wd any, a []string) error {
			reachable := uiSession(wd).BuildView().ProposalReachable
			want := a[0] == "reachable"
			if reachable != want {
				return fmt.Errorf("proposal reachable = %v, want %v", reachable, want)
			}
			return nil
		})
}

// buildTask returns the on-screen Stage-1 task row for the one-based number
// captured from a step.
func buildTask(wd any, numStr string) (webui.TaskRow, error) {
	n, err := taskNumber(numStr)
	if err != nil {
		return webui.TaskRow{}, err
	}
	rows := uiSession(wd).BuildView().Tasks
	if n < 1 || n > len(rows) {
		return webui.TaskRow{}, fmt.Errorf("task number %d is not on screen (have %d)", n, len(rows))
	}
	return rows[n-1], nil
}

// expectLock asserts a section's locked flag against the "locked"/"available"
// word captured from a step.
func expectLock(label string, locked bool, want string) error {
	wantLocked := want == "locked"
	if locked != wantLocked {
		return fmt.Errorf("%s locked = %v, want %v", label, locked, wantLocked)
	}
	return nil
}

// registerUIProposal registers the Stage-2 Proposal assertion steps. The
// proposal-request action itself ("requests a proposal with velocity …") is
// shared verbatim with the API-level proposal feature, so a single handler in
// steps_proposal.go serves both, routing to the UI session in UI mode.
func registerUIProposal(reg *runtime.Registry) {
	reg.Step(`^the proposal screen shows confidence (.+)$`,
		func(wd any, a []string) error {
			if got := uiSession(wd).ProposalView().Confidence; got != a[0] {
				return fmt.Errorf("confidence = %q, want %q", got, a[0])
			}
			return nil
		})

	reg.Step(`^the proposal screen shows contract (.+)$`,
		func(wd any, a []string) error {
			if got := uiSession(wd).ProposalView().Contract; got != a[0] {
				return fmt.Errorf("contract = %q, want %q", got, a[0])
			}
			return nil
		})

	reg.Step(`^the proposal screen shows a cost range from (\d+) to (\d+)$`,
		func(wd any, a []string) error {
			v := uiSession(wd).ProposalView()
			return expectUIRange("cost", v.CostLow, v.CostHigh, a[0], a[1])
		})

	reg.Step(`^the proposal screen shows a timeline range from (\d+) to (\d+) weeks$`,
		func(wd any, a []string) error {
			v := uiSession(wd).ProposalView()
			return expectUIRange("timeline", v.WeeksLow, v.WeeksHigh, a[0], a[1])
		})

	reg.Step(`^the proposal screen shows success probability (\d+) at the low figure and (\d+) at the high figure$`,
		func(wd any, a []string) error {
			v := uiSession(wd).ProposalView()
			return expectUIRange("success", v.SuccessLow, v.SuccessHigh, a[0], a[1])
		})

	reg.Step(`^the proposal screen scope includes the task (.+)$`,
		func(wd any, a []string) error {
			return expectContains("scope", uiSession(wd).ProposalView().Scope, a[0])
		})

	reg.Step(`^the proposal screen assumptions include (.+)$`,
		func(wd any, a []string) error {
			return expectContains("assumptions", uiSession(wd).ProposalView().Assumptions, a[0])
		})

	reg.Step(`^the proposal screen shows no estimate mechanics$`,
		func(wd any, _ []string) error {
			html, err := uiSession(wd).ProposalHTML()
			if err != nil {
				return err
			}
			for _, marker := range proposalMechanicsMarkers {
				if strings.Contains(html, marker) {
					return fmt.Errorf("proposal screen leaks mechanics marker %q", marker)
				}
			}
			return nil
		})

	reg.Step(`^the proposal screen shows no proposal$`,
		func(wd any, _ []string) error {
			if uiSession(wd).ProposalView().HasProposal {
				return fmt.Errorf("expected no proposal on screen")
			}
			return nil
		})

	reg.Step(`^the proposal screen shows the error (.+)$`,
		func(wd any, a []string) error {
			if got := uiSession(wd).ProposalView().Error; got != a[0] {
				return fmt.Errorf("proposal error = %q, want %q", got, a[0])
			}
			return nil
		})
}

// proposalMechanicsMarkers are the Stage-1-only raw-mechanics labels and the
// risk-band heading the Stage-2 Proposal screen must never show. Their absence
// on the rendered screen is the transparency boundary.
var proposalMechanicsMarkers = []string{
	"Optimistic", "Most Likely", "Pessimistic",
	"Standard Deviation", "Relative Standard",
	"Risk Band",
}

// expectRange asserts an on-screen low..high range against two captured
// expected values.
func expectUIRange(label string, gotLow, gotHigh int, wantLow, wantHigh string) error {
	lo, err := strconv.Atoi(wantLow)
	if err != nil {
		return fmt.Errorf("invalid %s low %q: %w", label, wantLow, err)
	}
	hi, err := strconv.Atoi(wantHigh)
	if err != nil {
		return fmt.Errorf("invalid %s high %q: %w", label, wantHigh, err)
	}
	if gotLow != lo || gotHigh != hi {
		return fmt.Errorf("%s = %d..%d, want %d..%d", label, gotLow, gotHigh, lo, hi)
	}
	return nil
}

// expectContains asserts a client-facing list shows the wanted entry.
func expectContains(label string, list []string, want string) error {
	for _, item := range list {
		if item == want {
			return nil
		}
	}
	return fmt.Errorf("%s %v does not include %q", label, list, want)
}
