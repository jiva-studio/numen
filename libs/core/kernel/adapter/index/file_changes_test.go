package index

import (
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/changefeed"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/filechange"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/frontmatter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func listChanges(t *testing.T, db *DB, v domain.Vault, after int64) []filechange.Change {
	t.Helper()
	head := getFeedHead(t, db, v)
	changes, err := db.FileChanges().ListFileChanges(t.Context(), v.ID, filechange.GetEvents(), after, head.Cursor.Seq, 1000)
	if err != nil {
		t.Fatal(err)
	}
	return changes
}

func getFeedHead(t *testing.T, db *DB, v domain.Vault) filechange.Head {
	t.Helper()
	head, err := db.FileChanges().GetHead(t.Context(), v.ID)
	if err != nil {
		t.Fatal(err)
	}
	return head
}

func getEvents(changes []filechange.Change) []string {
	out := make([]string, 0, len(changes))
	for _, one := range changes {
		out = append(out, one.Path+" "+one.Event)
	}
	return out
}

func getSourceID(t *testing.T, db *DB, v domain.Vault, path string) filechange.SourceID {
	t.Helper()
	ids := rowsOf(t, db, `SELECT s.id FROM sources s WHERE s.vault_id = (SELECT id FROM vaults WHERE identifier = ?) AND s.path = ?`, v.ID, path)
	if len(ids) != 1 {
		t.Fatalf("%s is held as %v", path, ids)
	}
	return filechange.SourceID(ids[0])
}

func TestAFileChangedTwiceHasOneRowForEachEvent(t *testing.T) {
	db := openDB(t)
	v := getFirstVault()
	saveNoteWithFrontmatter(t, db, v, "notes/river.md", map[string]any{"tags": "water"}, "River")
	saveNoteWithFrontmatter(t, db, v, "notes/river.md", map[string]any{"tags": "delta"}, "River runs")
	saveNoteWithFrontmatter(t, db, v, "notes/river.md", map[string]any{"tags": "sea"}, "River runs on")

	want := []string{"notes/river.md file.added", "notes/river.md file.changed", "notes/river.md note.frontmatterChanged"}
	got := getEvents(listChanges(t, db, v, 0))
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("the feed holds %v, want %v", got, want)
	}
}

func TestTheChangesOfAVaultAreNumberedInTheOrderTheyWereStored(t *testing.T) {
	db := openDB(t)
	saveNoteWithFrontmatter(t, db, getFirstVault(), "a.md", nil, "A")
	saveNoteWithFrontmatter(t, db, getSecond(), "x.md", nil, "X")
	saveNoteWithFrontmatter(t, db, getFirstVault(), "b.md", nil, "B")
	saveNoteWithFrontmatter(t, db, getFirstVault(), "a.md", nil, "A again")

	changes := listChanges(t, db, getFirstVault(), 0)
	seqs := make([]int64, 0, len(changes))
	for _, one := range changes {
		seqs = append(seqs, one.Seq)
	}
	if !slices.Equal(seqs, []int64{1, 2, 3}) {
		t.Errorf("the first vault's changes are numbered %v, want [1 2 3]", seqs)
	}
	want := []string{"a.md file.added", "b.md file.added", "a.md file.changed"}
	if got := getEvents(changes); !slices.Equal(got, want) {
		t.Errorf("the changes are %v, want %v", got, want)
	}
	if head := getFeedHead(t, db, getSecond()); head.Cursor.Seq != 1 {
		t.Errorf("the second vault's feed is at %d, want 1", head.Cursor.Seq)
	}
}

func TestARemovalReplacesTheOtherEventsOfItsFile(t *testing.T) {
	db := openDB(t)
	v := getFirstVault()
	saveNoteWithFrontmatter(t, db, v, "notes/river.md", map[string]any{"tags": "water"}, "River")
	id := getSourceID(t, db, v, "notes/river.md")

	if err := db.Notes().Remove(t.Context(), v.ID, []string{"notes/river.md"}); err != nil {
		t.Fatal(err)
	}

	changes := listChanges(t, db, v, 0)
	if len(changes) != 1 || changes[0].Event != manifest.EventFileRemoved || changes[0].SourceID != id || changes[0].Path != "notes/river.md" {
		t.Errorf("the feed holds %+v, want only the removal of source %d at the path it had", changes, id)
	}
}

