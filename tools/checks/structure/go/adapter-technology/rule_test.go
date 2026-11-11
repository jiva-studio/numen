package adaptertechnology

import (
	"go/ast"
	"go/types"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/tools/go/packages"
)

func getTechnologyFreeAdapters() map[string]string {
	return map[string]string{
		"internal/adapter/clock": "the machine clock itself, read through the time package",
	}
}

func isUseCaseImport(root, path string) bool {
	return load.IsWithin(path, root+"/usecase")
}

// getCalledObject returns what a call expression calls: the function or method, resolved by the type
// checker, or nil for a conversion, a built-in and a call of a function value.
func getCalledObject(info *types.Info, call *ast.CallExpr) types.Object {
	fun := ast.Unparen(call.Fun)
	switch generic := fun.(type) {
	case *ast.IndexExpr:
		fun = generic.X
	case *ast.IndexListExpr:
		fun = generic.X
	}
	switch callee := fun.(type) {
	case *ast.SelectorExpr:
		if selection, ok := info.Selections[callee]; ok {
			return selection.Obj()
		}
		return info.Uses[callee.Sel]
	case *ast.Ident:
		return info.Uses[callee]
	}
	return nil
}

// isCallingTechnology reports whether the package calls a function or a method that a technology package
// declares, directly or on a value of one of its types. Naming a type of a technology package, as an
// `http.Handler` in a signature, is no call.
func isCallingTechnology(pkg *packages.Package) bool {
	found := false
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || found {
				return !found
			}
			if fn, ok := getCalledObject(pkg.TypesInfo, call).(*types.Func); ok && fn.Pkg() != nil && isTechnologyCall(fn.Pkg().Path()) {
				found = true
			}
			return !found
		})
	}
	return found
}

func getAdapterTechnologyFindings(pkgs []*packages.Package, root string, free map[string]string) (found []string, driven int) {
	for _, pkg := range pkgs {
		rel := strings.TrimPrefix(strings.TrimPrefix(pkg.PkgPath, root), "/")
		isInternalAdapter := load.IsWithin(rel, "internal/adapter")
		if !isInternalAdapter && !load.IsWithin(rel, "adapter") {
			continue
		}
		isDriving := false
		for path := range pkg.Imports {
			isDriving = isDriving || isUseCaseImport(root, path)
		}
		switch {
		case isDriving && isInternalAdapter:
			found = append(found, rel+" calls a use case: a driving adapter lives under adapter/")
		case isDriving:
		default:
			driven++
			if _, isFree := free[rel]; !isFree && !isCallingTechnology(pkg) {
				found = append(found, rel+" is a driven adapter that calls no technology: it holds callbacks or names types, it integrates nothing")
			}
		}
	}
	slices.Sort(found)
	return found, driven
}

func checkDrivenAdaptersCallATechnology(t *testing.T, workspace load.WorkspacePackages) {
	t.Helper()
	pkgs := load.GetModulePackages(t, workspace, "libs/core/kernel")
	free := getTechnologyFreeAdapters()
	found, driven := getAdapterTechnologyFindings(pkgs, load.CoreImportPath, free)
	for _, one := range found {
		t.Error(one)
	}
	for name, reason := range free {
		if !slices.ContainsFunc(pkgs, func(pkg *packages.Package) bool { return pkg.PkgPath == load.CoreImportPath+"/"+name }) || reason == "" {
			t.Errorf("the entry %s names no package of the kernel or has no reason", name)
		}
	}
	if driven < 20 {
		t.Fatalf("%d driven adapters read: the rule is checked against nothing", driven)
	}
}

// isTechnologyCall reports whether a call into the package reaches outside the process. The libraries that
// frame the wire (the RPC framework and the protocol buffer runtime) only build and read messages: a
// package that calls nothing else integrates nothing.
func isTechnologyCall(path string) bool {
	for _, framing := range []string{"connectrpc.com/connect", "google.golang.org/protobuf"} {
		if load.IsWithin(path, framing) {
			return false
		}
	}
	return load.IsTechnologyImport(path)
}
