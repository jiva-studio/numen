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

func TestNoTypeExceedsItsLayersSize(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	checkNoTypeExceedsItsLayersSize(t, load.Workspace(t, "libs/core/kernel"))
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
