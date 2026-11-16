package adapterrole

import (
	"go/types"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
	"golang.org/x/tools/go/packages"
)

// adapterRole is what the structure of one adapter package shows it to be.
type adapterRole struct {
	pkgPath   string
	drivingBy load.Finding
	drivenBy  load.Finding
}

func (r adapterRole) isDriving() bool { return r.drivingBy.Name != "" }

func (r adapterRole) isDriven() bool { return r.drivenBy.Name != "" }

func getAdapterRoles(adapters, core []*packages.Package, root string) []adapterRole {
	ports := load.ListPortInterfaces(core, root)
	isUseCase := func(object types.Object) bool {
		return load.IsInLayer(object.Pkg().Path(), root, "usecase") && !load.IsPortType(object, ports)
	}
	roles := make([]adapterRole, 0, len(adapters))
	for _, pkg := range adapters {
		role := adapterRole{pkgPath: pkg.PkgPath, drivingBy: load.Finding{Position: "", Name: ""}, drivenBy: load.GetImplementation(pkg, ports)}
		if uses := load.ListReferences(pkg, isUseCase, load.GetDeclaredObject); len(uses) > 0 {
			role.drivingBy = uses[0]
		}
		roles = append(roles, role)
	}
	slices.SortFunc(roles, func(a, b adapterRole) int { return strings.Compare(a.pkgPath, b.pkgPath) })
	return roles
}

func getPortReferences(adapters, core []*packages.Package, roles []adapterRole, root string) []load.Finding {
	driving := map[string]bool{}
	for _, role := range roles {
		driving[role.pkgPath] = role.isDriving()
	}
	ports := load.ListPortInterfaces(core, root)
	isPort := func(object types.Object) bool {
		return load.IsInLayer(object.Pkg().Path(), root, "port") || load.IsPortType(object, ports)
	}
	var found []load.Finding
	for _, pkg := range adapters {
		if driving[pkg.PkgPath] {
			found = append(found, load.ListReferences(pkg, isPort, load.GetDeclaredObject)...)
		}
	}
	return load.SortFindings(found)
}

func loadAdapterUniverse(t *testing.T) (adapters, core []*packages.Package) {
	t.Helper()
	config := load.NewTypedConfig(load.RepoRootDir, nil)
	config.Tests = false
	pkgs, err := packages.Load(config, load.CoreImportPath+"/...", load.NumenRoot+"apps/desktop/...")
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range load.ListProductionPackages(pkgs) {
		for _, one := range pkg.Errors {
			t.Errorf("%s does not load: %v", pkg.ID, one)
		}
		switch {
		case load.IsWithin(pkg.PkgPath, load.CoreImportPath):
			core = append(core, pkg)
			if load.IsInLayer(pkg.PkgPath, load.CoreImportPath, "adapter") || load.IsInLayer(pkg.PkgPath, load.CoreImportPath, "internal/adapter") {
				adapters = append(adapters, pkg)
			}
		case load.IsWithin(pkg.PkgPath, load.NumenRoot+"apps/desktop"):
			adapters = append(adapters, pkg)
		}
	}
	return adapters, core
}

func checkAdapterHasOneRole(t *testing.T, roles []adapterRole) {
	t.Helper()
	driving, driven := 0, 0
	for _, role := range roles {
		if role.isDriving() {
			driving++
		}
		if role.isDriven() {
			driven++
		}
		if role.isDriving() && role.isDriven() {
			t.Errorf("%s is driving and driven: %s references %s, and %s %s",
				role.pkgPath, role.drivingBy.Position, role.drivingBy.Name, role.drivenBy.Position, role.drivenBy.Name)
		}
	}

	// A classification that finds no adapter of either kind is checked against nothing.
	if len(roles) == 0 || driving == 0 || driven == 0 {
		t.Fatalf("%d adapters read, %d driving and %d driven: the rule is checked against nothing", len(roles), driving, driven)
	}
}

func checkDrivingAdapterReferencesNoPort(t *testing.T, adapters, core []*packages.Package, roles []adapterRole) {
	t.Helper()
	for _, one := range getPortReferences(adapters, core, roles, load.CoreImportPath) {
		t.Errorf("%s references %s: a driving adapter reaches the core through a use case", one.Position, one.Name)
	}
	driving := 0
	for _, role := range roles {
		if role.isDriving() {
			driving++
		}
	}
	if driving < 5 {
		t.Fatalf("%d driving adapters read: the rule is checked against nothing", driving)
	}
}

func getAdapterRoleFixture() map[string]string {
	return map[string]string{
		"port/noteport/noteport.go": `package noteport

type Notes interface{ Read() string }

type Note struct{ Title string }
`,
		"usecase/find/find.go": `package find

import "fixture/port/noteport"

type Find struct{ notes noteport.Notes }

func New(notes noteport.Notes) *Find { return &Find{notes: notes} }

func (f *Find) Execute() string { return f.notes.Read() }
`,
		"usecase/find/ports.go": `package find

type Cache interface{ Get() string }

type Executor interface{ Execute() string }

func NewCached(c Cache) *Find { return &Find{} }
`,
		"adapter/cache/cache.go": `package cache

type Cache struct{}

func (Cache) Get() string { return "" }
`,
		"adapter/execs/execs.go": `package execs

type Exec struct{}

func (Exec) Execute() string { return "" }
`,
		"adapter/warm/warm.go": `package warm

import "fixture/usecase/find"

func Warm(f *find.Find, c find.Cache) {}
`,
		"usecase/find/findcontract/findcontract.go": `package findcontract

import "fixture/port/noteport"

type Note = noteport.Note
`,
		"adapter/cli/cli.go": `package cli

import "fixture/usecase/find"

func Run(f *find.Find) string { return f.Execute() }
`,
		"adapter/window/window.go": `package window

import (
	"fixture/usecase/find"
	"fixture/usecase/find/findcontract"
)

func Show(f *find.Find) findcontract.Note { return findcontract.Note{Title: f.Execute()} }
`,
		"adapter/store/store.go": `package store

type Store struct{}

func (*Store) Read() string { return "" }
`,
		"adapter/mixed/mixed.go": `package mixed

import "fixture/usecase/find"

type Memory struct{}

func (Memory) Read() string { return "" }

func Open() *find.Find { return find.New(Memory{}) }
`,
		"adapter/aliased/aliased.go": `package aliased

import "fixture/usecase/find/findcontract"

func Title(n findcontract.Note) string { return n.Title }
`,
	}
}

func describeRole(role adapterRole) []string {
	var kinds []string
	if role.isDriving() {
		kinds = append(kinds, "driving")
	}
	if role.isDriven() {
		kinds = append(kinds, "driven")
	}
	return kinds
}
