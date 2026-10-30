package index

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/adapter/index/vault"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
)

// openWithShortBusyTimeout is a write pool that waits a fifth of a second for the lock instead
// of the five seconds an installation waits, so a test can meet a locked
// database without spending a minute on it. Everything else about it is what
// Open builds.
func openWithShortBusyTimeout(t *testing.T, path string) *sql.DB {
	t.Helper()
	dsn := dsnOf(path, []string{"journal_mode(WAL)", "foreign_keys(1)", "busy_timeout(200)", shipped})
	db, err := sql.Open("sqlite", dsn+"&_txlock=immediate")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	testsupport.CloseAfter(t, db)
	return db
}

// SQLite has one write lock for the whole file. A write that meets it held by
// another process waits the busy timeout, is told the database is locked, and
// asks again.
func TestAWriteWaitsOutAWriterInAnotherProcess(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "index.db")
	heldDB := openDBAt(t, path)

	// The other process, mid-write: an immediate transaction holds the file's
	// one write lock until it ends.
	tx, err := heldDB.write.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, "PRAGMA user_version"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS holding (x)`); err != nil {
		t.Fatal(err)
	}

	// Longer than one wait for the lock and shorter than the three a write is
	// given, so the write lands on an ask after the first.
	let := time.AfterFunc(350*time.Millisecond, func() {
		if err := tx.Rollback(); err != nil {
			t.Error(err)
		}
	})
	defer let.Stop()

	if err := vault.NewRepository(openWithShortBusyTimeout(t, path)).Register(ctx, "01WAITED"); err != nil {
		t.Fatalf("a write that met the lock was not asked again: %v", err)
	}
}
