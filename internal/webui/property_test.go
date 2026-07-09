//go:build property

// Property tests for the web UI adapter's pure QA-priming parsers. Kept out of
// the normal unit suite (and coverage/mutation/CRAP) by the `property` build
// tag; run them with scripts/property.sh. The example-based prime_test.go pins
// specific strings; these assert the parsers' invariants over broad inputs:
// they never panic, they only ever emit clean (trimmed, non-empty) entries, and
// well-formed input round-trips through formatting and re-parsing unchanged.
package webui

import (
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"testing/quick"
)

const propertySeed = 20260707

func propertyConfig() *quick.Config {
	return &quick.Config{MaxCount: 1000, Rand: rand.New(rand.NewSource(propertySeed))}
}

// TestPropParsersNeverEmitDirtyEntries checks the cleanliness invariant that
// every parser guarantees regardless of input: a returned task/description is
// non-empty and carries no surrounding whitespace, and every estimate seed gets
// a non-empty placeholder reasoning. Arbitrary strings (including ones dense
// with ';' and ':' separators) must not break these or panic.
func TestPropParsersNeverEmitDirtyEntries(t *testing.T) {
	invariant := func(s string) bool {
		for _, task := range ParseTaskList(s) {
			if task == "" || task != strings.TrimSpace(task) {
				return false
			}
		}
		for _, seed := range ParseRiskSeeds(s) {
			if seed.Description == "" || seed.Description != strings.TrimSpace(seed.Description) {
				return false
			}
		}
		for _, seed := range ParseEstimateSeeds(s) {
			if seed.Reasoning == "" {
				return false
			}
		}
		return true
	}
	if err := quick.Check(invariant, propertyConfig()); err != nil {
		t.Fatal(err)
	}
}

// TestPropParsersTolerateSeparatorSoup checks robustness: strings built only
// from the separator characters the parsers key on (';', ':', '/', "task",
// spaces, digits) must be handled without panicking, and any survivors still
// satisfy the cleanliness invariants. This stresses the malformed-entry paths
// that random UTF-8 rarely reaches.
func TestPropParsersTolerateSeparatorSoup(t *testing.T) {
	rng := rand.New(rand.NewSource(propertySeed))
	tokens := []string{";", ":", "/", "task", " ", "1", "5", "13", "", "-2", "x", "//"}
	for iter := 0; iter < 1000; iter++ {
		var b strings.Builder
		for n := rng.Intn(12); n > 0; n-- {
			b.WriteString(tokens[rng.Intn(len(tokens))])
		}
		s := b.String()
		for _, task := range ParseTaskList(s) {
			if task == "" || task != strings.TrimSpace(task) {
				t.Fatalf("iter %d: ParseTaskList(%q) emitted dirty entry %q", iter, s, task)
			}
		}
		for _, seed := range ParseRiskSeeds(s) {
			if seed.Description == "" || seed.Description != strings.TrimSpace(seed.Description) {
				t.Fatalf("iter %d: ParseRiskSeeds(%q) emitted dirty description %q", iter, s, seed.Description)
			}
		}
		for _, seed := range ParseEstimateSeeds(s) {
			if seed.Reasoning == "" {
				t.Fatalf("iter %d: ParseEstimateSeeds(%q) emitted empty reasoning", iter, s)
			}
		}
	}
}

// TestPropTaskListRoundTrip checks that a list of clean descriptions (non-empty,
// trimmed, free of the ';' separator) survives formatting as a "; "-joined line
// and re-parsing: same count, same descriptions, same order.
func TestPropTaskListRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(propertySeed))
	for iter := 0; iter < 1000; iter++ {
		descs := randCleanDescriptions(rng, "task")
		got := ParseTaskList(strings.Join(descs, "; "))
		if len(got) != len(descs) {
			t.Fatalf("iter %d: got %d tasks, want %d: %q", iter, len(got), len(descs), got)
		}
		for i, want := range descs {
			if got[i] != want {
				t.Fatalf("iter %d: task %d = %q, want %q", iter, i, got[i], want)
			}
		}
	}
}

// TestPropRiskSeedRoundTrip checks that risk seeds with clean descriptions
// survive formatting as "task N: desc" entries and re-parsing with their task
// number and description intact, in order.
func TestPropRiskSeedRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(propertySeed))
	for iter := 0; iter < 1000; iter++ {
		descs := randCleanDescriptions(rng, "risk")
		entries := make([]string, len(descs))
		numbers := make([]int, len(descs))
		for i, d := range descs {
			numbers[i] = rng.Intn(200) - 50 // includes zero and negatives; the parser preserves them
			entries[i] = "task " + strconv.Itoa(numbers[i]) + ": " + d
		}
		got := ParseRiskSeeds(strings.Join(entries, "; "))
		if len(got) != len(descs) {
			t.Fatalf("iter %d: got %d seeds, want %d: %+v", iter, len(got), len(descs), got)
		}
		for i := range descs {
			if got[i].TaskNumber != numbers[i] || got[i].Description != descs[i] {
				t.Fatalf("iter %d: seed %d = %+v, want {%d %q}", iter, i, got[i], numbers[i], descs[i])
			}
		}
	}
}

// TestPropEstimateSeedRoundTrip checks that O/M/P triples survive formatting as
// "task N: O/M/P" entries and re-parsing with the number and all three values
// intact, in order, each carrying a non-empty reasoning.
func TestPropEstimateSeedRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(propertySeed))
	for iter := 0; iter < 1000; iter++ {
		count := rng.Intn(5)
		entries := make([]string, count)
		numbers := make([]int, count)
		triples := make([][3]int, count)
		for i := range entries {
			numbers[i] = rng.Intn(200) - 50
			triples[i] = [3]int{rng.Intn(100), rng.Intn(100), rng.Intn(100)}
			entries[i] = "task " + strconv.Itoa(numbers[i]) + ": " +
				strconv.Itoa(triples[i][0]) + "/" + strconv.Itoa(triples[i][1]) + "/" + strconv.Itoa(triples[i][2])
		}
		got := ParseEstimateSeeds(strings.Join(entries, "; "))
		if len(got) != count {
			t.Fatalf("iter %d: got %d seeds, want %d: %+v", iter, len(got), count, got)
		}
		for i := range entries {
			want := EstimateSeed{
				TaskNumber:  numbers[i],
				Optimistic:  triples[i][0],
				MostLikely:  triples[i][1],
				Pessimistic: triples[i][2],
				Reasoning:   got[i].Reasoning,
			}
			if got[i] != want || got[i].Reasoning == "" {
				t.Fatalf("iter %d: seed %d = %+v, want number %d triple %v with non-empty reasoning",
					iter, i, got[i], numbers[i], triples[i])
			}
		}
	}
}

// randCleanDescriptions builds 0..4 descriptions that are non-empty, carry no
// leading/trailing whitespace, and contain neither the ';' entry separator nor a
// ':' (so risk entries split at the intended colon). These are exactly the
// descriptions the parsers round-trip losslessly.
func randCleanDescriptions(rng *rand.Rand, prefix string) []string {
	descs := make([]string, rng.Intn(5))
	for i := range descs {
		descs[i] = prefix + "-" + strconv.Itoa(rng.Intn(10000))
	}
	return descs
}
