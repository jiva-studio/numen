// Package instants is the check `go/instants`, run on the tree with the planted violations that prove it.
package instants

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// TestCoreInstantsUseTimeType checks that an instant in the domain and port folders is a time.Time.
// A second count and a millisecond count are both integers, so using one where
// the other is expected compiles and is off by a factor of a thousand; the zero
// of either is a valid instant in 1970.
func TestCoreInstantsUseTimeType(t *testing.T) {
	wrong, read := getUntimedInstants(t, load.ListTypeRoots())
	for _, one := range wrong {
		t.Error(one + ": an instant the core carries is a time.Time")
	}

	// A walk that read no instant is a rule checked against nothing, and it
	// passes.
	if read == 0 {
		t.Fatal("the domain and port folders name no instant: the walk is not reading them")
	}
}

func getUntimedInstants(t *testing.T, roots []load.Root) (wrong []string, read int) {
	t.Helper()
	for _, root := range roots {
		for _, file := range load.ParseSourceFiles(t, root.Dir) {
			for _, found := range getInstants(file) {
				read++
				if found.Spelled != "time.Time" {
					wrong = append(wrong, fmt.Sprintf("%s/%s.%s is a %s",
						root.Name, found.In, found.Field, found.Spelled))
				}
			}
		}
	}
	return wrong, read
}

// The walk once read libs/note/domain in place of the kernel's domain, and an
// integer instant there passed.
func TestCoreInstants_ReadKernelDomain(t *testing.T) {
	dir, err := filepath.Abs(getKernelDomain(t).Dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(filepath.ToSlash(dir), "libs/core/kernel/domain") {
		t.Fatalf("kernel/domain is read from %s, want libs/core/kernel/domain", dir)
	}

	planted := t.TempDir()
	if err := os.MkdirAll(filepath.Join(planted, "vault"), 0o755); err != nil {
		t.Fatal(err)
	}
	source := "package vault\n\ntype Visit struct {\n\tSeenTime int64\n}\n"
	if err := os.WriteFile(filepath.Join(planted, "vault", "visit.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	wrong, _ := getUntimedInstants(t, []load.Root{{Name: "kernel/domain", Dir: planted}})
	if !slices.Equal(wrong, []string{"kernel/domain/Visit.SeenTime is a int64"}) {
		t.Errorf("the walk rejected %v, want the planted Visit.SeenTime", wrong)
	}
}

func getKernelDomain(t *testing.T) load.Root {
	t.Helper()
	for _, root := range load.ListTypeRoots() {
		if root.Name == "kernel/domain" {
			return root
		}
	}
	t.Fatal("the walk lists no kernel/domain")
	return load.Root{}
}

// finds the field and the type it is written in, and lets through the one that
// is a time and the ones that name no instant at all.
func TestInstantRule_RejectsTimeFields(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "instant.go", `package p

import "time"

type Reading struct {
	ModTime  time.Time
	SeenTime int64
	Pages    int
	runTime  int64
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	var rejected []string
	for _, one := range getInstants(file) {
		if one.Spelled != "time.Time" {
			rejected = append(rejected, one.Field)
		}
	}
	if !slices.Equal(rejected, []string{"SeenTime"}) {
		t.Errorf("rejected %v, want SeenTime", rejected)
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
