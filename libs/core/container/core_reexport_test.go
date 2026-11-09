package container

import (
	"go/types"
	"slices"
	"testing"

	"golang.org/x/tools/go/packages"
)

// getReexports returns each type alias of the packages whose target is declared in another of the
// packages, and the number of type names read.
func getReexports(pkgs []*packages.Package) ([]finding, int) {
	members := map[string]bool{}
	for _, pkg := range pkgs {
		members[pkg.PkgPath] = true
	}
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
			named, ok := types.Unalias(object.Type()).(*types.Named)
			if !ok || named.Obj().Pkg() == nil {
				continue
			}
			target := named.Obj().Pkg().Path()
			if target != pkg.PkgPath && members[target] {
				found = append(found, newFinding(pkg.Fset, object.Pos(),
					pkg.Types.Name()+"."+object.Name()+" = "+target+"."+named.Obj().Name()))
			}
		}
	}
	return sortFindings(found), read
}

func checkNoCoreTypeIsReexported(t *testing.T, workspace workspacePackages) {
	t.Helper()
	pkgs := getModulePackages(t, workspace, "libs/core")
	found, read := getReexports(pkgs)
	for _, one := range found {
		t.Errorf("%s re-exports %s: name the type in the package that declares it", one.position, one.name)
	}

	// A walk that read few packages or few type names checks nothing.
	if len(pkgs) < 50 || read < 300 {
		t.Fatalf("%d packages of libs/core and %d type names read: the rule is checked against nothing", len(pkgs), read)
	}
}

// The rule rejects an alias of a type another package of the module declares, and accepts an alias of a
// type of its own package or of a package outside the module.
func TestReexportRule_RejectsAnAliasOfAnotherPackagesType(t *testing.T) {
	dir := writeTreeFixture(t, map[string]string{
		"port/port.go": `package port

type Reader interface{ Read() string }

type Note struct{ Title string }
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
`,
	})
	found, read := getReexports(loadFixture(t, dir))
	if want := []string{"contract.go:9 contract.Reader = fixture/port.Reader"}; !slices.Equal(formatFindings(found), want) {
		t.Errorf("the rule rejects %v, want %v", formatFindings(found), want)
	}
	if read != 7 {
		t.Errorf("%d type names read, want 7", read)
	}
}
