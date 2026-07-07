//go:build property

// Property tests for the AI provider's pure response-parsing logic. Kept out of
// the normal unit suite (and coverage/mutation/CRAP) by the `property` build
// tag; run them with scripts/property.sh. They assert that parseTasks is a
// faithful, order-preserving projection of the model's tool payload over broad
// inputs: non-blank tasks are conserved in order, surrounding whitespace is
// trimmed, blank entries are dropped, and a payload with nothing usable yields
// ErrNoTasks.
package aiprovider

import (
	"encoding/json"
	"errors"
	"math/rand"
	"strconv"
	"strings"
	"testing"
)

const propertySeed = 20260707

// buildRaw marshals the task list into the {"tasks": [...]} shape the model
// returns, so the property exercises the real decode path rather than a
// hand-built string.
func buildRaw(t *testing.T, tasks []string) string {
	t.Helper()
	raw, err := json.Marshal(struct {
		Tasks []string `json:"tasks"`
	}{Tasks: tasks})
	if err != nil {
		t.Fatalf("marshal tasks: %v", err)
	}
	return string(raw)
}

// TestPropParseTasksFaithful checks that parseTasks trims and drops blanks while
// preserving the order and content of every usable task, and reports ErrNoTasks
// exactly when nothing usable remains.
func TestPropParseTasksFaithful(t *testing.T) {
	rng := rand.New(rand.NewSource(propertySeed))

	// Entry generators spanning every branch: normal tokens, tokens padded with
	// whitespace (must be trimmed), and blank entries (must be dropped).
	kinds := []func(i int) string{
		func(i int) string { return "task-" + strconv.Itoa(i) },
		func(i int) string { return "  task-" + strconv.Itoa(i) + "\t" },
		func(i int) string { return "" },
		func(i int) string { return "   " },
		func(i int) string { return "\t\n " },
	}

	for iter := 0; iter < 1000; iter++ {
		n := rng.Intn(7) // 0..6 entries; 0 exercises the empty payload
		in := make([]string, n)
		want := make([]string, 0, n)
		for i := range in {
			in[i] = kinds[rng.Intn(len(kinds))](iter*10 + i)
			if trimmed := strings.TrimSpace(in[i]); trimmed != "" {
				want = append(want, trimmed)
			}
		}

		got, err := parseTasks(buildRaw(t, in))

		if len(want) == 0 {
			if !errors.Is(err, ErrNoTasks) {
				t.Fatalf("iter %d: input %q -> err %v, want ErrNoTasks", iter, in, err)
			}
			if got != nil {
				t.Fatalf("iter %d: input %q -> tasks %q, want nil", iter, in, got)
			}
			continue
		}
		if err != nil {
			t.Fatalf("iter %d: input %q -> unexpected err %v", iter, in, err)
		}
		if len(got) != len(want) {
			t.Fatalf("iter %d: input %q -> %d tasks, want %d", iter, in, len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("iter %d: task[%d] = %q, want %q (input %q)", iter, i, got[i], want[i], in)
			}
		}
	}
}
