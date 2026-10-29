package index

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
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
	for _, name := range []string{"initial.sql", "_initial.sql", "abc_initial.sql", "0000_zero.sql", "0007"} {
		if _, err := parseVersion(name); err == nil {
			t.Errorf("%q was accepted", name)
		}
	}
	v, err := parseVersion("0007_add_column.sql")
	if err != nil || v != 7 {
		t.Errorf("0007 gave %d, %v", v, err)
	}
}

func TestStatementsIgnoreCommentsAndBlanks(t *testing.T) {
	got := splitStatements("-- a comment\n\nSELECT 1;\n-- another\n;\nSELECT 2;\n")
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
	// next start retries the whole migration and not resuming half-way.
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
		t.Error("failed migration partially applied")
	}
}

// An index written by a later build is emptied and rebuilt from the first
// migration file.
//
// Its schema holds data this build cannot read. The index is a cache: its
// content is derived from the vault, and the next scan reads the vault again,
// so the reset costs only that re-read.
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

	// the later build left is in it.
	var pos int
	if err := raw.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&pos); err != nil {
		t.Fatal(err)
	}
	if pos != getNewestVersion(t) {
		t.Errorf("the index is at schema %d, want %d", pos, getNewestVersion(t))
	}
	var vaults int
	if err := raw.QueryRowContext(ctx, `SELECT count(*) FROM vaults`).Scan(&vaults); err != nil {
		t.Fatal(err)
	}
	if vaults != 0 {
		t.Errorf("the index holds %d vaults, want none: it was not emptied", vaults)
	}
}

// An index whose version number does not match the schema it holds is reported
// as an error.
//
// A migration written for the preceding schema fails on a schema that lacks it.
// The failure is reported by this build, and the index is not modified.
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
	t.Logf("failed with: %v", err)

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

// PRAGMA user_version writes through the same page as an ordinary table
// change, and modernc.org/sqlite rolls it back with the rest of an explicit
// transaction that never commits.
func TestPragmaUserVersionRollsBackWithItsTransaction(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "index.db")

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "PRAGMA user_version = 99"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO nonexistent (x) VALUES (1)"); err == nil {
		t.Fatal("an insert into a table that does not exist reported success")
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	var version int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 0 {
		t.Errorf("user_version = %d after rollback, want 0", version)
	}
}

// breaksOn lists the statements that the breaking driver fails. A test that
// uses the breaking driver sets it through breakWhere, since the driver is
// registered once per process.
var breaksOn atomic.Pointer[func(query string) bool]

// errBroken is what every statement the breaking driver fails answers with.
var errBroken = errors.New("broken for the test")

// breakingDriver registers a driver that fails the statements breaksOn names and
// runs every other statement unchanged. A transaction's BEGIN is the statement
// "BEGIN".
var breakingDriver = sync.OnceValue(func() error {
	base, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return err
	}
	defer base.Close()
	sql.Register("sqlite-breaking", breaking{base.Driver()})
	return nil
})

// breakWhere fails, until the test ends, every statement the predicate holds for.
func breakWhere(t *testing.T, breaks func(query string) bool) {
	t.Helper()
	breaksOn.Store(&breaks)
	t.Cleanup(func() { breaksOn.Store(nil) })
}

// breakOn fails, until the test ends, every statement containing the text given.
func breakOn(t *testing.T, text string) {
	t.Helper()
	breakWhere(t, func(query string) bool { return strings.Contains(query, text) })
}

// openBreaking is the index file at path through the breaking driver, opened the
// way the write pool is.
func openBreaking(t *testing.T, path string) *sql.DB {
	t.Helper()
	if err := breakingDriver(); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite-breaking", writeDSN(path, settings{synchronous: shipped}))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	return db
}

func isBroken(query string) bool {
	breaks := breaksOn.Load()
	return breaks != nil && (*breaks)(query)
}

type breaking struct{ driver.Driver }

func (d breaking) Open(name string) (driver.Conn, error) {
	c, err := d.Driver.Open(name)
	if err != nil {
		return nil, err
	}
	return breakingConn{c}, nil
}

type breakingConn struct{ driver.Conn }

func (c breakingConn) Prepare(query string) (driver.Stmt, error) {
	s, err := c.Conn.Prepare(query)
	if err != nil {
		return nil, err
	}
	return breakingStmt{s, query}, nil
}

func (c breakingConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	prepares, ok := c.Conn.(driver.ConnPrepareContext)
	if !ok {
		return c.Prepare(query)
	}
	s, err := prepares.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return breakingStmt{s, query}, nil
}

// BeginTx is where database/sql starts every transaction on this driver.
func (c breakingConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if isBroken("BEGIN") {
		return nil, fmt.Errorf("%w: BEGIN", errBroken)
	}
	tx, err := c.Conn.(driver.ConnBeginTx).BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return breakingTx{tx}, nil
}

// breakingTx fails a commit while "COMMIT" is broken, and leaves the transaction
// rolled back.
type breakingTx struct{ driver.Tx }

func (t breakingTx) Commit() error {
	if isBroken("COMMIT") {
		_ = t.Tx.Rollback()
		return fmt.Errorf("%w: COMMIT", errBroken)
	}
	return t.Tx.Commit()
}

