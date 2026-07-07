package runtime

import (
	"fmt"
	"io"
	"regexp"
)

// HandlerFunc executes a step. It receives the current scenario execution's
// world/state object and the values captured by the handler's pattern, and
// returns a diagnostic error on failure.
type HandlerFunc func(world any, args []string) error

type stepDef struct {
	re *regexp.Regexp
	fn HandlerFunc
}

// Registry holds the project step handlers and a factory that produces a fresh
// world/state object for each scenario execution.
type Registry struct {
	factory func() any
	defs    []stepDef
}

// NewRegistry creates a registry whose executions each get a fresh world from
// factory.
func NewRegistry(factory func() any) *Registry {
	return &Registry{factory: factory}
}

// Step registers a handler for step text matching the anchored regular
// expression pattern. Captured groups are passed to the handler in order.
func (r *Registry) Step(pattern string, fn HandlerFunc) {
	r.defs = append(r.defs, stepDef{re: regexp.MustCompile(pattern), fn: fn})
}

var placeholderRe = regexp.MustCompile(`<([A-Za-z0-9_]+)>`)

// expand replaces every <name> placeholder in text with its value from the
// current example object. A placeholder with no example value is a failure.
func expand(text string, example map[string]string) (string, error) {
	var missing string
	out := placeholderRe.ReplaceAllStringFunc(text, func(m string) string {
		name := m[1 : len(m)-1]
		if v, ok := example[name]; ok {
			return v
		}
		missing = name
		return m
	})
	if missing != "" {
		return "", fmt.Errorf("missing example value for <%s>", missing)
	}
	return out, nil
}

// Run executes every scenario execution represented by the features, writing a
// PASS/FAIL line per execution, and returns the pass and fail counts.
func (r *Registry) Run(features []Feature, out io.Writer) (pass, fail int) {
	for _, f := range features {
		for _, sc := range f.Scenarios {
			executions := sc.Examples
			if len(executions) == 0 {
				executions = []map[string]string{{}}
			}
			for i, ex := range executions {
				name := fmt.Sprintf("%s :: %s/example_%d", f.Name, sc.Name, i+1)
				if err := r.runExecution(f, sc, ex); err != nil {
					fail++
					fmt.Fprintf(out, "FAIL %s: %v\n", name, err)
				} else {
					pass++
					fmt.Fprintf(out, "PASS %s\n", name)
				}
			}
		}
	}
	return pass, fail
}

func (r *Registry) runExecution(f Feature, sc Scenario, example map[string]string) error {
	world := r.factory()
	steps := make([]Step, 0, len(f.Background)+len(sc.Steps))
	steps = append(steps, f.Background...)
	steps = append(steps, sc.Steps...)
	for _, st := range steps {
		text, err := expand(st.Text, example)
		if err != nil {
			return fmt.Errorf("step %q: %w", st.Text, err)
		}
		if err := r.dispatch(world, text); err != nil {
			return fmt.Errorf("step %q: %w", text, err)
		}
	}
	return nil
}

func (r *Registry) dispatch(world any, text string) error {
	for _, d := range r.defs {
		if m := d.re.FindStringSubmatch(text); m != nil {
			return d.fn(world, m[1:])
		}
	}
	return fmt.Errorf("unsupported step: %q", text)
}

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-07T22:19:12+05:30","module_hash":"c681427eb38eda653414f2c09b3b2cd26f4504b63bb9f402d62a43d6ce2a741c","functions":[{"id":"func/NewRegistry","name":"NewRegistry","line":28,"end_line":30,"hash":"86e3f55e6e8bb98e4eed5ab1955d98deaf73f988bf3f44602f9872f17a2c0bd0"},{"id":"func/Registry.Step","name":"Registry.Step","line":34,"end_line":36,"hash":"788d41a5a3ab489510e2c59fad041bfc39be78475575540177e98e39f0b32580"},{"id":"func/expand","name":"expand","line":42,"end_line":56,"hash":"a31af1725d2ee49c13af85936ebacb618673628c774c8f32115949b34335d2cf"},{"id":"func/Registry.Run","name":"Registry.Run","line":60,"end_line":80,"hash":"171dfee61aa8d13c7cc68dc930970af76389ab99701769e16b459386c97ede63"},{"id":"func/Registry.runExecution","name":"Registry.runExecution","line":82,"end_line":97,"hash":"b657603514fcc14a5c5a74c2ec9b2134af3d6ffd20f1ac79c2ff07d69402fdac"},{"id":"func/Registry.dispatch","name":"Registry.dispatch","line":99,"end_line":106,"hash":"5dadc474b481212d79993b04bb5be08b0028765499b6a1b41ac903ddaad3336a"}]}
// mutate4go-manifest-end
