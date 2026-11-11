package portcalls

import (
	"go/ast"
	"go/types"
	"slices"
	"testing"
	"golang.org/x/tools/go/packages"
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

func TestAPortDeclaresAndDoesNotOrchestrate(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	checkPortDeclaresAndDoesNotOrchestrate(t, load.Workspace(t, "libs/core/kernel"))
}
