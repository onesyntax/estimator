// Package archtest provides a shared architecture guard for the project's
// adapter packages. Each adapter (internal/httpapi, internal/aiprovider) is a
// low-level, IO-near module that, within the project, may depend only on the
// high-level core domain (internal/wbs). The scanning mechanics are identical
// across adapters, so they live here once; each adapter's test still declares
// its own core package and label and thus asserts its own rule independently.
package archtest

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// projectPrefix marks intra-project imports; anything outside it (third-party
// SDKs, the standard library) is intentionally unconstrained.
const projectPrefix = "estimation/"

// scanForbiddenImports parses every production Go file (non-directory, .go,
// non-_test) in dir and returns, for each file importing a project package
// other than corePackage, a "<file> imports project package <path>" message,
// along with the number of production files inspected. Separating this pure
// scan from the *testing.T reporting keeps the layering logic directly testable.
func scanForbiddenImports(dir, corePackage string) (violations []string, checked int, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0, fmt.Errorf("reading package directory: %w", err)
	}

	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		checked++

		file, perr := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ImportsOnly)
		if perr != nil {
			return nil, 0, fmt.Errorf("parsing %s: %w", name, perr)
		}
		for _, imp := range file.Imports {
			path, uerr := strconv.Unquote(imp.Path.Value)
			if uerr != nil {
				return nil, 0, fmt.Errorf("%s: bad import literal %s", name, imp.Path.Value)
			}
			if strings.HasPrefix(path, projectPrefix) && path != corePackage {
				violations = append(violations, fmt.Sprintf("%s imports project package %q", name, path))
			}
		}
	}
	return violations, checked, nil
}

// AssertDependsOnlyOnCore fails t if any production Go file in the current
// package directory imports a project package other than corePackage. It must
// be called from a test in the adapter package being guarded, so that the
// test's working directory is that package's source directory. adapterLabel
// names the adapter in failure messages (e.g. "HTTP adapter"). The guard fails
// rather than passing vacuously if it finds no production files to check.
func AssertDependsOnlyOnCore(t *testing.T, adapterLabel, corePackage string) {
	t.Helper()

	violations, checked, err := scanForbiddenImports(".", corePackage)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range violations {
		t.Errorf("adapter file %s; the %s may depend only on the core %q", v, adapterLabel, corePackage)
	}
	if checked == 0 {
		t.Fatal("no production Go files found to check; the architecture guard would pass vacuously")
	}
}
