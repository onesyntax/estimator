package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeGo(t *testing.T, dir, name, src string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

// TestScanFlagsOnlyForbiddenProjectImports checks the full classification: the
// core package is allowed, a sibling project package is flagged, standard-library
// and third-party imports are ignored, and _test.go files are not inspected.
func TestScanFlagsOnlyForbiddenProjectImports(t *testing.T) {
	dir := t.TempDir()
	writeGo(t, dir, "ok.go", "package p\nimport _ \"estimation/internal/wbs\"\n")                     // core: allowed
	writeGo(t, dir, "bad.go", "package p\nimport _ \"estimation/internal/httpapi\"\n")                // sibling: forbidden
	writeGo(t, dir, "std.go", "package p\nimport _ \"strings\"\n")                                    // stdlib: ignored
	writeGo(t, dir, "ext.go", "package p\nimport _ \"github.com/anthropics/anthropic-sdk-go\"\n")     // third-party: ignored
	writeGo(t, dir, "guard_test.go", "package p\nimport _ \"estimation/internal/cmd/estimationd\"\n") // test file: skipped

	violations, checked, err := scanForbiddenImports(dir, "estimation/internal/wbs")
	if err != nil {
		t.Fatalf("scan error: %v", err)
	}
	if checked != 4 { // ok, bad, std, ext — guard_test.go is skipped
		t.Errorf("checked = %d, want 4", checked)
	}
	if len(violations) != 1 {
		t.Fatalf("violations = %v, want exactly one (bad.go)", violations)
	}
	if !strings.Contains(violations[0], "bad.go") || !strings.Contains(violations[0], "estimation/internal/httpapi") {
		t.Errorf("violation = %q, want it to name bad.go and the forbidden import", violations[0])
	}
}

// TestScanReportsNoProductionFiles covers the vacuous-guard signal: a directory
// with only a _test.go file yields zero inspected files.
func TestScanReportsNoProductionFiles(t *testing.T) {
	dir := t.TempDir()
	writeGo(t, dir, "only_test.go", "package p\n")

	violations, checked, err := scanForbiddenImports(dir, "estimation/internal/wbs")
	if err != nil {
		t.Fatalf("scan error: %v", err)
	}
	if checked != 0 {
		t.Errorf("checked = %d, want 0 (only a _test.go file present)", checked)
	}
	if len(violations) != 0 {
		t.Errorf("violations = %v, want none", violations)
	}
}
