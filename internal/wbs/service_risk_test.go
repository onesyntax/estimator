package wbs

import (
	"errors"
	"testing"
)

// approvedServiceWBS generates the standard 3-task WBS, approves it, and returns
// the service and the WBS id.
func approvedServiceWBS(t *testing.T) (*Service, string) {
	t.Helper()
	s := NewService()
	s.Prime([]string{"Login API", "Login UI", "Session store"})
	id, err := s.Generate(NewTextDocument("req"))
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if err := s.Approve(id); err != nil {
		t.Fatalf("Approve returned error: %v", err)
	}
	return s, id
}

func serviceNoteCount(t *testing.T, s *Service, id string, taskNumber int) int {
	t.Helper()
	w, err := s.Get(id)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	task, ok := w.TaskAt(taskNumber)
	if !ok {
		t.Fatalf("task number %d does not exist", taskNumber)
	}
	return len(task.RiskNotes)
}

func TestFlagRisksAppliesPrimedNotes(t *testing.T) {
	s, id := approvedServiceWBS(t)
	s.PrimeRisks([]RiskAssignment{
		{TaskNumber: 1, Description: "SQL injection"},
		{TaskNumber: 1, Description: "Rate limiting"},
		{TaskNumber: 2, Description: "XSS"},
	})

	if err := s.FlagRisks(id); err != nil {
		t.Fatalf("FlagRisks returned error: %v", err)
	}
	if got := serviceNoteCount(t, s, id, 1); got != 2 {
		t.Fatalf("task 1 note count = %d, want 2", got)
	}
	if got := serviceNoteCount(t, s, id, 2); got != 1 {
		t.Fatalf("task 2 note count = %d, want 1", got)
	}
	if got := serviceNoteCount(t, s, id, 3); got != 0 {
		t.Fatalf("task 3 note count = %d, want 0", got)
	}
}

func TestFlagRisksRejectsUnapprovedWBS(t *testing.T) {
	s := NewService()
	s.Prime([]string{"Login API", "Login UI", "Session store"})
	id, _ := s.Generate(NewTextDocument("req"))
	s.PrimeRisks([]RiskAssignment{{TaskNumber: 1, Description: "SQL injection"}})

	if err := s.FlagRisks(id); !errors.Is(err, ErrWBSNotApproved) {
		t.Fatalf("FlagRisks unapproved error = %v, want ErrWBSNotApproved", err)
	}
	if got := serviceNoteCount(t, s, id, 1); got != 0 {
		t.Fatalf("task 1 note count = %d, want 0 (no notes created)", got)
	}
}

func TestFlagRisksReportsUnknownWBS(t *testing.T) {
	s := NewService()
	if err := s.FlagRisks("does-not-exist"); !errors.Is(err, ErrWBSNotFound) {
		t.Fatalf("FlagRisks unknown WBS error = %v, want ErrWBSNotFound", err)
	}
}

func TestFlagRisksReplacesOnReFlag(t *testing.T) {
	s, id := approvedServiceWBS(t)
	s.PrimeRisks([]RiskAssignment{{TaskNumber: 1, Description: "SQL injection"}, {TaskNumber: 2, Description: "XSS"}})
	if err := s.FlagRisks(id); err != nil {
		t.Fatalf("first FlagRisks error: %v", err)
	}
	if err := s.AddRiskNote(id, 3, "Manual concern"); err != nil {
		t.Fatalf("AddRiskNote error: %v", err)
	}
	s.PrimeRisks([]RiskAssignment{{TaskNumber: 1, Description: "CSRF"}})
	if err := s.FlagRisks(id); err != nil {
		t.Fatalf("second FlagRisks error: %v", err)
	}

	if got := serviceNoteCount(t, s, id, 1); got != 1 {
		t.Fatalf("task 1 note count = %d, want 1", got)
	}
	if got := serviceNoteCount(t, s, id, 2); got != 0 {
		t.Fatalf("task 2 note count = %d, want 0", got)
	}
	if got := serviceNoteCount(t, s, id, 3); got != 0 {
		t.Fatalf("task 3 note count = %d, want 0 (manual note replaced)", got)
	}
}

