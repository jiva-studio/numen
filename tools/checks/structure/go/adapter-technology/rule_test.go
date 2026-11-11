package adaptertechnology

import (
	"go/ast"
	"go/token"
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
			if fn, ok := getCalledObject(pkg.TypesInfo, call).(*types.Func); ok && fn.Pkg() != nil && (isTechnologyCall(fn.Pkg().Path()) || isConnectClientCall(pkg.Fset, fn)) {
				found = true
			}
			return !found
		})
	}
	return found
}

func isConnectClientCall(fset *token.FileSet, fn *types.Func) bool {
	// A call across a process boundary through Connect is a method of the connect.Client type, or a constructor or a method of a client
	// generated into a *connect.go file. Handlers and routes of the server side are not client calls.
	signature, ok := fn.Type().(*types.Signature)
	if !ok {
		return false
	}
	receiver := getReceiverName(signature)
	if load.IsWithin(fn.Pkg().Path(), "connectrpc.com/connect") {
		return receiver == "Client"
	}
	if !strings.HasSuffix(fset.Position(fn.Pos()).Filename, "connect.go") {
		return false
	}
	if receiver == "" {
		return strings.HasPrefix(fn.Name(), "New") && strings.HasSuffix(fn.Name(), "Client")
	}
	return strings.HasSuffix(receiver, "Client") || strings.HasSuffix(receiver, "client")
}

func getReceiverName(signature *types.Signature) string {
	if signature.Recv() == nil {
		return ""
	}
	receiver := signature.Recv().Type()
	if pointer, ok := receiver.(*types.Pointer); ok {
		receiver = pointer.Elem()
	}
	if named, ok := receiver.(*types.Named); ok {
		return named.Obj().Name()
	}
	return ""
}

func isDrivenAdapter(pkg *packages.Package, rel string, ports []load.PortInterface) bool {
	// A driven adapter is in internal/adapter, in adapter/index or adapter/settings, or declares a type that implements an interface of port/.
	// The rest of adapter/ translates a wire or a command line into calls and has no technology of its own to wrap.
	switch {
	case load.IsWithin(rel, "internal/adapter"), load.IsWithin(rel, "adapter/index"), load.IsWithin(rel, "adapter/settings"):
		return true
	}
	return load.GetImplementation(pkg, ports).Name != ""
}

// getAdapterFamily is the folder of the adapters that belong together, internal/adapter/<technology>, for a package nested
// below it, and empty for any other package.
func getAdapterFamily(rel string) string {
	parts := strings.Split(rel, "/")
	if len(parts) < 4 || !load.IsWithin(rel, "internal/adapter") {
		return ""
	}
	return strings.Join(parts[:3], "/")
}

// getSharedPlumbing lists the packages nested in an adapter family that another package of the same family imports and
// that implement no port. They hold what the adapters of the family share, such as a lease or a null object, and
// are no adapter of a technology themselves.
func getSharedPlumbing(pkgs []*packages.Package, root string, ports []load.PortInterface) map[string]bool {
	shared := map[string]bool{}
	for _, pkg := range pkgs {
		rel := strings.TrimPrefix(strings.TrimPrefix(pkg.PkgPath, root), "/")
		family := getAdapterFamily(rel)
		if family == "" {
			continue
		}
		for path := range pkg.Imports {
			imported := strings.TrimPrefix(strings.TrimPrefix(path, root), "/")
			if imported != rel && getAdapterFamily(imported) == family {
				shared[imported] = true
			}
		}
	}
	for _, pkg := range pkgs {
		rel := strings.TrimPrefix(strings.TrimPrefix(pkg.PkgPath, root), "/")
		if shared[rel] && load.GetImplementation(pkg, ports).Name != "" {
			delete(shared, rel)
		}
	}
	return shared
}

func getAdapterTechnologyFindings(pkgs []*packages.Package, root string, free map[string]string) (found []string, driven int) {
	ports := load.ListPortInterfaces(pkgs, root)
	shared := getSharedPlumbing(pkgs, root, ports)
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
		case isDriving, shared[rel]:
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
	if driven == 0 {
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
