package index

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

// getStatements reads the SQL files that sit beside the repository in the folder.
func getStatements(folder string) map[string]string {
	return sqlfile.Load(os.DirFS(folder), "sql").All()
}

func getCoarse() []byte { return make([]byte, 128) }

func getExpectedPlans() []struct {
	statements map[string]string
	name       string
	args       []any
	endPos     []string
} {
	return []struct {
		statements map[string]string
		name       string
		args       []any
		endPos     []string
	}{
		{getStatements("note"), "candidates", []any{1, "a", 1, "b", 1, "c"}, []string{"notes_by_folded_name"}},
		{getStatements("note"), "backlink_candidates", []any{"id", 1, "base", 1}, []string{"links_by_target", "links_by_name"}},
		{getStatements("note"), "note_by_identifier", []any{"id"}, []string{"notes_by_identifier"}},
		{getStatements("note"), "links_of", []any{1}, []string{"(note_id=?)"}},
		{getStatements("note"), "identify", []any{1, "p"}, []string{"(vault_id=? AND path=?)"}},
		{getStatements("note"), "addressing", []any{1, "p"}, []string{"(vault_id=? AND path=?)"}},
		{getStatements("note"), "notes_of_type", []any{1, "sample"}, []string{"notes_by_type"}},
		{getStatements("note"), "types_at", []any{1, `["p"]`}, []string{"(vault_id=? AND path=?)"}},

		{getStatements("chunk"), "identify", []any{1, "book", "p"}, []string{"(vault_id=? AND path=?)"}},
		{getStatements("chunk"), "file_stats", []any{1, "book"}, []string{"sources_by_fingerprint"}},
		{getStatements("chunk"), "source_text", []any{1, "library/note.epub"}, []string{"(vault_id=? AND path=?)"}},
		{getStatements("chunk"), "file_stats_under", []any{1, "folder", 1, "folder/", "folder0"}, []string{"(vault_id=? AND path=?)", "(vault_id=? AND path>? AND path<?)"}},
		{getStatements("chunk"), "sources_under", []any{1, "folder", "folder/", "folder0"}, []string{"(vault_id=? AND path=?)", "(vault_id=? AND path>? AND path<?)"}},
		// A folder and everything under it are filed at their new paths through the
		// same two lookups, and the note at the path itself is renamed by its own.
		{getStatements("chunk"), "move_sources", []any{"science/folder", 7, 1, "folder", "folder/", "folder0"}, []string{"(vault_id=? AND path=?)", "(vault_id=? AND path>? AND path<?)"}},
		{getStatements("chunk"), "note_naming", []any{1, "folder/note-00001.md"}, []string{"(vault_id=? AND path=?)", "INTEGER PRIMARY KEY"}},
		{getStatements("chunk"), "rename_note", []any{"Entropy", "Entropy", 1, "folder/note-00001.md"}, []string{"(vault_id=? AND path=?)"}},
		{getStatements("chunk"), "unchunked", []any{1, "book", 50}, []string{"sources_by_fingerprint", "chunks_by_source"}},
		{getStatements("chunk"), "stale_recipe", []any{1, "book", `["epub-1","pdf-1"]`, 50}, []string{"sources_by_fingerprint"}},
		{getStatements("chunk"), "unembedded", []any{"model", 1, int64(math.MaxInt64), int64(math.MaxInt64), int64(math.MaxInt64), int64(math.MaxInt64), int64(math.MaxInt64), 0, 0, 50}, []string{"sources_by_modified", "vectors_by_hash"}},
		{getStatements("chunk"), "passage", []any{1, 1}, []string{"INTEGER PRIMARY KEY"}},
		{getStatements("chunk"), "enclosing", []any{1, 1}, []string{"INTEGER PRIMARY KEY"}},
		{getStatements("chunk"), "progress", []any{"model", 1, 0}, []string{"chunks_by_vault_parent", "vectors_by_hash"}},
		// A search by words reads the full-text index and then the row each hit
		// names. A virtual table reports itself as a scan and has no named index.
		{getStatements("chunk"), "lexical", []any{`"entropy"`, 1, 20, "notes/", "notes/"}, []string{"chunks_fts", "INTEGER PRIMARY KEY"}},
		{getStatements("chunk"), "lexical_kinds", []any{`"entropy"`, 1, `["note"]`, 20, "notes/", "notes/"}, []string{"chunks_fts", "INTEGER PRIMARY KEY"}},
		// A search by name reads its own full-text index the same way.
		{getStatements("chunk"), "sections", []any{`"entropy"`, 1, 20, "notes/", "notes/"}, []string{"sections_fts", "INTEGER PRIMARY KEY"}},
		{getStatements("chunk"), "sections_kinds", []any{`"entropy"`, 1, `["note"]`, 20, "notes/", "notes/"}, []string{"sections_fts", "INTEGER PRIMARY KEY"}},
		{getStatements("chunk"), "clear_sections", []any{1}, []string{"chunks_by_source"}},
		{getStatements("chunk"), "clear_fts", []any{1}, []string{"chunks_by_source"}},
		// Cutting a source again reads its current contents now, and then moves, writes or
		// takes out one row at a time.
		{getStatements("chunk"), "chunks_of", []any{1}, []string{"chunks_by_source"}},
		{getStatements("chunk"), "move_chunk", []any{0, 10, 1, nil, 1}, []string{"INTEGER PRIMARY KEY"}},
		{getStatements("chunk"), "delete_chunk", []any{1}, []string{"INTEGER PRIMARY KEY"}},
		// The coarse pass reads the vector index. That it stays inside one vault is
		// asserted in TestTheCoarsePassIsConstrainedInsideTheQuery and
		// TestTheCoarsePassStaysInsideItsVault.
		// The coarse pass under a path prefix finds the sources by the range of their
		// paths, their chunks by source, and the vector of each chunk by its key: the
		// "!" of the plan of the vector table is its lookup of one row by key.
		{getStatements("chunk"), "coarse_prefix", []any{getCoarse(), 1, "notes/", 10, "notes0", "notes/", "notes0"}, []string{"(vault_id=? AND path>? AND path<?)", "chunks_by_source", "SCAN v VIRTUAL TABLE INDEX 5:2!"}},
		{getStatements("chunk"), "coarse", []any{getCoarse(), 1, 10}, []string{"chunks_vec"}},
		// The cached hash of a file is its source row, found by vault and path. The
		// vault is given by its identifier, then the path; a save gives the record's
		// hash, size, modification time, file id and hash time, the vault and path, and
		// the size, modification time and hash the row must have.
		{getStatements("version"), "find_content_hash", []any{"first", "folder/note-00001.md"}, []string{"(vault_id=? AND path=?)"}},
		{getStatements("version"), "save_content_hash", []any{"h", 1, 1, 7, 9, "first", "folder/note-00001.md", 1, 1, "h"}, []string{"(vault_id=? AND path=?)"}},
	}
}

