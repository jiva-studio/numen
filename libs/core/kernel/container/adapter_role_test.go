package container

import (
	"go/types"
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// portInterface is an interface with methods that a package under port/ declares.
type portInterface struct {
	name  string
	iface *types.Interface
}

// adapterRole is what the structure of one adapter package shows it to be.
type adapterRole struct {
	pkgPath   string
	drivingBy finding
	drivenBy  finding
}

func (r adapterRole) isDriving() bool { return r.drivingBy.name != "" }

func (r adapterRole) isDriven() bool { return r.drivenBy.name != "" }

func isInLayer(path, root, layer string) bool {
	return isWithin(path, root+"/"+layer)
}

func listPortInterfaces(pkgs []*packages.Package, root string) []portInterface {
	var found []portInterface
	for _, pkg := range pkgs {
		if !isInLayer(pkg.PkgPath, root, "port") {
			continue
		}
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			object, ok := scope.Lookup(name).(*types.TypeName)
			if !ok || object.IsAlias() {
				continue
			}
			iface, ok := object.Type().Underlying().(*types.Interface)
			if !ok || !iface.IsMethodSet() || iface.NumMethods() == 0 {
				continue
			}
			found = append(found, portInterface{name: pkg.PkgPath + "." + name, iface: iface})
		}
	}
	return found
}

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

func listReferences(pkg *packages.Package, isDeclaredIn func(path string) bool, resolve func(types.Object) types.Object) []finding {
	var found []finding
	for ident, object := range pkg.TypesInfo.Uses {
		object = resolve(object)
		if object == nil || object.Pkg() == nil || !isDeclaredIn(object.Pkg().Path()) {
			continue
		}
		if strings.HasSuffix(pkg.Fset.Position(ident.Pos()).Filename, "_test.go") {
			continue
		}
		found = append(found, newFinding(pkg.Fset, ident.Pos(), object.Pkg().Path()+"."+object.Name()))
	}
	return sortFindings(found)
}

func getImplementation(pkg *packages.Package, ports []portInterface) finding {
	scope := pkg.Types.Scope()
	for _, name := range scope.Names() {
		object, ok := scope.Lookup(name).(*types.TypeName)
		if !ok || object.IsAlias() || types.IsInterface(object.Type()) {
			continue
		}
		if named, ok := object.Type().(*types.Named); ok && named.TypeParams().Len() > 0 {
			continue
		}
		for _, port := range ports {
			if types.Implements(object.Type(), port.iface) || types.Implements(types.NewPointer(object.Type()), port.iface) {
				return newFinding(pkg.Fset, object.Pos(), pkg.Types.Name()+"."+name+" implements "+port.name)
			}
		}
	}
	return finding{position: "", name: ""}
}

func getAdapterRoles(adapters, core []*packages.Package, root string) []adapterRole {
	ports := listPortInterfaces(core, root)
	isUseCase := func(path string) bool { return isInLayer(path, root, "usecase") }
	roles := make([]adapterRole, 0, len(adapters))
	for _, pkg := range adapters {
		role := adapterRole{pkgPath: pkg.PkgPath, drivingBy: finding{position: "", name: ""}, drivenBy: getImplementation(pkg, ports)}
		if uses := listReferences(pkg, isUseCase, getDeclaredObject); len(uses) > 0 {
			role.drivingBy = uses[0]
		}
		roles = append(roles, role)
	}
	slices.SortFunc(roles, func(a, b adapterRole) int { return strings.Compare(a.pkgPath, b.pkgPath) })
	return roles
}

func getPortReferences(adapters []*packages.Package, roles []adapterRole, root string) []finding {
	driving := map[string]bool{}
	for _, role := range roles {
		driving[role.pkgPath] = role.isDriving()
	}
	isPort := func(path string) bool { return isInLayer(path, root, "port") }
	var found []finding
	for _, pkg := range adapters {
		if driving[pkg.PkgPath] {
			found = append(found, listReferences(pkg, isPort, getDeclaredObject)...)
		}
	}
	return sortFindings(found)
}

