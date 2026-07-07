// Command acceptance-mutation-runner is the project-specific runner adapter for
// the APS gherkin-mutator. It runs as a persistent worker: the mutator starts it
// once and streams newline-delimited JSON mutation jobs on stdin, one response
// per line on stdout.
//
// For each job the adapter reads the mutated feature IR named by feature_json,
// runs it through the real acceptance runtime and project step handlers, and
// reports whether the generated tests passed (survived) or failed (killed). The
// generated *_gen.go entry points only register a baked-in IR; the actual test
// logic lives in the generic runtime and the project steps, so evaluating the
// supplied IR directly is equivalent to re-running the generated tests against
// it.
//
// This command is a thin, environmentally-unsuitable process shell; the testable
// behavior lives in estimation/acceptance/mutationrunner.
package main

import (
	"fmt"
	"os"

	"estimation/acceptance/mutationrunner"
)

func main() {
	if err := mutationrunner.Serve(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "acceptance-mutation-runner: %v\n", err)
		os.Exit(1)
	}
}
