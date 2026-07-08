//go:build hardening

// Hardening tests: added by the hardender to kill mutation survivors that the
// normal unit suite leaves alive. They live behind the `hardening` build tag so
// they stay out of the unit suite, coverage, CRAP, and DRY runs, and are
// exercised only by the language mutation tool via `-tags hardening`.
package wbs

import "testing"

// Kills the three Proposal `<= 0 -> <= 1` mutants on the team-input guard: 1 is
// the smallest positive value each of Velocity, Capacity, and Rate may take, so a
// proposal built with all three at 1 must be accepted. Each mutant rejects the
// boundary input it widened, failing this success expectation.
func TestHardeningProposalAcceptsBoundaryPositiveInputs(t *testing.T) {
	w := approvedEstimatedSet(t, [3]int{2, 5, 13}, [3]int{1, 2, 3}, [3]int{3, 8, 20})

	if _, err := w.Proposal(TeamInputs{Velocity: 1, Capacity: 1, Rate: 1}); err != nil {
		t.Fatalf("Proposal with the minimal positive inputs 1/1/1 err = %v, want it accepted", err)
	}
}
