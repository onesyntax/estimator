package webui

import (
	"testing"

	"estimation/internal/archtest"
)

// TestWebUIDependsInward enforces the dependency rule for the web UI delivery
// adapter: it is a low-level, IO-near module (HTTP handlers, html/template
// rendering), so within the project it may depend only on the high-level core
// (internal/wbs) whose service it drives. It must not import the entry point
// (cmd/...) or any sibling adapter such as internal/httpapi or
// internal/aiprovider. Standard-library imports (net/http, html/template) are
// outside the project prefix and are unconstrained. A future edit that reaches
// sideways or upward fails here instead of silently eroding the layering.
func TestWebUIDependsInward(t *testing.T) {
	archtest.AssertDependsOnlyOnCore(t, "web UI adapter", "estimation/internal/wbs")
}
