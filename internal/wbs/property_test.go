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
	"math"
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

// FIFO conservation for estimate priming: primed assignment lists are handed
// back in priming order, one shot each, and estimating past the queue yields no
// assignments without error. The tasks argument is ignored, so any input drives
// the same primed output.
func TestPropPrimedEstimateProviderFIFO(t *testing.T) {
	f := func(lists [][]EstimateAssignment) bool {
		p := &PrimedEstimateProvider{}
		for _, l := range lists {
			p.PrimeEstimates(l)
		}
		for _, want := range lists {
			got, err := p.Estimate(nil)
			if err != nil || !slices.Equal(got, want) {
				return false
			}
		}
		got, err := p.Estimate(nil)
		return err == nil && len(got) == 0
	}
	if err := quick.Check(f, propertyConfig()); err != nil {
		t.Error(err)
	}
}

// SetEstimates conservation and filtering: generation discards every prior
// estimate and reapplies exactly the assignments whose task number is in range
// (1..count); when several in-range assignments target the same task the last
// one wins. Out-of-range numbers are dropped, so a task ends with an estimate
// iff some in-range assignment named it; the set is left generated-but-unapproved
// and the WBS's own approval is untouched.
func TestPropSetEstimatesConservesInRangeAssignments(t *testing.T) {
	rng := rand.New(rand.NewSource(propertySeed))
	// Value pools are unvalidated on purpose: SetEstimates stores whatever it is
	// given (only OverrideEstimate validates), so the property is about routing,
	// not legality.
	values := []int{0, 1, 2, 3, 5, 8, 13, 20, 40, 100}
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
		assignments := make([]EstimateAssignment, m)
		expected := make([]*Estimate, n) // per task: the estimate the last in-range assignment sets, or nil
		for k := 0; k < m; k++ {
			// task numbers span -1..n+1 so both boundaries (0 and n+1) are exercised
			tn := rng.Intn(n+3) - 1
			a := EstimateAssignment{
				TaskNumber:  tn,
				Optimistic:  values[rng.Intn(len(values))],
				MostLikely:  values[rng.Intn(len(values))],
				Pessimistic: values[rng.Intn(len(values))],
				Reasoning:   "r-" + strconv.Itoa(k),
			}
			assignments[k] = a
			if tn >= 1 && tn <= n {
				expected[tn-1] = &Estimate{Optimistic: a.Optimistic, MostLikely: a.MostLikely, Pessimistic: a.Pessimistic, Reasoning: a.Reasoning}
			}
		}

		w.SetEstimates(assignments)

		if w.EstimatesApproved() {
			t.Fatalf("iter %d: SetEstimates must leave estimates unapproved", iter)
		}
		if !w.Approved() {
			t.Fatalf("iter %d: SetEstimates must not change WBS approval", iter)
		}
		for tnum := 1; tnum <= n; tnum++ {
			task, _ := w.TaskAt(tnum)
			want := expected[tnum-1]
			if want == nil {
				if task.Estimate != nil {
					t.Fatalf("iter %d: task %d has estimate %+v, want none", iter, tnum, *task.Estimate)
				}
				continue
			}
			if task.Estimate == nil || *task.Estimate != *want {
				t.Fatalf("iter %d: task %d estimate = %v, want %+v (last in-range wins)", iter, tnum, task.Estimate, *want)
			}
		}
	}
}

