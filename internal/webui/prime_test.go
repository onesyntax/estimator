package webui

import "testing"

func TestParseRiskSeeds(t *testing.T) {
	got := ParseRiskSeeds("task 1: SQL injection; task 2: XSS")
	want := []RiskSeed{{TaskNumber: 1, Description: "SQL injection"}, {TaskNumber: 2, Description: "XSS"}}
	if len(got) != len(want) {
		t.Fatalf("got %d seeds, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("seed %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestParseEstimateSeeds(t *testing.T) {
	got := ParseEstimateSeeds("task 1: 2/5/13; task 2: 1/2/3")
	if len(got) != 2 {
		t.Fatalf("got %d seeds, want 2: %+v", len(got), got)
	}
	if got[0].TaskNumber != 1 || got[0].Optimistic != 2 || got[0].MostLikely != 5 || got[0].Pessimistic != 13 {
		t.Errorf("seed 0 = %+v, want 1: 2/5/13", got[0])
	}
	if got[1].TaskNumber != 2 || got[1].Optimistic != 1 || got[1].MostLikely != 2 || got[1].Pessimistic != 3 {
		t.Errorf("seed 1 = %+v, want 2: 1/2/3", got[1])
	}
	if got[0].Reasoning == "" {
		t.Error("expected a non-empty placeholder reasoning")
	}
}

func TestParseTaskList(t *testing.T) {
	got := ParseTaskList("Login API; Login UI; Session store")
	want := []string{"Login API", "Login UI", "Session store"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("task %d = %q, want %q", i, got[i], want[i])
		}
	}
}
