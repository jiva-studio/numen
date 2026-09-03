package index

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/chunk"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/note"
)

// coarse is a vector of the shape the vector index holds: one bit per dimension
// over 1024 dimensions.
var coarse = make([]byte, 128)

// The questions the index is asked, and the index each one has to be answered
// through.
var expectedPlans = []struct {
	statements map[string]string
	name       string
	args       []any
	// An index name, or the columns for a primary key: SQLite names those
	// itself, and the name changes when a constraint is added.
	through []string
}{
	{note.Statements(), "candidates", []any{1, "a", 1, "b", 1, "c"}, []string{"notes_by_basename"}},
	{note.Statements(), "backlink_candidates", []any{"id", 1, "base", 1}, []string{"links_by_target", "links_by_name"}},
	{note.Statements(), "note_by_identifier", []any{"id"}, []string{"notes_by_identifier"}},
	{note.Statements(), "links_of", []any{1}, []string{"(note_id=?)"}},
	{note.Statements(), "identify", []any{1, "p"}, []string{"(vault_id=? AND path=?)"}},
	{note.Statements(), "addressing", []any{1, "p"}, []string{"(vault_id=? AND path=?)"}},
	{note.Statements(), "fingerprints", []any{1, "note"}, []string{"sources_by_fingerprint"}},
	{note.Statements(), "search", []any{`"entropy"`, 1, 20}, []string{"chunks_fts"}},
	{note.Statements(), "stencils", []any{1, "stencil"}, []string{"notes_by_type"}},
	{note.Statements(), "notes_of_type", []any{1, "deck"}, []string{"notes_by_type"}},
	{note.Statements(), "types_at", []any{1, `["p"]`}, []string{"(vault_id=? AND path=?)"}},

	{chunk.Statements(), "identify", []any{1, "book", "p"}, []string{"(vault_id=? AND path=?)"}},
	{chunk.Statements(), "fingerprints", []any{1, "book"}, []string{"sources_by_fingerprint"}},
	{chunk.Statements(), "reading", []any{1, "library/note.epub"}, []string{"(vault_id=? AND path=?)"}},
	{chunk.Statements(), "sources_under", []any{1, "folder", 1, "folder/", "folder0"}, []string{"(vault_id=? AND path=?)", "(vault_id=? AND path>? AND path<?)"}},
	{chunk.Statements(), "sources_at", []any{1, "folder", "folder/", "folder0"}, []string{"(vault_id=? AND path=?)", "(vault_id=? AND path>? AND path<?)"}},
	// A folder and everything under it are filed at their new paths through the
	// same two lookups, and the note at the path itself is renamed by its own.
	{chunk.Statements(), "move_sources", []any{"science/folder", 7, 1, "folder", "folder/", "folder0"}, []string{"(vault_id=? AND path=?)", "(vault_id=? AND path>? AND path<?)"}},
	{chunk.Statements(), "note_naming", []any{1, "folder/note-00001.md"}, []string{"(vault_id=? AND path=?)", "INTEGER PRIMARY KEY"}},
	{chunk.Statements(), "rename_note", []any{"Entropy", "Entropy", 1, "folder/note-00001.md"}, []string{"(vault_id=? AND path=?)"}},
	{chunk.Statements(), "unchunked", []any{1, "book", 50}, []string{"sources_by_fingerprint", "chunks_by_source"}},
	{chunk.Statements(), "stale_recipe", []any{1, "book", `["epub-1","pdf-1"]`, 50}, []string{"sources_by_fingerprint"}},
	{chunk.Statements(), "unembedded", []any{"model", 1, 0, 50}, []string{"chunks_by_vault", "vectors_of"}},
	{chunk.Statements(), "passage", []any{1, 1}, []string{"INTEGER PRIMARY KEY"}},
	{chunk.Statements(), "enclosing", []any{1, 1}, []string{"INTEGER PRIMARY KEY"}},
	{chunk.Statements(), "progress", []any{"model", 1}, []string{"chunks_by_vault_parent", "vectors_of"}},
	// A search by words reads the full-text index and then the row each hit
	// names. A virtual table reports itself as a scan and has no named index.
	{chunk.Statements(), "lexical", []any{`"entropy"`, 1, `[]`, 20}, []string{"chunks_fts", "INTEGER PRIMARY KEY"}},
	// A search by name reads its own full-text index the same way.
	{chunk.Statements(), "sections", []any{`"entropy"`, 1, `[]`, 20}, []string{"parts_fts", "INTEGER PRIMARY KEY"}},
	{chunk.Statements(), "clear_parts", []any{1}, []string{"chunks_by_source"}},
	{chunk.Statements(), "clear_fts", []any{1}, []string{"chunks_by_source"}},
	// Cutting a source again reads what it holds now, and then moves, writes or
	// takes out one row at a time.
	{chunk.Statements(), "chunks_of", []any{1}, []string{"chunks_by_source"}},
	{chunk.Statements(), "move_chunk", []any{0, 10, 1, nil, 1}, []string{"INTEGER PRIMARY KEY"}},
	{chunk.Statements(), "delete_chunk", []any{1}, []string{"INTEGER PRIMARY KEY"}},
	// The coarse pass reads the vector index. That it stays inside one vault is
	// asserted in TestTheCoarsePassIsConstrainedInsideTheQuery and
	// TestTheCoarsePassStaysInsideItsVault.
	{chunk.Statements(), "search", []any{coarse, 1, 10}, []string{"chunks_vec"}},
}

