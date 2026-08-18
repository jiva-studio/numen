package index

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
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
		`INSERT INTO vaults (identifier, name, path) VALUES ('01AAA', 'x', '/tmp/x')`); err != nil {
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

func TestTheChunksOfAnOlderIndexKeepTheirVectors(t *testing.T) {
	// A chunk written before it carried the hash of its text keeps its row, and
	// the vector made from it. The hash it carries is the empty string, which no
	// text hashes to, so the row is replaced the next time its source is cut.
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "index.db")

	db, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		t.Fatal(err)
	}
	available, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range available {
		if m.version >= 6 {
			break
		}
		if err := apply(ctx, db, m); err != nil {
			t.Fatal(err)
		}
	}
	for _, statement := range []string{
		`INSERT INTO vaults (id, identifier, name, path) VALUES (1, '01AAA', 'kept', '/notes')`,
		`INSERT INTO sources (id, vault_id, path, kind, size, modified_at)
		 VALUES (1, 1, 'notes/Entropy.md', 'note', 10, 1)`,
		`INSERT INTO chunks (id, source_id, vault_id, start, length, parent) VALUES (1, 1, 1, 0, 10, NULL)`,
		`INSERT INTO chunks (id, source_id, vault_id, start, length, parent) VALUES (2, 1, 1, 0, 10, 1)`,
		`INSERT INTO vectors (chunk_id, model, dims, kind, v) VALUES (2, 'model', 1024, 'int8', x'00')`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("%s: %v", statement, err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	upgraded, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("migrating an index cut before chunks carried a hash: %v", err)
	}
	defer upgraded.Close()

	var chunks, vectors int
	if err := upgraded.read.QueryRowContext(ctx,
		`SELECT (SELECT COUNT(*) FROM chunks WHERE hash = ''), (SELECT COUNT(*) FROM vectors)`).
		Scan(&chunks, &vectors); err != nil {
		t.Fatal(err)
	}
	if chunks != 2 || vectors != 1 {
		t.Errorf("%d chunks and %d vectors survived the migration, want 2 and 1", chunks, vectors)
	}
}

func TestAnOlderIndexIsMigratedRatherThanRebuilt(t *testing.T) {
	// The point of migrations: a schema change must not cost the user a rescan
	// of every vault. This builds a database at version 1, puts a row in it, and
	// checks the row survives the upgrade.
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "index.db")

	db, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		t.Fatal(err)
	}
	available, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	if len(available) < 2 {
		t.Skip("only one migration so far, nothing to upgrade from")
	}
	if err := apply(ctx, db, available[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO vaults (identifier, name, path) VALUES ('01AAA', 'kept', '/notes')`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	upgraded, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("migrating a version-1 index: %v", err)
	}
	defer upgraded.Close()

	var name string
	if err := upgraded.write.QueryRowContext(ctx,
		`SELECT name FROM vaults WHERE identifier = '01AAA'`).Scan(&name); err != nil {
		t.Fatalf("the row did not survive the migration: %v", err)
	}
	if name != "kept" {
		t.Errorf("name = %q", name)
	}

	var version int
	if err := upgraded.write.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if want := available[len(available)-1].version; version != want {
		t.Errorf("version = %d, want %d", version, want)
	}
}

// A version number says how many migrations ran, and nothing about which. An
// index migrated by other texts under the same numbers has a schema its number
// does not describe, and no later migration can be written to expect either one.
//
// This is what an edited migration leaves behind, and what a database written by
// another build of this application looks like.
func TestAnIndexMigratedByOtherMigrationsIsBuiltAgain(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "index.db")

	// A database that believes three migrations ran, holding a table none of
	// this binary's migrations create and lacking every one they do.
	raw, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`CREATE TABLE strangers (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE applied (version INTEGER PRIMARY KEY, name TEXT NOT NULL, hash TEXT NOT NULL)`,
		`INSERT INTO applied VALUES (1, '0001_initial.sql', 'another build wrote this')`,
		`PRAGMA user_version = 3`,
	} {
		if _, err := raw.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("an index of another build could not be opened: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	var version int
	if err := db.write.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != newest(t) {
		t.Errorf("the index is at version %d", version)
	}
	// Built from the first migration, so what it holds is what they create.
	for _, table := range []string{"vaults", "notes", "sources", "chunks", "vectors"} {
		var held int
		if err := db.write.QueryRowContext(ctx,
			`SELECT count(*) FROM sqlite_master WHERE type='table' AND name = ?`, table).Scan(&held); err != nil {
			t.Fatal(err)
		}
		if held != 1 {
			t.Errorf("%s is not there", table)
		}
	}
	var strangers int
	if err := db.write.QueryRowContext(ctx,
		`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='strangers'`).Scan(&strangers); err != nil {
		t.Fatal(err)
	}
	if strangers != 0 {
		t.Error("a table no migration creates survived")
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
	if err := db.Vaults().Save(ctx, domain.Vault{ID: "01KEPT", Name: "kept", Path: "/kept"}); err != nil {
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

// newest is the version the migrations this binary holds reach.
func newest(t *testing.T) int {
	t.Helper()
	available, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	return available[len(available)-1].version
}
