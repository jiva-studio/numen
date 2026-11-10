package container

import (
	"go/ast"
	"go/types"
	"slices"
	"testing"

	"golang.org/x/tools/go/packages"
)

func getCalledInterfaceMethod(call *ast.CallExpr, info *types.Info) *types.Func {
	selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	selection := info.Selections[selector]
	if selection == nil || selection.Kind() == types.FieldVal {
		return nil
	}
	method, ok := selection.Obj().(*types.Func)
	if !ok {
		return nil
	}
	recv := method.Signature().Recv()
	if recv == nil || !types.IsInterface(recv.Type()) {
		return nil
	}
	return method
}

func getPortInterfaceCalls(pkgs []*packages.Package, root string) ([]finding, int) {
	var found []finding
	read := 0
	for _, pkg := range pkgs {
		if !isInLayer(pkg.PkgPath, root, "port") {
			continue
		}
		for _, file := range pkg.Syntax {
			if ast.IsGenerated(file) {
				continue
			}
			for _, decl := range file.Decls {
				function, ok := decl.(*ast.FuncDecl)
				if !ok || function.Body == nil {
					continue
				}
				read++
				ast.Inspect(function.Body, func(node ast.Node) bool {
					call, ok := node.(*ast.CallExpr)
					if !ok {
						return true
					}
					if method := getCalledInterfaceMethod(call, pkg.TypesInfo); method != nil {
						found = append(found, newFinding(pkg.Fset, call.Pos(),
							pkg.Types.Name()+"."+function.Name.Name+" calls "+types.TypeString(method.Signature().Recv().Type(), nil)+"."+method.Name()))
					}
					return true
				})
			}
		}
	}
	return sortFindings(found), read
}

func checkPortDeclaresAndDoesNotOrchestrate(t *testing.T, workspace workspacePackages) {
	t.Helper()
	pkgs := getModulePackages(t, workspace, "libs/core/kernel")
	found, _ := getPortInterfaceCalls(pkgs, coreImportPath)
	for _, one := range found {
		t.Errorf("%s: %s; a use case calls the ports, a port declares them", one.position, one.name)
	}

	// A port declares and does not orchestrate, so it holds few functions to read. A walk that loaded few
	// port interfaces did not reach the ports; the fixture test proves the rule fires on a call.
	if ports := len(listPortInterfaces(pkgs, coreImportPath)); ports < 40 {
		t.Fatalf("%d port interfaces read: the rule is checked against nothing", ports)
	}
}

// The rule rejects a call through an interface in a function or method of a port, a promoted one and one
// inside a closure included, and accepts calls of functions and of concrete methods.
func TestPortCallRule_RejectsInterfaceCalls(t *testing.T) {
	dir := writeTreeFixture(t, map[string]string{
		"port/noteport/noteport.go": `package noteport

import "strings"

type Reader interface{ Read(path string) string }

type Note struct{ Title string }

func (n Note) Words() int { return len(strings.Fields(n.Title)) }

func ReadTitle(r Reader, path string) Note { return Note{Title: r.Read(path)} }

type Wrapped struct{ Reader }

func (w Wrapped) Twice(path string) string {
	read := func() string { return w.Read(path) }
	return read() + read()
}

func Describe(err error) string { return err.Error() }

func Count(n Note) int { return n.Words() }
`,
		"usecase/read/read.go": `package read

import "fixture/port/noteport"

func Execute(r noteport.Reader) string { return r.Read("a.md") }
`,
	})
	found, read := getPortInterfaceCalls(loadFixture(t, dir), "fixture")
	want := []string{
		"noteport.go:11 noteport.ReadTitle calls fixture/port/noteport.Reader.Read",
		"noteport.go:16 noteport.Twice calls fixture/port/noteport.Reader.Read",
		"noteport.go:20 noteport.Describe calls error.Error",
	}
	if got := formatFindings(found); !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
	if read != 5 {
		t.Errorf("%d functions read, want 5", read)
	}
}