type breakingStmt struct {
	driver.Stmt
	query string
}

// ExecContext and QueryContext are where every statement in this package runs,
// so they are the one place that needs to see the fault. modernc.org/sqlite
// implements both on every statement.
func (s breakingStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	if isBroken(s.query) {
		return nil, fmt.Errorf("%w: %s", errBroken, strings.TrimSpace(s.query))
	}
	return s.Stmt.(driver.StmtExecContext).ExecContext(ctx, args)
}

func (s breakingStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	if isBroken(s.query) {
		return nil, fmt.Errorf("%w: %s", errBroken, strings.TrimSpace(s.query))
	}
	rows, err := s.Stmt.(driver.StmtQueryContext).QueryContext(ctx, args)
	if err != nil {
		return nil, err
	}
	if fault := rowsBreak.Load(); fault != nil && strings.Contains(s.query, fault.text) {
		return breakingRows{rows, fault.unreadable}, nil
	}
	return rows, nil
}

// rowFault fails the reading of the rows returned by a statement containing
// text: when unreadable is set, the first row holds a value that no destination
// accepts; otherwise the iteration ends in an error.
type rowFault struct {
	text       string
	unreadable bool
}

var rowsBreak atomic.Pointer[rowFault]

// breakRows fails the reading of the rows of every statement containing text,
// until the test ends.
func breakRows(t *testing.T, text string, unreadable bool) {
	t.Helper()
	rowsBreak.Store(&rowFault{text: text, unreadable: unreadable})
	t.Cleanup(func() { rowsBreak.Store(nil) })
}

type breakingRows struct {
	driver.Rows
	unreadable bool
}

func (r breakingRows) Next(dest []driver.Value) error {
	if !r.unreadable {
		return errBroken
	}
	if err := r.Rows.Next(dest); err != nil {
		return err
	}
	dest[0] = complex128(1)
	return nil
}

// remember's rename runs as one transaction: a statement that fails partway
// through leaves the schema_migrations table that was there before remember
// was called.
func TestRememberPreservesTheOriginalTableWhenRenamingFails(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "index.db")

	raw, err := sql.Open("sqlite", dsn(path, settings{synchronous: shipped}))
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at TEXT)`,
		`INSERT INTO schema_migrations (version, name, applied_at) VALUES (1, '0001_index.sql', '2024-01-01')`,
	} {
		if _, err := raw.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	db := openBreaking(t, path)
	breakOn(t, "RENAME TO")

	if err := createBookkeeping(ctx, db); !errors.Is(err, errBroken) {
		t.Fatalf("remember answered %v over a rename it could not make", err)
	}

	back, err := sql.Open("sqlite", dsn(path, settings{synchronous: shipped}))
	if err != nil {
		t.Fatal(err)
	}
	defer back.Close()

	var name string
	if err := back.QueryRowContext(ctx,
		`SELECT name FROM schema_migrations WHERE version = 1`).Scan(&name); err != nil {
		t.Fatalf("the original schema_migrations did not survive the failed rename: %v", err)
	}
	if name != "0001_index.sql" {
		t.Errorf("schema_migrations held %q, want the original row", name)
	}

	var next int
	if err := back.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'schema_migrations_next'`).Scan(&next); err != nil {
		t.Fatal(err)
	}
	if next != 0 {
		t.Error("schema_migrations_next table retained after rollback")
	}
}

func TestMigrationToPrefixIndex(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "index.db")

	raw, err := sql.Open("sqlite", dsn(path, settings{synchronous: shipped}))
	if err != nil {
		t.Fatal(err)
	}

	m1, err := files.ReadFile("migration/0001_index.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err := apply(ctx, raw, migration{version: 1, name: "0001_index.sql", body: string(m1)}); err != nil {
		t.Fatal(err)
	}

	if _, err := raw.ExecContext(ctx, `INSERT INTO vaults (id, identifier) VALUES (1, '01TEST')`); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `INSERT INTO sources (id, vault_id, path, kind, size, modified_at) VALUES (1, 1, 'notes/entropy.md', 'note', 100, 123456)`); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `INSERT INTO notes (source_id, vault_id, folded_name, title, type) VALUES (1, 1, 'entropy', 'Entropy', 'note')`); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `INSERT INTO headings (id, note_id, line, level, text) VALUES (1, 1, 2, 2, 'Thermodynamics')`); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `INSERT INTO titles_fts (rowid, text) VALUES (1, 'Entropy')`); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `INSERT INTO headings_fts (rowid, text) VALUES (1, 'Thermodynamics')`); err != nil {
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	matches, err := db.NoteQueries().GetNameMatches(ctx, "01TEST", "thermo", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].Heading != "Thermodynamics" {
		t.Fatalf("expected heading match, got %+v", matches)
	}

	var modTime int64
	if err := db.read.QueryRowContext(ctx, `SELECT modified_at FROM sources WHERE id = 1`).Scan(&modTime); err != nil {
		t.Fatal(err)
	}
	if modTime != 0 {
		t.Errorf("modified_at = %d, want 0", modTime)
	}
}
