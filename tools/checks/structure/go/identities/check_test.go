// Package identities is the check `go/identities`, run on the tree with the planted violations that prove it.
package identities

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

// TestCoreIdentitiesUseDomainTypes checks that every identity in the core is a type of the core's own.
// A row number belongs to the store: it is assigned by whatever counts rows and
// can be ordered and paged, which the entity it names cannot. A core carrying
// one imports the store's numbering into scenario logic, such as a use case
// looping over ascending integer keys.
func TestCoreIdentitiesUseDomainTypes(t *testing.T) {
	wrong, read := getNumberedIdentities(t, load.ListTypeRoots())
	for _, one := range wrong {
		t.Error(one + ": an identity the core carries is a word of its own")
	}

	// A walk that read no identity is a rule checked against nothing, and it
	// passes.
	if read == 0 {
		t.Fatal("the domain and port folders name no identity: the walk is not reading them")
	}
}

func getNumberedIdentities(t *testing.T, roots []load.Root) (wrong []string, read int) {
	t.Helper()
	for _, root := range roots {
		for _, file := range load.ParseSourceFiles(t, root.Dir) {
			for _, found := range getIdentities(file) {
				read++
				if getNumberedTypes()[found.Spelled] {
					wrong = append(wrong, fmt.Sprintf("%s/%s.%s is a %s",
						root.Name, found.In, found.Field, found.Spelled))
				}
			}
		}
	}
	return wrong, read
}

// The walk once read libs/note/domain in place of the kernel's domain, and an
// integer identity there passed.
func TestCoreIdentities_ReadKernelDomain(t *testing.T) {
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
	source := "package vault\n\ntype Vault struct {\n\tID int64\n}\n"
	if err := os.WriteFile(filepath.Join(planted, "vault", "vault.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	wrong, _ := getNumberedIdentities(t, []load.Root{{Name: "kernel/domain", Dir: planted}})
	if !slices.Equal(wrong, []string{"kernel/domain/Vault.ID is a int64"}) {
		t.Errorf("the walk rejected %v, want the planted Vault.ID", wrong)
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

// has to find the field and the type it is spelled in, and let through the ones
// that are the core's own word and the ones that name no identity at all.
func TestIdentityRule_RejectsIdentityFields(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "identity.go", `package p

type Thing struct {
	ID      VaultID
	ChunkID int64
	OtherID domain.ChunkID
	Rows    int64
	chunkID int64
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	var rejected []string
	for _, one := range getIdentities(file) {
		if getNumberedTypes()[one.Spelled] {
			rejected = append(rejected, one.Field)
		}
	}
	if !slices.Equal(rejected, []string{"ChunkID"}) {
		t.Errorf("the rule rejected %v, want the one identity spelled as a number", rejected)
	}
}

// The same rule applied to port signatures. An identity passed through a port as
// a number brings the store's numbering into the core's language, and a signature
// is where the crossing occurs.
func TestPortMethodIdentitiesUseDomainTypes(t *testing.T) {
	var wrong []string
	var read int
	dir := filepath.Join("..", "port")
	for _, file := range load.ParseSourceFiles(t, dir) {
		for _, found := range getMethodIdentities(file) {
			read++
			if getNumberedTypes()[found.Spelled] {
				wrong = append(wrong, fmt.Sprintf("port/%s(%s %s)",
					found.In, found.Field, found.Spelled))
			}
		}
	}
	for _, one := range wrong {
		t.Error(one + ": an identity a port asks for is a word of the core's own")
	}

	// A walk that read no identity is a rule checked against nothing, and it
	// passes.
	if read == 0 {
		t.Fatal("port/ asks for no identity by name: the walk is not reading it")
	}
}

// checks parameters and results alike, accepts the type that is the core's own,
// and ignores unnamed parameters.
func TestIdentityRule_RejectsIdentitySignatureParams(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "asking.go", `package p

import "context"

type Store interface {
	Read(ctx context.Context, vaultID domain.VaultID, chunkID int64) error
	Write(ctx context.Context, rowID int) (itemID domain.ItemID, err error)
	Count(context.Context, int64) int
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	var rejected []string
	for _, one := range getMethodIdentities(file) {
		if getNumberedTypes()[one.Spelled] {
			rejected = append(rejected, one.Field)
		}
	}
	if !slices.Equal(rejected, []string{"chunkID", "rowID"}) {
		t.Errorf("the rule rejected %v, want the two identities spelled as numbers", rejected)
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
