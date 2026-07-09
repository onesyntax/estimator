package webui

import "testing"

func TestAddTaskShownOnScreen(t *testing.T) {
	s := primeAndGenerate(t)
	s.AddTask("Password reset")

	v := s.BuildView()
	if len(v.Tasks) != 4 {
		t.Fatalf("task count = %d, want 4", len(v.Tasks))
	}
	if v.Tasks[0].Description != "Login API" {
		t.Fatalf("task 1 = %q, want Login API", v.Tasks[0].Description)
	}
	if v.Tasks[3].Description != "Password reset" {
		t.Fatalf("task 4 = %q, want Password reset", v.Tasks[3].Description)
	}
}

func TestEditTaskShownOnScreen(t *testing.T) {
	s := primeAndGenerate(t)
	s.EditTask(1, "SSO API")

	v := s.BuildView()
	if len(v.Tasks) != 3 || v.Tasks[0].Description != "SSO API" {
		t.Fatalf("edit not reflected: %+v", v.Tasks)
	}
}

func TestDeleteTaskShownOnScreen(t *testing.T) {
	s := primeAndGenerate(t)
	s.DeleteTask(1)

	v := s.BuildView()
	if len(v.Tasks) != 2 || v.Tasks[0].Description != "Login UI" {
		t.Fatalf("delete not reflected: %+v", v.Tasks)
	}
}

func TestApproveEmptyWBSRefusedInlineAndKeepsGateClosed(t *testing.T) {
	s := primeAndGenerate(t)
	s.DeleteAllTasks()
	s.ApproveWBS()

	v := s.BuildView()
	if v.Error != "cannot approve an empty WBS" {
		t.Fatalf("error = %q, want %q", v.Error, "cannot approve an empty WBS")
	}
	if !v.RisksLocked {
		t.Fatal("risks section should stay locked after a failed approval")
	}
}
