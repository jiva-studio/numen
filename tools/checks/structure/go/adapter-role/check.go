package adapterrole

import (
	"go/types"
	"slices"
	"strings"
	"testing"
	"golang.org/x/tools/go/packages"
	"go/ast"
	"go/token"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// adapterRole is what the structure of one adapter package shows it to be.
type adapterRole struct {
	pkgPath   string
	drivingBy load.Finding
	drivenBy  load.Finding
}

func (r adapterRole) isDriving() bool { return r.drivingBy.Name != "" }

func (r adapterRole) isDriven() bool { return r.drivenBy.Name != "" }

func getDeclaredObject(object types.Object) types.Object {
	name, ok := object.(*types.TypeName)
	if !ok || !name.IsAlias() {
		return object
	}
	if named, ok := types.Unalias(name.Type()).(*types.Named); ok {
		return named.Obj()
	}
	return object
}

func listReferences(pkg *packages.Package, isDeclaredIn func(path string) bool, resolve func(types.Object) types.Object) []load.Finding {
	var found []load.Finding
	for ident, object := range pkg.TypesInfo.Uses {
		object = resolve(object)
		if object == nil || object.Pkg() == nil || !isDeclaredIn(object.Pkg().Path()) {
			continue
		}
		if strings.HasSuffix(pkg.Fset.Position(ident.Pos()).Filename, "_test.go") {
			continue
		}
		found = append(found, load.NewFinding(pkg.Fset, ident.Pos(), object.Pkg().Path()+"."+object.Name()))
	}
	return load.SortFindings(found)
}

func getAdapterRoles(adapters, core []*packages.Package, root string) []adapterRole {
	ports := load.ListPortInterfaces(core, root)
	isUseCase := func(path string) bool { return load.IsInLayer(path, root, "usecase") }
	roles := make([]adapterRole, 0, len(adapters))
	for _, pkg := range adapters {
		role := adapterRole{pkgPath: pkg.PkgPath, drivingBy: load.Finding{Position: "", Name: ""}, drivenBy: load.GetImplementation(pkg, ports)}
		if uses := listReferences(pkg, isUseCase, getDeclaredObject); len(uses) > 0 {
			role.drivingBy = uses[0]
		}
		roles = append(roles, role)
	}
	slices.SortFunc(roles, func(a, b adapterRole) int { return strings.Compare(a.pkgPath, b.pkgPath) })
	return roles
}

func getPortReferences(adapters []*packages.Package, roles []adapterRole, root string) []load.Finding {
	driving := map[string]bool{}
	for _, role := range roles {
		driving[role.pkgPath] = role.isDriving()
	}
	isPort := func(path string) bool { return load.IsInLayer(path, root, "port") }
	var found []load.Finding
	for _, pkg := range adapters {
		if driving[pkg.PkgPath] {
			found = append(found, listReferences(pkg, isPort, getDeclaredObject)...)
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
	if len(roles) < 20 || driving == 0 || driven == 0 {
		t.Fatalf("%d adapters read, %d driving and %d driven: the rule is checked against nothing", len(roles), driving, driven)
	}
}

func checkDrivingAdapterReferencesNoPort(t *testing.T, adapters []*packages.Package, roles []adapterRole) {
	t.Helper()
	for _, one := range getPortReferences(adapters, roles, load.CoreImportPath) {
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

func checkDrivingFunctionCallsOneUseCase(t *testing.T, adapters []*packages.Package) {
	t.Helper()
	found, functions := load.ListManyUseCaseCalls(adapters, load.CoreImportPath)
	for _, one := range found {
		t.Errorf("%s %s: a driving adapter calls one use case", one.Position, one.Name)
	}
	if functions == 0 {
		t.Fatalf("%d driving-adapter functions reach a use case: the walk is not reading them", functions)
	}
}
