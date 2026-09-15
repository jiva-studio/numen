package index

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/note"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// ran is how many questions have been put to a database opened through the
// tallying driver. A question is one statement executed, so a statement
// prepared once and asked a hundred times is a hundred.
var ran atomic.Int64

// tallyingDriver registers the driver the index opens, wrapped so that every
// statement executed through it is counted. It is registered once, because a
// driver name is registered for the life of the process.
var tallyingDriver = sync.OnceValue(func() error {
	base, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return err
	}
	defer base.Close()
	sql.Register("sqlite-tallying", tallying{base.Driver()})
	return nil
})

type tallying struct{ driver.Driver }

func (d tallying) Open(name string) (driver.Conn, error) {
	c, err := d.Driver.Open(name)
	if err != nil {
		return nil, err
	}
	return tallyingConn{c}, nil
}

// tallyingConn hands out statements that count themselves. It offers no query
// of its own, so everything asked of this connection is prepared first and
// counted where it runs.
type tallyingConn struct{ driver.Conn }

func (c tallyingConn) Prepare(query string) (driver.Stmt, error) {
	s, err := c.Conn.Prepare(query)
	if err != nil {
		return nil, err
	}
	return tallyingStmt{s}, nil
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
	return tallyingStmt{s}, nil
}

type tallyingStmt struct{ driver.Stmt }

func (s tallyingStmt) Query(args []driver.Value) (driver.Rows, error) {
	ran.Add(1)
	return s.Stmt.Query(args)
}

func (s tallyingStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	ran.Add(1)
	queries, ok := s.Stmt.(driver.StmtQueryContext)
	if !ok {
		values := make([]driver.Value, 0, len(args))
		for _, a := range args {
			values = append(values, a.Value)
		}
		return s.Stmt.Query(values)
	}
	return queries.QueryContext(ctx, args)
}

// openCountedDB is an index whose questions are counted: the database itself,
// and the note queries over a pool of its own that does the counting.
func openCountedDB(t *testing.T) (*DB, *note.Queries) {
	t.Helper()
	if err := tallyingDriver(); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "index.db")
	migrated.CopyTo(t, path)
	db, err := Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	for _, v := range []domain.Vault{first, second} {
		if err := db.Vaults().Register(t.Context(), v.ID); err != nil {
			t.Fatal(err)
		}
	}

	pool, err := sql.Open("sqlite-tallying", dsn(path))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Close() })
	return db, note.NewQueries(pool)
}

// questions is how many statements a call ran.
func questions(t *testing.T, ask func() error) int64 {
	t.Helper()
	ran.Store(0)
	if err := ask(); err != nil {
		t.Fatal(err)
	}
	return ran.Load()
}

// A folder is drawn from what the index says each of its notes is, and a
// question per entry is a folder of five hundred notes asking five hundred
// times. What it costs to open a folder does not grow with the folder.
func TestWhatEveryNoteOfAFolderIsCostsTheSameHoweverManyThereAre(t *testing.T) {
	db, queries := openCountedDB(t)
	ctx := t.Context()

	paths := make([]string, 0, 6)
	for i := range 6 {
		path := fmt.Sprintf("cards/deck-%d.md", i)
		saveTypedNote(t, db, first, path, fmt.Sprintf("Deck %d", i), domain.TypeDeck)
		paths = append(paths, path)
	}

	// The pool is opened and the vault looked up before anything is counted.
	if _, err := queries.Types(ctx, first.ID, paths[:1]); err != nil {
		t.Fatal(err)
	}

	few := questions(t, func() error {
		_, err := queries.Types(ctx, first.ID, paths[:2])
		return err
	})
	many := questions(t, func() error {
		_, err := queries.Types(ctx, first.ID, paths)
		return err
	})
	if many != few {
		t.Errorf("two notes cost %d questions and six cost %d", few, many)
	}
}

// A rename reads every deck of the vault, and the index says which notes those
// are. Asking for them by type is one question whatever the vault holds.
func TestTheNotesOfOneTypeAreOneQuestion(t *testing.T) {
	db, queries := openCountedDB(t)
	ctx := t.Context()

	saveTypedNote(t, db, first, "decks/mammals.md", "Mammals", domain.TypeDeck)
	saveTypedNote(t, db, first, "decks/birds.md", "Birds", domain.TypeDeck)
	saveTypedNote(t, db, first, "stencils/animal.md", "Animal", domain.TypeStencil)
	saveTypedNote(t, db, first, "notes/entropy.md", "Entropy", domain.TypeNote)
	saveTypedNote(t, db, second, "decks/quasars.md", "Quasars", domain.TypeDeck)

	if _, err := queries.OfType(ctx, first.ID, domain.TypeDeck); err != nil {
		t.Fatal(err)
	}
	if ran := questions(t, func() error {
		_, err := queries.OfType(ctx, first.ID, domain.TypeDeck)
		return err
	}); ran > 2 {
		t.Errorf("the decks of a vault cost %d questions", ran)
	}

	found, err := queries.OfType(ctx, first.ID, domain.TypeDeck)
	if err != nil {
		t.Fatal(err)
	}
	// By path, so that two runs over one vault answer in one order.
	if len(found) != 2 || found[0] != "decks/birds.md" || found[1] != "decks/mammals.md" {
		t.Errorf("the decks of the vault came back as %v", found)
	}
	// One database holds every vault, and the second vault's deck is not this
	// vault's.
	other, err := queries.OfType(ctx, second.ID, domain.TypeDeck)
	if err != nil {
		t.Fatal(err)
	}
	if len(other) != 1 || other[0] != "decks/quasars.md" {
		t.Errorf("the other vault's decks came back as %v", other)
	}
}
