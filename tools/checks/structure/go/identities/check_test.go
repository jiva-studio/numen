// Package identities is the check `go/identities`, run on the tree with the planted violations that prove it.
package identities

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// TestCoreIdentitiesUseDomainTypes checks that every identity in the core is a type of the core's own.
// A row number belongs to the store: it is assigned by whatever counts rows and
// can be ordered and paged, which the entity it names cannot. A core carrying
// one imports the store's numbering into scenario logic, such as a use case
// looping over ascending integer keys.
func TestCoreIdentitiesUseDomainTypes(t *testing.T) {
	var wrong []string
	var read int
	for _, item := range []string{"domain", "port"} {
		dir := load.GetPackageDir(item)
		for _, file := range load.ParseSourceFiles(t, dir) {
			for _, found := range getIdentities(file) {
				read++
				if getNumberedTypes()[found.Spelled] {
					wrong = append(wrong, fmt.Sprintf("%s/%s.%s is a %s",
						item, found.In, found.Field, found.Spelled))
				}
			}
		}
	}
	for _, one := range wrong {
		t.Error(one + ": an identity the core carries is a word of its own")
	}

	// A walk that read no identity is a rule checked against nothing, and it
	// passes.
	if read == 0 {
		t.Fatal("domain/ and port/ name no identity: the walk is not reading them")
	}
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
