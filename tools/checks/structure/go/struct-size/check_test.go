// Package structsize is the check `go/struct-size`, run on the tree with the planted violations that prove it.
package structsize

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// The rule counts the methods a type takes from embedded interfaces and from embedded structs, counts a
// function-typed field as a field, and reads no interface and no type outside its layers.
func TestStructSizeRule_CountsPromotedMethodsAndFunctionFields(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"adapter/store/store.go": `package store

type Reader interface{ Read() string }

type Writer interface{ Write(string) }

type Both struct {
	Reader
	Writer
}

type Outer struct{ Both }

type Callbacks struct {
	A, B, C func()
}

type Small struct{ name string }

func (Small) One() {}

type Wide interface {
	Reader
	Writer
}
`,
		"domain/domain.go": `package domain

type Free struct{ A, B, C, D int }
`,
	})
	getLimit := func(path string) (structLimit, bool) {
		return structLimit{methods: 1, fields: 2}, strings.HasPrefix(path, "fixture/adapter")
	}
	found, read := getOversizedTypes(load.LoadFixture(t, dir), getLimit)
	want := []string{
		"store.go:12 store.Outer has 2 methods, more than 1",
		"store.go:14 store.Callbacks has 3 fields, more than 2",
		"store.go:7 store.Both has 2 methods, more than 1",
	}
	if got := load.FormatFindings(found); !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
	if read != 4 {
		t.Errorf("%d types read, want 4", read)
	}
}

func TestStructSizeRule_CountsFieldsOfEmbeddedStructs(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"adapter/store/store.go": `package store

type Places struct{ a, b int }

type Stores struct {
	Places
	c int
}

type Wiring struct {
	Stores
	d int
}

type Plain struct{ a, b int }
`,
	})
	getLimit := func(path string) (structLimit, bool) {
		return structLimit{methods: 100, fields: 3}, strings.HasPrefix(path, "fixture/adapter")
	}
	found, _ := getOversizedTypes(load.LoadFixture(t, dir), getLimit)
	want := []string{"store.go:10 store.Wiring has 4 fields, more than 3"}
	if got := load.FormatFindings(found); !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
}

// A type that groups its ports into sub-structs of its package counts their leaf fields, however deep the
// grouping goes; a struct of another package that a field holds stays one field.
func TestStructSizeRule_CountsLeafFieldsOfNamedStructFields(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"domain/domain.go": `package domain

type Note struct{ ID, Title, Body, Path string }
`,
		"usecase/session/session.go": `package session

import "fixture/domain"

type WindowPorts struct{ a, b, c int }

type VaultPorts struct {
	d    int
	Deep DeepPorts
}

type DeepPorts struct{ e, f int }

type Deps struct {
	Window WindowPorts
	Vault  *VaultPorts
	Note   domain.Note
	out    int
}

type Tree struct {
	Next *Tree
	a    int
}

type Flat struct{ a, b, c int }
`,
	})
	getLimit := func(path string) (structLimit, bool) {
		return structLimit{methods: 100, fields: 5}, strings.HasPrefix(path, "fixture/usecase")
	}
	found, _ := getOversizedTypes(load.LoadFixture(t, dir), getLimit)
	want := []string{"session.go:14 session.Deps has 8 fields, more than 5"}
	if got := load.FormatFindings(found); !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
}

// A container type counts the leaf fields of the groups it names, like a type of any other layer.
func TestStructSizeRule_CountsLeafFieldsOfGroupsInTheContainer(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"container/wiring.go": `package container

type A struct{ a1, a2, a3, a4, a5 int }
type B struct{ b1, b2, b3, b4, b5 int }
type C struct{ c1, c2, c3, c4, c5 int }

type Holder struct {
	A A
	B B
	C C
}

type Small struct {
	A A
	x int
}
`,
	})
	getLimit := func(path string) (structLimit, bool) {
		return structLimit{methods: 100, fields: 12}, path == "fixture/container"
	}
	found, _ := getOversizedTypes(load.LoadFixture(t, dir), getLimit)
	want := []string{"wiring.go:7 container.Holder has 15 fields, more than 12"}
	if got := load.FormatFindings(found); !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
}

func TestNoTypeExceedsItsLayersSize(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	checkNoTypeExceedsItsLayersSize(t, load.Workspace(t, "libs/core/kernel"))
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
