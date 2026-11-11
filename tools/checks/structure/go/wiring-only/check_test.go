package wiringonly

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strconv"
	"strings"
	"testing"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/types/typeutil"
	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// The rule rejects an import of the machine, of randomness and of cryptography, a call that prints, and a
// function with more decisions than wiring takes, and accepts a function that selects and connects.
func TestWiringOnlyRule_RejectsTechnologyPrintingAndDecisions(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"container/wire.go": `package container

import (
	"crypto/rand"
	"fmt"
	"os"
)

func Connect(a, b func() int) int { return a() + b() }

func Guard(a func() int) int {
	if a == nil {
		return 0
	}
	return a()
}

func Choose(values []int, limit int) int {
	total := 0
	for _, value := range values {
		if value > limit && value%2 == 0 {
			total += value
		}
	}
	return total
}

func Print() {
	fmt.Println(os.Args, rand.Reader)
}
`,
		"container/wire_other.go": "package container\n\nfunc Other() {}\n",
		"other/other.go": `package other

import "os"

func Read() []string { return os.Args }
`,
	})
	found, functions := getWiringFindings(load.LoadFixture(t, dir), "fixture/container")
	want := []string{
		"wire.go:18 Choose has complexity 4",
		"wire.go:29 calls fmt.Println",
		"wire.go:4 imports crypto/rand",
		"wire.go:6 imports os",
	}
	if got := load.FormatFindings(found); !slices.Equal(got, want) || functions != 5 {
		t.Errorf("the rule rejects %v with %d functions read, want %v with 5", got, functions, want)
	}
}

func TestTheCompositionRootHoldsWiringOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	checkTheCompositionRootHoldsWiringOnly(t, load.Workspace(t, "libs/core/kernel"))
}
