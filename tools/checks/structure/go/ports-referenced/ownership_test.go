package portsreferenced

import (
	"go/ast"
	"go/types"
	"slices"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/tools/go/packages"
)

// portConsumers records the production packages that take each contract across shipped platforms.
type portConsumers map[string]map[string]bool

func (c portConsumers) addPackages(root, dir string, pkgs []*packages.Package) {
	contracts := map[*types.TypeName]string{}
	for _, pkg := range pkgs {
		if !isUnderDir(pkg, dir) {
			continue
		}
		for _, name := range pkg.Types.Scope().Names() {
			t, ok := pkg.Types.Scope().Lookup(name).(*types.TypeName)
			if ok && t.Exported() {
				key := pkg.Name + "." + name
				contracts[t] = key
				if c[key] == nil {
					c[key] = map[string]bool{}
				}
			}
		}
	}
	for _, pkg := range pkgs {
		if len(pkg.GoFiles) == 0 || isUnderDir(pkg, dir) {
			continue
		}
		owner := load.GetPackage(root, pkg.GoFiles[0])
		if owner == "container" || load.IsUnder(owner, "internal/testsupport") || load.IsUseCaseTestSupport(owner) {
			continue
		}
		record := func(t types.Type) {
			if n, ok := types.Unalias(t).(*types.Named); ok {
				if key, ok := contracts[n.Obj()]; ok {
					c[key][owner] = true
				}
			}
		}
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(node ast.Node) bool {
				if spec, ok := node.(*ast.ValueSpec); ok && !slices.ContainsFunc(spec.Names, func(name *ast.Ident) bool { return name.Name != "_" }) {
					return false
				}
				expr, ok := node.(ast.Expr)
				if !ok {
					return true
				}
				if value, ok := pkg.TypesInfo.Types[expr]; ok {
					record(value.Type)
				}
				selector, ok := expr.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if selection := pkg.TypesInfo.Selections[selector]; selection != nil {
					record(selection.Recv())
					// A call through an embedding interface also consumes the contract that declares the method.
					if method, ok := selection.Obj().(*types.Func); ok {
						record(method.Signature().Recv().Type())
					}
				}
				return true
			})
		}
	}
}

func (c portConsumers) listSingleConsumerContracts() []string {
	var unshared []string
	for name, owners := range c {
		if len(owners) < 2 {
			unshared = append(unshared, name)
		}
	}
	slices.Sort(unshared)
	return unshared
}
