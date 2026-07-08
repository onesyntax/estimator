package wbs

import (
	"errors"
	"testing"
)

// approvedEstimatesServiceWBS returns a service whose standard 3-task WBS has an
// approved estimate set, ready for a proposal.
func approvedEstimatesServiceWBS(t *testing.T) (*Service, string) {
	t.Helper()
	s, id := approvedServiceWBS(t)
	s.PrimeEstimates(loginEstimates())
	if err := s.GenerateEstimates(id); err != nil {
		t.Fatalf("GenerateEstimates error: %v", err)
	}
	if err := s.ApproveEstimates(id); err != nil {
		t.Fatalf("ApproveEstimates error: %v", err)
	}
	return s, id
}

func TestServiceProposalDelegatesToWBS(t *testing.T) {
	s, id := approvedEstimatesServiceWBS(t)

	p, err := s.Proposal(id, TeamInputs{Velocity: 10, Capacity: 40, Rate: 100})
	if err != nil {
		t.Fatalf("Proposal error: %v", err)
	}
	if len(p.Scope) != 3 {
		t.Fatalf("proposal scope = %d items, want 3", len(p.Scope))
	}
	if p.Confidence == "" || p.Contract == "" {
		t.Fatalf("proposal = %+v, want non-empty confidence and contract", p)
	}
}

func TestServiceProposalReportsUnknownWBS(t *testing.T) {
	s := NewService()
	if _, err := s.Proposal("nope", TeamInputs{Velocity: 10, Capacity: 40, Rate: 100}); !errors.Is(err, ErrWBSNotFound) {
		t.Fatalf("Proposal on unknown WBS error = %v, want ErrWBSNotFound", err)
	}
}

func TestServiceProposalPropagatesDomainError(t *testing.T) {
	s, id := approvedServiceWBS(t) // approved WBS but no estimates approved yet
	if _, err := s.Proposal(id, TeamInputs{Velocity: 10, Capacity: 40, Rate: 100}); !errors.Is(err, ErrEstimatesNotApprovedForProposal) {
		t.Fatalf("Proposal without approved estimates error = %v, want ErrEstimatesNotApprovedForProposal", err)
	}
}
