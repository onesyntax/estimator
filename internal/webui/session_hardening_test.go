//go:build hardening

// Hardening tests for the session controller, behind the `hardening` build tag
// so they stay out of the unit suite, coverage, CRAP, and DRY runs.
package webui

import "testing"

// TestBuildViewTreatsMissingWBSAsAbsent covers Session.current's error path: an
// active id that the service cannot resolve must read as "no WBS", not as a
// present-but-nil WBS. This both covers the otherwise-unexercised
// `return nil, false` and kills its `false` -> `true` mutation, which would make
// BuildView dereference a nil WBS and panic.
func TestBuildViewTreatsMissingWBSAsAbsent(t *testing.T) {
	s := NewSession()
	s.wbsID = "does-not-exist"

	v := s.BuildView()
	if v.HasWBS {
		t.Fatal("BuildView should report no WBS when the active id does not resolve")
	}
	if len(v.Tasks) != 0 {
		t.Fatalf("tasks = %v, want none for an unresolved WBS", v.Tasks)
	}
}

// TestDeleteAllTasksStopsAtEmptyWithoutError pins DeleteAllTasks's loop bound:
// it must stop once the WBS is empty rather than attempt one delete too many.
// This kills the `TaskCount() > 0` -> `>= 0` mutation, which loops once past
// empty, deletes a nonexistent task, and leaves an inline error.
func TestDeleteAllTasksStopsAtEmptyWithoutError(t *testing.T) {
	s := primeAndGenerate(t)
	s.DeleteAllTasks()

	v := s.BuildView()
	if v.Error != "" {
		t.Fatalf("error = %q, want none after emptying the WBS", v.Error)
	}
	if len(v.Tasks) != 0 {
		t.Fatalf("tasks = %v, want none after DeleteAllTasks", v.Tasks)
	}
}

// TestRejectedProposalClearsHasProposal pins the reset at the top of
// RequestProposal: a rejected request must leave no active proposal. This kills
// the `s.hasProposal = false` -> `= true` mutation on the error path, which would
// leave hasProposal set even though no proposal was built.
func TestRejectedProposalClearsHasProposal(t *testing.T) {
	s := approvedEstimates(t, mixedSeeds())
	s.RequestProposal(TeamInput{Velocity: 0, Capacity: 30, Rate: 120})

	if s.hasProposal {
		t.Fatal("hasProposal should be false after a rejected proposal request")
	}
	if s.proposal != nil {
		t.Fatalf("proposal = %+v, want nil after a rejected request", s.proposal)
	}
}
