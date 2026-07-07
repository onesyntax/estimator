// Command acceptance-entrypoint-generator turns parser JSON IR into a Go
// acceptance entry point. Each generated file embeds the JSON IR and registers
// it with the acceptance runtime. It writes thin entry points only; step
// handlers and the runtime are hand-written elsewhere.
//
// Usage:
//
//	acceptance-entrypoint-generator <json-ir> <generated-test-output-dir>
//
// The optional environment variable ACCEPTANCE_FEATURE_PATH records the source
// feature path in the generated metadata.
//
// This is a thin shell; the generation behavior lives in the testable
// estimation/acceptance/generator package.
package main

import (
	"os"

	"estimation/acceptance/generator"
)

func main() {
	os.Exit(generator.Run(os.Args[1:], os.Getenv, os.Stderr))
}
