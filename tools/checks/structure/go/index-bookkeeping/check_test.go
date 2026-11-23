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

// The rule rejects a statement in a string literal that drops a table it names at run time, changes the migration
// table, reads its shape or names a copy of it, and passes a drop of a named table, the statements that create,
// record and read the migration table, a delete of rows, the same words in a comment and a test file.
func TestBookkeepingRule_RejectsRewritesOfTheIndex(t *testing.T) {
	root := load.WriteTreeFixture(t, map[string]string{
		"migrate.go": "package index\n\n// DROP TABLE and ALTER TABLE schema_migrations are not run.\n\n" +
			"const create = `CREATE TABLE IF NOT EXISTS schema_migrations (\n\tversion INTEGER PRIMARY KEY\n)`\n\n" +
			"const record = `INSERT INTO schema_migrations (version) VALUES (?)`\n\n" +
			"const ran = `SELECT version FROM schema_migrations`\n\n" +
			"const clear = `DELETE FROM chunks WHERE id = ?`\n",
		"db.go": "package index\n\nconst next = `CREATE TABLE schema_migrations_next (version INTEGER)`\n\n" +
			"const shape = `SELECT name FROM pragma_table_info('schema_migrations')`\n",
		"chunk/chunk.go": "package chunk\n\nfunc drop(name string) string { return \"drop table \" + name }\n\n" +
			"const widen = \"ALTER TABLE schema_migrations ADD x\"\n",
		"vectors.go": "package index\n\nconst refit = `DROP TABLE IF EXISTS chunks_vec`\n",
		"db_test.go": "package index\n\nconst reset = `DROP TABLE chunks`\n",
	})
	wrong, read, err := findBookkeepingRewrites(root)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(wrong)
	want := []string{
		"chunk/chunk.go changes the migration table",
		"chunk/chunk.go drops a table it names at run time",
		"db.go changes the migration table",
		"db.go changes the migration table",
	}
	if !slices.Equal(wrong, want) {
		t.Errorf("the rule rejects %v, want %v", wrong, want)
	}
	if read != 4 {
		t.Errorf("%d files read, want 4", read)
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.RunTests(m))
}
