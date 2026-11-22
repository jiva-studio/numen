// Package usecaseports is the check `go/use-case-ports`, run on the tree with the planted violations that prove it.
package usecaseports

import (
	"os"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func TestInterfaceRule_RejectsAUseCaseOfAnotherPackage(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"usecase/vault/open.go": `package vault

type Open struct{}

func (Open) Execute() error { return nil }

type Opener interface{ Execute() error }

type Close struct{}

func (*Close) ExecuteAll() {}

type Closer interface{ ExecuteAll() }

type Name interface{ Name() string }
`,
		"usecase/window/start.go": `package window

import "fixture/usecase/vault"

type Opener interface{ Execute() error }

type Start struct{ open Opener }

func NewStart(open vault.Open) Start { return Start{open: open} }

func (Start) Execute() {}

type Namer interface{ Name() string }
`,
		"port/noteport/reader.go": `package noteport

type Reader interface{ ExecuteAll() }

type Clock interface{ Now() int }
`,
	})
	found, interfaces, useCases := listUseCasesBehindInterfaces(load.LoadFixture(t, dir))
	want := []string{
		"reader.go:3 Reader is implemented by the use case vault.Close",
		"start.go:5 Opener is implemented by the use case vault.Open",
	}
	slices.Sort(want)
	if got := load.FormatFindings(found); !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
	if interfaces != 7 || useCases != 3 {
		t.Errorf("%d interfaces and %d use cases read, want 7 and 3", interfaces, useCases)
	}
}

func TestFunctionRule_RejectsAMethodValueOfAnotherUseCasePackage(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"usecase/vault/open.go": `package vault

type Open struct{}

func (Open) Execute() error { return nil }

func (Open) Check() error { return nil }
`,
		"usecase/window/start.go": `package window

type Start struct {
	Open  func() error
	Other func() error
}

func NewStart(open func() error) Start { return Start{Open: open} }

func (Start) Execute() {}

func (Start) ExecuteSame() func() { return Start{}.Execute }
`,
		"adapter/api/api.go": `package api

import "fixture/usecase/vault"

type Api struct{ Open func() error }

func New() Api { return Api{Open: vault.Open{}.Execute} }
`,
		"container/wire.go": `package container

import (
	"fixture/usecase/vault"
	"fixture/usecase/window"
)

func Wire() window.Start {
	open := vault.Open{}.Execute
	check := vault.Open{}.Check
	_ = check
	start := window.NewStart(open)
	start.Other = vault.Open{}.Execute
	return window.Start{Open: vault.Open{}.Execute, Other: check}
}
`,
	})
	found, values := listFunctionSeams(load.LoadFixture(t, dir))
	want := []string{
		"wire.go:12 vault.Execute is passed as a function to the function NewStart of the use case package window",
		"wire.go:13 vault.Execute is passed as a function to the field Other of the use case package window",
		"wire.go:14 vault.Execute is passed as a function to the field Open of the use case package window",
	}
	slices.Sort(want)
	if got := load.FormatFindings(found); !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
	if values != 5 {
		t.Errorf("%d method values read, want 5", values)
	}
}

func TestNoUseCaseImplementsAnInterfaceOfAnotherPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the kernel")
	}
	checkNoUseCaseIsBehindAnInterface(t, load.Workspace(t, "libs/core/kernel"))
}

func TestNoUseCaseIsPassedAsAFunction(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the kernel")
	}
	checkNoUseCaseIsPassedAsAFunction(t, load.Workspace(t, "libs/core/kernel"))
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
