package container

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The vault a window shows, how a switch of it is ordered, how long a page is waited for and what is
// locked while that happens are decisions of the window session use case. A window adapter maps a
// request onto a use case; it holds no lock and sets no bound on a wait.

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
		pkg := getPackage("..", path)
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

func TestNoWindowAdapterBoundsAWait(t *testing.T) {
	var wrong []string
	files, packages := walkWindowAdapters(t, func(_, path string) {
		for _, call := range getWaitFindings(t, path, nil) {
			wrong = append(wrong, path+" calls "+call)
		}
	})
	for _, one := range wrong {
		t.Error(one + ": a bound on a wait is a decision of a use case")
	}

	if files == 0 || packages == 0 {
		t.Fatalf("%d files in %d packages read: the walk is not reading the window adapters", files, packages)
	}
}

func TestEditorWindowImportsNoSync(t *testing.T) {
	var wrong []string
	files, _ := walkWindowAdapters(t, func(pkg, path string) {
		if !isUnder(pkg, "adapter/window/editor") {
			return
		}
		for range getLockImports(t, path, nil) {
			wrong = append(wrong, path+" imports sync")
		}
	})
	for _, one := range wrong {
		t.Error(one + ": what is locked while a window switches vaults belongs to the window session use case")
	}
	if files == 0 {
		t.Fatalf("%d files of the editor window read: the walk is not reading it", files)
	}
}

func TestTheWaitRuleSeesABoundOnAWait(t *testing.T) {
	for _, tc := range []struct {
		name, src string
		want      int
	}{
		{"timeout", "package p\nimport (\"context\"; \"time\")\nfunc f(ctx context.Context) { context.WithTimeout(ctx, time.Second) }", 1},
		{"deadline", "package p\nimport (\"context\"; \"time\")\nfunc f(ctx context.Context) { context.WithDeadline(ctx, time.Now()) }", 1},
		{"after", "package p\nimport \"time\"\nfunc f() { <-time.After(time.Second) }", 1},
		{"sleep", "package p\nimport \"time\"\nfunc f() { time.Sleep(time.Second) }", 1},
		{"ticker of a stream", "package p\nimport \"time\"\nfunc f() { time.NewTicker(time.Second) }", 0},
		{"cancel without a bound", "package p\nimport \"context\"\nfunc f(ctx context.Context) { context.WithCancel(ctx) }", 0},
	} {
		if got := getWaitFindings(t, tc.name, tc.src); len(got) != tc.want {
			t.Errorf("%s: %d calls found, want %d", tc.name, len(got), tc.want)
		}
	}
}

func TestTheLockRuleSeesTheSyncImport(t *testing.T) {
	if got := getLockImports(t, "a.go", "package p\nimport \"sync\"\nvar mu sync.Mutex"); len(got) != 1 {
		t.Errorf("%d imports of sync found in a file that imports it, want 1", len(got))
	}
	if got := getLockImports(t, "b.go", "package p\nimport \"sync/atomic\"\nvar n atomic.Int32"); len(got) != 0 {
		t.Errorf("%d imports of sync found in a file that imports sync/atomic only, want 0", len(got))
	}
}
