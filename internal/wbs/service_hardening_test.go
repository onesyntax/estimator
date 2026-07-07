//go:build hardening

package wbs

import "testing"

// Kills NewServiceWithProvider `nextID: 1 -> 0`: generated WBS ids start at
// wbs-1 and increment by one per generation.
func TestHardeningServiceGeneratesSequentialIDsFromOne(t *testing.T) {
	s := NewService()
	s.Prime([]string{"A"})
	s.Prime([]string{"B"})

	first, err := s.Generate(NewTextDocument("req one"))
	if err != nil {
		t.Fatalf("first Generate error: %v", err)
	}
	if first != "wbs-1" {
		t.Fatalf("first id = %q, want wbs-1", first)
	}

	second, err := s.Generate(NewTextDocument("req two"))
	if err != nil {
		t.Fatalf("second Generate error: %v", err)
	}
	if second != "wbs-2" {
		t.Fatalf("second id = %q, want wbs-2", second)
	}
}
