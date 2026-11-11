// Package drivingoneusecase is the check `go/driving-one-use-case`, run on the tree with the planted violations that prove it.
package drivingoneusecase

import (
	"os"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// A driving adapter maps a request, calls one use case and maps the result. A function that reaches two use
// cases, itself or through the functions of its package, decides between them or sequences them, which is a
// scenario and belongs in a use case. An adapter drives the core when it imports a use case package, wherever
// it lies below adapter/ and internal/adapter/. A use case is reached through a field of a type a use case
// package declares, through an interface with an Execute method, through a field that holds a function of
// the signature of an Execute method, and through an Execute method taken as a value.
func TestDrivingFunctionCallsOneUseCase(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks libs/core/kernel")
	}
	pkgs := load.GetModulePackages(t, load.Workspace(t, "libs/core/kernel"), "libs/core/kernel")
	found, functions := load.ListManyUseCaseCalls(pkgs, load.CoreImportPath)
	for _, one := range found {
		t.Errorf("%s %s: a driving adapter calls one use case", one.Position, one.Name)
	}
	if functions == 0 {
		t.Fatalf("%d driving-adapter functions reach a use case: the walk is not reading them", functions)
	}
}

func TestDrivingCallRule_RejectsAFunctionCallingTwoUseCases(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"usecase/find/find.go": "package find\n\ntype Find struct{}\n\nfunc (Find) Execute() {}\n\ntype Save struct{}\n\nfunc (Save) Execute() {}\n",
		"adapter/window/one/one.go": `package one

import "fixture/usecase/find"

type Reader interface{ Execute() }

type Helper struct{ save *find.Save }

func (h *Helper) Write() { h.save.Execute() }

type Handler struct {
	find   find.Find
	save   *find.Save
	reader Reader
	helper *Helper
	other  int
}

func (h *Handler) Both()    { h.find.Execute(); h.save.Execute() }
func (h *Handler) Passed()  { run(h.find.Execute, h.reader.Execute) }
func (h *Handler) Through() { h.find.Execute(); h.helper.Write() }
func (h *Handler) Above()   { h.Through() }
func (h *Handler) Again()   { h.find.Execute(); h.find.Execute() }
func (h *Handler) One()     { h.save.Execute() }
func (h *Handler) None()    { h.other++ }

func (h *Handler) Handed() { hand(h.find, h.save) }

func run(...func()) {}

func hand(...any) {}
`,
	})
	found, functions := load.ListManyUseCaseCalls(load.LoadFixture(t, dir), "fixture")
	want := []string{"one.go:19 Both calls find, save", "one.go:20 Passed calls find, reader", "one.go:21 Through calls find, save", "one.go:27 Handed calls find, save"}
	if !slices.Equal(load.FormatFindings(found), want) {
		t.Errorf("the rule refuses %v, want %v", load.FormatFindings(found), want)
	}
	if functions != 8 {
		t.Errorf("%d functions reach a use case, want 8", functions)
	}
}

// The driving adapters are the packages that import a use case package, not a list of folders; the use cases
// a function reaches count through a field that holds a function of an Execute signature and through an
// Execute method taken as a value, and a callback of another signature is no use case.
func TestDrivingCallRule_CountsUseCasesReachedAsFunctionValues(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"usecase/note/note.go": `package note

import "context"

type Open struct{}

func (Open) Execute(ctx context.Context, id string) (string, error) { return id, nil }

type Close struct{}

func (Close) Execute(ctx context.Context, id string) error { return nil }
`,
		"adapter/anywhere/deep/deep.go": `package deep

import (
	"context"

	"fixture/usecase/note"
)

type Handler struct {
	open     func(ctx context.Context, id string) (string, error)
	close    func(ctx context.Context, id string) error
	onChange func(id string)
	opening  note.Open
}

func (h *Handler) Function() {
	_, _ = h.open(context.Background(), "a")
	_ = h.close(context.Background(), "a")
}

func (h *Handler) Callback() {
	h.onChange("a")
	_ = h.close(context.Background(), "a")
}

func (h *Handler) Value() {
	var closing note.Close
	run(closing.Execute, h.opening.Execute)
}

func (h *Handler) Method() {
	_ = note.Open.Execute
	_ = note.Close.Execute
}

func (h *Handler) Local() {
	var closing note.Close
	_ = closing.Execute(context.Background(), "a")
	_ = closing.Execute(context.Background(), "b")
}

func run(...any) {}
`,
		"adapter/free/free.go": `package free

import "context"

type Handler struct {
	open  func(ctx context.Context, id string) (string, error)
	close func(ctx context.Context, id string) error
}

func (h *Handler) Both() { _, _ = h.open(context.Background(), ""); _ = h.close(context.Background(), "") }
`,
	})
	found, _ := load.ListManyUseCaseCalls(load.LoadFixture(t, dir), "fixture")
	want := []string{
		"deep.go:16 Function calls close, open",
		"deep.go:26 Value calls note.Close.Execute, opening",
		"deep.go:31 Method calls note.Close.Execute, note.Open.Execute",
	}
	if !slices.Equal(load.FormatFindings(found), want) {
		t.Errorf("the rule refuses %v, want %v", load.FormatFindings(found), want)
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
