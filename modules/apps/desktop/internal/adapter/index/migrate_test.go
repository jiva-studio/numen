package index

import (
	"database/sql"
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
		`INSERT INTO vaults (id, name, path) VALUES ('01AAA', 'x', '/tmp/x')`); err != nil {
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
