package index

import (
	"path/filepath"
	"testing"
)

func TestMigrationFileKindFilesEverySourceButANoteAsAFile(t *testing.T) {
	ctx := t.Context()
	db := openBreaking(t, filepath.Join(t.TempDir(), "index.db"))
	all, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	var fileKind migration
	for _, m := range all {
		if m.name == "0007_file_kind.sql" {
			fileKind = m
			break
		}
		if err := apply(ctx, db, m); err != nil {
			t.Fatal(err)
		}
	}
	for _, statement := range []string{
		`INSERT INTO vaults (id, identifier) VALUES (1, 'v')`,
		`INSERT INTO sources (vault_id, path, kind, size, modified_at) VALUES (1, 'a.md', 'note', 1, 1)`,
		`INSERT INTO sources (vault_id, path, kind, size, modified_at) VALUES (1, 'b.sample', 'sample', 1, 1)`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}

	if err := apply(ctx, db, fileKind); err != nil {
		t.Fatal(err)
	}
	var kinds string
	if err := db.QueryRowContext(ctx, `SELECT group_concat(kind, ',') FROM (SELECT kind FROM sources ORDER BY path)`).Scan(&kinds); err != nil {
		t.Fatal(err)
	}
	if kinds != "note,file" {
		t.Errorf("kinds = %q, want the note kept and the other source filed as a file", kinds)
	}
}
