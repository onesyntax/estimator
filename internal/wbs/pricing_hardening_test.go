//go:build hardening

// Hardening tests: added by the hardender to kill mutation survivors that the
// normal unit suite leaves alive. They live behind the `hardening` build tag so
// they stay out of the unit suite, coverage, CRAP, and DRY runs, and are
// exercised only by the language mutation tool via `-tags hardening`.
package wbs

import "testing"

// Kills bandForRSD `rsd < 10 -> rsd <= 10`: RSD 10 is the inclusive lower edge of
// the yellow band, not green. Hitting an exact integer project RSD from estimates
// is impractical, so the band mapping is asserted directly (as pricing_test.go
// notes it prefers).
func TestHardeningBandForRSDLowerEdgeIsYellow(t *testing.T) {
	if b := bandForRSD(10); b.flag != "yellow" {
		t.Fatalf("bandForRSD(10).flag = %q, want yellow (10 is the inclusive lower edge, not green)", b.flag)
	}
	if b := bandForRSD(9); b.flag != "green" {
		t.Fatalf("bandForRSD(9).flag = %q, want green", b.flag)
	}
}

// Kills bandForRSD green `invitesRenegotiation: false -> true`: a green (low-risk)
// project never invites renegotiation. The field is consumed only by Proposal, so
// it is asserted here on the band directly.
func TestHardeningBandForRSDGreenDoesNotInviteRenegotiation(t *testing.T) {
	if b := bandForRSD(9); b.invitesRenegotiation {
		t.Fatal("bandForRSD(9).invitesRenegotiation = true, want false for a green project")
	}
}
