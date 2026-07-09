package webui

import (
	"strings"
	"testing"
)

// mechanicsMarkers are the Stage-1-only raw-mechanics labels and the risk-band
// heading the Stage-2 Proposal screen must never show. They are the on-screen
// evidence of the transparency boundary: none of them appears in the client
// view.
var mechanicsMarkers = []string{
	"Optimistic", "Most Likely", "Pessimistic",
	"Standard Deviation", "Relative Standard",
	"Risk Band",
}

func TestBuildHTMLShowsTasksAndMechanics(t *testing.T) {
	s := estimatedWBS(t, uniformSeeds(2, 3, 5))

	html, err := s.BuildHTML()
	if err != nil {
		t.Fatalf("BuildHTML: %v", err)
	}
	for _, want := range []string{"Login API", "Optimistic", "green", "fixed-price"} {
		if !strings.Contains(html, want) {
			t.Errorf("build HTML missing %q", want)
		}
	}
}

func TestProposalHTMLHidesMechanics(t *testing.T) {
	s := approvedWBS(t)
	s.PrimeRisks([]RiskSeed{{TaskNumber: 1, Description: "SQL injection"}})
	s.FlagRisks()
	s.PrimeEstimates(mixedSeeds())
	s.GenerateEstimates()
	s.ApproveEstimates()
	s.RequestProposal(TeamInput{Velocity: 3, Capacity: 30, Rate: 120})

	html, err := s.ProposalHTML()
	if err != nil {
		t.Fatalf("ProposalHTML: %v", err)
	}
	// The client-clean view shows scope and assumptions.
	for _, want := range []string{"Login API", "SQL injection", "medium", "fixed-price-with-buffer"} {
		if !strings.Contains(html, want) {
			t.Errorf("proposal HTML missing %q", want)
		}
	}
	// And none of the raw mechanics or risk-band flags.
	for _, marker := range mechanicsMarkers {
		if strings.Contains(html, marker) {
			t.Errorf("proposal HTML leaks mechanics marker %q", marker)
		}
	}
}
