package index

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func saveNoteWithContent(t *testing.T, db *DB, v domain.Vault, path, hash string, size int64, modifiedAt time.Time) {
	t.Helper()
	err := saveCut(t.Context(), db, v.ID, []domain.Note{{
		FileStat: domain.FileStat{Path: path, Kind: domain.KindNote, Size: size, ModTime: modifiedAt, Hash: hash},
		Title:    path,
		Type:     domain.TypeNote,
	}})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAChangeCarriesTheHashSizeAndModificationTimeOfItsSource(t *testing.T) {
	db := openDB(t)
	v := getFirstVault()
	modifiedAt := time.Unix(1_700_000_000, 123)
	hash := domain.HashContent([]byte("River"))
	saveNoteWithContent(t, db, v, "notes/river.md", hash, 5, modifiedAt)

	changes := listChanges(t, db, v, 0)

	if len(changes) != 1 {
		t.Fatalf("the feed holds %+v, want one change", changes)
	}
	if got := changes[0]; got.Hash != hash || got.Size != 5 || got.ModifiedAt != modifiedAt.UnixNano() {
		t.Errorf("the change has hash %q, size %d, modified at %d, want %q, 5, %d", got.Hash, got.Size, got.ModifiedAt, hash, modifiedAt.UnixNano())
	}
}

func TestAChangeAfterASaveHasTheLatestHashSizeAndModificationTimeOfItsSource(t *testing.T) {
	db := openDB(t)
	v := getFirstVault()
	saveNoteWithContent(t, db, v, "notes/river.md", domain.HashContent([]byte("River")), 5, time.Unix(1_700_000_000, 0))
	latest := domain.HashContent([]byte("River runs"))
	saveNoteWithContent(t, db, v, "notes/river.md", latest, 10, time.Unix(1_700_000_050, 0))

	for _, one := range listChanges(t, db, v, 0) {
		if one.Hash != latest || one.Size != 10 || one.ModifiedAt != time.Unix(1_700_000_050, 0).UnixNano() {
			t.Errorf("%s of %s has hash %q, size %d, modified at %d, want the latest save", one.Event, one.Path, one.Hash, one.Size, one.ModifiedAt)
		}
	}
}

func TestAChangeOfASourceWithoutAHashHasAnEmptyHash(t *testing.T) {
	db := openDB(t)
	v := getFirstVault()
	saveNoteWithContent(t, db, v, "notes/river.md", "", 5, time.Unix(1_700_000_000, 0))

	changes := listChanges(t, db, v, 0)

	if len(changes) != 1 || changes[0].Hash != "" || changes[0].Size != 5 {
		t.Errorf("the feed holds %+v, want an empty hash and size 5", changes)
	}
}

func TestARemovalHasNoHashSizeOrModificationTime(t *testing.T) {
	db := openDB(t)
	v := getFirstVault()
	saveNoteWithContent(t, db, v, "notes/river.md", domain.HashContent([]byte("River")), 5, time.Unix(1_700_000_000, 0))
	if err := db.Notes().Remove(t.Context(), v.ID, []string{"notes/river.md"}); err != nil {
		t.Fatal(err)
	}

	changes := listChanges(t, db, v, 0)

	if len(changes) != 1 || changes[0].Hash != "" || changes[0].Size != 0 || changes[0].ModifiedAt != 0 {
		t.Errorf("the feed holds %+v, want one removal with empty values", changes)
	}
}
