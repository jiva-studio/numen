package index

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/index/note"
)

// TestResolutionUsesIndexes asks the database how it intends to answer the
// questions link resolution asks. A scan in one of these is not a matter of
// taste: it is paid once per link, on every note opened, and it is invisible
// until a vault is large enough for someone to complain.
//
// The schema comes from the migrations rather than from a copy in this file,
// and the plans are read from a populated database: with an empty table SQLite
// has no statistics and answers by heuristics, which is not what it will do on
// a real index.
func TestResolutionUsesIndexes(t *testing.T) {
	ctx := t.Context()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

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
	if _, err := db.write.ExecContext(ctx, "ANALYZE"); err != nil {
		t.Fatal(err)
	}

	statements := note.Statements()
	for _, q := range []struct {
		name string
		args []any
	}{
		{"candidates", []any{"v", "a", "v", "b", "v", "c"}},
		{"backlink_candidates", []any{"v", "id", "v", "base"}},
		{"note_by_id", []any{"id"}},
		{"links_of", []any{"v", "p"}},
	} {
		rows, err := db.read.QueryContext(ctx, "EXPLAIN QUERY PLAN "+statements[q.name], q.args...)
		if err != nil {
			t.Fatalf("%s: %v", q.name, err)
		}
		var plan []string
		for rows.Next() {
			var a, b, c int
			var detail string
			if err := rows.Scan(&a, &b, &c, &detail); err != nil {
				t.Fatal(err)
			}
			plan = append(plan, detail)
		}
		rows.Close()

		t.Logf("%s:\n    %s", q.name, strings.Join(plan, "\n    "))
		for _, step := range plan {
			if strings.HasPrefix(step, "SCAN") {
				t.Errorf("%s reads a whole table: %s", q.name, step)
			}
		}
	}
}
