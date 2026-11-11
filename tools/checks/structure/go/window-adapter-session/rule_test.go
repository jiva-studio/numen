package windowadaptersession

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func getWaitCalls() map[string]map[string]bool {
	return map[string]map[string]bool{
		"context": {"WithTimeout": true, "WithDeadline": true, "WithTimeoutCause": true, "WithDeadlineCause": true},
		"time":    {"After": true, "Sleep": true, "NewTimer": true, "Tick": true},
	}
}

func getWaitFindings(t *testing.T, name string, src any) []string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), name, src, 0)
	if err != nil {
		t.Fatal(err)
	}
	var found []string
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if pkg, ok := selector.X.(*ast.Ident); ok && getWaitCalls()[pkg.Name][selector.Sel.Name] {
			found = append(found, pkg.Name+"."+selector.Sel.Name)
		}
		return true
	})
	return found
}

func getLockImports(t *testing.T, name string, src any) []string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), name, src, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	var found []string
	for _, spec := range file.Imports {
		if path, err := strconv.Unquote(spec.Path.Value); err == nil && path == "sync" {
			found = append(found, path)
		}
	}
	return found
}

func walkWindowAdapters(t *testing.T, visit func(pkg, path string)) (files, packages int) {
	t.Helper()
	packageSet := map[string]bool{}
	err := filepath.WalkDir("../adapter/window", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		pkg := load.GetPackage("..", path)
		packageSet[pkg] = true
		files++
		visit(pkg, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files, len(packageSet)
}
