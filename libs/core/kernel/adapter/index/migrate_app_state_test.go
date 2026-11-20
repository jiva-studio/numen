package index

import (
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func TestReplacingOpenTabsStartsWithEmptyAppState(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "index.db")
	db := openRawDB(t, path)
	defer testsupport.Close(t, db)
	steps, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range steps {
		if step.version >= 7 {
			break
		}
		if err := apply(ctx, db, step); err != nil {
			t.Fatal(err)
		}
	}
	const row = `INSERT INTO open_tabs (vault, position, id, kind, path, title, state, is_front)
		VALUES ('one', 0, 'a', 'note', 'a.md', 'A', '{}', 1)`
	if _, err := db.ExecContext(ctx, row); err != nil {
		t.Fatal(err)
	}
	if err := migrate(ctx, db, path); err != nil {
		t.Fatal(err)
	}
	if hasTable(t, db, "open_tabs") {
		t.Fatal("legacy open_tabs table remains after migration")
	}
	var count int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM app_state").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("new app state contains %d entries, want zero", count)
	}
}
