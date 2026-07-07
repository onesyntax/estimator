package aiprovider

import (
	"testing"

	"estimation/internal/archtest"
)

// TestAIProviderDependsInward enforces the dependency rule for the AI provider
// adapter: it is a low-level, IO-near module (it calls an external AI service),
// so within the project it may depend only on the high-level core (internal/wbs)
// whose Provider port it implements. It must not import the entry point (cmd/...)
// or any sibling adapter such as internal/httpapi. Third-party imports (the
// Anthropic SDK) are outside the project prefix and are unconstrained. A future
// edit that reaches sideways or upward fails here instead of silently eroding
// the layering.
func TestAIProviderDependsInward(t *testing.T) {
	archtest.AssertDependsOnlyOnCore(t, "AI provider adapter", "estimation/internal/wbs")
}
