// Package corereexport is the check `go/core-reexport`, run on the tree with the planted violations that prove it.
package corereexport

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// A re-export is an alias of a named type or of an unnamed interface, an exported variable or constant set
// to a member of another package of the repository, or an exported function that calls a function of
// another such package with its own parameters and has its signature. An alias of a type of its own package
// or of a package outside the repository, and a function with another signature, are accepted.
func TestReexportRule_RejectsAnAliasOfAnotherPackagesType(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"port/port.go": `package port

type Reader interface{ Read() string }

type Note struct{ Title string }

const Limit = 3

var Default = Note{}

func Open(name string) Reader { return nil }

func Close() {}
`,
		"contract/contract.go": `package contract

import (
	"strings"

	"fixture/port"
)

type Reader = port.Reader

type Own struct{}

type Local = Own

type Builder = strings.Builder

type Notes = []port.Note

type Anonymous = interface{ Read() string }

const Limit = port.Limit

var Default = port.Default

func Open(name string) port.Reader { return port.Open(name) }

func Close() { port.Close() }

func Trim(name string) port.Reader { return port.Open(name + "x") }

func Swap(a, b string) port.Reader { return port.Open(b) }

func Wrap(name string) any { return port.Open(name) }
`,
	})
	found, read := getReexports(load.LoadFixture(t, dir), func(path string) bool { return strings.HasPrefix(path, "fixture/") })
	want := []string{
		"contract.go:9 contract.Reader = fixture/port.Reader",
		"contract.go:19 contract.Anonymous is an alias of an unnamed interface",
		"contract.go:21 contract.Limit = fixture/port.Limit",
		"contract.go:23 contract.Default = fixture/port.Default",
		"contract.go:25 contract.Open forwards to fixture/port.Open",
		"contract.go:27 contract.Close forwards to fixture/port.Close",
	}
	got := load.FormatFindings(found)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
	if read != 8 {
		t.Errorf("%d type names read, want 8", read)
	}
}

func TestNoCoreTypeIsReexported(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	checkNoCoreTypeIsReexported(t, load.Workspace(t, "libs/core/kernel"))
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
