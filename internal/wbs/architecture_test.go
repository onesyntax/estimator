package wbs

import (
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// modulePath is the Go module path; intra-project imports start with this.
const modulePath = "estimation/"

// TestCoreIsDependencyLeaf enforces the dependency rule for the core domain
// package: internal/wbs is high-level policy far from IO, so it must not depend
// on any other package in this project (adapters, acceptance pipeline, delivery
// layers). Dependencies point inward toward the core, never outward from it.
// A future edit that makes the core import a lower-level project package fails
// here instead of silently inverting the dependency direction.
func TestCoreIsDependencyLeaf(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading package directory: %v", err)
	}

	fset := token.NewFileSet()
	checked := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		if strings.HasSuffix(name, "_test.go") {
			continue // tests may reach for anything; only production code is constrained
		}
		checked++

		file, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parsing %s: %v", name, err)
		}
		for _, imp := range file.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatalf("%s: bad import literal %s", name, imp.Path.Value)
			}
			if strings.HasPrefix(path, modulePath) {
				t.Errorf("core package file %s imports project package %q; the domain core must not depend on other project packages", name, path)
			}
		}
	}

	if checked == 0 {
		t.Fatal("no production Go files found to check; the architecture guard would pass vacuously")
	}
}
