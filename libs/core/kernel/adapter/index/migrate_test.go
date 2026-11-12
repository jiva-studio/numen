package index

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
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
	defer testsupport.Close(t, db)

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
	defer testsupport.Close(t, second)

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
	defer testsupport.Close(t, db)

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
		testsupport.Close(t, db)
		t.Fatal("an index whose schema does not match its number was opened")
	}
	t.Logf("failed with: %v", err)

	// What it held, it still holds.
	back, err := sql.Open("sqlite", dsn(path, settings{synchronous: shipped}))
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, back)
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
	testsupport.CloseAfter(t, again)

	var vaults int
	if err := again.write.QueryRowContext(ctx, `SELECT count(*) FROM vaults`).Scan(&vaults); err != nil {
		t.Fatal(err)
	}
	if vaults != 1 {
		t.Errorf("%d vaults survived opening the index a second time", vaults)
	}
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
	defer testsupport.Close(t, db)

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

// faults are the statements and rows a breaking database fails, set by the test that opened it.
type faults struct {
	breaks atomic.Pointer[func(query string) bool]
	rows   atomic.Pointer[rowFault]
}

// errBroken is what every statement the breaking driver fails answers with.
var errBroken = errors.New("broken for the test")

func (f *faults) breakWhere(t *testing.T, breaks func(query string) bool) {
	t.Helper()
	f.breaks.Store(&breaks)
	t.Cleanup(f.stop)
}

func (f *faults) breakOn(t *testing.T, text string) {
	t.Helper()
	f.breakWhere(t, func(query string) bool { return strings.Contains(query, text) })
}

func (f *faults) stop() { f.breaks.Store(nil) }

func openBreakingDB(t *testing.T, path string) (*sql.DB, *faults) {
	t.Helper()
	base, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	wrapped := breaking{Driver: base.Driver(), faults: &faults{}}
	if err := base.Close(); err != nil {
		t.Fatal(err)
	}
	db := sql.OpenDB(connector{name: writeDSN(path, settings{synchronous: shipped}), driver: wrapped})
	db.SetMaxOpenConns(1)
	testsupport.CloseAfter(t, db)
	return db, wrapped.faults
}

// connector opens the connections of a database through one driver value, which no name
// registers.
type connector struct {
	name   string
	driver driver.Driver
}

func (c connector) Connect(context.Context) (driver.Conn, error) { return c.driver.Open(c.name) }

func (c connector) Driver() driver.Driver { return c.driver }

func (f *faults) isBroken(query string) bool {
	breaks := f.breaks.Load()
	return breaks != nil && (*breaks)(query)
}

type breaking struct {
	driver.Driver
	faults *faults
}

func (d breaking) Open(name string) (driver.Conn, error) {
	c, err := d.Driver.Open(name)
	if err != nil {
		return nil, err
	}
	return breakingConn{c, d.faults}, nil
}

type breakingConn struct {
	driver.Conn
	faults *faults
}

func (c breakingConn) Prepare(query string) (driver.Stmt, error) {
	s, err := c.Conn.Prepare(query)
	if err != nil {
		return nil, err
	}
	return breakingStmt{s, query, c.faults}, nil
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
	return breakingStmt{s, query, c.faults}, nil
}

// BeginTx is where database/sql starts every transaction on this driver.
func (c breakingConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if c.faults.isBroken("BEGIN") {
		return nil, fmt.Errorf("%w: BEGIN", errBroken)
	}
	begin, ok := c.Conn.(driver.ConnBeginTx)
	if !ok {
		return nil, errors.New("the wrapped connection cannot begin a transaction with options")
	}
	return c.wrapTx(begin.BeginTx(ctx, opts))
}

func (c breakingConn) wrapTx(tx driver.Tx, err error) (driver.Tx, error) {
	if err != nil {
		return nil, err
	}
	return breakingTx{tx, c.faults}, nil
}

// breakingTx fails a commit while "COMMIT" is broken, and leaves the transaction
// rolled back.
type breakingTx struct {
	driver.Tx
	faults *faults
}

func (t breakingTx) Commit() error {
	if t.faults.isBroken("COMMIT") {
		if err := t.Tx.Rollback(); err != nil {
			return errors.Join(fmt.Errorf("%w: COMMIT", errBroken), err)
		}
		return fmt.Errorf("%w: COMMIT", errBroken)
	}
	return t.Tx.Commit()
}

type breakingStmt struct {
	driver.Stmt
	query  string
	faults *faults
}

// ExecContext and QueryContext are where every statement in this package runs,
// so they are the one place that needs to see the fault. modernc.org/sqlite
// implements both on every statement.
func (s breakingStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	if s.faults.isBroken(s.query) {
		return nil, fmt.Errorf("%w: %s", errBroken, strings.TrimSpace(s.query))
	}
	exec, ok := s.Stmt.(driver.StmtExecContext)
	if !ok {
		return nil, errors.New("the wrapped statement cannot execute with a context")
	}
	return exec.ExecContext(ctx, args)
}

func (s breakingStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	if s.faults.isBroken(s.query) {
		return nil, fmt.Errorf("%w: %s", errBroken, strings.TrimSpace(s.query))
	}
	query, ok := s.Stmt.(driver.StmtQueryContext)
	if !ok {
		return nil, errors.New("the wrapped statement cannot query with a context")
	}
	rows, err := query.QueryContext(ctx, args)
	if err != nil {
		return nil, err
	}
	if fault := s.faults.rows.Load(); fault != nil && strings.Contains(s.query, fault.text) {
		return breakingRows{rows, fault.unreadable}, nil
	}
	return rows, nil
}

// rowFault fails the reading of the rows returned by a statement containing
// text: when unreadable is set, the first row holds a value that no destination
// accepts; when it is not set, the iteration ends in an error.
type rowFault struct {
	text       string
	unreadable bool
}

func (f *faults) breakRows(t *testing.T, text string, unreadable bool) {
	t.Helper()
	f.rows.Store(&rowFault{text: text, unreadable: unreadable})
	t.Cleanup(func() { f.rows.Store(nil) })
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
