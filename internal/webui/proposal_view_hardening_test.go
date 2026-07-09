//go:build hardening

// Hardening tests for the Proposal view model, behind the `hardening` build tag
// so they stay out of the unit suite, coverage, CRAP, and DRY runs.
package webui

import (
	"testing"

	"estimation/internal/wbs"
)

// TestProposalViewHiddenUnlessFlagged pins the guard on ProposalView: the screen
// stays empty unless the session has actually flagged a proposal, even if a stale
// proposal pointer is present. This kills the `!s.hasProposal || s.proposal == nil`
// -> `&& ` mutation, which would render a proposal whenever hasProposal is false
// but the pointer is non-nil, dereferencing a proposal that was never published.
func TestProposalViewHiddenUnlessFlagged(t *testing.T) {
	s := &Session{proposal: &wbs.Proposal{Confidence: "high"}, hasProposal: false}

	v := s.ProposalView()
	if v.HasProposal {
		t.Fatal("ProposalView should report no proposal when hasProposal is false")
	}
	if v.Confidence != "" {
		t.Fatalf("confidence = %q, want empty when no proposal is flagged", v.Confidence)
	}
}
