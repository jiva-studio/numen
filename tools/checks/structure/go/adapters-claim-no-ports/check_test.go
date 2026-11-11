// Package adaptersclaimnoports is the check `go/adapters-claim-no-ports`, run on the tree with the planted violations that prove it.
package adaptersclaimnoports

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// TestAdaptersDoNotClaimSatisfiedPorts checks that no adapter declares `var _ port.X = …`,
// which duplicates the binding the composition root owns.
// A method returning `port.X` is not such a declaration: where a port opens
// another (`VaultReaders.Open` returns a `VaultReader`), the return type is part
// of the port's signature. The rule rejects only free-standing assertions.
func TestAdaptersDoNotClaimSatisfiedPorts(t *testing.T) {
	wrong, err := findPortClaims("..")
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one)
	}
}

// TestPortClaimWalk_FindsThePlantedClaim verifies that the walk finds the claim of an adapter and
// passes the same line outside the adapters.
func TestPortClaimWalk_FindsThePlantedClaim(t *testing.T) {
	claim := "package p\n\nvar _ vaultport.VaultReader = (*Reader)(nil)\n"
	root := load.WriteTreeFixture(t, map[string]string{
		"adapter/index/index.go": claim,
		"usecase/note/note.go":   claim,
	})
	wrong, err := findPortClaims(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(wrong) != 1 || !strings.Contains(wrong[0], "adapter/index/index.go names port.VaultReader") {
		t.Errorf("the walk found %v, want the claim in adapter/index/index.go only", wrong)
	}
}

// TestPortClaimWalk_FailsOnATreeWithoutAdapters verifies that a root without an adapter file is an error, not a pass.
func TestPortClaimWalk_FailsOnATreeWithoutAdapters(t *testing.T) {
	if _, err := findPortClaims(t.TempDir()); err == nil {
		t.Error("a walk over a tree without an adapter passed")
	}
}

func findPortClaims(root string) ([]string, error) {
	var wrong []string
	var read int
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		pkg := load.GetPackage(root, path)
		if strings.HasSuffix(path, "_test.go") || !load.IsAdapter(pkg) && !load.IsShared(pkg) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		read++
		for _, one := range getClaimedPorts(file) {
			wrong = append(wrong, path+" names port."+one)
		}
		return nil
	})
	if err == nil && read == 0 {
		err = fmt.Errorf("no adapter file read under %s: the walk checks nothing", root)
	}
	return wrong, err
}

// shapes of the claim are one claim, and the walk has to reach the one written
// inside a function; a variable holding a port is left alone.
func TestPortClaimRule_RejectsPortClaims(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "claim.go", `package p

var _ vaultport.VaultReader = (*Reader)(nil)

var _, _ vaultport.VaultWriter = (*Writer)(nil), (*Writer)(nil)

var storePort vaultport.IndexClaim

func mount() {
	var _ noteport.NoteRefs = (*sound)(nil)
	var agentPort agentport.Agent
	_, _ = storePort, agentPort
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"VaultReader", "VaultWriter", "NoteRefs"}
	if got := getClaimedPorts(file); !slices.Equal(got, want) {
		t.Errorf("the rule refuses %v, want %v", got, want)
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