func TestAMovedFileKeepsItsSourceAndHasOneMoveChange(t *testing.T) {
	db := openDB(t)
	v := getFirstVault()
	saveNoteWithFrontmatter(t, db, v, "physics/Entropy.md", map[string]any{"tags": "heat"}, "Entropy")
	id := getSourceID(t, db, v, "physics/Entropy.md")
	before := getFeedHead(t, db, v).Cursor.Seq

	if err := db.Sources().MoveSources(t.Context(), v.ID, "physics/Entropy.md", "science/Entropy.md"); err != nil {
		t.Fatal(err)
	}

	changes := listChanges(t, db, v, before)
	want := []filechange.Change{{
		SourceID: id, Path: "science/Entropy.md", Event: manifest.EventFileMoved, Seq: before + 1,
		NoteTypes: []domain.NoteType{domain.TypeNote}, Size: int64(len("Entropy")), ModifiedAt: getWalkedTime().UnixNano(),
	}}
	if !reflect.DeepEqual(changes, want) {
		t.Errorf("the move recorded %+v, want %+v", changes, want)
	}
}

func TestTrimmedRemovalsMakeAnOlderCursorReset(t *testing.T) {
	db := openDB(t)
	v := getFirstVault()
	saveNoteWithFrontmatter(t, db, v, "a.md", nil, "A")
	start := getFeedHead(t, db, v).Cursor
	if err := db.Notes().Remove(t.Context(), v.ID, []string{"a.md"}); err != nil {
		t.Fatal(err)
	}
	removal := getFeedHead(t, db, v).Cursor

	if err := changefeed.Trim(t.Context(), db.write, time.Hour); err != nil {
		t.Fatal(err)
	}
	if got := listChanges(t, db, v, 0); len(got) != 1 {
		t.Fatalf("a removal of now was trimmed: the feed holds %+v", got)
	}

	if err := changefeed.Trim(t.Context(), db.write, -time.Hour); err != nil {
		t.Fatal(err)
	}
	if got := listChanges(t, db, v, 0); len(got) != 0 {
		t.Errorf("the trimmed feed holds %+v", got)
	}
	head := getFeedHead(t, db, v)
	if !head.IsReset(start) {
		t.Errorf("a cursor before a trimmed removal does not reset: head %+v, cursor %+v", head, start)
	}
	if head.IsReset(removal) {
		t.Errorf("a cursor at the trimmed removal resets: head %+v, cursor %+v", head, removal)
	}
}

func TestANewIndexStartsAFeedOfAnotherID(t *testing.T) {
	first := openDB(t)
	second := openDB(t)
	a, b := getFeedHead(t, first, getFirstVault()), getFeedHead(t, second, getFirstVault())
	if a.Cursor.FeedID == "" || a.Cursor.FeedID == b.Cursor.FeedID {
		t.Errorf("two indexes have the feeds %q and %q", a.Cursor.FeedID, b.Cursor.FeedID)
	}
	if !b.IsReset(a.Cursor) {
		t.Error("a cursor of another feed does not reset")
	}
	if err := first.Vaults().Register(t.Context(), getFirstVault().ID); err != nil {
		t.Fatal(err)
	}
	if again := getFeedHead(t, first, getFirstVault()); again.Cursor.FeedID != a.Cursor.FeedID {
		t.Error("registering a vault again gave it another feed")
	}
}

func TestAVaultOfAnIndexOfTheVersionBeforeGetsAFeed(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "index.db")
	raw := openRawDB(t, path)
	one, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	if err := apply(ctx, raw, one[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `INSERT INTO vaults (identifier) VALUES (?)`, string(getFirstVault().ID)); err != nil {
		t.Fatal(err)
	}
	testsupport.Close(t, raw)

	db, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, db)
	if head := getFeedHead(t, db, getFirstVault()); head.Cursor.FeedID == "" {
		t.Error("the vault has no feed after the migration")
	}
}