func TestPrimedRiskProviderConsumesOneShotFIFO(t *testing.T) {
	p := &PrimedRiskProvider{}
	p.PrimeRisks([]RiskAssignment{{TaskNumber: 1, Description: "first"}})
	p.PrimeRisks([]RiskAssignment{{TaskNumber: 2, Description: "second"}})

	first, _ := p.FlagRisks(nil)
	if len(first) != 1 || first[0].Description != "first" {
		t.Fatalf("first flag = %v, want [first]", first)
	}
	second, _ := p.FlagRisks(nil)
	if len(second) != 1 || second[0].Description != "second" {
		t.Fatalf("second flag = %v, want [second]", second)
	}
	empty, _ := p.FlagRisks(nil)
	if len(empty) != 0 {
		t.Fatalf("third flag = %v, want empty (nothing primed)", empty)
	}
}

func TestServiceRiskNoteEditsDelegateToWBS(t *testing.T) {
	s, id := approvedServiceWBS(t)
	s.PrimeRisks([]RiskAssignment{{TaskNumber: 1, Description: "SQL injection"}})
	if err := s.FlagRisks(id); err != nil {
		t.Fatalf("FlagRisks error: %v", err)
	}

	if err := s.EditRiskNote(id, 1, 1, "SQL injection via login"); err != nil {
		t.Fatalf("EditRiskNote error: %v", err)
	}
	w, _ := s.Get(id)
	task, _ := w.TaskAt(1)
	if task.RiskNotes[0].Description != "SQL injection via login" {
		t.Fatalf("note = %q, want SQL injection via login", task.RiskNotes[0].Description)
	}

	if err := s.DeleteRiskNote(id, 1, 1); err != nil {
		t.Fatalf("DeleteRiskNote error: %v", err)
	}
	if got := serviceNoteCount(t, s, id, 1); got != 0 {
		t.Fatalf("task 1 note count = %d, want 0 after delete", got)
	}
}

// failingRiskProvider is a RiskProvider whose FlagRisks always fails, used to
// verify the service propagates a provider error without mutating the WBS.
type failingRiskProvider struct{ err error }

func (p failingRiskProvider) FlagRisks(tasks []Task) ([]RiskAssignment, error) {
	return nil, p.err
}

func TestFlagRisksPropagatesProviderError(t *testing.T) {
	wantErr := errors.New("risk provider unavailable")
	s := NewServiceWithProviders(&PrimedProvider{}, failingRiskProvider{err: wantErr})
	s.Prime([]string{"Login API", "Login UI", "Session store"})
	id, err := s.Generate(NewTextDocument("req"))
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if err := s.Approve(id); err != nil {
		t.Fatalf("Approve returned error: %v", err)
	}

	if err := s.FlagRisks(id); !errors.Is(err, wantErr) {
		t.Fatalf("FlagRisks provider error = %v, want %v", err, wantErr)
	}
	if got := serviceNoteCount(t, s, id, 1); got != 0 {
		t.Fatalf("task 1 note count = %d, want 0 (no notes applied on provider error)", got)
	}
}

func TestServiceRiskOpsReportUnknownWBS(t *testing.T) {
	s := NewService()
	for name, op := range map[string]func() error{
		"AddRiskNote":    func() error { return s.AddRiskNote("nope", 1, "x") },
		"EditRiskNote":   func() error { return s.EditRiskNote("nope", 1, 1, "x") },
		"DeleteRiskNote": func() error { return s.DeleteRiskNote("nope", 1, 1) },
	} {
		if err := op(); !errors.Is(err, ErrWBSNotFound) {
			t.Fatalf("%s on unknown WBS error = %v, want ErrWBSNotFound", name, err)
		}
	}
}
