// Package portcalls is the check `go/port-calls`, run on the tree with the planted violations that prove it.
package portcalls

import (
	"os"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// The rule rejects a call through an interface in a function or method of a port, a promoted one and one
// inside a closure included, and accepts calls of functions and of concrete methods.
func TestPortCallRule_RejectsInterfaceCalls(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"port/noteport/noteport.go": `package noteport

import "strings"

type Reader interface{ Read(path string) string }

type Note struct{ Title string }

func (n Note) Words() int { return len(strings.Fields(n.Title)) }

func ReadTitle(r Reader, path string) Note { return Note{Title: r.Read(path)} }

type Wrapped struct{ Reader }

func (w Wrapped) Twice(path string) string {
	read := func() string { return w.Read(path) }
	return read() + read()
}

func Describe(err error) string { return err.Error() }

func Count(n Note) int { return n.Words() }
`,
		"usecase/read/read.go": `package read

import "fixture/port/noteport"

func Execute(r noteport.Reader) string { return r.Read("a.md") }
`,
	})
	found, read := getPortInterfaceCalls(load.LoadFixture(t, dir), "fixture")
	want := []string{
		"noteport.go:11 noteport.ReadTitle calls fixture/port/noteport.Reader.Read",
		"noteport.go:16 noteport.Twice calls fixture/port/noteport.Reader.Read",
		"noteport.go:20 noteport.Describe calls error.Error",
	}
	if got := load.FormatFindings(found); !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
	if read != 5 {
		t.Errorf("%d functions read, want 5", read)
	}
}

// A function of a use case package that returns a port is read as a function of a port; one that returns anything
// else, and the use case itself, is not.
func TestPortCallRule_ReadsAFunctionThatBuildsAPort(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"usecase/find/find.go": `package find

type Cache interface{ Get() string }

type Reader interface{ Read() string }

type Find struct{ cache Cache }

func New(c Cache) *Find { return &Find{cache: c} }

func (f *Find) Execute() string { return f.cache.Get() }

func Wrap(c Cache, r Reader) Cache { _ = r.Read(); return c }

func Count(r Reader) int { return len(r.Read()) }

func WrapArray(c Cache, r Reader) [1]Cache { _ = r.Read(); return [1]Cache{c} }

func WrapChannel(r Reader) <-chan Cache { _ = r.Read(); return nil }
`,
	})
	found, read := getPortInterfaceCalls(load.LoadFixture(t, dir), "fixture")
	want := []string{
		"find.go:13 find.Wrap calls fixture/usecase/find.Reader.Read",
		"find.go:17 find.WrapArray calls fixture/usecase/find.Reader.Read",
		"find.go:19 find.WrapChannel calls fixture/usecase/find.Reader.Read",
	}
	if got := load.FormatFindings(found); !slices.Equal(got, want) || read != len(want) {
		t.Errorf("the rule rejects %v of %d functions read, want %v of %d", got, read, want, len(want))
	}
}

func TestAPortDeclaresAndDoesNotOrchestrate(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	checkPortDeclaresAndDoesNotOrchestrate(t, load.LoadWorkspace(t, "libs/core/kernel"))
}

func TestMain(m *testing.M) {
	os.Exit(load.RunTests(m))
}