func TestAChangeDuringAFullReadIsAfterTheCursorTakenBeforeIt(t *testing.T) {
	db := openDB(t)
	v := getFirstVault()
	saveNoteWithFrontmatter(t, db, v, "a.md", map[string]any{"tags": "one"}, "A")
	cursor := getFeedHead(t, db, v).Cursor

	saveNoteWithFrontmatter(t, db, v, "b.md", map[string]any{"tags": "two"}, "B")
	if _, err := db.NoteQueryReads().ListFrontmatter(t.Context(), v.ID, frontmatter.Query{Limit: 1}); err != nil {
		t.Fatal(err)
	}

	got := getEvents(listChanges(t, db, v, cursor.Seq))
	if !slices.Contains(got, "b.md note.frontmatterChanged") {
		t.Errorf("the feed after the cursor holds %v, want the frontmatter of b.md", got)
	}
}

func TestTheStartOfTheFeedReadsAfterTrimmedRemovals(t *testing.T) {
	db := openDB(t)
	v := getFirstVault()
	saveNoteWithFrontmatter(t, db, v, "a.md", nil, "A")
	if err := db.Notes().Remove(t.Context(), v.ID, []string{"a.md"}); err != nil {
		t.Fatal(err)
	}
	if err := changefeed.Trim(t.Context(), db.write, -time.Hour); err != nil {
		t.Fatal(err)
	}
	head := getFeedHead(t, db, v)
	if start := (filechange.Cursor{FeedID: head.Cursor.FeedID, Seq: 0}); head.IsReset(start) {
		t.Errorf("the start of the feed resets: head %+v", head)
	}
}

func TestFeedKeepsChangedNoteTypesSinceTheCursor(t *testing.T) {
	db := openDB(t)
	v := getFirstVault()
	const path = "source.md"
	write := func(kind domain.NoteType, body string) {
		t.Helper()
		note := domain.Note{
			FileStat: domain.FileStat{Path: path, Kind: domain.KindNote, Size: int64(len(body)), ModTime: getWalkedTime()},
			Type:     kind, Body: body, Frontmatter: map[string]any{"type": string(kind), "title": body},
		}
		if err := saveCut(t.Context(), db, v.ID, []domain.Note{note}); err != nil {
			t.Fatal(err)
		}
	}
	check := func(after int64, kinds ...domain.NoteType) {
		t.Helper()
		files, err := db.FileChanges().ListChangedFiles(t.Context(), v.ID, after, getFeedHead(t, db, v).Cursor.Seq)
		if err != nil {
			t.Fatal(err)
		}
		var got []domain.NoteType
		for _, file := range files {
			if file.Path != path {
				t.Errorf("changed path = %q, want %q", file.Path, path)
			}
			got = append(got, file.NoteType)
		}
		slices.Sort(got)
		if !slices.Equal(got, kinds) {
			t.Errorf("changed note types = %v, want %v", got, kinds)
		}
	}
	write("link", "one")
	cursor := getFeedHead(t, db, v).Cursor.Seq
	write(domain.TypeNote, "second")
	changed := getFeedHead(t, db, v).Cursor.Seq
	write(domain.TypeNote, "another title")
	check(cursor, "link", domain.TypeNote)
	check(changed, domain.TypeNote)
	if err := db.Notes().Remove(t.Context(), v.ID, []string{path}); err != nil {
		t.Fatal(err)
	}
	check(cursor, "link", domain.TypeNote)
	check(changed, domain.TypeNote)
	page := listChanges(t, db, v, changed)
	if len(page) != 1 || !slices.Equal(page[0].NoteTypes, []domain.NoteType{domain.TypeNote}) {
		t.Fatalf("changes since the note was retyped = %+v", page)
	}
}