// growing is the tables that grow with the vault. A question answered by
// reading one of them whole is a question that gets slower as the user writes.
var growing = []string{"sources", "notes", "links", "chunks", "vectors"}

// readsEverything says a plan step reads a table whole.
//
// A plan names the alias a statement gave the table, so `SCAN c` is how reading
// every chunk appears. Every scan is a failure except a full-text or
// nearest-neighbour match, which report themselves as a scan of their virtual
// table, and `vaults`, of which there are never many.
func readsEverything(step string) bool {
	return strings.HasPrefix(step, "SCAN ") &&
		!strings.Contains(step, "VIRTUAL TABLE") &&
		!strings.Contains(step, "vaults")
}

// TestEveryQuestionIsAnsweredThroughAnIndex asks the database how it intends to
// answer.
//
// The schema comes from the migrations, the database is populated because an
// empty one is answered from rules of thumb, and it is measured through the
// call a scan makes.
func TestEveryQuestionIsAnsweredThroughAnIndex(t *testing.T) {
	ctx := t.Context()
	db := populated(t)

	if err := (Statistics{db.write}).Changed(ctx); err != nil {
		t.Fatal(err)
	}

	for _, q := range expectedPlans {
		plan := planFor(ctx, t, db, q.statements[q.name], q.args)
		t.Logf("%s:\n    %s", q.name, strings.Join(plan, "\n    "))

		joined := strings.Join(plan, "\n")
		for _, want := range q.through {
			if !strings.Contains(joined, want) {
				t.Errorf("%s is not answered through %s", q.name, want)
			}
		}
		for _, step := range plan {
			if readsEverything(step) {
				t.Errorf("%s reads a table whole, and every table it reads is one of %v: %s",
					q.name, growing, step)
			}
		}
	}
}

