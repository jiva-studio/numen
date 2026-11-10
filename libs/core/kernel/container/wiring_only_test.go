package container

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/types/typeutil"
)

// maxWiringComplexity is the cyclomatic complexity of the most complex function of the composition root: a
// function that selects and connects takes at most two decisions, such as a guard and a loop-free choice.
// Anything that decides more over data is a rule, and a rule belongs to a use case or an adapter.
const maxWiringComplexity = 3

// minWiringFunctions is how many functions the walk of the composition root reads at least; fewer means the
// check read the wrong package.
const minWiringFunctions = 100

func isRandomImport(path string) bool {
	return isWithin(path, "math/rand") || isWithin(path, "crypto")
}

func isPrintCall(fn *types.Func) bool {
	return fn.Pkg() != nil && fn.Pkg().Path() == "fmt" && (strings.HasPrefix(fn.Name(), "Fprint") || strings.HasPrefix(fn.Name(), "Print"))
}

func getComplexity(fn *ast.FuncDecl) int {
	complexity := 1
	ast.Inspect(fn, func(node ast.Node) bool {
		switch one := node.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt:
			complexity++
		case *ast.CaseClause:
			if one.List != nil {
				complexity++
			}
		case *ast.CommClause:
			if one.Comm != nil {
				complexity++
			}
		case *ast.BinaryExpr:
			if one.Op == token.LAND || one.Op == token.LOR {
				complexity++
			}
		}
		return true
	})
	return complexity
}

// getWiringFindings returns what the files of the package at path hold that wiring does not: an import of a
// package that reaches the machine, a call that prints, and a function that decides more than
// maxWiringComplexity, with the number of functions read.
func getWiringFindings(pkgs []*packages.Package, path string) (found []finding, functions int) {
	for _, pkg := range pkgs {
		if pkg.PkgPath != path {
			continue
		}
		for _, file := range pkg.Syntax {
			for _, spec := range file.Imports {
				imported, err := strconv.Unquote(spec.Path.Value)
				if err == nil && (isTechnologyImport(imported) || isRandomImport(imported)) {
					found = append(found, newFinding(pkg.Fset, spec.Pos(), "imports "+imported))
				}
			}
			ast.Inspect(file, func(node ast.Node) bool {
				call, isCall := node.(*ast.CallExpr)
				if !isCall {
					return true
				}
				if fn, isFunc := typeutil.Callee(pkg.TypesInfo, call).(*types.Func); isFunc && isPrintCall(fn) {
					found = append(found, newFinding(pkg.Fset, call.Pos(), "calls fmt."+fn.Name()))
				}
				return true
			})
			for _, decl := range file.Decls {
				fn, isFn := decl.(*ast.FuncDecl)
				if !isFn || fn.Body == nil {
					continue
				}
				functions++
				if complexity := getComplexity(fn); complexity > maxWiringComplexity {
					found = append(found, newFinding(pkg.Fset, fn.Pos(), fn.Name.Name+" has complexity "+strconv.Itoa(complexity)))
				}
			}
		}
	}
	return sortFindings(found), functions
}

func checkTheCompositionRootHoldsWiringOnly(t *testing.T, workspace workspacePackages) {
	t.Helper()
	found, functions := getWiringFindings(getModulePackages(t, workspace, "libs/core/kernel"), coreImportPath+"/container")
	for _, one := range found {
		t.Errorf("%s: %s; the composition root connects and holds no rule", one.position, one.name)
	}
	if functions < minWiringFunctions {
		t.Fatalf("%d functions of the composition root read: the rule is checked against nothing", functions)
	}
}

// The rule rejects an import of the machine, of randomness and of cryptography, a call that prints, and a
// function with more decisions than wiring takes, and accepts a function that selects and connects.
func TestWiringOnlyRule_RejectsTechnologyPrintingAndDecisions(t *testing.T) {
	dir := writeTreeFixture(t, map[string]string{
		"container/wire.go": `package container

import (
	"crypto/rand"
	"fmt"
	"os"
)

func Connect(a, b func() int) int { return a() + b() }

func Guard(a func() int) int {
	if a == nil {
		return 0
	}
	return a()
}

func Choose(values []int, limit int) int {
	total := 0
	for _, value := range values {
		if value > limit && value%2 == 0 {
			total += value
		}
	}
	return total
}

func Print() {
	fmt.Println(os.Args, rand.Reader)
}
`,
		"container/wire_other.go": "package container\n\nfunc Other() {}\n",
		"other/other.go": `package other

import "os"

func Read() []string { return os.Args }
`,
	})
	found, functions := getWiringFindings(loadFixture(t, dir), "fixture/container")
	want := []string{
		"wire.go Choose has complexity 4",
		"wire.go calls fmt.Println",
		"wire.go imports crypto/rand",
		"wire.go imports os",
	}
	if got := formatFindings(found); !slices.Equal(got, want) || functions != 5 {
		t.Errorf("the rule rejects %v with %d functions read, want %v with 5", got, functions, want)
	}
}
