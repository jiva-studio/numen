package adaptersclaimnoports

import (
	"go/parser"
	"go/token"
	"io/fs"
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
	var wrong []string
	var read int
	reached := map[string]bool{}
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		pkg := load.GetPackage("..", path)
		if strings.HasSuffix(path, "_test.go") || !load.IsAdapter(pkg) && !load.IsShared(pkg) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		read++
		reached[load.GetFamily(pkg)] = true
		for _, one := range getClaimedPorts(file) {
			wrong = append(wrong, path+" names port."+one)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one)
	}

	// A walk that reads no file of an adapter checks nothing and passes.
	if read < 50 {
		t.Fatalf("%d files of the adapters read: the walk is not reading them", read)
	}
	if len(reached) < 10 {
		t.Fatalf("%d adapters reached: the walk stops short of the adapters", len(reached))
	}
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
