package index

import (
	"path/filepath"
	"testing"
)

func TestMigrationLinkMarkerAddsTheMarkerColumnToLinks(t *testing.T) {
	ctx := t.Context()
	db := openBreaking(t, filepath.Join(t.TempDir(), "index.db"))
	all, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	var marker migration
	for _, m := range all {
		if m.name == "0009_link_marker.sql" {
			marker = m
			break
		}
		if err := apply(ctx, db, m); err != nil {
			t.Fatal(err)
		}
	}
	if marker.name == "" {
		t.Fatal("migration 0009_link_marker.sql is not embedded")
	}

	count := func() int {
		var n int
		if err := db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM pragma_table_info('links') WHERE name = 'marker'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if count() != 0 {
		t.Fatal("the column exists before the migration that adds it")
	}
	if err := apply(ctx, db, marker); err != nil {
		t.Fatal(err)
	}
	if got := count(); got != 1 {
		t.Errorf("links has %d marker columns, want 1", got)
	}
}
