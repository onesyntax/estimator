package webui

import "testing"

func approvedWBS(t *testing.T) *Session {
	t.Helper()
	s := primeAndGenerate(t)
	s.ApproveWBS()
	if v := s.BuildView(); v.RisksLocked {
		t.Fatalf("expected an approved WBS, risks still locked: %+v", v)
	}
	return s
}

func TestFlagRisksRendersNotesUnderTasks(t *testing.T) {
	s := approvedWBS(t)
	s.PrimeRisks([]RiskSeed{{TaskNumber: 1, Description: "SQL injection"}, {TaskNumber: 2, Description: "XSS"}})
	s.FlagRisks()

	v := s.BuildView()
	if got := v.Tasks[0].RiskNotes; len(got) != 1 || got[0] != "SQL injection" {
		t.Fatalf("task 1 risk notes = %v, want [SQL injection]", got)
	}
	if got := v.Tasks[1].RiskNotes; len(got) != 1 || got[0] != "XSS" {
		t.Fatalf("task 2 risk notes = %v, want [XSS]", got)
	}
}

func TestAddRiskNoteShownOnScreen(t *testing.T) {
	s := approvedWBS(t)
	s.PrimeRisks([]RiskSeed{{TaskNumber: 1, Description: "SQL injection"}})
	s.FlagRisks()
	s.AddRiskNote(2, "Manual concern")

	v := s.BuildView()
	if got := v.Tasks[1].RiskNotes; len(got) != 1 || got[0] != "Manual concern" {
		t.Fatalf("task 2 risk notes = %v, want [Manual concern]", got)
	}
}