// OverrideEstimate validation, isolation, and state invariants over a random
// candidate on an approved estimate set. A candidate is accepted iff all three
// values are on the Fibonacci scale, strictly increasing, and the reasoning is
// non-blank; the first rule broken selects the error (off-scale, then
// not-increasing, then empty reasoning). On success only the target task's
// estimate changes — to the candidate with trimmed reasoning — and the set drops
// to unapproved; on any rejection no estimate changes and approval is preserved.
func TestPropOverrideEstimateValidationInvariants(t *testing.T) {
	rng := rand.New(rand.NewSource(propertySeed))
	onScale := []int{0, 1, 2, 3, 5, 8, 13, 20, 40, 100}
	offScale := []int{-5, -1, 4, 6, 7, 9, 21, 34, 50, 99}
	inScale := func(v int) bool { return slices.Contains(onScale, v) }
	pool := append(append([]int{}, onScale...), offScale...)
	reasonings := []string{"solid reasoning", "", "   ", "\t\n"}
	baseline := Estimate{Optimistic: 2, MostLikely: 5, Pessimistic: 13, Reasoning: "baseline"}

	for iter := 0; iter < 1000; iter++ {
		n := 1 + rng.Intn(5)
		descs := make([]string, n)
		base := make([]EstimateAssignment, n)
		for i := range descs {
			descs[i] = "task-" + strconv.Itoa(i+1)
			base[i] = EstimateAssignment{TaskNumber: i + 1, Optimistic: baseline.Optimistic, MostLikely: baseline.MostLikely, Pessimistic: baseline.Pessimistic, Reasoning: baseline.Reasoning}
		}
		w := NewWBS("w", descs)
		if err := w.Approve(); err != nil {
			t.Fatalf("iter %d: Approve failed: %v", iter, err)
		}
		w.SetEstimates(base)
		if err := w.ApproveEstimates(); err != nil {
			t.Fatalf("iter %d: ApproveEstimates failed: %v", iter, err)
		}

		target := 1 + rng.Intn(n)
		cand := Estimate{
			Optimistic:  pool[rng.Intn(len(pool))],
			MostLikely:  pool[rng.Intn(len(pool))],
			Pessimistic: pool[rng.Intn(len(pool))],
			Reasoning:   reasonings[rng.Intn(len(reasonings))],
		}

		// Independent oracle mirroring the documented rule priority.
		var wantErr error
		switch {
		case !inScale(cand.Optimistic) || !inScale(cand.MostLikely) || !inScale(cand.Pessimistic):
			wantErr = ErrOffFibonacciScale
		case !(cand.Optimistic < cand.MostLikely && cand.MostLikely < cand.Pessimistic):
			wantErr = ErrEstimatesNotIncreasing
		case strings.TrimSpace(cand.Reasoning) == "":
			wantErr = ErrEmptyReasoning
		}

		err := w.OverrideEstimate(target, cand)

		if wantErr != nil {
			if !errors.Is(err, wantErr) {
				t.Fatalf("iter %d: override %+v error = %v, want %v", iter, cand, err, wantErr)
			}
			if w.EstimatesApproved() != true {
				t.Fatalf("iter %d: a rejected override must preserve approval", iter)
			}
			for tnum := 1; tnum <= n; tnum++ {
				if got := estimateAt(w, tnum); got != baseline {
					t.Fatalf("iter %d: rejected override changed task %d to %+v", iter, tnum, got)
				}
			}
			continue
		}

		if err != nil {
			t.Fatalf("iter %d: override %+v unexpectedly failed: %v", iter, cand, err)
		}
		if w.EstimatesApproved() {
			t.Fatalf("iter %d: an accepted override must unapprove the set", iter)
		}
		wantStored := cand
		wantStored.Reasoning = strings.TrimSpace(cand.Reasoning)
		for tnum := 1; tnum <= n; tnum++ {
			got := estimateAt(w, tnum)
			want := baseline
			if tnum == target {
				want = wantStored
			}
			if got != want {
				t.Fatalf("iter %d: task %d estimate = %+v, want %+v", iter, tnum, got, want)
			}
		}
	}
}

