package mutableusecase

import (
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"golang.org/x/tools/go/packages"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func TestWithMethodRule_RejectsAUseCaseChangedAfterConstruction(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"usecase/find/find.go": `package find

type Find struct{ limit int }

func (*Find) Execute() {}

func (f *Find) WithLimit(limit int) { f.limit = limit }

func (f Find) WithCopy(limit int) Find { f.limit = limit; return f }

func (f *Find) WithName(name string) string { return name }

func (f *Find) Limit() int { return f.limit }

type Helper struct{ limit int }

func (h *Helper) WithLimit(limit int) { h.limit = limit }
`,
	})
	found, useCases := listMutatingWithMethods(load.LoadFixture(t, dir))
	want := []string{
		"find.go:7 Find.WithLimit changes the use case after construction",
		"find.go:9 Find.WithCopy changes the use case after construction",
	}
	if got := load.FormatFindings(found); !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
	if useCases != 1 {
		t.Errorf("%d use cases read, want 1", useCases)
	}
}

func TestFunctionFieldRule_RejectsAnAssignmentAfterConstruction(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"adapter/store/store.go": `package store

type Store struct {
	Notify func()
	unexportedHook func()
	Name   string
}

func NewStore() *Store {
	store := &Store{}
	store.Notify = func() {}
	return store
}

func (s *Store) Attach(notify func()) { s.Notify = notify }

func (s *Store) Rename(name string) { s.Name = name; s.unexportedHook = nil }
`,
		"container/wire.go": `package container

import "fixture/adapter/store"

func Wire() *store.Store {
	fresh := store.NewStore()
	fresh.Notify = nil
	return fresh
}
`,
	})
	found, fields := listLateFunctionFields(load.LoadFixture(t, dir))
	want := []string{
		"store.go:15 assigns the function field Notify after construction",
		"wire.go:7 assigns the function field Notify after construction",
	}
	if got := load.FormatFindings(found); !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
	if fields != 1 {
		t.Errorf("%d function fields read, want 1", fields)
	}
}

func TestConstructorsReportCounts(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks every module of the workspace")
	}
	workspace := load.Workspace(t)
	kernelDir, err := filepath.Abs(filepath.Join(load.RepoRootDir, "libs/core/kernel"))
	if err != nil {
		t.Fatal(err)
	}
	withError, withoutError := 0, 0
	for _, pkg := range load.ListProductionPackages(workspace.ListModule(t, kernelDir)) {
		if !isUseCaseOrAdapterPath(pkg.PkgPath) {
			continue
		}
		for _, file := range pkg.Syntax {
			for _, declaration := range file.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok || function.Recv != nil || !strings.HasPrefix(function.Name.Name, "New") {
					continue
				}
				results := function.Type.Results
				if results != nil && len(results.List) > 1 && types.Identical(pkg.TypesInfo.TypeOf(results.List[len(results.List)-1].Type), types.Universe.Lookup("error").Type()) {
					withError++
				} else {
					withoutError++
				}
			}
		}
	}
	t.Logf("constructors of use cases and adapters that return an error: %d, that do not: %d", withError, withoutError)
}

func TestNoUseCaseHasAMutatingWithMethod(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	checkNoUseCaseHasAMutatingWithMethod(t, load.Workspace(t))
}

func TestNoFunctionFieldIsAssignedAfterConstruction(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	checkNoFunctionFieldIsAssignedAfterConstruction(t, load.Workspace(t))
}
