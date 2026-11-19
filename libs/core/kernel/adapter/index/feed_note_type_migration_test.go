package index

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/changefeed"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/filechange"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func TestFeedMigrationBackfillsLiveNotesAndResetsBeforeOldRemovals(t *testing.T) {
	ctx := t.Context()
	db := openRawDB(t, filepath.Join(t.TempDir(), "index.db"))
	defer testsupport.Close(t, db)
	steps, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range steps {
		if step.version >= 6 {
			break
		}
		if err := apply(ctx, db, step); err != nil {
			t.Fatal(err)
		}
	}
	const fixture = `
	INSERT INTO vaults (id, identifier) VALUES (1, 'garden');
	INSERT INTO change_feeds (vault_id, feed_id, last_seq) VALUES (1, 'feed', 3);
	INSERT INTO sources (id, vault_id, path, kind, size, modified_at) VALUES (1, 1, 'link.md', 'note', 10, 0);
	INSERT INTO notes (source_id, vault_id, folded_name, title, type) VALUES (1, 1, 'link', 'Link', 'link');
	INSERT INTO file_changes (vault_id, source_id, event, seq, change_time, path) VALUES
	    (1, 1, 'file.removed', 2, 1, 'old.md'),
	    (1, 1, 'file.added', 3, 2, 'link.md');`
	if _, err := db.ExecContext(ctx, fixture); err != nil {
		t.Fatal(err)
	}
	if err := migrate(ctx, db, "index.db"); err != nil {
		t.Fatal(err)
	}
	feed := changefeed.NewFeed(db)
	head, err := feed.GetHead(ctx, "garden")
	if err != nil {
		t.Fatal(err)
	}
	if !head.IsReset(filechange.Cursor{FeedID: "feed", Seq: 1}) || head.IsReset(filechange.Cursor{FeedID: "feed", Seq: 2}) {
		t.Errorf("reset boundary after migration = %+v", head)
	}
	changes, err := feed.ListFileChanges(ctx, "garden", filechange.GetEvents(), 0, 3, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 || len(changes[0].NoteTypes) != 0 || !slices.Equal(changes[1].NoteTypes, []domain.NoteType{"link"}) {
		t.Errorf("migrated changes = %+v", changes)
	}
}
