// Package scancomplete is the check `go/scan-complete`, run on the tree with the planted violations that prove it.
package scancomplete

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// The layers are the folders domain, port, usecase, adapter, internal/adapter, internal/adapterlib,
// internal/testsupport and container. A package belongs to exactly one, so a new folder cannot sit outside
// every rule that reads a layer. The pure packages that belong to none are listed with the reason each needs
// no layer: they hold no scenario and no port, they compute over values or hand out a grant.
func TestEveryKernelPackageBelongsToOneKnownLayer(t *testing.T) {
	dirs, err := listProductionDirs("..")
	if err != nil {
		t.Fatal(err)
	}
	helpers := getHelperPackages()
	for _, one := range getUnplacedPackages(dirs, helpers) {
		t.Error(one)
	}

	// Every helper must exist, or its entry hides a package that is gone and may return unseen.
	for helper, reason := range helpers {
		if !slices.Contains(dirs, helper) || reason == "" {
			t.Errorf("the helper %s is not a package of the kernel or has no reason", helper)
		}
	}
	if len(dirs) < 100 {
		t.Fatalf("%d packages read: the walk is not reading the kernel", len(dirs))
	}
}

// A package outside every group fails, a package in a helper entry passes, and adapter and adapterlib are
// separate groups although one name is the prefix of the other.
func TestScanRule_RejectsAPackageOutsideEveryGroup(t *testing.T) {
	helpers := map[string]string{"internal/csp": "policy string"}
	got := getUnplacedPackages([]string{
		"domain/theme", "port", "port/noteport", "usecase/note/query", "adapter/window/editor", "internal/adapter/clock",
		"internal/adapterlib/wire", "internal/testsupport/indexfake", "container", "internal/csp",
		"internal/vaultreading", "internal/adapterx", "scratch", "internal/csp/sub",
	}, helpers)
	want := []string{
		`internal/vaultreading is in "" groups, want exactly one`,
		`internal/adapterx is in "" groups, want exactly one`,
		`scratch is in "" groups, want exactly one`,
		`internal/csp/sub is in "" groups, want exactly one`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
}

func TestUseCaseImportRule_RejectsASharedPackageOutsideDomainPortAndHelpers(t *testing.T) {
	helpers := map[string]string{"internal/text": "ranges"}
	got := getBadUseCaseImports(map[string][]string{
		"usecase/note": {"domain/theme", "port/noteport", "internal/text", "internal/vaultreading", "adapter/index", "usecase/note/query"},
	}, helpers)
	want := []string{
		"usecase/note imports adapter/index (adapter): a use case imports the domain, ports and its own helpers",
		`usecase/note imports internal/vaultreading (): a use case imports the domain, ports and its own helpers`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
}

func TestUseCasesImportOnlyDomainAndPorts(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	checkUseCasesImportOnlyDomainAndPorts(t, load.Workspace(t, "libs/core/kernel"))
}

// A helper package is a leaf: it imports no port, no use case and no technology package, and exports no type
// whose methods change its state. A helper that does is an adapter or a use case under a helper's name.
func TestHelperPackagesImportNothingAndKeepNoState(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	pkgs := load.GetModulePackages(t, load.Workspace(t, "libs/core/kernel"), "libs/core/kernel")
	helpers := getHelperPackages()
	found, read := getHelperFindings(pkgs, load.CoreImportPath, helpers)
	for _, one := range found {
		t.Error(one)
	}
	for _, one := range getHelperReasonFindings(pkgs, load.CoreImportPath, helpers) {
		t.Error(one)
	}
	if read < 10 {
		t.Fatalf("%d helper packages read: the rule is checked against nothing", read)
	}
}

// The rule refuses a helper that imports a port, a use case or a technology package and one that exports a type
// whose method assigns to a field, increments, deletes from a map or writes through its receiver; it lets
// through a pure function, a method that only reads, and a type of another layer.
func TestHelperRule_RejectsAHelperWithStateOrOutsideImports(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"port/noteport/noteport.go": "package noteport\n\ntype Reader interface{ Read() string }\n",
		"usecase/find/find.go":      "package find\n\ntype Find struct{}\n",
		"internal/adapterlib/pure/p.go": `package pure

func Add(a, b int) int { return a + b }

type Range struct{ A, B int }

func (r Range) Len() int { return r.B - r.A }

func (r *Range) Size() int {
	local := r.B
	local++
	return local
}
`,
		"internal/adapterlib/bus/bus.go": `package bus

type Bus struct {
	subscribers map[string]int
	count       int
	last        *int
}

func (b *Bus) Subscribe(id string) { b.subscribers[id] = 1 }
func (b *Bus) Count()              { b.count++ }
func (b *Bus) Forget(id string)    { delete(b.subscribers, id) }
func (b *Bus) Peek() int           { return b.count }
func (b *Bus) Store(v int)         { *b.last = v }

type Reset struct{ n int }

func (r *Reset) Zero() { *r = Reset{} }

type hidden struct{ n int }

func (h *hidden) Add() { h.n++ }
`,
		"internal/adapterlib/reach/reach.go": `package reach

import (
	"os"

	"fixture/port/noteport"
	"fixture/usecase/find"
)

var _ = os.Args
var _ noteport.Reader
var _ find.Find
`,
		"internal/testsupport/fake/fake.go": "package fake\n\nimport \"fixture/port/noteport\"\n\ntype Reader struct{}\n\nfunc (Reader) Read() string { return \"\" }\n\nvar _ noteport.Reader = Reader{}\n",
		"internal/csp/csp.go":               "package csp\n\nfunc Build() {}\n\ntype Mutable struct{ n int }\n\nfunc (m *Mutable) Set() { m.n = 1 }\n",
		"adapter/index/index.go":            "package index\n\ntype Index struct{ n int }\n\nfunc (i *Index) Set() { i.n = 1 }\n",
	})
	found, read := getHelperFindings(load.LoadFixture(t, dir), "fixture", map[string]string{"internal/csp": "builds the policy"})
	want := []string{
		"internal/adapterlib/bus exports Bus.Count, which changes the state of its receiver: a helper holds no state",
		"internal/adapterlib/bus exports Bus.Forget, which changes the state of its receiver: a helper holds no state",
		"internal/adapterlib/bus exports Bus.Store, which changes the state of its receiver: a helper holds no state",
		"internal/adapterlib/bus exports Bus.Subscribe, which changes the state of its receiver: a helper holds no state",
		"internal/adapterlib/bus exports Reset.Zero, which changes the state of its receiver: a helper holds no state",
		"internal/adapterlib/reach imports os: a helper reaches no technology",
		"internal/adapterlib/reach imports port/noteport: a helper depends on no port and no use case",
		"internal/adapterlib/reach imports usecase/find: a helper depends on no port and no use case",
		"internal/csp exports Mutable.Set, which changes the state of its receiver: a helper holds no state",
		"internal/testsupport/fake imports port/noteport: a helper depends on no port and no use case",
	}
	if !slices.Equal(found, want) {
		t.Errorf("the rule rejects\n%s\nwant\n%s", strings.Join(found, "\n"), strings.Join(want, "\n"))
	}
	if read != 5 {
		t.Errorf("%d helper packages read, want 5", read)
	}
}

// A reason that shares no word with the name, the documentation and the exported identifiers of its package
// describes another package, or none.
func TestHelperRule_RejectsAReasonThatNamesNothingOfThePackage(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"internal/ranges/ranges.go": "// Package ranges works on byte ranges of text.\npackage ranges\n\nfunc Overlap() {}\n",
		"internal/colors/colors.go": "package colors\n\nfunc MixHue() {}\n",
	})
	helpers := map[string]string{
		"internal/ranges":  "byte-range arithmetic over text",
		"internal/colors":  "a reason that is a sentence",
		"internal/missing": "gone",
	}
	got := getHelperReasonFindings(load.LoadFixture(t, dir), "fixture", helpers)
	want := []string{
		"the helper internal/missing is not a package of the kernel",
		"the reason of the helper internal/colors names nothing the package declares or documents: a reason that is a sentence",
	}
	if !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
