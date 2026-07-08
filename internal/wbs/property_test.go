//go:build property

// Package-level property tests for the WBS core. These are kept out of the
// normal unit suite (and therefore out of coverage, mutation, and CRAP runs) by
// the `property` build tag; run them with scripts/property.sh. They assert
// invariants over broad input ranges rather than single examples: parsing
// round trips, idempotence, FIFO conservation, stable numbering, and the
// approval/edit invariants of the aggregate.
package wbs

import (
	"bytes"
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

// taskEqual compares two tasks by value. Task is no longer directly comparable
// once it carries a RiskNote slice, so equality walks the notes (which are
// comparable) explicitly rather than relying on == or slices.Equal on []Task.
func taskEqual(a, b Task) bool {
	return a.ID == b.ID && a.Description == b.Description && slices.Equal(a.RiskNotes, b.RiskNotes)
}

// PDF pass-through: a PDF requirement is never text-extracted by the domain. A
// minimal PDF with drawn content passes its raw bytes through unmodified, and a
// content-free PDF (drawn from blank text) yields the empty-requirement error.
// The encoder draws content exactly when the input has non-whitespace text.
func TestPropPDFDocumentPassesContentOrRejectsEmpty(t *testing.T) {
	f := func(s string) bool {
		pdf := EncodeMinimalPDF(s)
		req, err := NewPDFDocument(pdf).Requirement()
		if strings.TrimSpace(s) == "" {
			return errors.Is(err, ErrEmptyRequirement)
		}
		return err == nil && req.Text == "" && bytes.Equal(req.PDF, pdf)
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
			got, err := p.Generate(Requirement{Text: "req"})
			if err != nil || !slices.Equal(got, want) {
				return false
			}
		}
		got, err := p.Generate(Requirement{Text: "req"})
		return err == nil && len(got) == 0
	}
	if err := quick.Check(f, propertyConfig()); err != nil {
		t.Error(err)
	}
}

// FIFO conservation for risk priming: primed assignment lists are handed back in
// priming order, one shot each, and flagging past the queue yields no assignments
// without error. The tasks argument is ignored, so any input drives the same
// primed output.
func TestPropPrimedRiskProviderFIFO(t *testing.T) {
	f := func(lists [][]RiskAssignment) bool {
		p := &PrimedRiskProvider{}
		for _, l := range lists {
			p.PrimeRisks(l)
		}
		for _, want := range lists {
			got, err := p.FlagRisks(nil)
			if err != nil || !slices.Equal(got, want) {
				return false
			}
		}
		got, err := p.FlagRisks(nil)
		return err == nil && len(got) == 0
	}
	if err := quick.Check(f, propertyConfig()); err != nil {
		t.Error(err)
	}
}

// ReplaceRiskNotes conservation and filtering: flagging discards all prior notes
// and reapplies exactly the assignments whose task number is in range (1..count),
// placed on their task in assignment order. Out-of-range numbers are dropped, so
// the total note count equals the number of in-range assignments; every note id is
// non-empty and unique; and approval is left unchanged.
func TestPropReplaceRiskNotesConservesInRangeAssignments(t *testing.T) {
	rng := rand.New(rand.NewSource(propertySeed))
	for iter := 0; iter < 1000; iter++ {
		n := 1 + rng.Intn(6)
		descs := make([]string, n)
		for i := range descs {
			descs[i] = "task-" + strconv.Itoa(i+1)
		}
		w := NewWBS("w", descs)
		if err := w.Approve(); err != nil {
			t.Fatalf("iter %d: Approve of non-empty WBS failed: %v", iter, err)
		}

		m := rng.Intn(9)
		assignments := make([]RiskAssignment, m)
		expected := make([][]string, n) // per task: expected note descriptions, in order
		inRange := 0
		for k := 0; k < m; k++ {
			// task numbers span -1..n+1 so both boundaries (0 and n+1) are exercised
			tn := rng.Intn(n+3) - 1
			desc := "r-" + strconv.Itoa(k)
			assignments[k] = RiskAssignment{TaskNumber: tn, Description: desc}
			if tn >= 1 && tn <= n {
				expected[tn-1] = append(expected[tn-1], desc)
				inRange++
			}
		}

		w.ReplaceRiskNotes(assignments)

		if !w.Approved() {
			t.Fatalf("iter %d: ReplaceRiskNotes must not change approval", iter)
		}
		total := 0
		ids := make(map[string]bool)
		for tnum := 1; tnum <= n; tnum++ {
			if got := noteDescriptions(w, tnum); !slices.Equal(got, expected[tnum-1]) {
				t.Fatalf("iter %d: task %d notes = %v, want %v", iter, tnum, got, expected[tnum-1])
			}
			task, _ := w.TaskAt(tnum)
			for _, note := range task.RiskNotes {
				if note.ID == "" {
					t.Fatalf("iter %d: task %d has an empty risk-note id", iter, tnum)
				}
				if ids[note.ID] {
					t.Fatalf("iter %d: duplicate risk-note id %q", iter, note.ID)
				}
				ids[note.ID] = true
			}
			total += len(task.RiskNotes)
		}
		if total != inRange {
			t.Fatalf("iter %d: total notes = %d, want %d in-range assignments", iter, total, inRange)
		}
	}
}

