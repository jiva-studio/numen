package portsreferenced

import (
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
		for _, value := range pkg.TypesInfo.Types {
			record(value.Type)
		}
		// A call through an embedding interface also consumes the contract that declares the method.
		for _, selection := range pkg.TypesInfo.Selections {
			record(selection.Recv())
			if m, ok := selection.Obj().(*types.Func); ok {
				record(m.Signature().Recv().Type())
			}
		}
	}
}

func (c portConsumers) listUnshared() []string {
	var unshared []string
	for name, owners := range c {
		if len(owners) < 2 {
			unshared = append(unshared, name)
		}
	}
	slices.Sort(unshared)
	return unshared
}
