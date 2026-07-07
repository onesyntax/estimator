//go:build hardening

package runtime

import "testing"

// Covers and hardens MustRegister/Registered, the generated-entrypoint
// registration glue that the unit suite never exercises (it runs only from
// generated init functions). Kills MustRegister `err != nil -> err == nil`:
// a well-formed IR must register without panicking, and its feature must become
// visible through Registered.
func TestHardeningMustRegisterAddsWellFormedFeature(t *testing.T) {
	before := len(Registered())

	MustRegister(`{"name":"Registered Feature","scenarios":[]}`)

	after := Registered()
	if len(after) != before+1 {
		t.Fatalf("Registered length = %d, want %d", len(after), before+1)
	}
	if after[len(after)-1].Name != "Registered Feature" {
		t.Fatalf("last registered feature = %q, want Registered Feature", after[len(after)-1].Name)
	}
}

// Kills the same mutant from the other direction: a malformed IR is a
// programming error and must panic, leaving the registry unchanged.
func TestHardeningMustRegisterPanicsOnMalformedIR(t *testing.T) {
	before := len(Registered())
	defer func() {
		if recover() == nil {
			t.Fatal("MustRegister must panic on malformed IR")
		}
		if len(Registered()) != before {
			t.Fatalf("registry changed after failed register: len=%d, want %d", len(Registered()), before)
		}
	}()

	MustRegister(`{ this is not valid json `)
}
