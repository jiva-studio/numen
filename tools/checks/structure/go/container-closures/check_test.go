// Package containerclosures is the check `go/container-closures`, run on the tree with the planted violations that prove it.
package containerclosures

import (
	"os"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func TestCapturedAssignmentRule_RejectsAClosureAssigningACapturedVariable(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"container/wire.go": `package container

type Config struct{ Versions []string }

var global int

func Wire(c *Config) func() {
	count := 0
	local := func() {
		inner := 0
		inner++
		var other []int
		other = append(other, inner)
		_ = other
	}
	local()
	return func() {
		c.Versions = nil
		count++
		global = 1
		a, b := 1, 2
		a = b
		count, a = a, count
	}
}
`,
	})
	found := listCapturedAssignments(load.LoadFixture(t, dir)[0])
	want := []string{
		"wire.go:18 assigns c, a variable the function literal captures",
		"wire.go:19 assigns count, a variable the function literal captures",
		"wire.go:23 assigns count, a variable the function literal captures",
	}
	if got := load.FormatFindings(found); !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
}

func TestGenericFunctionRule_RejectsAFunctionWithATypeParameter(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"container/wire.go": `package container

func Set[T any](target *T, value T) { *target = value }

func Plain(target *int) { *target = 1 }

type Box[T any] struct{ value T }

func (b Box[T]) Get() T { return b.value }
`,
	})
	found, functions := listGenericFunctions(load.LoadFixture(t, dir)[0])
	want := []string{"wire.go:3 Set has a type parameter"}
	if got := load.FormatFindings(found); !slices.Equal(got, want) {
		t.Errorf("the rule rejects %v, want %v", got, want)
	}
	if functions != 3 {
		t.Errorf("%d functions read, want 3", functions)
	}
}

func TestTheContainerAssignsNoCapturedVariable(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	checkTheContainerAssignsNoCapturedVariable(t, load.Workspace(t, "libs/core/kernel"))
}

func TestTheContainerDeclaresNoGenericFunction(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	checkTheContainerDeclaresNoGenericFunction(t, load.Workspace(t, "libs/core/kernel"))
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
