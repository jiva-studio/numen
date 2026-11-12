package index

import (
	"database/sql"
	"embed"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func getProbe() migration {
	return migration{
		version: 2,
		name:    "0002_probe.sql",
		body:    "CREATE TABLE probe (x INTEGER);\nINSERT INTO probe (x) VALUES (1);",
	}
}

func containing(text string) func(string) bool {
	return func(query string) bool { return strings.Contains(query, text) }
}

func openAtVersionOne(t *testing.T) (*sql.DB, *faults) {
	t.Helper()
	db, faults := openBreakingDB(t, filepath.Join(t.TempDir(), "index.db"))
	start := migration{version: 1, name: "0001_start.sql", body: "CREATE TABLE start (x INTEGER);"}
	if err := apply(t.Context(), db, start); err != nil {
		t.Fatal(err)
	}
	return db, faults
}

func readUserVersion(t *testing.T, db *sql.DB) int {
	t.Helper()
	var version int
	if err := db.QueryRowContext(t.Context(), "PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	return version
}

func hasTable(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var n int
	if err := db.QueryRowContext(t.Context(),
		`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, name).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n == 1
}

// A migration whose any step fails is reported with what failed, and the index
// stays at the version it was at, holding nothing of the migration.
func TestAMigrationStepThatFailsPreservesTheIndexAtItsVersion(t *testing.T) {
	for _, step := range []struct {
		name   string
		breaks func(string) bool
	}{
		{"making the bookkeeping", containing("CREATE TABLE IF NOT EXISTS schema_migrations")},
		{"beginning the transaction", func(query string) bool { return query == "BEGIN" }},
		{"a statement of the migration", containing("CREATE TABLE probe")},
		{"writing the version", containing("PRAGMA user_version = 2")},
		{"recording what ran", containing("INSERT INTO schema_migrations (")},
	} {
		t.Run(step.name, func(t *testing.T) {
			db, faults := openAtVersionOne(t)
			faults.breakWhere(t, step.breaks)

			err := apply(t.Context(), db, getProbe())
			faults.stop()
			if !errors.Is(err, errBroken) {
				t.Fatalf("the migration answered %v", err)
			}
			if pos := readUserVersion(t, db); pos != 1 {
				t.Errorf("the index is at version %d, want 1", pos)
			}
			if hasTable(t, db, "probe") {
				t.Error("the failed migration left its table behind")
			}
		})
	}
}

// An index whose version cannot be read is reported, and nothing is built in it.
func TestAnIndexWhoseVersionCannotBeReadIsNotMigrated(t *testing.T) {
	db, faults := openBreakingDB(t, filepath.Join(t.TempDir(), "index.db"))
	faults.breakWhere(t, func(query string) bool { return query == "PRAGMA user_version" })

	err := migrate(t.Context(), db, "index.db")
	faults.stop()
	if !errors.Is(err, errBroken) {
		t.Fatalf("migrating answered %v", err)
	}
	if hasTable(t, db, "vaults") {
		t.Error("the migrations ran over a version nobody read")
	}
}

func TestAFailedMigrationIsNamedAndCarriesItsCause(t *testing.T) {
	db, faults := openBreakingDB(t, filepath.Join(t.TempDir(), "index.db"))
	faults.breakOn(t, "PRAGMA user_version = 1")

	err := migrate(t.Context(), db, "index.db")
	faults.stop()
	if !errors.Is(err, errBroken) {
		t.Fatalf("migrating answered %v", err)
	}
	if !strings.Contains(err.Error(), "0001_index.sql") {
		t.Errorf("the report %q does not name the migration", err)
	}
	if pos := readUserVersion(t, db); pos != 0 {
		t.Errorf("the index is at version %d, want 0", pos)
	}
}

// A file name that is not a migration's names no version.
func TestAFileNameThatIsNotAMigrationsProducesVersionZero(t *testing.T) {
	for _, name := range []string{"initial.sql", "abc_initial.sql", "0000_zero.sql", "-3_negative.sql"} {
		if v, err := parseVersion(name); err == nil || v != 0 {
			t.Errorf("%q gave %d, %v", name, v, err)
		}
	}
}

// Comment lines go and every other line stays, each on a line of its own.
func TestCommentsAreTakenOutLineByLine(t *testing.T) {
	got := stripComments("SELECT 1\n  -- note\nFROM x")
	if got != "SELECT 1\nFROM x\n" {
		t.Errorf("got %q", got)
	}
}

func TestMigrationsThatCannotBeReadMigrateNothing(t *testing.T) {
	db, _ := openBreakingDB(t, filepath.Join(t.TempDir(), "index.db"))
	saved := files
	files = embed.FS{}
	t.Cleanup(func() { files = saved })

	if _, err := loadMigrations(); err == nil {
		t.Fatal("migrations were loaded from nothing")
	}
	if err := migrate(t.Context(), db, "index.db"); err == nil {
		t.Fatal("an index was migrated by nothing")
	}
	if pos := readUserVersion(t, db); pos != 0 {
		t.Errorf("the index is at version %d, want 0", pos)
	}
}
