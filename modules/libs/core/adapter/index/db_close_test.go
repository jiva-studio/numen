package index

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// refusingToClose is a pool whose connection answers a close with the error
// given, and nothing when it is nil.
type refusingToClose struct{ err error }

func (r refusingToClose) Connect(context.Context) (driver.Conn, error) {
	return closeRefusal(r), nil
}

func (refusingToClose) Driver() driver.Driver { return nil }

type closeRefusal struct{ err error }

func (closeRefusal) Prepare(string) (driver.Stmt, error) { return nil, errors.New("no statements") }
func (c closeRefusal) Close() error                      { return c.err }
func (closeRefusal) Begin() (driver.Tx, error)           { return nil, errors.New("no transactions") }

// poolRefusing is a pool with one open connection that answers a close with err.
func poolRefusing(t *testing.T, err error) *sql.DB {
	t.Helper()
	pool := sql.OpenDB(refusingToClose{err})
	conn, connErr := pool.Conn(t.Context())
	if connErr != nil {
		t.Fatal(connErr)
	}
	if connErr := conn.Close(); connErr != nil {
		t.Fatal(connErr)
	}
	return pool
}

func TestClosingAnIndexReturnsWhatEitherPoolRefused(t *testing.T) {
	errWrite := errors.New("the write pool is stuck")
	errRead := errors.New("the read pool is stuck")
	for name, c := range map[string]struct {
		write, read error
		want        error
	}{
		"neither pool refuses": {nil, nil, nil},
		"the write pool alone": {errWrite, nil, errWrite},
		"the read pool alone":  {nil, errRead, errRead},
		"both pools refuse":    {errWrite, errRead, errWrite},
	} {
		db := &DB{write: poolRefusing(t, c.write), read: poolRefusing(t, c.read)}
		if got := db.Close(); !errors.Is(got, c.want) {
			t.Errorf("%s: Close returned %v, want %v", name, got, c.want)
		}
	}
}

func TestClosingAnIndexClosesTheReadPoolWhenTheWritePoolRefuses(t *testing.T) {
	read := poolRefusing(t, nil)
	db := &DB{write: poolRefusing(t, errors.New("stuck")), read: read}
	_ = db.Close()
	if _, err := read.Conn(t.Context()); err == nil {
		t.Error("the read pool is still open")
	}
}

// openFilesUnder is how many files below dir this process holds open.
func openFilesUnder(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Skip("this machine does not list the files a process holds open")
	}
	held := 0
	for _, e := range entries {
		target, err := os.Readlink(filepath.Join("/proc/self/fd", e.Name()))
		if err == nil && strings.HasPrefix(target, dir) {
			held++
		}
	}
	return held
}

func TestAnIndexThatCannotBeMigratedHoldsNoFileOpen(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "index.db")

	// A table the first migration creates is already there.
	seed, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := seed.ExecContext(t.Context(), "CREATE TABLE vaults (id TEXT)"); err != nil {
		t.Fatal(err)
	}
	if err := seed.Close(); err != nil {
		t.Fatal(err)
	}

	if db, err := Open(t.Context(), path); err == nil {
		db.Close()
		t.Fatal("an index whose first migration cannot run was opened")
	}
	if held := openFilesUnder(t, dir); held != 0 {
		t.Errorf("%d files of the index stayed open", held)
	}
}
