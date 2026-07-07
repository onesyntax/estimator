package httpapi

import (
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// projectPrefix marks intra-project imports; core is the single project package
// this adapter is allowed to depend on.
const (
	projectPrefix = "estimation/"
	corePackage   = "estimation/internal/wbs"
)

// TestHTTPAdapterDependsInward enforces the dependency rule for the HTTP
// delivery adapter: it is a low-level module near IO, so within the project it
// may depend only on the high-level core (internal/wbs). It must not import the
// entry point (cmd/...) or any sibling adapter, which would either invert the
// dependency direction or couple adapters to each other. A future edit that
// reaches sideways or upward fails here instead of silently eroding the layering.
func TestHTTPAdapterDependsInward(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading package directory: %v", err)
	}

	fset := token.NewFileSet()
	checked := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
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
			if strings.HasPrefix(path, projectPrefix) && path != corePackage {
				t.Errorf("adapter file %s imports project package %q; the HTTP adapter may depend only on the core %q", name, path, corePackage)
			}
		}
	}

	if checked == 0 {
		t.Fatal("no production Go files found to check; the architecture guard would pass vacuously")
	}
}
