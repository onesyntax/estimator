package wbs

import (
	"errors"
	"testing"
)

func loginEstimates() []EstimateAssignment {
	return []EstimateAssignment{
		{TaskNumber: 1, Optimistic: 2, MostLikely: 5, Pessimistic: 13, Reasoning: "clean"},
		{TaskNumber: 2, Optimistic: 1, MostLikely: 2, Pessimistic: 3, Reasoning: "trivial"},
		{TaskNumber: 3, Optimistic: 3, MostLikely: 8, Pessimistic: 20, Reasoning: "legacy"},
	}
}

func serviceEstimate(t *testing.T, s *Service, id string, taskNumber int) *Estimate {
	t.Helper()
	w, err := s.Get(id)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	task, ok := w.TaskAt(taskNumber)
	if !ok {
		t.Fatalf("task number %d does not exist", taskNumber)
	}
	return task.Estimate
}

func TestGenerateEstimatesAppliesPrimedEstimates(t *testing.T) {
	s, id := approvedServiceWBS(t)
	s.PrimeEstimates(loginEstimates())

	if err := s.GenerateEstimates(id); err != nil {
		t.Fatalf("GenerateEstimates returned error: %v", err)
	}
	e := serviceEstimate(t, s, id, 1)
	if e == nil || e.Optimistic != 2 || e.MostLikely != 5 || e.Pessimistic != 13 || e.Reasoning != "clean" {
		t.Fatalf("task 1 estimate = %+v, want 2/5/13 clean", e)
	}
	w, _ := s.Get(id)
	if w.EstimatesApproved() {
		t.Fatal("generated estimates must be unapproved")
	}
}

func TestGenerateEstimatesRejectsUnapprovedWBS(t *testing.T) {
	s := NewService()
	s.Prime([]string{"Login API", "Login UI", "Session store"})
	id, _ := s.Generate(NewTextDocument("req"))
	s.PrimeEstimates(loginEstimates())

	if err := s.GenerateEstimates(id); !errors.Is(err, ErrWBSNotApprovedForEstimation) {
		t.Fatalf("GenerateEstimates unapproved error = %v, want ErrWBSNotApprovedForEstimation", err)
	}
	if e := serviceEstimate(t, s, id, 1); e != nil {
		t.Fatalf("task 1 estimate = %+v, want nil (none created)", e)
	}
}

func TestGenerateEstimatesReportsUnknownWBS(t *testing.T) {
	s := NewService()
	if err := s.GenerateEstimates("does-not-exist"); !errors.Is(err, ErrWBSNotFound) {
		t.Fatalf("GenerateEstimates unknown WBS error = %v, want ErrWBSNotFound", err)
	}
}

func TestPrimedEstimateProviderConsumesOneShotFIFO(t *testing.T) {
	p := &PrimedEstimateProvider{}
	p.PrimeEstimates([]EstimateAssignment{{TaskNumber: 1, Optimistic: 2, MostLikely: 5, Pessimistic: 13, Reasoning: "first"}})
	p.PrimeEstimates([]EstimateAssignment{{TaskNumber: 1, Optimistic: 1, MostLikely: 2, Pessimistic: 3, Reasoning: "second"}})

	first, _ := p.Estimate(nil)
	if len(first) != 1 || first[0].Reasoning != "first" {
		t.Fatalf("first estimate = %v, want [first]", first)
	}
	second, _ := p.Estimate(nil)
	if len(second) != 1 || second[0].Reasoning != "second" {
		t.Fatalf("second estimate = %v, want [second]", second)
	}
	empty, _ := p.Estimate(nil)
	if len(empty) != 0 {
		t.Fatalf("third estimate = %v, want empty (nothing primed)", empty)
	}
}

func TestServiceOverrideAndApproveDelegateToWBS(t *testing.T) {
	s, id := approvedServiceWBS(t)
	s.PrimeEstimates(loginEstimates())
	if err := s.GenerateEstimates(id); err != nil {
		t.Fatalf("GenerateEstimates error: %v", err)
	}

	if err := s.OverrideEstimate(id, 1, Estimate{Optimistic: 3, MostLikely: 8, Pessimistic: 20, Reasoning: "team"}); err != nil {
		t.Fatalf("OverrideEstimate error: %v", err)
	}
	if e := serviceEstimate(t, s, id, 1); e.MostLikely != 8 || e.Reasoning != "team" {
		t.Fatalf("task 1 estimate = %+v, want 3/8/20 team", e)
	}

	if err := s.ApproveEstimates(id); err != nil {
		t.Fatalf("ApproveEstimates error: %v", err)
	}
	w, _ := s.Get(id)
	if !w.EstimatesApproved() {
		t.Fatal("estimates must be approved after ApproveEstimates")
	}
}

func TestServiceEstimateOpsReportUnknownWBS(t *testing.T) {
	s := NewService()
	for name, op := range map[string]func() error{
		"OverrideEstimate": func() error {
			return s.OverrideEstimate("nope", 1, Estimate{Optimistic: 1, MostLikely: 2, Pessimistic: 3, Reasoning: "x"})
		},
		"ApproveEstimates": func() error { return s.ApproveEstimates("nope") },
	} {
		if err := op(); !errors.Is(err, ErrWBSNotFound) {
			t.Fatalf("%s on unknown WBS error = %v, want ErrWBSNotFound", name, err)
		}
	}
}

// failingEstimateProvider always fails, to verify the service propagates a
// provider error without mutating the WBS.
type failingEstimateProvider struct{ err error }

func (p failingEstimateProvider) Estimate(tasks []Task) ([]EstimateAssignment, error) {
	return nil, p.err
}

func TestGenerateEstimatesPropagatesProviderError(t *testing.T) {
	wantErr := errors.New("estimate provider unavailable")
	s := NewServiceWithProviders(&PrimedProvider{}, &PrimedRiskProvider{}, failingEstimateProvider{err: wantErr})
	s.Prime([]string{"Login API", "Login UI", "Session store"})
	id, _ := s.Generate(NewTextDocument("req"))
	if err := s.Approve(id); err != nil {
		t.Fatalf("Approve error: %v", err)
	}

	if err := s.GenerateEstimates(id); !errors.Is(err, wantErr) {
		t.Fatalf("GenerateEstimates provider error = %v, want %v", err, wantErr)
	}
	if e := serviceEstimate(t, s, id, 1); e != nil {
		t.Fatalf("task 1 estimate = %+v, want nil (no estimate on provider error)", e)
	}
}
