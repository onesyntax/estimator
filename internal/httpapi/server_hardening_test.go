//go:build hardening

// Hardening tests for the HTTP adapter, behind the `hardening` build tag so they
// stay out of the unit suite, coverage, CRAP, and DRY runs.
package httpapi

import (
	"net/http"
	"testing"
)

// Covers resolveTaskNumber's WBS-not-found path — editing a task on an unknown
// WBS — which the unit suite leaves unexercised, and pins the mapping to a 404
// "WBS not found" rather than "task not found".
func TestHardeningEditTaskOnUnknownWBSReportsWBSNotFound(t *testing.T) {
	c := newClient(t, true)
	status, body := c.json(http.MethodPut, "/wbs/does-not-exist/tasks/t1", map[string]any{"description": "X"})
	if status != http.StatusNotFound || body["error"] != "WBS not found" {
		t.Fatalf("edit task on unknown WBS = %d %v, want 404 WBS not found", status, body)
	}
}

// Covers the same path for deletion.
func TestHardeningDeleteTaskOnUnknownWBSReportsWBSNotFound(t *testing.T) {
	c := newClient(t, true)
	status, body := c.do(http.MethodDelete, "/wbs/does-not-exist/tasks/t1", "", nil)
	if status != http.StatusNotFound || body["error"] != "WBS not found" {
		t.Fatalf("delete task on unknown WBS = %d %v, want 404 WBS not found", status, body)
	}
}
