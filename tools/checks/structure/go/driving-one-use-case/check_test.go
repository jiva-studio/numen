// Package drivingoneusecase is the check `go/driving-one-use-case`, run on the tree with the planted violations that prove it.
package drivingoneusecase

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

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

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
