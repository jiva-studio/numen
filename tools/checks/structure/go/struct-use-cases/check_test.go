// Package structusecases is the check `go/struct-use-cases`, run on the tree with the planted violations that prove it.
package structusecases

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// The rule counts value, pointer and embedded fields of use case types, and not a field of another type.
func TestStructUseCaseRule_RejectsAStructOfMoreThanTheLimit(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"usecase/find/find.go":   "package find\n\ntype Find struct{}\n",
		"usecase/store/store.go": "package store\n\ntype Store struct{}\n",
		"adapter/window/window.go": `package window

import (
	"fixture/usecase/find"
	"fixture/usecase/store"
)

type Crowded struct {
	first  *find.Find
	second *find.Find
	third  *store.Store
	fourth *store.Store
	fifth  find.Find
	store.Store
}

type Full struct {
	first  *find.Find
	second *find.Find
	third  *store.Store
	fourth *store.Store
	fifth  *store.Store
	name   string
	names  []string
}
`,
	})
	found, count := getCrowdedStructs(load.LoadFixture(t, dir), 5)
	if want := []string{"window.go:8 window.Crowded holds 6 use cases"}; !slices.Equal(load.FormatFindings(found), want) {
		t.Errorf("the rule rejects %v, want %v", load.FormatFindings(found), want)
	}
	if count.useCases != 11 {
		t.Errorf("%d use case fields read, want 11", count.useCases)
	}
}

func TestStructUseCaseRule_CountsMethodValuesAndFunctionFields(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"usecase/find/find.go": `package find

type Find struct{}

func NewFind() *Find { return &Find{} }

func (*Find) Execute() {}

func (*Find) Look() {}
`,
		"adapter/window/window.go": `package window

import "fixture/usecase/find"

type Wired struct {
	one   func()
	two   func()
	three func()
	four  func()
	five  func()
	six   func()
	plain func()
}

func New(registry *find.Find) *Wired {
	wired := &Wired{
		one:   registry.Look,
		two:   find.NewFind().Execute,
		three: registry.Execute,
		plain: func() {},
	}
	wired.four = registry.Look
	return wired
}

func newPositional(registry *find.Find) Wired {
	return Wired{nil, nil, nil, nil, registry.Execute, registry.Look, nil}
}
`,
	})
	found, count := getCrowdedStructs(load.LoadFixture(t, dir), 5)
	if want := []string{"window.go:5 window.Wired holds 6 use cases"}; !slices.Equal(load.FormatFindings(found), want) {
		t.Errorf("the rule rejects %v, want %v", load.FormatFindings(found), want)
	}
	if count.useCases != 6 {
		t.Errorf("%d use case fields read, want 6", count.useCases)
	}
}

func TestNoStructHoldsMoreThanFiveUseCases(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	checkNoStructHoldsMoreThanFiveUseCases(t, load.Workspace(t))
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