func getGrowing() []string { return []string{"sources", "notes", "links", "chunks", "vectors"} }

func readsEverything(step string) bool {
	return strings.HasPrefix(step, "SCAN ") &&
		!strings.Contains(step, "VIRTUAL TABLE") &&
		!strings.Contains(step, "vaults") &&
		!strings.HasPrefix(step, "SCAN h")
}

// TestQueryPlans_UseIndexes reads the database's query plan.
//
// The schema comes from the migrations. The database is populated because the
// planner uses rules of thumb for an empty one, and the plans are obtained
// through the call a scan makes.
func TestQueryPlans_UseIndexes(t *testing.T) {
	ctx := t.Context()
	db := openPopulatedDB(t)

	if err := (DatabaseMaintenance{db.write}).ReportChanges(ctx); err != nil {
		t.Fatal(err)
	}

	for _, q := range getExpectedPlans() {
		statement, ok := q.statements[q.name]
		if !ok {
			t.Errorf("there is no statement %s", q.name)
			continue
		}
		plan := planFor(ctx, t, db, statement, q.args)
		t.Logf("%s:\n    %s", q.name, strings.Join(plan, "\n    "))

		joined := strings.Join(plan, "\n")
		for _, want := range q.endPos {
			if !strings.Contains(joined, want) {
				t.Errorf("%s is not answered through %s", q.name, want)
			}
		}
		for _, step := range plan {
			if readsEverything(step) {
				t.Errorf("%s reads a table whole, and every table it reads is one of %v: %s",
					q.name, getGrowing(), step)
			}
		}
	}
}

func openPopulatedDB(t *testing.T) *DB {
	t.Helper()
	ctx := t.Context()
	db, err := Open(ctx, newMigrated().Path(t))
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)

	tx, err := db.write.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			t.Error(err)
		}
	}()

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

	exec(`INSERT INTO vaults (id, identifier)
	      VALUES (1, 'first'), (2, 'second')`)

	const notes = 1500
	for vault, prefix := range map[int]string{1: "note", 2: "quasar"} {
		for i := range notes {
			name := fmt.Sprintf("%s-%05d", prefix, i)
			source := insert(
				`INSERT INTO sources (vault_id, path, kind, size, modified_at)
				 VALUES (?, ?, 'note', 1, 1) RETURNING id`, vault, "folder/"+name+".md")
			// A few notes are forms, so queries on the form type run against a
			// table that contains both values.
			targetVal := "note"
			if i%250 == 0 {
				targetVal = "form"
			}
			exec(`INSERT INTO notes (source_id, vault_id, folded_name, title, type, identifier)
			      VALUES (?, ?, ?, ?, ?, ?)`,
				source, vault, name, name, targetVal, fmt.Sprintf("01M%d%022d", vault, i))
			for j := range 3 {
				target := fmt.Sprintf("%s-%05d", prefix, (i+j+1)%notes)
				exec(`INSERT INTO links (note_id, position, scheme, target, folded_name, role)
				      VALUES (?, ?, 'name', ?, ?, 'ref')`, source, j, target, target)
			}
			// Every second note is chunked, so queries for unchunked notes run
			// against a table that contains both chunked and unchunked notes.
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

func cut(t *testing.T, tx *sql.Tx, source, vault int64) {
	t.Helper()
	ctx := t.Context()

	var large int64
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO chunks (source_id, vault_id, start, length, parent_id, location, hash)
		 VALUES (?, ?, 0, 100, NULL, 'chapter 1', hex(randomblob(32))) RETURNING id`, source, vault).Scan(&large); err != nil {
		t.Fatal(err)
	}
	index(t, tx, large, "entropy and the observer, at length")
	for j := range 2 {
		var small int64
		if err := tx.QueryRowContext(ctx,
			`INSERT INTO chunks (source_id, vault_id, start, length, parent_id, location, hash)
			 VALUES (?, ?, ?, 50, ?, NULL, hex(randomblob(32))) RETURNING id`, source, vault, j*50, large).Scan(&small); err != nil {
			t.Fatal(err)
		}
		index(t, tx, small, "entropy and the observer")
		if _, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO vectors (hash, recipe, embedding) VALUES (unhex((SELECT hash FROM chunks WHERE id = ?)), 'model', ?)`,
			small, getCoarse()); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO chunks_vec (chunk_id, vault_id, coarse) VALUES (?, ?, vec_bit(?))`,
			small, vault, getCoarse()); err != nil {
			t.Fatal(err)
		}
	}
}

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
	defer func() {
		if err := rows.Close(); err != nil {
			t.Error(err)
		}
	}()

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
