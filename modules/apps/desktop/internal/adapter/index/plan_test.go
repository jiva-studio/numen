package index

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/index/note"
)

// The questions link resolution asks, and the index each one has to be answered
// through.
var expectedPlans = []struct {
	name string
	args []any
	// An index name, or the columns for a primary key: SQLite names those
	// itself, and the name changes when a constraint is added.
	through []string
}{
	{"candidates", []any{"v", "a", "v", "b", "v", "c"}, []string{"notes_by_basename"}},
	{"backlink_candidates", []any{"v", "id", "v", "base"}, []string{"links_by_target", "links_by_name"}},
	{"note_by_id", []any{"id"}, []string{"notes_by_id"}},
	{"links_of", []any{"v", "p"}, []string{"(vault_id=? AND from_path=?)"}},
}

// TestResolutionUsesIndexes asks the database how it intends to answer.
//
// The schema comes from the migrations, the database is populated because an
// empty one is answered from rules of thumb, and it is measured through the
// call a scan makes.
func TestResolutionUsesIndexes(t *testing.T) {
	ctx := t.Context()
	db := populated(t)

	if err := (Statistics{db.write}).Changed(ctx); err != nil {
		t.Fatal(err)
	}

	statements := note.Statements()
	for _, q := range expectedPlans {
		plan := planFor(ctx, t, db, statements[q.name], q.args)
		t.Logf("%s:\n    %s", q.name, strings.Join(plan, "\n    "))

		joined := strings.Join(plan, "\n")
		for _, want := range q.through {
			if !strings.Contains(joined, want) {
				t.Errorf("%s is not answered through %s", q.name, want)
			}
		}
		for _, step := range plan {
			if strings.HasPrefix(step, "SCAN") {
				t.Errorf("%s reads a whole table: %s", q.name, step)
			}
		}
	}
}

func populated(t *testing.T) *DB {
	t.Helper()
	ctx := t.Context()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := db.write.ExecContext(ctx,
		`INSERT INTO vaults (id, name, path) VALUES ('v', 'v', '/v')`); err != nil {
		t.Fatal(err)
	}
	for i := range 3000 {
		name := fmt.Sprintf("note-%05d", i)
		path := "folder/" + name + ".md"
		if _, err := db.write.ExecContext(ctx,
			`INSERT INTO files (vault_id, path, size, mtime) VALUES ('v', ?, 1, 1)`, path); err != nil {
			t.Fatal(err)
		}
		if _, err := db.write.ExecContext(ctx,
			`INSERT INTO notes (vault_id, path, title, basename, note_id) VALUES ('v', ?, ?, ?, ?)`,
			path, name, name, fmt.Sprintf("01M%023d", i)); err != nil {
			t.Fatal(err)
		}
		for j := range 3 {
			target := fmt.Sprintf("note-%05d", (i+j+1)%3000)
			if _, err := db.write.ExecContext(ctx,
				`INSERT INTO links (vault_id, from_path, scheme, value, role, value_base, position)
				 VALUES ('v', ?, 'name', ?, 'ref', ?, ?)`, path, target, target, j); err != nil {
				t.Fatal(err)
			}
		}
	}
	return db
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
