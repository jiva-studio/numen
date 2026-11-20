package index

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/opentabs"
	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func TestAppStateMigrationPreservesTheWorkspaceOfEachVault(t *testing.T) {
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
	const oldState = "{ \"value\" : \"a\\\"b\" }"
	const rows = `INSERT INTO open_tabs (vault,position,id,kind,path,title,state,is_front) VALUES
		('one', 1, 'b', 'reader', 'Docs/b.pdf', 'B', ?, 1),
		('two', 0, 'c', 'new', '', 'C', '', 0),
		('one', 0, 'a', 'notes', 'a.md', 'A', '', 0)`
	if _, err := db.ExecContext(ctx, rows, oldState); err != nil {
		t.Fatal(err)
	}
	if err := migrate(ctx, db, path); err != nil {
		t.Fatal(err)
	}
	if hasTable(t, db, "open_tabs") {
		t.Fatal("legacy open_tabs table remains after migration")
	}
	repo := opentabs.NewRepository(db, db)
	for id, want := range map[domain.VaultID]windowdomain.OpenTabs{
		"one": {Tabs: []windowdomain.Tab{
			{ID: "a", Kind: "notes", Path: "a.md", Title: "A"},
			{ID: "b", Kind: "reader", Path: "Docs/b.pdf", Title: "B", State: oldState},
		}, ActiveID: "b"},
		"two": {Tabs: []windowdomain.Tab{{ID: "c", Kind: "new", Title: "C"}}},
	} {
		got, err := repo.ReadOpenTabs(ctx, domain.Vault{ID: id})
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("vault %s = %+v, %v, want %+v", id, got, err, want)
		}
	}
}
