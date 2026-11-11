package corereexport

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

func getReexports(pkgs []*packages.Package, isRepo func(path string) bool) ([]load.Finding, int) {
	var found []load.Finding
	read := 0
	for _, pkg := range pkgs {
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			object, ok := scope.Lookup(name).(*types.TypeName)
			if !ok {
				continue
			}
			read++
			if !object.IsAlias() {
				continue
			}
			switch target := types.Unalias(object.Type()).(type) {
			case *types.Named:
				if target.Obj().Pkg() == nil {
					continue
				}
				path := target.Obj().Pkg().Path()
				if path != pkg.PkgPath && isRepo(path) {
					found = append(found, load.NewFinding(pkg.Fset, object.Pos(),
						pkg.Types.Name()+"."+object.Name()+" = "+path+"."+target.Obj().Name()))
				}
			case *types.Interface:
				found = append(found, load.NewFinding(pkg.Fset, object.Pos(),
					pkg.Types.Name()+"."+object.Name()+" is an alias of an unnamed interface"))
			}
		}
		found = append(found, getForwarders(pkg, isRepo)...)
	}
	return load.SortFindings(found), read
}

func getForwardedPackage(pkg *packages.Package, expr ast.Expr, isRepo func(string) bool) (string, bool) {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	ident, ok := selector.X.(*ast.Ident)
	if !ok {
		return "", false
	}
	imported, ok := pkg.TypesInfo.Uses[ident].(*types.PkgName)
	if !ok || imported.Imported().Path() == pkg.PkgPath || !isRepo(imported.Imported().Path()) {
		return "", false
	}
	return imported.Imported().Path() + "." + selector.Sel.Name, true
}

func getForwarders(pkg *packages.Package, isRepo func(string) bool) []load.Finding {
	var found []load.Finding
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			switch decl := decl.(type) {
			case *ast.GenDecl:
				if decl.Tok != token.VAR && decl.Tok != token.CONST {
					continue
				}
				for _, spec := range decl.Specs {
					value, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for i, expr := range value.Values {
						if target, ok := getForwardedPackage(pkg, expr, isRepo); ok && i < len(value.Names) && value.Names[i].IsExported() {
							found = append(found, load.NewFinding(pkg.Fset, value.Names[i].Pos(),
								pkg.Types.Name()+"."+value.Names[i].Name+" = "+target))
						}
					}
				}
			case *ast.FuncDecl:
				if target, ok := getForwardedCall(pkg, decl, isRepo); ok {
					found = append(found, load.NewFinding(pkg.Fset, decl.Name.Pos(),
						pkg.Types.Name()+"."+decl.Name.Name+" forwards to "+target))
				}
			}
		}
	}
	return found
}

func getForwardedCall(pkg *packages.Package, decl *ast.FuncDecl, isRepo func(string) bool) (string, bool) {
	if decl.Body == nil || len(decl.Body.List) != 1 || decl.Recv != nil || !decl.Name.IsExported() {
		return "", false
	}
	var call ast.Expr
	switch stmt := decl.Body.List[0].(type) {
	case *ast.ReturnStmt:
		if len(stmt.Results) != 1 {
			return "", false
		}
		call = stmt.Results[0]
	case *ast.ExprStmt:
		call = stmt.X
	default:
		return "", false
	}
	invoked, ok := call.(*ast.CallExpr)
	if !ok {
		return "", false
	}
	target, ok := getForwardedPackage(pkg, invoked.Fun, isRepo)
	if !ok {
		return "", false
	}
	selector, ok := invoked.Fun.(*ast.SelectorExpr)
	callee, isFunc := pkg.TypesInfo.Uses[selector.Sel].(*types.Func)
	own, isOwn := pkg.TypesInfo.Defs[decl.Name].(*types.Func)
	if !ok || !isFunc || !isOwn || !types.Identical(own.Type(), callee.Type()) {
		return "", false
	}
	var params []string
	for _, field := range decl.Type.Params.List {
		for _, name := range field.Names {
			params = append(params, name.Name)
		}
	}
	var args []string
	for _, arg := range invoked.Args {
		ident, ok := arg.(*ast.Ident)
		if !ok {
			return "", false
		}
		args = append(args, ident.Name)
	}
	if !slices.Equal(params, args) {
		return "", false
	}
	return target, true
}

func isRepoPath(path string) bool {
	return strings.HasPrefix(path, load.NumenRoot)
}

func checkNoCoreTypeIsReexported(t *testing.T, workspace load.WorkspacePackages) {
	t.Helper()
	pkgs := load.GetModulePackages(t, workspace, "libs/core/kernel")
	found, read := getReexports(pkgs, isRepoPath)
	for _, one := range found {
		t.Errorf("%s re-exports %s: name the type in the package that declares it", one.Position, one.Name)
	}

	// A walk that read few packages or few type names checks nothing.
	if len(pkgs) < 50 || read < 300 {
		t.Fatalf("%d packages of libs/core/kernel and %d type names read: the rule is checked against nothing", len(pkgs), read)
	}
}
