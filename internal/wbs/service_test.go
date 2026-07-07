package wbs

import (
	"errors"
	"testing"
)

func TestServiceGeneratesUnapprovedWBS(t *testing.T) {
	s := NewService()
	s.Prime([]string{"Login API", "Login UI", "Session store"})

	id, err := s.Generate(NewTextDocument("valid requirement"))
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	w, err := s.Get(id)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if w.Approved() {
		t.Fatal("generated WBS must be unapproved")
	}
	if w.TaskCount() != 3 {
		t.Fatalf("task count = %d, want 3", w.TaskCount())
	}
	if task, _ := w.TaskAt(1); task.Description != "Login API" {
		t.Fatalf("task 1 = %q, want Login API", task.Description)
	}
}

func TestServiceGeneratesFromPDF(t *testing.T) {
	s := NewService()
	s.Prime([]string{"Charge API", "Refund API"})

	id, err := s.Generate(NewPDFDocument(EncodeMinimalPDF("requirement")))
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	w, _ := s.Get(id)
	if w.TaskCount() != 2 {
		t.Fatalf("task count = %d, want 2", w.TaskCount())
	}
	if task, _ := w.TaskAt(1); task.Description != "Charge API" {
		t.Fatalf("task 1 = %q, want Charge API", task.Description)
	}
}

func TestServiceRejectsEmptyDocumentAndCreatesNoWBS(t *testing.T) {
	s := NewService()
	s.Prime([]string{"Should", "Not", "Appear"})

	_, err := s.Generate(NewTextDocument(""))
	if !errors.Is(err, ErrEmptyRequirement) {
		t.Fatalf("Generate empty error = %v, want ErrEmptyRequirement", err)
	}
	if got := s.Count(); got != 0 {
		t.Fatalf("stored WBS count = %d, want 0", got)
	}
}

func TestServiceRejectsCorruptPDFAndCreatesNoWBS(t *testing.T) {
	s := NewService()

	_, err := s.Generate(NewPDFDocument([]byte("garbage")))
	if !errors.Is(err, ErrUnreadableDocument) {
		t.Fatalf("Generate corrupt error = %v, want ErrUnreadableDocument", err)
	}
	if got := s.Count(); got != 0 {
		t.Fatalf("stored WBS count = %d, want 0", got)
	}
}

func TestServiceReportsUnknownWBSAsNotFound(t *testing.T) {
	s := NewService()
	_, err := s.Get("does-not-exist")
	if !errors.Is(err, ErrWBSNotFound) {
		t.Fatalf("Get unknown error = %v, want ErrWBSNotFound", err)
	}
}

func TestServiceEditingUnknownWBSIsNotFound(t *testing.T) {
	s := NewService()
	for _, op := range []func() error{
		func() error { return s.AddTask("nope", "x") },
		func() error { return s.EditTask("nope", 1, "x") },
		func() error { return s.DeleteTask("nope", 1) },
		func() error { return s.Approve("nope") },
	} {
		if err := op(); !errors.Is(err, ErrWBSNotFound) {
			t.Fatalf("operation on unknown WBS error = %v, want ErrWBSNotFound", err)
		}
	}
}

func TestServiceDelegatesEditsToWBS(t *testing.T) {
	s := NewService()
	s.Prime([]string{"Login API", "Login UI", "Session store"})
	id, _ := s.Generate(NewTextDocument("req"))

	if err := s.AddTask(id, "Password reset"); err != nil {
		t.Fatalf("AddTask error: %v", err)
	}
	w, _ := s.Get(id)
	if w.TaskCount() != 4 {
		t.Fatalf("task count = %d, want 4", w.TaskCount())
	}
	if task, _ := w.TaskAt(4); task.Description != "Password reset" {
		t.Fatalf("task 4 = %q, want Password reset", task.Description)
	}
}
