// Package indexbookkeeping is the check `go/index-bookkeeping`, run on the tree with the planted violations that
// prove it. The index has one schema and migrates forward only: no code of it empties the index or reshapes its
// migration table.
package indexbookkeeping

import (
	"os"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func TestNoCodeOfTheIndexEmptiesItOrReshapesItsBookkeeping(t *testing.T) {
	wrong, read, err := findBookkeepingRewrites("../adapter/index")
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one)
	}
	if read == 0 {
		t.Fatal("no file of the index read: the rule is checked against nothing")
	}
}

// The rule rejects a rewriting statement in a string literal, a removed function of the index package and the
// removed identifier anywhere, and passes the same words in a comment, a method of a removed name, a removed
// function name in a subpackage and a test file.
func TestBookkeepingRule_RejectsRewritesOfTheIndex(t *testing.T) {
	root := load.WriteTreeFixture(t, map[string]string{
		"db.go": "package index\n\n// resetIndex and ALTER TABLE schema_migrations are gone.\n\n" +
			"const next = `CREATE TABLE schema_migrations_next (version INTEGER)`\n\n" +
			"func resetIndex() {}\n\ntype store struct{}\n\nfunc (store) getDropping() {}\n",
		"chunk/chunk.go": "package chunk\n\nfunc getDropping() {}\n\nfunc findDroppableSQL() string { return \"ALTER TABLE schema_migrations ADD x\" }\n",
		"db_test.go":     "package index\n\nfunc dropEverything() {}\n",
	})
	wrong, read, err := findBookkeepingRewrites(root)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(wrong)
	want := []string{
		`chunk/chunk.go holds "ALTER TABLE schema_migrations"`,
		"chunk/chunk.go names findDroppableSQL",
		`db.go declares resetIndex`,
		`db.go holds "schema_migrations_next"`,
	}
	if !slices.Equal(wrong, want) {
		t.Errorf("the rule rejects %v, want %v", wrong, want)
	}
	if read != 2 {
		t.Errorf("%d files read, want 2", read)
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
