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

// isDrivenAdapter reports whether a package under adapter/ is a driven adapter: it is in internal/adapter, in
// adapter/index or adapter/settings, or declares a type that implements an interface of port/. The rest of
// adapter/ translates a wire or a command line into calls and has no technology of its own to wrap.
func isDrivenAdapter(pkg *packages.Package, rel string, ports []load.PortInterface) bool {
	switch {
	case load.IsWithin(rel, "internal/adapter"), load.IsWithin(rel, "adapter/index"), load.IsWithin(rel, "adapter/settings"):
		return true
	}
	return load.GetImplementation(pkg, ports).Name != ""
}

func getAdapterTechnologyFindings(pkgs []*packages.Package, root string, free map[string]string) (found []string, driven int) {
	ports := load.ListPortInterfaces(pkgs, root)
	for _, pkg := range pkgs {
		rel := strings.TrimPrefix(strings.TrimPrefix(pkg.PkgPath, root), "/")
		isInternalAdapter := load.IsWithin(rel, "internal/adapter")
		if !isInternalAdapter && !load.IsWithin(rel, "adapter") {
			continue
		}
		if !isInternalAdapter && !isDrivenAdapter(pkg, rel, ports) {
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

func isTechnologyCall(path string) bool {
	for _, framing := range []string{"connectrpc.com/connect", "google.golang.org/protobuf"} {
		if load.IsWithin(path, framing) {
			return false
		}
	}
	return load.IsTechnologyImport(path)
}
