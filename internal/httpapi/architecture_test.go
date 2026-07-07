package httpapi

import (
	"testing"

	"estimation/internal/archtest"
)

// TestHTTPAdapterDependsInward enforces the dependency rule for the HTTP
// delivery adapter: it is a low-level module near IO, so within the project it
// may depend only on the high-level core (internal/wbs). It must not import the
// entry point (cmd/...) or any sibling adapter, which would either invert the
// dependency direction or couple adapters to each other. A future edit that
// reaches sideways or upward fails here instead of silently eroding the layering.
func TestHTTPAdapterDependsInward(t *testing.T) {
	archtest.AssertDependsOnlyOnCore(t, "HTTP adapter", "estimation/internal/wbs")
}
