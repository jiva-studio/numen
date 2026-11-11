package portcalls

import (
	"go/ast"
	"go/types"
	"slices"
	"testing"
	"golang.org/x/tools/go/packages"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
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

func getPortInterfaceCalls(pkgs []*packages.Package, root string) ([]load.Finding, int) {
	var found []load.Finding
	read := 0
	for _, pkg := range pkgs {
		if !load.IsInLayer(pkg.PkgPath, root, "port") {
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
						found = append(found, load.NewFinding(pkg.Fset, call.Pos(),
							pkg.Types.Name()+"."+function.Name.Name+" calls "+types.TypeString(method.Signature().Recv().Type(), nil)+"."+method.Name()))
					}
					return true
				})
			}
		}
	}
	return load.SortFindings(found), read
}

func checkPortDeclaresAndDoesNotOrchestrate(t *testing.T, workspace load.WorkspacePackages) {
	t.Helper()
	pkgs := load.GetModulePackages(t, workspace, "libs/core/kernel")
	found, _ := getPortInterfaceCalls(pkgs, load.CoreImportPath)
	for _, one := range found {
		t.Errorf("%s: %s; a use case calls the ports, a port declares them", one.Position, one.Name)
	}

	// A port declares and does not orchestrate, so it holds few functions to read. A walk that loaded few
	// port interfaces did not reach the ports; the fixture test proves the rule fires on a call.
	if ports := len(load.ListPortInterfaces(pkgs, load.CoreImportPath)); ports < 40 {
		t.Fatalf("%d port interfaces read: the rule is checked against nothing", ports)
	}
}
