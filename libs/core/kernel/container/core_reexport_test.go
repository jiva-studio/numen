package container

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

func getReexports(pkgs []*packages.Package, isRepo func(path string) bool) ([]finding, int) {
	var found []finding
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
					found = append(found, newFinding(pkg.Fset, object.Pos(),
						pkg.Types.Name()+"."+object.Name()+" = "+path+"."+target.Obj().Name()))
				}
			case *types.Interface:
				found = append(found, newFinding(pkg.Fset, object.Pos(),
					pkg.Types.Name()+"."+object.Name()+" is an alias of an unnamed interface"))
			}
		}
		found = append(found, getForwarders(pkg, isRepo)...)
	}
	return sortFindings(found), read
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

func getForwarders(pkg *packages.Package, isRepo func(string) bool) []finding {
	var found []finding
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
							found = append(found, newFinding(pkg.Fset, value.Names[i].Pos(),
								pkg.Types.Name()+"."+value.Names[i].Name+" = "+target))
						}
					}
				}
			case *ast.FuncDecl:
				if target, ok := getForwardedCall(pkg, decl, isRepo); ok {
					found = append(found, newFinding(pkg.Fset, decl.Name.Pos(),
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
	return strings.HasPrefix(path, numenRoot)
}

func checkNoCoreTypeIsReexported(t *testing.T, workspace workspacePackages) {
	t.Helper()
	pkgs := getModulePackages(t, workspace, "libs/core/kernel")
	found, read := getReexports(pkgs, isRepoPath)
	for _, one := range found {
		t.Errorf("%s re-exports %s: name the type in the package that declares it", one.position, one.name)
	}

	// A walk that read few packages or few type names checks nothing.
	if len(pkgs) < 50 || read < 300 {
		t.Fatalf("%d packages of libs/core/kernel and %d type names read: the rule is checked against nothing", len(pkgs), read)
	}
}

// A re-export is an alias of a named type, an alias of an unnamed interface (a second spelling of a port that
// hides its name), an exported variable or constant set to a member of another package, or an exported
// function whose whole body calls a function of another package of the repository with its own parameters
// and has the same signature.
// The rule accepts an alias of a type of its own package or of a package outside the repository, and a
// function whose signature differs from the callee's.
func TestReexportRule_RejectsAnAliasOfAnotherPackagesType(t *testing.T) {
	dir := writeTreeFixture(t, map[string]string{
		"port/port.go": `package port

type Reader interface{ Read() string }

type Note struct{ Title string }

const Limit = 3

var Default = Note{}

func Open(name string) Reader { return nil }

func Close() {}
`,
		"contract/contract.go": `package contract

import (
	"strings"

	"fixture/port"
)

type Reader = port.Reader

type Own struct{}

type Local = Own

type Builder = strings.Builder

type Notes = []port.Note

type Anonymous = interface{ Read() string }

const Limit = port.Limit

var Default = port.Default

func Open(name string) port.Reader { return port.Open(name) }

func Close() { port.Close() }

func Trim(name string) port.Reader { return port.Open(name + "x") }

func Swap(a, b string) port.Reader { return port.Open(b) }

func Wrap(name string) any { return port.Open(name) }
`,
	})
	found, read := getReexports(loadFixture(t, dir), func(path string) bool { return strings.HasPrefix(path, "fixture/") })
	want := []string{
		"contract.go:9 contract.Reader = fixture/port.Reader",
		"contract.go:19 contract.Anonymous is an alias of an unnamed interface",
		"contract.go:21 contract.Limit = fixture/port.Limit",
		"contract.go:23 contract.Default = fixture/port.Default",
		"contract.go:25 contract.Open forwards to fixture/port.Open",
		"contract.go:27 contract.Close forwards to fixture/port.Close",
	}
	got := formatFindings(found)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
	if read != 8 {
		t.Errorf("%d type names read, want 8", read)
	}
}
