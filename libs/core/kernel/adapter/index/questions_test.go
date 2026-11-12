package index

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/note"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// tallying is the driver the index opens, wrapped to count in ran every statement executed
// through it. A query is one statement executed, so a statement prepared once and executed a
// hundred times counts a hundred.
type tallying struct {
	driver.Driver
	ran *atomic.Int64
}

func (d tallying) Open(name string) (driver.Conn, error) {
	c, err := d.Driver.Open(name)
	if err != nil {
		return nil, err
	}
	return tallyingConn{c, d.ran}, nil
}

// tallyingConn returns statements that count themselves. It offers no query of
// its own, so every statement on this connection is prepared first and counted
// where it runs.
type tallyingConn struct {
	driver.Conn
	ran *atomic.Int64
}

func (c tallyingConn) Prepare(query string) (driver.Stmt, error) {
	s, err := c.Conn.Prepare(query)
	if err != nil {
		return nil, err
	}
	return tallyingStmt{s, c.ran}, nil
}

func (c tallyingConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	prepares, ok := c.Conn.(driver.ConnPrepareContext)
	if !ok {
		return c.Prepare(query)
	}
	s, err := prepares.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	return tallyingStmt{s, c.ran}, nil
}

// errPositionalQuery is what a statement answers when the driver offers no QueryContext.
var errPositionalQuery = errors.New("the driver offers no QueryContext")

type tallyingStmt struct {
	driver.Stmt
	ran *atomic.Int64
}

// Query is never called: database/sql calls QueryContext on a statement that has it.
func (s tallyingStmt) Query([]driver.Value) (driver.Rows, error) {
	return nil, errPositionalQuery
}

func (s tallyingStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	s.ran.Add(1)
	queries, ok := s.Stmt.(driver.StmtQueryContext)
	if !ok {
		return nil, errPositionalQuery
	}
	return queries.QueryContext(ctx, args)
}

func openCountedDB(t *testing.T) (*DB, *note.Queries, *atomic.Int64) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "index.db")
	newMigratedTemplate().CopyTo(t, path)
	db, err := Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	for _, v := range []domain.Vault{getFirstVault(), getSecond()} {
		if err := db.Vaults().Register(t.Context(), v.ID); err != nil {
			t.Fatal(err)
		}
	}

	base, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	tallied := tallying{Driver: base.Driver(), ran: &atomic.Int64{}}
	if err := base.Close(); err != nil {
		t.Fatal(err)
	}
	pool := sql.OpenDB(connector{name: dsn(path, settings{synchronous: shipped}), driver: tallied})
	testsupport.CloseAfter(t, pool)
	return db, note.NewQueries(pool), tallied.ran
}

func questions(t *testing.T, ran *atomic.Int64, ask func() error) int64 {
	t.Helper()
	ran.Store(0)
	if err := ask(); err != nil {
		t.Fatal(err)
	}
	return ran.Load()
}

// A folder is rendered from what the index reports for each of its notes, and
// a query per entry makes a folder of five hundred notes issue five hundred
// queries. The cost of opening a folder does not grow with the folder.
func TestWhatEveryNoteOfAFolderIsCostsTheSameHoweverManyThereAre(t *testing.T) {
	db, queries, ran := openCountedDB(t)
	ctx := t.Context()

	paths := make([]string, 0, 6)
	for i := range 6 {
		path := fmt.Sprintf("samples/sample-%d.md", i)
		saveTypedNote(t, db, getFirstVault(), path, fmt.Sprintf("Sample %d", i), sampleType)
		paths = append(paths, path)
	}

	// The pool is opened and the vault looked up before anything is counted.
	if _, err := queries.GetTypes(ctx, getFirstVault().ID, paths[:1]); err != nil {
		t.Fatal(err)
	}

	few := questions(t, ran, func() error {
		_, err := queries.GetTypes(ctx, getFirstVault().ID, paths[:2])
		return err
	})
	many := questions(t, ran, func() error {
		_, err := queries.GetTypes(ctx, getFirstVault().ID, paths)
		return err
	})
	if many != few {
		t.Errorf("two notes cost %d questions and six cost %d", few, many)
	}
}

// A rename reads every note of one type in the vault, and the index reports which notes those
// are. Querying them by type is one query regardless of the vault's size.
func TestTheNotesOfOneTypeAreOneQuestion(t *testing.T) {
	db, queries, ran := openCountedDB(t)
	ctx := t.Context()

	saveTypedNote(t, db, getFirstVault(), "samples/mammals.md", "Mammals", sampleType)
	saveTypedNote(t, db, getFirstVault(), "samples/birds.md", "Birds", sampleType)
	saveTypedNote(t, db, getFirstVault(), "forms/animal.md", "Animal", formType)
	saveTypedNote(t, db, getFirstVault(), "notes/entropy.md", "Entropy", domain.TypeNote)
	saveTypedNote(t, db, getSecond(), "samples/quasars.md", "Quasars", sampleType)

	if _, err := queries.GetPathsOfType(ctx, getFirstVault().ID, sampleType); err != nil {
		t.Fatal(err)
	}
	if questionCount := questions(t, ran, func() error {
		_, err := queries.GetPathsOfType(ctx, getFirstVault().ID, sampleType)
		return err
	}); questionCount > 2 {
		t.Errorf("the samples of a vault cost %d questions", questionCount)
	}

	found, err := queries.GetPathsOfType(ctx, getFirstVault().ID, sampleType)
	if err != nil {
		t.Fatal(err)
	}
	// By path, so two runs over one vault return one order.
	if len(found) != 2 || found[0] != "samples/birds.md" || found[1] != "samples/mammals.md" {
		t.Errorf("the samples of the vault came back as %v", found)
	}
	// One database holds every vault, and the second vault's sample is not this
	// vault's.
	other, err := queries.GetPathsOfType(ctx, getSecond().ID, sampleType)
	if err != nil {
		t.Fatal(err)
	}
	if len(other) != 1 || other[0] != "samples/quasars.md" {
		t.Errorf("the other vault's samples came back as %v", other)
	}
}
