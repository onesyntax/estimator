package webui

import (
	"testing"

	"estimation/internal/wbs"
)

// primeAndGenerate is the common Stage-1 setup: prime the next WBS with three
// tasks and generate it from a valid text requirement.
func primeAndGenerate(t *testing.T) *Session {
	t.Helper()
	s := NewSession()
	s.PrimeWBS([]string{"Login API", "Login UI", "Session store"})
	s.GenerateWBS(wbs.NewTextDocument("build a login system"))
	if v := s.BuildView(); v.Error != "" {
		t.Fatalf("setup generate: unexpected error %q", v.Error)
	}
	return s
}

func TestGenerateWBSFromTextShowsTasks(t *testing.T) {
	s := primeAndGenerate(t)

	v := s.BuildView()
	if !v.HasWBS {
		t.Fatal("expected the build view to show a WBS")
	}
	if len(v.Tasks) != 3 {
		t.Fatalf("task count = %d, want 3", len(v.Tasks))
	}
	if v.Tasks[0].Description != "Login API" {
		t.Fatalf("task 1 = %q, want Login API", v.Tasks[0].Description)
	}
}

func TestGenerateWBSFromPDFShowsTasks(t *testing.T) {
	s := NewSession()
	s.PrimeWBS([]string{"Login API", "Login UI", "Session store"})
	s.GenerateWBS(wbs.NewPDFDocument(wbs.EncodeMinimalPDF("build a login system")))

	v := s.BuildView()
	if !v.HasWBS || len(v.Tasks) != 3 || v.Tasks[0].Description != "Login API" {
		t.Fatalf("PDF generation did not populate the WBS: %+v", v)
	}
}

func TestEmptyRequirementShowsErrorAndNoWBS(t *testing.T) {
	s := NewSession()
	s.GenerateWBS(wbs.NewTextDocument(""))

	v := s.BuildView()
	if v.Error != "requirement is empty" {
		t.Fatalf("error = %q, want %q", v.Error, "requirement is empty")
	}
	if v.HasWBS {
		t.Fatal("expected no WBS after an empty requirement")
	}
}

func TestCorruptPDFShowsUnreadableError(t *testing.T) {
	s := NewSession()
	s.GenerateWBS(wbs.NewPDFDocument([]byte("not a valid pdf")))

	v := s.BuildView()
	if v.Error != "document could not be read" {
		t.Fatalf("error = %q, want %q", v.Error, "document could not be read")
	}
	if v.HasWBS {
		t.Fatal("expected no WBS after a corrupt PDF")
	}
}

func TestRisksAndEstimatesLockedUntilWBSApproved(t *testing.T) {
	s := primeAndGenerate(t)

	if v := s.BuildView(); !v.RisksLocked || !v.EstimatesLocked {
		t.Fatalf("sections should be locked before approval: %+v", v)
	}

	s.ApproveWBS()

	if v := s.BuildView(); v.RisksLocked || v.EstimatesLocked {
		t.Fatalf("sections should be available after approval: %+v", v)
	}
}
