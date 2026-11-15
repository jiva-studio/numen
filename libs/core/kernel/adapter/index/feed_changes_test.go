package index

import (
	"slices"
	"testing"
)

func TestFeedWatcherReadsTheCommittedChanges(t *testing.T) {
	db := openDB(t)
	v := getFirstVault()
	var seen [][]string
	t.Cleanup(db.FeedChanges().Watch(v.ID, func() {
		seen = append(seen, getEvents(listChanges(t, db, v, 0)))
	}))
	unwatched := 0
	t.Cleanup(db.FeedChanges().Watch(getSecond().ID, func() { unwatched++ }))

	saveNoteWithFrontmatter(t, db, v, "notes/river.md", nil, "River")

	if len(seen) != 1 || !slices.Equal(seen[0], []string{"notes/river.md file.added"}) {
		t.Errorf("the watcher read %v", seen)
	}
	if unwatched != 0 {
		t.Errorf("the watcher of another vault was called %d times", unwatched)
	}
}

func TestStoppedFeedWatcherIsNotCalled(t *testing.T) {
	db := openDB(t)
	v := getFirstVault()
	told := 0
	db.FeedChanges().Watch(v.ID, func() { told++ })()

	saveNoteWithFrontmatter(t, db, v, "notes/river.md", nil, "River")

	if told != 0 {
		t.Errorf("a stopped watcher was called %d times", told)
	}
}
