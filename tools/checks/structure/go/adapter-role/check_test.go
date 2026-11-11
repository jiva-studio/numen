// Package adapterrole is the check `go/adapter-role`, run on the tree with the planted violations that prove it.
package adapterrole

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/tools/go/packages"
)

// An adapter is driving when it calls a use case and driven when it implements a port. A driving adapter
// reaches the core only through use cases. Both rules read the role from the type-checked code.
func TestEveryAdapterHasOneRoleAndADrivingOneReachesNoPort(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks libs/core/kernel and apps/desktop")
	}
	adapters, core := loadAdapterUniverse(t)
	roles := getAdapterRoles(adapters, core, load.CoreImportPath)
	t.Run("AnAdapterIsDrivingOrDrivenNotBoth", func(t *testing.T) {
		checkAdapterHasOneRole(t, roles)
	})
	t.Run("ADrivingAdapterReferencesNoPort", func(t *testing.T) {
		checkDrivingAdapterReferencesNoPort(t, adapters, roles)
	})
}

// A package is classified by what it references and what it implements, a pointer receiver included;
// a package with both roles is reported, an alias declared under usecase/ counts as the port type it
// stands for, and a driving package's reference to a port is found through such an alias.
func TestAdapterRoleRule_ClassifiesByStructure(t *testing.T) {
	pkgs := load.LoadFixture(t, load.WriteTreeFixture(t, getAdapterRoleFixture()))
	var adapters []*packages.Package
	for _, pkg := range pkgs {
		if load.IsInLayer(pkg.PkgPath, "fixture", "adapter") {
			adapters = append(adapters, pkg)
		}
	}
	roles := getAdapterRoles(adapters, pkgs, "fixture")
	var got []string
	for _, role := range roles {
		got = append(got, role.pkgPath+" "+strings.Join(describeRole(role), "+"))
	}
	want := []string{
		"fixture/adapter/aliased ",
		"fixture/adapter/cli driving",
		"fixture/adapter/mixed driving+driven",
		"fixture/adapter/store driven",
		"fixture/adapter/window driving",
	}
	if !slices.Equal(got, want) {
		t.Errorf("the roles are %v, want %v", got, want)
	}
	refs := load.FormatFindings(getPortReferences(adapters, roles, "fixture"))
	if want := []string{"window.go:8 fixture/port/noteport.Note", "window.go:8 fixture/port/noteport.Title"}; !slices.Equal(refs, want) {
		t.Errorf("the port references are %v, want %v", refs, want)
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
