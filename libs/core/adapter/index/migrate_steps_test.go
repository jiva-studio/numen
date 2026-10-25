package index

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// probe is a migration of these tests' own, applied over an index at version 1.
var probe = migration{
	version: 2,
	name:    "0002_probe.sql",
	body:    "CREATE TABLE probe (x INTEGER);\nINSERT INTO probe (x) VALUES (1);",
}

// containing holds for every statement carrying the text given.
func containing(text string) func(string) bool {
	return func(query string) bool { return strings.Contains(query, text) }
}

// stopBreaking stops the breaking driver failing anything.
func stopBreaking() { breaksOn.Store(nil) }

// openAtVersionOne is a new index file, through the breaking driver, that one
// migration of the test's own has been applied to.
func openAtVersionOne(t *testing.T) *sql.DB {
	t.Helper()
	db := openBreaking(t, filepath.Join(t.TempDir(), "index.db"))
	start := migration{version: 1, name: "0001_start.sql", body: "CREATE TABLE start (x INTEGER);"}
	if err := apply(t.Context(), db, start); err != nil {
		t.Fatal(err)
	}
	return db
}

// openLegacyBookkeeping is a new index file, through the breaking driver, whose
// schema_migrations carries a column this build does not write.
func openLegacyBookkeeping(t *testing.T) *sql.DB {
	t.Helper()
	db := openBreaking(t, filepath.Join(t.TempDir(), "index.db"))
	for _, statement := range []string{
		`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at TEXT)`,
		`INSERT INTO schema_migrations (version, name, applied_at) VALUES (1, '0001_index.sql', '2024-01-01')`,
	} {
		if _, err := db.ExecContext(t.Context(), statement); err != nil {
			t.Fatal(err)
		}
	}
	return db
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

// readColumns is the columns of a table, in the order they were declared.
func readColumns(t *testing.T, db *sql.DB, table string) string {
	t.Helper()
	var columns string
	if err := db.QueryRowContext(t.Context(),
		`SELECT group_concat(name, ',') FROM (SELECT name FROM pragma_table_info(?) ORDER BY cid)`, table).Scan(&columns); err != nil {
		t.Fatal(err)
	}
	return columns
}

// A migration whose any step fails is reported with what failed, and the index
// stays at the version it was at, holding nothing of the migration.
func TestAMigrationStepThatFailsPreservesTheIndexAtItsVersion(t *testing.T) {
	for _, step := range []struct {
		name   string
		breaks func(string) bool
	}{
		{"making the bookkeeping", containing("CREATE TABLE IF NOT EXISTS schema_migrations")},
		{"reading the bookkeeping's shape", containing("pragma_table_info('schema_migrations')")},
		{"beginning the transaction", func(query string) bool { return query == "BEGIN" }},
		{"a statement of the migration", containing("CREATE TABLE probe")},
		{"writing the version", containing("PRAGMA user_version = 2")},
		{"recording what ran", containing("INSERT INTO schema_migrations (")},
	} {
		t.Run(step.name, func(t *testing.T) {
			db := openAtVersionOne(t)
			breakWhere(t, step.breaks)

			err := apply(t.Context(), db, probe)
			stopBreaking()
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

// Bookkeeping already in the shape this build writes is used currently.
func TestBookkeepingInShapeIsNotRebuilt(t *testing.T) {
	db := openAtVersionOne(t)
	breakOn(t, "schema_migrations_next")

	if err := apply(t.Context(), db, probe); err != nil {
		t.Fatalf("a migration over bookkeeping in shape rebuilt it: %v", err)
	}
	stopBreaking()
	if pos := readUserVersion(t, db); pos != 2 {
		t.Errorf("the index is at version %d, want 2", pos)
	}
}

// Bookkeeping carrying a column this build does not write is brought to the
// shape this build writes, keeping every record.
func TestBookkeepingWithAColumnThisBuildDoesNotWriteIsReshaped(t *testing.T) {
	db := openLegacyBookkeeping(t)

	if err := createBookkeeping(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	if got := readColumns(t, db, "schema_migrations"); got != "version,name" {
		t.Errorf("schema_migrations holds columns %s, want version,name", got)
	}
	var name string
	if err := db.QueryRowContext(t.Context(),
		`SELECT name FROM schema_migrations WHERE version = 1`).Scan(&name); err != nil {
		t.Fatalf("the record of what ran did not survive: %v", err)
	}
	if name != "0001_index.sql" {
		t.Errorf("schema_migrations holds %q, want the original record", name)
	}
}

// A reshape that cannot begin is reported, and the bookkeeping stays as it was.
func TestBookkeepingWhoseReshapeCannotBeginIsLeftAsItWas(t *testing.T) {
	db := openLegacyBookkeeping(t)
	breakWhere(t, func(query string) bool { return query == "BEGIN" })

	err := createBookkeeping(t.Context(), db)
	stopBreaking()
	if !errors.Is(err, errBroken) {
		t.Fatalf("remember answered %v", err)
	}
	if got := readColumns(t, db, "schema_migrations"); got != "version,name,applied_at" {
		t.Errorf("schema_migrations holds columns %s, want the ones it had", got)
	}
}

// Emptying an index reports the step that failed, and the version goes back to
// nothing only once everything is out.
func TestEmptyingAnIndexReportsTheStepThatFailed(t *testing.T) {
	for _, step := range []struct {
		name   string
		breaks string
	}{
		{"turning references off", "PRAGMA foreign_keys = off"},
		{"reading what the index holds", "SELECT name FROM sqlite_master"},
		{"dropping an object", "DROP "},
		{"turning references back on", "PRAGMA foreign_keys = on"},
		{"putting the version back", "PRAGMA user_version = 0"},
	} {
		t.Run(step.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "index.db")
			migrated.CopyTo(t, path)
			db := openBreaking(t, path)
			breakOn(t, step.breaks)

			err := resetIndex(t.Context(), db)
			stopBreaking()
			if !errors.Is(err, errBroken) {
				t.Fatalf("emptying answered %v", err)
			}
			if pos := readUserVersion(t, db); pos == 0 {
				t.Error("an index that was not emptied says it holds no schema")
			}
		})
	}
}

// An index whose version cannot be read is reported, and nothing is built in it.
func TestAnIndexWhoseVersionCannotBeReadIsNotMigrated(t *testing.T) {
	db := openBreaking(t, filepath.Join(t.TempDir(), "index.db"))
	breakWhere(t, func(query string) bool { return query == "PRAGMA user_version" })

	err := migrate(t.Context(), db)
	stopBreaking()
	if !errors.Is(err, errBroken) {
		t.Fatalf("migrating answered %v", err)
	}
	if hasTable(t, db, "vaults") {
		t.Error("the migrations ran over a version nobody read")
	}
}

// An index from a later build that cannot be emptied is reported, and stays at
// the version that build left.
func TestAnIndexFromALaterBuildThatCannotBeEmptiedIsReported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.db")
	migrated.CopyTo(t, path)
	db := openBreaking(t, path)
	later := getNewestVersion(t) + 3
	if _, err := db.ExecContext(t.Context(), fmt.Sprintf("PRAGMA user_version = %d", later)); err != nil {
		t.Fatal(err)
	}
	breakOn(t, "PRAGMA user_version = 0")

	err := migrate(t.Context(), db)
	stopBreaking()
	if !errors.Is(err, errBroken) {
		t.Fatalf("migrating answered %v", err)
	}
	if pos := readUserVersion(t, db); pos != later {
		t.Errorf("the index is at version %d, want %d", pos, later)
	}
}

// A migration that fails is named in the report, which carries the cause.
func TestAFailedMigrationIsNamedAndCarriesItsCause(t *testing.T) {
	db := openBreaking(t, filepath.Join(t.TempDir(), "index.db"))
	breakOn(t, "PRAGMA user_version = 1")

	err := migrate(t.Context(), db)
	stopBreaking()
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

// Bookkeeping that cannot be committed is reported and left as it was.
func TestBookkeepingWhoseReshapeCannotBeCommittedIsLeftAsItWas(t *testing.T) {
	db := openLegacyBookkeeping(t)
	breakOn(t, "COMMIT")

	err := createBookkeeping(t.Context(), db)
	stopBreaking()
	if !errors.Is(err, errBroken) {
		t.Fatalf("remembering answered %v", err)
	}
	if !strings.Contains(readColumns(t, db, "schema_migrations"), "applied_at") {
		t.Error("bookkeeping that was not reshaped lost its column")
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

// An index whose migrations cannot be read is not migrated.
func TestMigrationsThatCannotBeReadMigrateNothing(t *testing.T) {
	db := openBreaking(t, filepath.Join(t.TempDir(), "index.db"))
	saved := files
	files = embed.FS{}
	t.Cleanup(func() { files = saved })

	if _, err := loadMigrations(); err == nil {
		t.Fatal("migrations were loaded from nothing")
	}
	if err := migrate(t.Context(), db); err == nil {
		t.Fatal("an index was migrated by nothing")
	}
	if pos := readUserVersion(t, db); pos != 0 {
		t.Errorf("the index is at version %d, want 0", pos)
	}
}
