package webui

import "testing"

func TestTaskRowExposesPerTaskMetricsAndRiskCount(t *testing.T) {
	s := approvedWBS(t)
	s.PrimeRisks([]RiskSeed{{TaskNumber: 1, Description: "SQL injection"}})
	s.FlagRisks()
	s.PrimeEstimates(mixedSeeds()) // task 1: 2/5/13
	s.GenerateEstimates()

	row := s.BuildView().Tasks[0]
	if row.RiskCount != 1 {
		t.Errorf("RiskCount = %d, want 1", row.RiskCount)
	}
	if row.Metrics == nil {
		t.Fatal("expected per-task metrics once estimated")
	}
	// 2/5/13 → PERT mean (2+20+13)/6 = 5.83→6, SD (13-2)/6 = 1.83→2, RSD 2/6→31%.
	if row.Metrics.Expected != 6 || row.Metrics.StandardDeviation != 2 {
		t.Errorf("per-task metrics = E%d SD%d, want E6 SD2", row.Metrics.Expected, row.Metrics.StandardDeviation)
	}
}

func TestTaskRowNoMetricsBeforeEstimates(t *testing.T) {
	s := approvedWBS(t)
	row := s.BuildView().Tasks[0]
	if row.Metrics != nil {
		t.Errorf("expected no per-task metrics before estimates, got %+v", row.Metrics)
	}
	if row.RiskCount != 0 {
		t.Errorf("RiskCount = %d, want 0", row.RiskCount)
	}
}
