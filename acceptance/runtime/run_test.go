package runtime

import (
	"io"
	"strings"
	"testing"
)

type recorder struct{ steps []string }

const sampleIR = `{
  "name": "Sample",
  "background": [ { "keyword": "Given", "text": "a fresh counter" } ],
  "scenarios": [
    {
      "name": "adds",
      "steps": [
        { "keyword": "When", "text": "add <n>", "parameters": ["n"] },
        { "keyword": "Then", "text": "the total is <total>", "parameters": ["total"] }
      ],
      "examples": [ { "n": "2", "total": "2" }, { "n": "5", "total": "5" } ]
    }
  ]
}`

func newSampleRegistry(worlds *[]*recorder) *Registry {
	reg := NewRegistry(func() any {
		w := &recorder{}
		*worlds = append(*worlds, w)
		return w
	})
	reg.Step(`^a fresh counter$`, func(world any, _ []string) error {
		world.(*recorder).steps = append(world.(*recorder).steps, "fresh")
		return nil
	})
	reg.Step(`^add (\d+)$`, func(world any, args []string) error {
		world.(*recorder).steps = append(world.(*recorder).steps, "add:"+args[0])
		return nil
	})
	reg.Step(`^the total is (\d+)$`, func(world any, args []string) error {
		w := world.(*recorder)
		w.steps = append(w.steps, "total:"+args[0])
		return nil
	})
	return reg
}

func TestRunExpandsExamplesPrependsBackgroundAndIsolatesWorlds(t *testing.T) {
	f, err := ParseFeature([]byte(sampleIR))
	if err != nil {
		t.Fatalf("ParseFeature: %v", err)
	}
	var worlds []*recorder
	reg := newSampleRegistry(&worlds)

	pass, fail := reg.Run([]Feature{f}, io.Discard)
	if pass != 2 || fail != 0 {
		t.Fatalf("pass=%d fail=%d, want 2 pass 0 fail", pass, fail)
	}
	if len(worlds) != 2 {
		t.Fatalf("got %d worlds, want one fresh world per example", len(worlds))
	}
	if got := strings.Join(worlds[0].steps, ","); got != "fresh,add:2,total:2" {
		t.Fatalf("execution 1 steps = %q", got)
	}
	if got := strings.Join(worlds[1].steps, ","); got != "fresh,add:5,total:5" {
		t.Fatalf("execution 2 steps = %q", got)
	}
}

func TestRunFailsUnsupportedStep(t *testing.T) {
	f := Feature{
		Name: "F",
		Scenarios: []Scenario{{
			Name:  "s",
			Steps: []Step{{Keyword: "When", Text: "do something unknown"}},
		}},
	}
	reg := NewRegistry(func() any { return nil })
	var out strings.Builder
	pass, fail := reg.Run([]Feature{f}, &out)
	if pass != 0 || fail != 1 {
		t.Fatalf("pass=%d fail=%d, want 0 pass 1 fail", pass, fail)
	}
	if !strings.Contains(out.String(), "unsupported step") {
		t.Fatalf("output = %q, want unsupported step diagnostic", out.String())
	}
}

func TestRunFailsMissingExampleValue(t *testing.T) {
	f := Feature{
		Name: "F",
		Scenarios: []Scenario{{
			Name:     "s",
			Steps:    []Step{{Keyword: "Then", Text: "the total is <total>", Parameters: []string{"total"}}},
			Examples: []map[string]string{{}},
		}},
	}
	reg := NewRegistry(func() any { return nil })
	reg.Step(`^the total is (\d+)$`, func(any, []string) error { return nil })
	var out strings.Builder
	_, fail := reg.Run([]Feature{f}, &out)
	if fail != 1 {
		t.Fatalf("fail=%d, want 1", fail)
	}
	if !strings.Contains(out.String(), "missing example value") {
		t.Fatalf("output = %q, want missing example value diagnostic", out.String())
	}
}