func loadAdapterUniverse(t *testing.T) (adapters, core []*packages.Package) {
	t.Helper()
	config := newTypedConfig(repoRootDir, nil)
	config.Tests = false
	pkgs, err := packages.Load(config, coreImportPath+"/...", numenRoot+"apps/desktop/...")
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range listProductionPackages(pkgs) {
		for _, one := range pkg.Errors {
			t.Errorf("%s does not load: %v", pkg.ID, one)
		}
		switch {
		case isWithin(pkg.PkgPath, coreImportPath):
			core = append(core, pkg)
			if isInLayer(pkg.PkgPath, coreImportPath, "adapter") || isInLayer(pkg.PkgPath, coreImportPath, "internal/adapter") {
				adapters = append(adapters, pkg)
			}
		case isWithin(pkg.PkgPath, numenRoot+"apps/desktop"):
			adapters = append(adapters, pkg)
		}
	}
	return adapters, core
}

// An adapter is driving when it calls a use case and driven when it implements a port. A driving adapter
// reaches the core only through use cases. Both rules read the role from the type-checked code.
func TestEveryAdapterHasOneRoleAndADrivingOneReachesNoPort(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks libs/core/kernel and apps/desktop")
	}
	adapters, core := loadAdapterUniverse(t)
	roles := getAdapterRoles(adapters, core, coreImportPath)
	t.Run("AnAdapterIsDrivingOrDrivenNotBoth", func(t *testing.T) {
		checkAdapterHasOneRole(t, roles)
	})
	t.Run("ADrivingAdapterReferencesNoPort", func(t *testing.T) {
		checkDrivingAdapterReferencesNoPort(t, adapters, roles)
	})
	// A driving adapter maps a request, calls one use case and maps the result. A function that reaches the
	// fields of two use cases, itself or through the functions of its package, decides between them or
	// sequences them, which is a scenario and belongs in a use case. A use case reaches the adapter through a
	// field of a type a use case package declares or of an interface with an Execute method; a use case passed
	// in as a function value, or through a field that is not a use case, is not seen.
	t.Run("ADrivingFunctionCallsOneUseCase", func(t *testing.T) {
		checkDrivingFunctionCallsOneUseCase(t, adapters)
	})
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
				role.pkgPath, role.drivingBy.position, role.drivingBy.name, role.drivenBy.position, role.drivenBy.name)
		}
	}

	// A classification that finds no adapter of either kind is checked against nothing.
	if len(roles) < 20 || driving == 0 || driven == 0 {
		t.Fatalf("%d adapters read, %d driving and %d driven: the rule is checked against nothing", len(roles), driving, driven)
	}
}

func checkDrivingAdapterReferencesNoPort(t *testing.T, adapters []*packages.Package, roles []adapterRole) {
	t.Helper()
	for _, one := range getPortReferences(adapters, roles, coreImportPath) {
		t.Errorf("%s references %s: a driving adapter reaches the core through a use case", one.position, one.name)
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

// A package is classified by what it references and what it implements, a pointer receiver included;
// a package with both roles is reported, an alias declared under usecase/ counts as the port type it
// stands for, and a driving package's reference to a port is found through such an alias.
func TestAdapterRoleRule_ClassifiesByStructure(t *testing.T) {
	pkgs := loadFixture(t, writeTreeFixture(t, getAdapterRoleFixture()))
	var adapters []*packages.Package
	for _, pkg := range pkgs {
		if isInLayer(pkg.PkgPath, "fixture", "adapter") {
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
	refs := formatFindings(getPortReferences(adapters, roles, "fixture"))
	if want := []string{"window.go:8 fixture/port/noteport.Note", "window.go:8 fixture/port/noteport.Title"}; !slices.Equal(refs, want) {
		t.Errorf("the port references are %v, want %v", refs, want)
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
