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
// through. Naming the index rather than merely forbidding a full scan is the
// point of the test: the wrong choice here is not a scan, it is the index that
// narrows to the vault and then reads every link in it — which reports itself
// as a search, looks entirely reasonable in a plan, and grows with the vault.
var expectedPlans = []struct {
	name    string
	args    []any
	indexes []string
}{
	{"candidates", []any{"v", "a", "v", "b", "v", "c"}, []string{"notes_by_basename"}},
	{"backlink_candidates", []any{"v", "id", "v", "base"}, []string{"links_by_target", "links_by_name"}},
	{"note_by_id", []any{"id"}, []string{"notes_by_id"}},
	{"links_of", []any{"v", "p"}, []string{"sqlite_autoindex_links_1"}},
}

// TestResolutionUsesIndexes asks the database how it intends to answer the
// questions link resolution asks. The wrong plan is paid once per link, on
// every note opened, and is invisible until a vault is large enough for
// someone to complain.
//
// The schema comes from the migrations rather than from a copy in this file,
// the database is populated because an empty one is answered from rules of
// thumb, and it is measured through the same call a scan makes — a test that
// measured the database itself would be certifying a plan the application never
// gets.
func TestResolutionUsesIndexes(t *testing.T) {
	ctx := t.Context()
	db := populated(t)

	if err := (Statistics{db.write}).Update(ctx); err != nil {
		t.Fatal(err)
	}

	statements := note.Statements()
	for _, q := range expectedPlans {
		plan := planFor(t, db, statements[q.name], q.args)
		t.Logf("%s:\n    %s", q.name, strings.Join(plan, "\n    "))

		joined := strings.Join(plan, "\n")
		for _, want := range q.indexes {
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

func planFor(t *testing.T, db *DB, statement string, args []any) []string {
	t.Helper()
	rows, err := db.read.QueryContext(context.Background(), "EXPLAIN QUERY PLAN "+statement, args...)
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
