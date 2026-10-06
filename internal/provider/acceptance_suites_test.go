package provider

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The acceptance CI harness reads this file to pick the job that runs each
// TestAcc function.
const acceptanceSuitesPath = "testdata/acceptance-suites.json"

type acceptanceSuites struct {
	Suites   map[string][]string `json:"suites"`
	Excluded map[string]string   `json:"excluded"`
}

func TestSuiteListCoversEveryAcceptanceTest(t *testing.T) {
	body, err := os.ReadFile(acceptanceSuitesPath)
	if err != nil {
		t.Fatal(err)
	}
	var list acceptanceSuites
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("%s: %v", acceptanceSuitesPath, err)
	}

	listed := map[string][]string{}
	for suite, names := range list.Suites {
		for _, name := range names {
			listed[name] = append(listed[name], suite)
		}
	}
	for name, reason := range list.Excluded {
		listed[name] = append(listed[name], "excluded")
		if strings.TrimSpace(reason) == "" {
			t.Errorf("%s: %s is excluded with no reason", acceptanceSuitesPath, name)
		}
	}

	files, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatal(err)
	}
	accTests := map[string]bool{}
	fset := token.NewFileSet()
	for _, path := range files {
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !strings.HasPrefix(fn.Name.Name, "Test") {
				continue
			}
			if strings.HasPrefix(fn.Name.Name, "TestAcc") {
				accTests[fn.Name.Name] = true
			} else if callsResourceTest(fn) {
				t.Errorf("%s calls resource.Test or resource.ParallelTest without the TestAcc prefix, so no CI suite runs it", fn.Name.Name)
			}
		}
	}

	for _, name := range slices.Sorted(maps.Keys(accTests)) {
		if len(listed[name]) == 0 {
			t.Errorf("%s: %s is in no suite; add it to one, or to excluded with a reason", acceptanceSuitesPath, name)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(listed)) {
		if places := listed[name]; len(places) > 1 {
			t.Errorf("%s: %s is listed more than once: %s", acceptanceSuitesPath, name, strings.Join(places, ", "))
		}
		if !accTests[name] {
			t.Errorf("%s: %s names no TestAcc function", acceptanceSuitesPath, name)
		}
	}
}

func callsResourceTest(fn *ast.FuncDecl) bool {
	found := false
	ast.Inspect(fn, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return !found
		}
		if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "resource" && (sel.Sel.Name == "Test" || sel.Sel.Name == "ParallelTest") {
			found = true
		}
		return !found
	})
	return found
}