// populated fills two vaults. One vault cannot show which index a question is
// answered through: a filter on a column that only ever holds one value is free
// whichever way the planner applies it, and it chooses on what it is told is
// there.
func populated(t *testing.T) *DB {
	t.Helper()
	ctx := t.Context()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	tx, err := db.write.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	exec := func(statement string, args ...any) {
		if _, err := tx.ExecContext(ctx, statement, args...); err != nil {
			t.Fatalf("%s: %v", statement, err)
		}
	}
	insert := func(statement string, args ...any) int64 {
		var row int64
		if err := tx.QueryRowContext(ctx, statement, args...).Scan(&row); err != nil {
			t.Fatalf("%s: %v", statement, err)
		}
		return row
	}

	exec(`INSERT INTO vaults (id, identifier, name, path)
	      VALUES (1, 'first', 'first', '/first'), (2, 'second', 'second', '/second')`)

	const notes = 1500
	for vault, prefix := range map[int]string{1: "note", 2: "quasar"} {
		for i := range notes {
			name := fmt.Sprintf("%s-%05d", prefix, i)
			source := insert(
				`INSERT INTO sources (vault_id, path, kind, size, modified_at)
				 VALUES (?, ?, 'note', 1, 1) RETURNING id`, vault, "folder/"+name+".md")
			// A few of them are stencils, so that a question about the stencils
			// of a vault is answered against a table where both answers occur.
			held := "note"
			if i%250 == 0 {
				held = "stencil"
			}
			exec(`INSERT INTO notes (source_id, vault_id, basename, title, type, identifier)
			      VALUES (?, ?, ?, ?, ?, ?)`,
				source, vault, name, name, held, fmt.Sprintf("01M%d%022d", vault, i))
			for j := range 3 {
				target := fmt.Sprintf("%s-%05d", prefix, (i+j+1)%notes)
				exec(`INSERT INTO links (note_id, position, scheme, value, value_base, role)
				      VALUES (?, ?, 'name', ?, ?, 'ref')`, source, j, target, target)
			}
			// Every second note is cut, so that a question about what is not cut
			// yet is answered against a table where both answers occur.
			if i%2 == 0 {
				cut(t, tx, source, int64(vault))
			}
		}
		// A book, cut and embedded like the notes beside it.
		book := insert(
			`INSERT INTO sources (vault_id, path, kind, size, modified_at, recipe)
			 VALUES (?, ?, 'book', 1000000, 1, 'epub') RETURNING id`,
			vault, fmt.Sprintf("library/%s.epub", prefix))
		cut(t, tx, book, int64(vault))
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return db
}

// cut writes one source's chunks: a large one, and two small ones inside it
// carrying a vector each. Every chunk is indexed for its words, as the
// repository indexes them.
func cut(t *testing.T, tx *sql.Tx, source, vault int64) {
	t.Helper()
	ctx := t.Context()

	var large int64
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO chunks (source_id, vault_id, start, length, parent, location, hash)
		 VALUES (?, ?, 0, 100, NULL, 'chapter 1', hex(randomblob(32))) RETURNING id`, source, vault).Scan(&large); err != nil {
		t.Fatal(err)
	}
	index(t, tx, large, "entropy and the observer, at length")
	for j := range 2 {
		var small int64
		if err := tx.QueryRowContext(ctx,
			`INSERT INTO chunks (source_id, vault_id, start, length, parent, location, hash)
			 VALUES (?, ?, ?, 50, ?, NULL, hex(randomblob(32))) RETURNING id`, source, vault, j*50, large).Scan(&small); err != nil {
			t.Fatal(err)
		}
		index(t, tx, small, "entropy and the observer")
		if _, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO vectors (fingerprint, recipe, v) VALUES (unhex((SELECT hash FROM chunks WHERE id = ?)), 'model', ?)`,
			small, coarse); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO chunks_vec (chunk_id, vault_id, embedding) VALUES (?, ?, vec_bit(?))`,
			small, vault, coarse); err != nil {
			t.Fatal(err)
		}
	}
}

// index puts one chunk's words in the full-text index, under the chunk's own
// number.
func index(t *testing.T, tx *sql.Tx, chunkID int64, text string) {
	t.Helper()
	if _, err := tx.ExecContext(t.Context(),
		`INSERT INTO chunks_fts (rowid, text) VALUES (?, ?)`, chunkID, text); err != nil {
		t.Fatal(err)
	}
}

func planFor(ctx context.Context, t *testing.T, db *DB, statement string, args []any) []string {
	t.Helper()
	rows, err := db.read.QueryContext(ctx, "EXPLAIN QUERY PLAN "+statement, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var plan []string
	for rows.Next() {
		var id, parent, notUsed int
		var detail string
		if err := rows.Scan(&id, &parent, &notUsed, &detail); err != nil {
			t.Fatal(err)
		}
		plan = append(plan, detail)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return plan
}
