package indexfile_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport/indexfile"
)

// A copy is at the schema the migrations leave behind, and says what ran, so a
// migration added later starts from where a real index would.
func TestACopyIsAMigratedIndex(t *testing.T) {
	t.Parallel()

	copied := indexfile.Path(t)

	scratch := filepath.Join(t.TempDir(), "index.db")
	opened, err := index.Open(t.Context(), scratch)
	if err != nil {
		t.Fatal(err)
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}

	if held, want := versionOf(t, copied), versionOf(t, scratch); held != want {
		t.Errorf("a copy is at schema %d, and an index opened from nothing is at %d", held, want)
	}
	held, want := readAppliedMigrations(t, copied), readAppliedMigrations(t, scratch)
	if len(want) == 0 {
		t.Fatal("an index opened from nothing records nothing as applied")
	}
	if !equal(held, want) {
		t.Errorf("a copy says %v was applied, and an index opened from nothing says %v", held, want)
	}
}

// Two copies are two databases: what is written to one is not in the other.
func TestCopiesAreIndependent(t *testing.T) {
	t.Parallel()

	one, err := index.Open(t.Context(), indexfile.Path(t), indexfile.SkipFlush())
	if err != nil {
		t.Fatal(err)
	}
	defer one.Close()

	other := indexfile.Path(t)
	two, err := index.Open(t.Context(), other)
	if err != nil {
		t.Fatal(err)
	}
	defer two.Close()

	if err := one.Vaults().Register(t.Context(), "abcdefghijklmnop"); err != nil {
		t.Fatal(err)
	}

	if held := vaultsIn(t, other); held != 0 {
		t.Errorf("the second copy holds %d vaults, and was written to only through the first", held)
	}
}

func versionOf(t *testing.T, path string) int {
	t.Helper()
	var version int
	if err := openFile(t, path).QueryRowContext(t.Context(), "PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	return version
}

func vaultsIn(t *testing.T, path string) int {
	t.Helper()
	var count int
	if err := openFile(t, path).QueryRowContext(t.Context(), "SELECT COUNT(*) FROM vaults").Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func readAppliedMigrations(t *testing.T, path string) []string {
	t.Helper()
	rows, err := openFile(t, path).QueryContext(t.Context(),
		"SELECT name FROM schema_migrations ORDER BY version")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		out = append(out, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func openFile(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
