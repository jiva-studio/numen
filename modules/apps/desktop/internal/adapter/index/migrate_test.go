package index

import (
	"database/sql"
	"errors"
	"fmt"
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

// An index a later build wrote is reported and left where it stands.
//
// Its schema holds what this build cannot read. Nothing is repaired, nothing is
// emptied: what this build owes the person is the two numbers and their own
// copy of their index, untouched.
func TestAnIndexFromALaterBuildIsRefused(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "index.db")

	db, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Vaults().Save(ctx, domain.Vault{ID: "01LATER", Name: "later", Path: "/later"}); err != nil {
		t.Fatal(err)
	}
	// A schema this build does not carry, written by one that does.
	if _, err := db.write.ExecContext(ctx,
		fmt.Sprintf("PRAGMA user_version = %d", newest(t)+3)); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	again, err := Open(ctx, path)
	if err == nil {
		again.Close()
		t.Fatal("an index from a later build was opened")
	}
	var ahead *Ahead
	if !errors.As(err, &ahead) {
		t.Fatalf("the error is %v, which does not say the index is ahead", err)
	}
	if ahead.Held != newest(t)+3 || ahead.Known != newest(t) {
		t.Errorf("said %d and %d, want %d and %d", ahead.Held, ahead.Known, newest(t)+3, newest(t))
	}

	// The index is as it was left. Nothing was repaired by deleting.
	raw, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	var vaults int
	if err := raw.QueryRowContext(ctx, `SELECT count(*) FROM vaults`).Scan(&vaults); err != nil {
		t.Fatal(err)
	}
	if vaults != 1 {
		t.Errorf("the index holds %d vaults, want the one it was left with", vaults)
	}
}

// An index whose number does not describe the schema it holds is reported.
//
// A migration written to expect what came before it fails on a schema that does
// not have it. The failure is what this build says; the index is not touched.
func TestAnIndexWhoseSchemaDoesNotMatchItsNumberIsRefused(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "index.db")

	available, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	// Every migration but the newest claimed, and none of what they build, so
	// the newest runs against a schema without what it was written to expect.
	behind := available[len(available)-1].version - 1

	raw, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`CREATE TABLE strangers (id INTEGER PRIMARY KEY)`,
		fmt.Sprintf(`PRAGMA user_version = %d`, behind),
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

	// What it held, it still holds.
	back, err := sql.Open("sqlite", dsn(path))
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

// A source read before the column held a producer stands on the same reading
// after.
//
// The column held a whole name and now holds the producer that name began with.
// A row keeping the whole name composes a name from it and finds nothing under
// it, and the passages of that book come back empty with nothing saying why.
func TestASourceReadBeforeTheColumnMeantAProducerKeepsItsReading(t *testing.T) {
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
	var through []migration
	for _, one := range available {
		if one.version >= 3 {
			break
		}
		through = append(through, one)
	}
	if len(through) == len(available) {
		t.Skip("the column does not name a producer yet")
	}
	for _, one := range through {
		if err := apply(ctx, db, one); err != nil {
			t.Fatal(err)
		}
	}
	const hash = "0ce540f592f6df89d36c636a898be70be0a557f468bff30137666607de9c580c"
	if _, err := db.ExecContext(ctx,
		`INSERT INTO vaults (identifier, name, path) VALUES ('01AAA', 'kept', '/notes')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO sources (vault_id, path, kind, size, modified_at, hash, text_path)
		 SELECT id, 'library/scan.pdf', 'book', 1, 1, ?, ? FROM vaults WHERE identifier = '01AAA'`,
		hash, "ocr/"+hash+".txt"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	upgraded, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer upgraded.Close()

	var from string
	if err := upgraded.write.QueryRowContext(ctx,
		`SELECT text_from FROM sources WHERE path = 'library/scan.pdf'`).Scan(&from); err != nil {
		t.Fatal(err)
	}
	if from != "ocr" {
		t.Errorf("text_from = %q, and a name composed from it names nothing", from)
	}
}