// estimateAt returns the estimate stored on the one-based task, failing the test
// if the task or its estimate is absent.
func estimateAt(w *WBS, taskNumber int) Estimate {
	task, ok := w.TaskAt(taskNumber)
	if !ok || task.Estimate == nil {
		panic("estimateAt: task or estimate missing")
	}
	return *task.Estimate
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

// fibScale mirrors the planning-poker deck the domain accepts; the metrics
// property tests draw strictly-increasing estimates from it.
var fibScale = []int{0, 1, 2, 3, 5, 8, 13, 20, 40, 100}

// randValidEstimate draws a strictly-increasing on-scale estimate (O < M < P) by
// taking three distinct scale positions in ascending order.
func randValidEstimate(rng *rand.Rand) Estimate {
	idx := rng.Perm(len(fibScale))[:3]
	slices.Sort(idx)
	return Estimate{Optimistic: fibScale[idx[0]], MostLikely: fibScale[idx[1]], Pessimistic: fibScale[idx[2]], Reasoning: "r"}
}

// Per-task metric bounds: for any strictly-increasing estimate the expected
// value lies within [O, P] (PERT expected is a weighted mean of the three
// points), and neither the standard deviation nor the relative standard
// deviation is ever negative. These hold whatever the exact rounding, so they
// guard the shape of Metrics independently of the formula.
func TestPropEstimateMetricsBounds(t *testing.T) {
	rng := rand.New(rand.NewSource(propertySeed))
	for iter := 0; iter < 1000; iter++ {
		e := randValidEstimate(rng)
		m := e.Metrics()
		if m.Expected < e.Optimistic || m.Expected > e.Pessimistic {
			t.Fatalf("iter %d: expected %d out of [%d,%d] for %+v", iter, m.Expected, e.Optimistic, e.Pessimistic, e)
		}
		if m.StandardDeviation < 0 || m.RelativeStandardDeviation < 0 {
			t.Fatalf("iter %d: negative metric %+v for %+v", iter, m, e)
		}
	}
}

// Standard deviation depends only on the spread: PERT SD = (P - O)/6, so two
// valid estimates with the same P - O round to the same StandardDeviation
// however their most-likely values differ. This pins the SD relationship
// without restating its rounding, catching any change that pulls M into SD.
func TestPropEstimateSDDependsOnlyOnSpread(t *testing.T) {
	rng := rand.New(rand.NewSource(propertySeed))
	for iter := 0; iter < 1000; iter++ {
		a := randValidEstimate(rng)
		// A second valid estimate with the same O and P but M moved to whatever
		// scale value sits strictly between them (a itself proves one exists).
		b := Estimate{Optimistic: a.Optimistic, MostLikely: a.MostLikely, Pessimistic: a.Pessimistic, Reasoning: "r2"}
		for _, v := range fibScale {
			if v > a.Optimistic && v < a.Pessimistic {
				b.MostLikely = v
			}
		}
		if got, want := b.Metrics().StandardDeviation, a.Metrics().StandardDeviation; got != want {
			t.Fatalf("iter %d: SD changed with M only: %+v gave %d, %+v gave %d", iter, b, got, a, want)
		}
	}
}

// Half-up sixth rounding: roundSixthsHalfUp(n) rounds n/6 to the nearest whole
// number with exact ties (n = 6q + 3) going up. An independent quotient/remainder
// oracle checks every remainder class exhaustively, so the exact-tie behavior the
// integer arithmetic exists to protect is pinned directly.
func TestPropRoundSixthsHalfUp(t *testing.T) {
	for n := 0; n <= 6000; n++ {
		q, r := n/6, n%6
		want := q
		if 2*r >= 6 { // r >= 3: the tie (r == 3) and everything above rounds up
			want = q + 1
		}
		if got := roundSixthsHalfUp(n); got != want {
			t.Fatalf("roundSixthsHalfUp(%d) = %d, want %d", n, got, want)
		}
	}
}

// Project rollup invariants over a random estimated subset: ok is reported iff
// at least one task carries an estimate; the rolled-up expected value is the
// half-up rounding of the summed sixths over exactly the estimated tasks; the
// rolled-up standard deviation is the half-up rounding of sqrt of the summed
// per-task variances (variances add, not standard deviations); and a lone
// estimated task rolls up to its own metrics.
func TestPropProjectMetricsRollup(t *testing.T) {
	rng := rand.New(rand.NewSource(propertySeed))
	for iter := 0; iter < 1000; iter++ {
		n := 1 + rng.Intn(6)
		descs := make([]string, n)
		for i := range descs {
			descs[i] = "task-" + strconv.Itoa(i+1)
		}
		w := NewWBS("w", descs)
		if err := w.Approve(); err != nil {
			t.Fatalf("iter %d: Approve failed: %v", iter, err)
		}

		var assignments []EstimateAssignment
		var estimated []Estimate
		for tnum := 1; tnum <= n; tnum++ {
			if rng.Intn(2) == 0 {
				continue // leave this task unestimated so rollup filtering is exercised
			}
			e := randValidEstimate(rng)
			assignments = append(assignments, EstimateAssignment{TaskNumber: tnum, Optimistic: e.Optimistic, MostLikely: e.MostLikely, Pessimistic: e.Pessimistic, Reasoning: e.Reasoning})
			estimated = append(estimated, e)
		}
		w.SetEstimates(assignments)

		got, ok := w.ProjectMetrics()
		if ok != (len(estimated) > 0) {
			t.Fatalf("iter %d: ok = %v, want %v (%d estimated)", iter, ok, len(estimated) > 0, len(estimated))
		}
		if !ok {
			continue
		}

		sumSixths := 0
		var sumVariance float64
		for _, e := range estimated {
			v := e.pert()
			sumSixths += v.expectedSixths
			sumVariance += v.stdDev * v.stdDev
		}
		if want := roundSixthsHalfUp(sumSixths); got.Expected != want {
			t.Fatalf("iter %d: rollup expected = %d, want %d", iter, got.Expected, want)
		}
		if want := roundHalfUp(math.Sqrt(sumVariance)); got.StandardDeviation != want {
			t.Fatalf("iter %d: rollup SD = %d, want %d (variances add)", iter, got.StandardDeviation, want)
		}
		if len(estimated) == 1 && got != estimated[0].Metrics() {
			t.Fatalf("iter %d: single-task rollup = %+v, want its own metrics %+v", iter, got, estimated[0].Metrics())
		}
	}
}

// pricingOracle restates the risk-band spec independently of the domain's band
// table, keyed off the whole-number project RSD. atLow is the success percentage
// at the low end of the proposal range (the normal-curve probability at the
// band's low SD multiplier: 50% at the mean for green, 84% at mean+1SD for
// yellow and red).
type pricingOracle struct {
	flag, riskLevel, contract, confidence string
	invites                               bool
	hasFixedBasis                         bool
	atLow                                 int
}

// expectBand is the independent oracle for the risk band of a project RSD: green
// below 10, yellow through 20 inclusive, red above.
func expectBand(rsd int) pricingOracle {
	switch {
	case rsd < 10:
		return pricingOracle{"green", "low", "fixed-price", "high", false, true, 50}
	case rsd <= 20:
		return pricingOracle{"yellow", "medium", "fixed-price-with-buffer", "medium", false, true, 84}
	default:
		return pricingOracle{"red", "high", "time-and-materials", "lower", true, false, 84}
	}
}

// randEstimatedWBS builds an approved WBS of n tasks with a valid estimate on
// every task, generated but not yet approved.
func randEstimatedWBS(t *testing.T, rng *rand.Rand, n int) *WBS {
	t.Helper()
	descs := make([]string, n)
	for i := range descs {
		descs[i] = "task-" + strconv.Itoa(i+1)
	}
	w := NewWBS("w", descs)
	if err := w.Approve(); err != nil {
		t.Fatalf("Approve failed: %v", err)
	}
	assignments := make([]EstimateAssignment, 0, n)
	for tnum := 1; tnum <= n; tnum++ {
		e := randValidEstimate(rng)
		assignments = append(assignments, EstimateAssignment{TaskNumber: tnum, Optimistic: e.Optimistic, MostLikely: e.MostLikely, Pessimistic: e.Pessimistic, Reasoning: e.Reasoning})
	}
	w.SetEstimates(assignments)
	return w
}

// Risk-band classification boundaries: bandForRSD maps a whole-number project RSD
// to exactly one band, with the cutoffs green < 10 <= yellow <= 20 < red and a
// fixed pricing basis for every band but red. Checked exhaustively against an
// independent oracle so a shifted edge (< vs <=) is caught at the exact 9/10/20/21
// values random projects rarely hit.
func TestPropBandForRSDBoundaries(t *testing.T) {
	for rsd := 0; rsd <= 1000; rsd++ {
		o := expectBand(rsd)
		b := bandForRSD(rsd)
		if b.flag != o.flag || b.riskLevel != o.riskLevel || b.contract != o.contract || b.confidence != o.confidence || b.invitesRenegotiation != o.invites || b.hasFixedBasis != o.hasFixedBasis {
			t.Fatalf("bandForRSD(%d) = %+v, want flag=%s risk=%s contract=%s confidence=%s invites=%v fixedBasis=%v", rsd, b, o.flag, o.riskLevel, o.contract, o.confidence, o.invites, o.hasFixedBasis)
		}
	}
}

// PricingStrategy reflects the project RSD: it is reported iff at least one task
// is estimated; its flag, risk level, and contract match the band of the project
// RSD; and a recommended basis is present for every band but red, where a fixed
// price is refused. The strategy needs no approval.
func TestPropPricingStrategyReflectsProjectRSD(t *testing.T) {
	rng := rand.New(rand.NewSource(propertySeed))
	for iter := 0; iter < 1000; iter++ {
		n := 1 + rng.Intn(6)
		descs := make([]string, n)
		for i := range descs {
			descs[i] = "task-" + strconv.Itoa(i+1)
		}
		w := NewWBS("w", descs)
		if err := w.Approve(); err != nil {
			t.Fatalf("iter %d: Approve failed: %v", iter, err)
		}
		var assignments []EstimateAssignment
		estimatedCount := 0
		for tnum := 1; tnum <= n; tnum++ {
			if rng.Intn(2) == 0 {
				continue // leave this task unestimated
			}
			e := randValidEstimate(rng)
			assignments = append(assignments, EstimateAssignment{TaskNumber: tnum, Optimistic: e.Optimistic, MostLikely: e.MostLikely, Pessimistic: e.Pessimistic, Reasoning: e.Reasoning})
			estimatedCount++
		}
		w.SetEstimates(assignments)

		ps, ok := w.PricingStrategy()
		if ok != (estimatedCount > 0) {
			t.Fatalf("iter %d: pricing ok = %v, want %v", iter, ok, estimatedCount > 0)
		}
		if !ok {
			continue
		}
		m, _ := w.ProjectMetrics()
		o := expectBand(m.RelativeStandardDeviation)
		if ps.Flag != o.flag || ps.RiskLevel != o.riskLevel || ps.Contract != o.contract {
			t.Fatalf("iter %d: strategy %+v, want band %+v for rsd %d", iter, ps, o, m.RelativeStandardDeviation)
		}
		if (ps.RecommendedBasis == nil) != (o.flag == "red") {
			t.Fatalf("iter %d: basis-absent=%v for flag %s (only red refuses a fixed basis)", iter, ps.RecommendedBasis == nil, o.flag)
		}
	}
}

// Proposal guards apply in priority order: the estimate set must be approved
// before any proposal (else ErrEstimatesNotApprovedForProposal, whatever the
// inputs), and only then are the team inputs required to be positive (else
// ErrNonPositiveTeamInputs). An approved set with all-positive inputs succeeds.
func TestPropProposalGuards(t *testing.T) {
	rng := rand.New(rand.NewSource(propertySeed))
	inputPool := []int{-3, -1, 0, 1, 2, 5, 40}
	for iter := 0; iter < 1000; iter++ {
		w := randEstimatedWBS(t, rng, 1+rng.Intn(4))
		approve := rng.Intn(2) == 0
		if approve {
			if err := w.ApproveEstimates(); err != nil {
				t.Fatalf("iter %d: ApproveEstimates failed: %v", iter, err)
			}
		}
		inputs := TeamInputs{
			Velocity: inputPool[rng.Intn(len(inputPool))],
			Capacity: inputPool[rng.Intn(len(inputPool))],
			Rate:     inputPool[rng.Intn(len(inputPool))],
		}

		var wantErr error
		switch {
		case !approve:
			wantErr = ErrEstimatesNotApprovedForProposal
		case inputs.Velocity <= 0 || inputs.Capacity <= 0 || inputs.Rate <= 0:
			wantErr = ErrNonPositiveTeamInputs
		}

		_, err := w.Proposal(inputs)
		if wantErr != nil {
			if !errors.Is(err, wantErr) {
				t.Fatalf("iter %d: proposal(approved=%v, %+v) err = %v, want %v", iter, approve, inputs, err, wantErr)
			}
			continue
		}
		if err != nil {
			t.Fatalf("iter %d: proposal(approved=%v, %+v) unexpected err %v", iter, approve, inputs, err)
		}
	}
}

// Proposal structure and monotonicity on a successful build: the scope lists one
// item per task in order; the assumptions concatenate every risk note in task
// then note order; the cost and timeline ranges are ordered low <= high; the
// success probabilities are the documented normal-curve figures (98% at
// mean+2SD, and 50% at the mean for green or 84% at mean+1SD otherwise) with
// atLow <= atHigh; and the confidence, contract, and renegotiation flag follow
// the band.
func TestPropProposalStructure(t *testing.T) {
	rng := rand.New(rand.NewSource(propertySeed))
	for iter := 0; iter < 1000; iter++ {
		n := 1 + rng.Intn(5)
		descs := make([]string, n)
		for i := range descs {
			descs[i] = "task-" + strconv.Itoa(i+1)
		}
		w := NewWBS("w", descs)
		if err := w.Approve(); err != nil {
			t.Fatalf("iter %d: Approve failed: %v", iter, err)
		}
		// Seed a random number of risk notes per task so assumptions aggregation
		// is exercised in task then note order.
		var seed []RiskAssignment
		var wantAssumptions []string
		for tnum := 1; tnum <= n; tnum++ {
			for j := 0; j < rng.Intn(3); j++ {
				d := "risk-" + strconv.Itoa(tnum) + "-" + strconv.Itoa(j)
				seed = append(seed, RiskAssignment{TaskNumber: tnum, Description: d})
				wantAssumptions = append(wantAssumptions, d)
			}
		}
		w.ReplaceRiskNotes(seed)

		assignments := make([]EstimateAssignment, 0, n)
		for tnum := 1; tnum <= n; tnum++ {
			e := randValidEstimate(rng)
			assignments = append(assignments, EstimateAssignment{TaskNumber: tnum, Optimistic: e.Optimistic, MostLikely: e.MostLikely, Pessimistic: e.Pessimistic, Reasoning: e.Reasoning})
		}
		w.SetEstimates(assignments)
		if err := w.ApproveEstimates(); err != nil {
			t.Fatalf("iter %d: ApproveEstimates failed: %v", iter, err)
		}

		inputs := TeamInputs{Velocity: 1 + rng.Intn(10), Capacity: 1 + rng.Intn(40), Rate: 1 + rng.Intn(200)}
		p, err := w.Proposal(inputs)
		if err != nil {
			t.Fatalf("iter %d: Proposal failed: %v", iter, err)
		}

		tasks := w.Tasks()
		if len(p.Scope) != len(tasks) {
			t.Fatalf("iter %d: scope len %d, want %d", iter, len(p.Scope), len(tasks))
		}
		for i, s := range p.Scope {
			if s.ID != tasks[i].ID || s.Description != tasks[i].Description {
				t.Fatalf("iter %d: scope[%d] = %+v, want id/desc of %+v", iter, i, s, tasks[i])
			}
		}
		if len(wantAssumptions) == 0 {
			if len(p.AssumptionsAndExclusions) != 0 {
				t.Fatalf("iter %d: assumptions = %v, want none", iter, p.AssumptionsAndExclusions)
			}
		} else if !slices.Equal(p.AssumptionsAndExclusions, wantAssumptions) {
			t.Fatalf("iter %d: assumptions = %v, want %v", iter, p.AssumptionsAndExclusions, wantAssumptions)
		}
		if p.Cost.Low > p.Cost.High {
			t.Fatalf("iter %d: cost range inverted %+v", iter, p.Cost)
		}
		if p.TimelineWeeks.Low > p.TimelineWeeks.High {
			t.Fatalf("iter %d: timeline range inverted %+v", iter, p.TimelineWeeks)
		}
		m, _ := w.ProjectMetrics()
		o := expectBand(m.RelativeStandardDeviation)
		if p.SuccessProbability.AtHigh != 98 {
			t.Fatalf("iter %d: atHigh = %d, want 98", iter, p.SuccessProbability.AtHigh)
		}
		if p.SuccessProbability.AtLow != o.atLow {
			t.Fatalf("iter %d: atLow = %d, want %d for %s", iter, p.SuccessProbability.AtLow, o.atLow, o.flag)
		}
		if p.SuccessProbability.AtLow > p.SuccessProbability.AtHigh {
			t.Fatalf("iter %d: atLow %d > atHigh %d", iter, p.SuccessProbability.AtLow, p.SuccessProbability.AtHigh)
		}
		if p.Confidence != o.confidence || p.Contract != o.contract || p.InvitesRenegotiation != o.invites {
			t.Fatalf("iter %d: band labels {%s,%s,%v}, want {%s,%s,%v} for rsd %d", iter, p.Confidence, p.Contract, p.InvitesRenegotiation, o.confidence, o.contract, o.invites, m.RelativeStandardDeviation)
		}
	}
}
