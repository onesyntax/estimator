// Command generated is the acceptance runner adapter. It runs every feature
// registered by the generated entry points against the project step handlers
// and exits non-zero when any scenario execution fails.
//
// The *_gen.go files in this package are produced by
// acceptance-entrypoint-generator and register the embedded JSON IR.
package main

import (
	"fmt"
	"os"

	"estimation/acceptance/runtime"
	"estimation/acceptance/steps"
)

func main() {
	features := runtime.Registered()
	if len(features) == 0 {
		fmt.Fprintln(os.Stderr, "no acceptance features registered; run the generator first")
		os.Exit(1)
	}
	reg := steps.NewRegistry()
	pass, fail := reg.Run(features, os.Stdout)
	fmt.Printf("\nacceptance: %d passed, %d failed\n", pass, fail)
	if fail > 0 {
		os.Exit(1)
	}
}