// Risk-note edit invariants over a random valid Add/Edit/Delete on a task that
// already has notes: Add raises that task's note count by one and preserves the
// existing notes as a prefix; Edit keeps the count and the edited note's id while
// changing only its description; Delete lowers the count by one and preserves the
// order and ids of the survivors. Note ids stay unique across the whole WBS.
func TestPropRiskNoteEditInvariants(t *testing.T) {
	rng := rand.New(rand.NewSource(propertySeed))
	for iter := 0; iter < 1000; iter++ {
		n := 1 + rng.Intn(4)
		descs := make([]string, n)
		for i := range descs {
			descs[i] = "task-" + strconv.Itoa(i+1)
		}
		w := NewWBS("w", descs)
		if err := w.Approve(); err != nil {
			t.Fatalf("iter %d: Approve of non-empty WBS failed: %v", iter, err)
		}

		// Seed every task with 1..3 notes so edit and delete always have a target.
		var seed []RiskAssignment
		for tnum := 1; tnum <= n; tnum++ {
			for j := 0; j < 1+rng.Intn(3); j++ {
				seed = append(seed, RiskAssignment{TaskNumber: tnum, Description: "seed-" + strconv.Itoa(tnum) + "-" + strconv.Itoa(j)})
			}
		}
		w.ReplaceRiskNotes(seed)

		task := 1 + rng.Intn(n)
		before, _ := w.TaskAt(task)
		beforeNotes := before.RiskNotes

		switch rng.Intn(3) {
		case 0: // Add appends one note, existing notes stay as a prefix.
			if err := w.AddRiskNote(task, "added"); err != nil {
				t.Fatalf("iter %d: AddRiskNote failed: %v", iter, err)
			}
			after, _ := w.TaskAt(task)
			if len(after.RiskNotes) != len(beforeNotes)+1 {
				t.Fatalf("iter %d: AddRiskNote count %d -> %d, want +1", iter, len(beforeNotes), len(after.RiskNotes))
			}
			if !slices.Equal(after.RiskNotes[:len(beforeNotes)], beforeNotes) {
				t.Fatalf("iter %d: AddRiskNote did not preserve existing notes as a prefix", iter)
			}
			if after.RiskNotes[len(beforeNotes)].Description != "added" {
				t.Fatalf("iter %d: appended note description = %q, want %q", iter, after.RiskNotes[len(beforeNotes)].Description, "added")
			}
		case 1: // Edit changes only the target note's description, keeping its id.
			pos := 1 + rng.Intn(len(beforeNotes))
			if err := w.EditRiskNote(task, pos, "changed"); err != nil {
				t.Fatalf("iter %d: EditRiskNote failed: %v", iter, err)
			}
			after, _ := w.TaskAt(task)
			if len(after.RiskNotes) != len(beforeNotes) {
				t.Fatalf("iter %d: EditRiskNote changed count", iter)
			}
			if after.RiskNotes[pos-1].ID != beforeNotes[pos-1].ID {
				t.Fatalf("iter %d: EditRiskNote changed the note id", iter)
			}
			if after.RiskNotes[pos-1].Description != "changed" {
				t.Fatalf("iter %d: EditRiskNote did not update the description", iter)
			}
			for i := range beforeNotes {
				if i != pos-1 && after.RiskNotes[i] != beforeNotes[i] {
					t.Fatalf("iter %d: EditRiskNote altered an untouched note at %d", iter, i)
				}
			}
		case 2: // Delete removes the target note, preserving survivor order and ids.
			pos := 1 + rng.Intn(len(beforeNotes))
			if err := w.DeleteRiskNote(task, pos); err != nil {
				t.Fatalf("iter %d: DeleteRiskNote failed: %v", iter, err)
			}
			after, _ := w.TaskAt(task)
			if len(after.RiskNotes) != len(beforeNotes)-1 {
				t.Fatalf("iter %d: DeleteRiskNote count %d -> %d, want -1", iter, len(beforeNotes), len(after.RiskNotes))
			}
			want := append(append([]RiskNote{}, beforeNotes[:pos-1]...), beforeNotes[pos:]...)
			if !slices.Equal(after.RiskNotes, want) {
				t.Fatalf("iter %d: DeleteRiskNote did not preserve survivor order/ids", iter)
			}
		}

		ids := make(map[string]bool)
		for _, tsk := range w.Tasks() {
			for _, note := range tsk.RiskNotes {
				if note.ID == "" {
					t.Fatalf("iter %d: empty risk-note id after edit", iter)
				}
				if ids[note.ID] {
					t.Fatalf("iter %d: duplicate risk-note id %q after edit", iter, note.ID)
				}
				ids[note.ID] = true
			}
		}
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
			if at, ok := w.TaskAt(i + 1); !ok || !taskEqual(at, task) {
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
			if !slices.EqualFunc(w.Tasks(), want, taskEqual) {
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
