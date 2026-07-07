package aiprovider

import (
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

// projectPrefix marks intra-project imports; core is the single project package
// this adapter is allowed to depend on. Third-party imports (the Anthropic SDK)
// are outside this prefix and are intentionally unconstrained.
const (
	projectPrefix = "estimation/"
	corePackage   = "estimation/internal/wbs"
)

// TestAIProviderDependsInward enforces the dependency rule for the AI provider
// adapter: it is a low-level, IO-near module (it calls an external AI service),
// so within the project it may depend only on the high-level core (internal/wbs)
// whose Provider port it implements. It must not import the entry point (cmd/...)
// or any sibling adapter such as internal/httpapi. A future edit that reaches
// sideways or upward fails here instead of silently eroding the layering.
func TestAIProviderDependsInward(t *testing.T) {
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
				t.Errorf("adapter file %s imports project package %q; the AI provider adapter may depend only on the core %q", name, path, corePackage)
			}
		}
	}

	if checked == 0 {
		t.Fatal("no production Go files found to check; the architecture guard would pass vacuously")
	}
}
