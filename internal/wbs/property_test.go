//go:build property

// Package-level property tests for the WBS core. These are kept out of the
// normal unit suite (and therefore out of coverage, mutation, and CRAP runs) by
// the `property` build tag; run them with scripts/property.sh. They assert
// invariants over broad input ranges rather than single examples: parsing
// round trips, idempotence, FIFO conservation, stable numbering, and the
// approval/edit invariants of the aggregate.
package wbs

import (
	"errors"
	"math/rand"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/quick"
)

// propertySeed keeps property runs deterministic so a failure reproduces and CI
// never flakes on a lucky input.
const propertySeed = 20260707

func propertyConfig() *quick.Config {
	return &quick.Config{MaxCount: 1000, Rand: rand.New(rand.NewSource(propertySeed))}
}

// stripParens mirrors the encoder's escaping so the round-trip property can
// predict the recoverable text for any input.
var stripParens = strings.NewReplacer("(", "", ")", "")

// Round trip / parsing stability: text encoded into a minimal PDF and read back
// out recovers the normalized (paren-stripped, trimmed) input, and yields the
// empty-requirement error exactly when nothing recoverable remains.
func TestPropPDFRoundTrip(t *testing.T) {
	f := func(s string) bool {
		want := strings.TrimSpace(stripParens.Replace(s))
		got, err := extractPDFText(EncodeMinimalPDF(s))
		if want == "" {
			return errors.Is(err, ErrEmptyRequirement)
		}
		return err == nil && got == want
	}
	if err := quick.Check(f, propertyConfig()); err != nil {
		t.Error(err)
	}
}

// Idempotence: requireDescription trims, and a value it accepts is a fixed point
// of a second application. Blank inputs are rejected exactly when they trim away.
func TestPropRequireDescriptionIdempotent(t *testing.T) {
	f := func(s string) bool {
		first, err := requireDescription(s)
		if err != nil {
			return errors.Is(err, ErrEmptyDescription) && strings.TrimSpace(s) == ""
		}
		second, err := requireDescription(first)
		return err == nil && second == first && first == strings.TrimSpace(s)
	}
	if err := quick.Check(f, propertyConfig()); err != nil {
		t.Error(err)
	}
}

// FIFO conservation: primed task lists are handed back in priming order, one
// shot each, and generation past the queue yields an empty list without error.
func TestPropPrimedProviderFIFO(t *testing.T) {
	f := func(lists [][]string) bool {
		p := &PrimedProvider{}
		for _, l := range lists {
			p.Prime(l)
		}
		for _, want := range lists {
			got, err := p.Generate("req")
			if err != nil || !slices.Equal(got, want) {
				return false
			}
		}
		got, err := p.Generate("req")
		return err == nil && len(got) == 0
	}
	if err := quick.Check(f, propertyConfig()); err != nil {
		t.Error(err)
	}
}

// Numbering and ordering invariants: NewWBS preserves description order, assigns
// unique one-based IDs, keeps TaskAt consistent with Tasks, and starts
// unapproved with out-of-range numbers rejected.
func TestPropNewWBSInvariants(t *testing.T) {
	f := func(descs []string) bool {
		w := NewWBS("wbs", descs)
		if w.TaskCount() != len(descs) || w.Approved() {
			return false
		}
		seen := make(map[string]bool, len(descs))
		for i, task := range w.Tasks() {
			if task.Description != descs[i] {
				return false // order preserved
			}
			if task.ID != "t"+strconv.Itoa(i+1) || seen[task.ID] {
				return false // stable, unique numbering
			}
			seen[task.ID] = true
			if at, ok := w.TaskAt(i + 1); !ok || at != task {
				return false // TaskAt agrees with Tasks
			}
		}
		if _, ok := w.TaskAt(0); ok {
			return false
		}
		if _, ok := w.TaskAt(len(descs) + 1); ok {
			return false
		}
		return true
	}
	if err := quick.Check(f, propertyConfig()); err != nil {
		t.Error(err)
	}
}

// Aggregate invariants under a random successful structural edit: adds and
// deletes conserve the count by exactly one, deletes preserve the order of the
// survivors, every structural edit unapproves, and an unapproved WBS never
// exposes an approved task list.
func TestPropStructuralEditInvariants(t *testing.T) {
	rng := rand.New(rand.NewSource(propertySeed))
	for iter := 0; iter < 1000; iter++ {
		n := 1 + rng.Intn(6)
		descs := make([]string, n)
		for i := range descs {
			descs[i] = "task-" + strconv.Itoa(rng.Intn(1000))
		}
		w := NewWBS("w", descs)
		if err := w.Approve(); err != nil {
			t.Fatalf("iter %d: Approve of non-empty WBS failed: %v", iter, err)
		}
		if !w.Approved() {
			t.Fatalf("iter %d: WBS not approved after Approve", iter)
		}

		switch rng.Intn(3) {
		case 0:
			before := w.TaskCount()
			if err := w.AddTask("added"); err != nil {
				t.Fatalf("iter %d: AddTask failed: %v", iter, err)
			}
			if w.TaskCount() != before+1 {
				t.Fatalf("iter %d: AddTask changed count %d -> %d, want +1", iter, before, w.TaskCount())
			}
		case 1:
			pos := 1 + rng.Intn(w.TaskCount())
			before := w.TaskCount()
			if err := w.EditTask(pos, "changed"); err != nil {
				t.Fatalf("iter %d: EditTask failed: %v", iter, err)
			}
			if w.TaskCount() != before {
				t.Fatalf("iter %d: EditTask changed count", iter)
			}
			if got, _ := w.TaskAt(pos); got.Description != "changed" {
				t.Fatalf("iter %d: EditTask did not update description", iter)
			}
		case 2:
			pos := 1 + rng.Intn(w.TaskCount())
			before := w.Tasks()
			if err := w.DeleteTask(pos); err != nil {
				t.Fatalf("iter %d: DeleteTask failed: %v", iter, err)
			}
			if w.TaskCount() != len(before)-1 {
				t.Fatalf("iter %d: DeleteTask changed count %d -> %d, want -1", iter, len(before), w.TaskCount())
			}
			want := append(append([]Task{}, before[:pos-1]...), before[pos:]...)
			if !slices.Equal(w.Tasks(), want) {
				t.Fatalf("iter %d: DeleteTask did not preserve survivor order", iter)
			}
		}

		if w.Approved() {
			t.Fatalf("iter %d: structural edit must unapprove the WBS", iter)
		}
		if w.ApprovedTasks() != nil {
			t.Fatalf("iter %d: unapproved WBS must not expose an approved task list", iter)
		}
	}
}
