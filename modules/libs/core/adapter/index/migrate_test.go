package index

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
)

func TestEveryMigrationIsNamedAndOrdered(t *testing.T) {
	got, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("no migrations were embedded")
	}
	for i, m := range got {
		if m.version <= 0 {
			t.Errorf("%s has version %d", m.name, m.version)
		}
		if i > 0 && got[i-1].version >= m.version {
			t.Errorf("%s and %s are out of order", got[i-1].name, m.name)
		}
	}
}

func TestVersionOfRejectsUnnumberedFiles(t *testing.T) {
	for _, name := range []string{"initial.sql", "_initial.sql", "abc_initial.sql", "0000_zero.sql"} {
		if _, err := versionOf(name); err == nil {
			t.Errorf("%q was accepted", name)
		}
	}
	v, err := versionOf("0007_add_column.sql")
	if err != nil || v != 7 {
		t.Errorf("0007 gave %d, %v", v, err)
	}
}

func TestStatementsIgnoreCommentsAndBlanks(t *testing.T) {
	got := statements("-- a comment\n\nSELECT 1;\n-- another\n;\nSELECT 2;\n")
	if len(got) != 2 {
		t.Fatalf("got %d statements: %q", len(got), got)
	}
}

func TestFreshDatabaseIsAtTheNewestVersion(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "index.db")

	db, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	available, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	newest := available[len(available)-1].version

	var version int
	if err := db.write.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != newest {
		t.Errorf("version = %d, want %d", version, newest)
	}
}

func TestReopeningAppliesNothingAndKeepsData(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "index.db")

	first, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.write.ExecContext(ctx,
		`INSERT INTO vaults (identifier) VALUES ('01AAA')`); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	// Re-running an applied migration would fail on CREATE TABLE, so this also
	// proves migrations are not applied twice.
	second, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer second.Close()

	var count int
	if err := second.write.QueryRowContext(ctx, `SELECT COUNT(*) FROM vaults`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("%d vaults after reopening, want 1 — data did not survive", count)
	}
}

func TestMigrationsRunInOneTransactionEach(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "index.db")

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// A statement that cannot apply must leave the version where it was, so the
	// next start retries the whole migration rather than resuming half-way.
	broken := migration{version: 99, name: "0099_broken.sql", body: "CREATE TABLE ok (x INTEGER);\nNOT SQL AT ALL;"}
	if err := apply(ctx, db, broken); err == nil {
		t.Fatal("a broken migration reported success")
	}

	var version int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 0 {
		t.Errorf("version = %d after a failed migration, want 0", version)
	}
	var tables int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='ok'`).Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables != 0 {
		t.Error("the first half of a failed migration was left behind")
	}
}

// An index a later build wrote is emptied and built again from the first file.
//
// Its schema holds what this build cannot read. The index is a cache — what it
// holds is a reading of the vault, and the next scan reads the vault again — so
// it costs the person that reading and stops nothing.
func TestAnIndexFromALaterBuildIsEmptiedAndBuiltAgain(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "index.db")

	db, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Vaults().Register(ctx, "01LATER"); err != nil {
		t.Fatal(err)
	}
	// A schema this build does not carry, written by one that does.
	if _, err := db.write.ExecContext(ctx,
		fmt.Sprintf("PRAGMA user_version = %d", getNewestVersion(t)+3)); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	again, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("an index a later build wrote stopped this one: %v", err)
	}
	if err := again.Close(); err != nil {
		t.Fatal(err)
	}

	raw, err := sql.Open("sqlite", dsn(path, settings{synchronous: shipped}))
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()

	// It stands at the schema this build carries, and nothing of the reading
	// the later build left is in it.
	var at int
	if err := raw.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&at); err != nil {
		t.Fatal(err)
	}
	if at != getNewestVersion(t) {
		t.Errorf("the index is at schema %d, want %d", at, getNewestVersion(t))
	}
	var vaults int
	if err := raw.QueryRowContext(ctx, `SELECT count(*) FROM vaults`).Scan(&vaults); err != nil {
		t.Fatal(err)
	}
	if vaults != 0 {
		t.Errorf("the index holds %d vaults, want none: it was not emptied", vaults)
	}
}

// An index whose number does not describe the schema it holds is reported.
//
// A migration written to expect what came before it fails on a schema that does
// not have it. The failure is what this build says; the index is not touched.
func TestAnIndexWhoseSchemaDoesNotMatchItsNumberIsRefused(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "index.db")

	raw, err := sql.Open("sqlite", dsn(path, settings{synchronous: shipped}))
	if err != nil {
		t.Fatal(err)
	}
	// A number saying nothing has been applied, over a schema already holding a
	// name the first migration builds. It runs, and it cannot.
	for _, statement := range []string{
		`CREATE TABLE strangers (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE vaults (id INTEGER PRIMARY KEY)`,
		`PRAGMA user_version = 0`,
	} {
		if _, err := raw.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := Open(ctx, path)
	if err == nil {
		db.Close()
		t.Fatal("an index whose schema does not match its number was opened")
	}
	t.Logf("refused with: %v", err)

	// What it held, it still holds.
	back, err := sql.Open("sqlite", dsn(path, settings{synchronous: shipped}))
	if err != nil {
		t.Fatal(err)
	}
	defer back.Close()
	var strangers int
	if err := back.QueryRowContext(ctx,
		`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='strangers'`).Scan(&strangers); err != nil {
		t.Fatal(err)
	}
	if strangers != 1 {
		t.Error("a table this build does not know was taken out of somebody's index")
	}
}

// An index this binary migrated is left where it is.
func TestAnIndexOfItsOwnIsNotBuiltAgain(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "index.db")

	db, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Vaults().Register(ctx, "01KEPT"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	again, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { again.Close() })

	var vaults int
	if err := again.write.QueryRowContext(ctx, `SELECT count(*) FROM vaults`).Scan(&vaults); err != nil {
		t.Fatal(err)
	}
	if vaults != 1 {
		t.Errorf("%d vaults survived opening the index a second time", vaults)
	}
}

// getNewestVersion is the version the migrations this binary holds reach.
func getNewestVersion(t *testing.T) int {
	t.Helper()
	available, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	return available[len(available)-1].version
}
