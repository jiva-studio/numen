package index

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
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

// A vector kept under a key that does not say where it was made goes.
//
// Nothing in the index says which of the places serving one model name made it,
// and a vector taken for one made somewhere else is answered with as though the
// two agreed to the last digit.
func TestAVectorThatDoesNotSayWhereItWasMadeIsDropped(t *testing.T) {
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
		if m.version >= 5 {
			break
		}
		if err := apply(ctx, db, m); err != nil {
			t.Fatal(err)
		}
	}
	// Both keys a vector has been kept under: the first named an address that
	// was empty for a model run on this machine, and the second named none. One
	// text each, because the two keys become one key on the way here.
	for _, kept := range []struct {
		print  []byte
		recipe string
	}{
		{[]byte{0x01}, "https://api.openai.com/v1|text-embedding-3-small|1536|0|int8"},
		{[]byte{0x02}, "text-embedding-3-small|1536|0|mean|int8"},
	} {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO vectors (fingerprint, recipe, v) VALUES (?, ?, x'02')`,
			kept.print, kept.recipe); err != nil {
			t.Fatal(err)
		}
	}
	// The coarse form of one of them, which is a reading of a vector that is
	// about to go.
	if _, err := db.ExecContext(ctx,
		`INSERT INTO chunks_vec (chunk_id, vault_id, embedding) VALUES (1, 1, vec_bit(?))`,
		make([]byte, 128)); err != nil {
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

	var held int
	if err := upgraded.write.QueryRowContext(ctx, `SELECT count(*) FROM vectors`).Scan(&held); err != nil {
		t.Fatal(err)
	}
	if held != 0 {
		t.Errorf("the index kept %d vectors that do not say where they were made", held)
	}
	var coarse int
	if err := upgraded.write.QueryRowContext(ctx, `SELECT count(*) FROM chunks_vec`).Scan(&coarse); err != nil {
		t.Fatal(err)
	}
	if coarse != 0 {
		t.Errorf("the coarse index kept %d readings of vectors that went", coarse)
	}

	// Nor is one of them handed to the model now configured.
	asked := port.EmbeddingModel{
		Name: "text-embedding-3-small", Dimensions: 1536, Pooling: "mean",
		From: "service:https://api.openai.com/v1/text-embedding-3-small",
	}.Recipe()
	kept, err := upgraded.ChunkQueries().Kept(ctx, asked, [][]byte{{0x01}, {0x02}})
	if err != nil {
		t.Fatal(err)
	}
	if len(kept) != 0 {
		t.Errorf("a vector made nobody knows where answered for %s: %v", asked, kept)
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

	raw, err := sql.Open("sqlite", dsn(path))
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

// A note indexed before the index held what a note is stays readable, and is a
// note.
//
// The key did not exist when it was read, so nothing about the row says the
// column is missing and no scan would notice.
func TestANoteIndexedBeforeTheIndexHeldWhatANoteIsIsANote(t *testing.T) {
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
		if one.version >= 7 {
			break
		}
		through = append(through, one)
	}
	if len(through) == len(available) {
		t.Skip("the index does not hold what a note is yet")
	}
	for _, one := range through {
		if err := apply(ctx, db, one); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO vaults (id, identifier, name, path) VALUES (1, '01AAA', 'kept', '/notes')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO sources (id, vault_id, path, kind, size, modified_at)
		 VALUES (1, 1, 'notes/entropy.md', 'note', 100, 1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO notes (source_id, vault_id, basename, title)
		 VALUES (1, 1, 'entropy', 'Entropy')`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	upgraded, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("migrating an index written before the column: %v", err)
	}
	defer upgraded.Close()

	var title, held string
	if err := upgraded.write.QueryRowContext(ctx,
		`SELECT title, type FROM notes WHERE source_id = 1`).Scan(&title, &held); err != nil {
		t.Fatalf("the note did not survive the migration: %v", err)
	}
	if title != "Entropy" {
		t.Errorf("title = %q", title)
	}
	if held != string(domain.TypeNote) {
		t.Errorf("type = %q, want %q", held, domain.TypeNote)
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

// A file that already said what it was before the key was indexed is read
// again.
//
// A scan skips a file whose size and modification time still match, so a note
// filed as `type: deck` before the upgrade would keep the default the column
// was added with until somebody edited the file. The fingerprint of every note
// goes with the column, and the next scan reads them.
func TestTheTypeOfAFileAlreadyIndexedIsReadAgain(t *testing.T) {
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
	const typed = 7
	for _, m := range available {
		if m.version >= typed {
			break
		}
		if err := apply(ctx, db, m); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO vaults (id, identifier, name, path) VALUES (1, '01AAA', 'kept', '/notes')`); err != nil {
		t.Fatal(err)
	}
	// The deck, and a document beside it whose text nothing in this migration
	// is about.
	if _, err := db.ExecContext(ctx,
		`INSERT INTO sources (id, vault_id, path, kind, size, modified_at)
		 VALUES (1, 1, 'decks/mammals.md', 'note', 120, 4), (2, 1, 'library/scan.pdf', 'book', 900, 5)`,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO notes (source_id, vault_id, basename, title) VALUES (1, 1, 'mammals', 'Mammals')`,
	); err != nil {
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

	known, err := upgraded.NoteQueries().Fingerprints(ctx, "01AAA")
	if err != nil {
		t.Fatal(err)
	}
	onDisk := domain.FileRef{Path: "decks/mammals.md", Kind: domain.KindNote, Size: 120, MTime: 4}
	if known[onDisk.Path].Unchanged(onDisk) {
		t.Errorf("the file is skipped by the next scan, so what it says it is is never read: %+v",
			known[onDisk.Path])
	}

	var size, modified int64
	if err := upgraded.write.QueryRowContext(ctx,
		`SELECT size, modified_at FROM sources WHERE path = 'library/scan.pdf'`).Scan(&size, &modified); err != nil {
		t.Fatal(err)
	}
	if size != 900 || modified != 5 {
		t.Errorf("a document is read again for a key only a note carries: %d bytes, %d", size, modified)
	}
}
