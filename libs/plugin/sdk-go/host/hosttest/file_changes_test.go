package hosttest_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
)

func getAllEvents() []string {
	return []string{host.EventFileAdded, host.EventFileChanged, host.EventFileMoved, host.EventFileRemoved, host.EventNoteFrontmatterChanged}
}

func readFeed(t *testing.T, v *hosttest.Vault, since host.FileChangeCursor) host.FileChangePage {
	t.Helper()
	page, err := v.ListFileChanges(t.Context(), host.FileChangeQuery{Since: since, Events: getAllEvents()})
	if err != nil {
		t.Fatal(err)
	}
	return page
}

func describe(changes []host.FileChange) []string {
	out := make([]string, 0, len(changes))
	for _, one := range changes {
		out = append(out, one.Path+" "+one.Event)
	}
	return out
}

func TestListFileChangesReturnsTheLatestEventOfEachKindOfEachFile(t *testing.T) {
	v := hosttest.New("garden")
	start := readFeed(t, v, host.FileChangeCursor{}).Next
	v.WriteFile("a.md", "---\ntags: lake\n---\nA\n")
	v.WriteFile("scan.pdf", "%PDF")
	v.WriteFile("a.md", "---\ntags: lake\n---\nA, longer\n")
	v.WriteFile("a.md", "---\ntags: river\n---\nA, longer\n")

	got := describe(readFeed(t, v, start).Changes)
	want := []string{"a.md file.added", "scan.pdf file.added", "a.md file.changed", "a.md note.frontmatterChanged"}
	if !slices.Equal(got, want) {
		t.Errorf("the feed holds %v, want %v", got, want)
	}
}

func TestAMovedNoteKeepsItsSourceAndARemovalDropsTheOtherEvents(t *testing.T) {
	v := hosttest.New("garden")
	v.WriteFile("a.md", "---\ntags: lake\n---\nA\n")
	v.WriteFile("b.md", "B\n")
	page, err := v.ReadFrontmatter(t.Context(), host.FrontmatterQuery{Paths: []string{"a.md"}})
	if err != nil {
		t.Fatal(err)
	}
	id, cursor := page.Notes[0].SourceID, page.Cursor

	if _, err := v.MoveNotes(t.Context(), []string{"a.md"}, "lakes"); err != nil {
		t.Fatal(err)
	}
	v.Remove("b.md")

	moved, err := v.StatFile(t.Context(), "lakes/a.md")
	if err != nil {
		t.Fatal(err)
	}
	changes := readFeed(t, v, cursor).Changes
	wantMoved := host.FileChange{
		SourceID: id, Path: "lakes/a.md", Event: host.EventFileMoved, Seq: cursor.Seq + 1,
		Hash: moved.Hash, Size: moved.Size, ModifiedAt: moved.ModTime.UnixNano(),
	}
	if len(changes) != 2 || changes[0] != wantMoved ||
		changes[1].Event != host.EventFileRemoved || changes[1].Path != "b.md" {
		t.Errorf("after the move and the removal the feed holds %+v", changes)
	}
	if got := describe(readFeed(t, v, host.FileChangeCursor{FeedID: cursor.FeedID}).Changes); slices.Contains(got, "b.md file.added") {
		t.Errorf("the removed file keeps its other events: %v", got)
	}
}

func TestAResetFeedOrTrimmedRemovalsResetACursor(t *testing.T) {
	v := hosttest.New("garden")
	v.WriteFile("a.md", "A\n")
	before := readFeed(t, v, host.FileChangeCursor{}).Next
	v.Remove("a.md")
	v.TrimRemovals()
	if page := readFeed(t, v, before); !page.IsReset {
		t.Errorf("a cursor before trimmed removals answered %+v", page)
	}
	at := readFeed(t, v, host.FileChangeCursor{}).Next
	v.ResetFeed()
	if page := readFeed(t, v, at); !page.IsReset || page.Next.FeedID == at.FeedID {
		t.Errorf("a cursor of the feed before answered %+v", page)
	}
	if _, err := v.ListFileChanges(t.Context(), host.FileChangeQuery{}); !errors.Is(err, host.ErrInvalidFileChangeQuery) {
		t.Errorf("a query without events answered %v", err)
	}
}

func TestTheStartOfTheFeedListsTheFilesAfterTrimmedRemovals(t *testing.T) {
	v := hosttest.New("garden")
	v.WriteFile("a.md", "A\n")
	v.WriteFile("b.md", "B\n")
	v.Remove("a.md")
	v.TrimRemovals()
	head := readFeed(t, v, host.FileChangeCursor{}).Next
	page := readFeed(t, v, host.FileChangeCursor{FeedID: head.FeedID})
	if page.IsReset || !slices.Equal(describe(page.Changes), []string{"b.md file.added"}) {
		t.Errorf("the start of the feed answered %+v", page)
	}
}

func TestAChangeHasTheHashSizeAndModificationTimeOfTheFile(t *testing.T) {
	v := hosttest.New("garden")
	start := readFeed(t, v, host.FileChangeCursor{}).Next
	v.WriteFile("a.md", "A\n")
	stat, err := v.StatFile(t.Context(), "a.md")
	if err != nil {
		t.Fatal(err)
	}

	changes := readFeed(t, v, start).Changes

	if len(changes) != 1 {
		t.Fatalf("the feed holds %+v, want one change", changes)
	}
	if got := changes[0]; got.Hash != domain.HashContent([]byte("A\n")) || got.Size != int64(len("A\n")) || got.ModifiedAt != stat.ModTime.UnixNano() {
		t.Errorf("the change has hash %q, size %d, modified at %d, want those of the file: %q, %d, %d",
			got.Hash, got.Size, got.ModifiedAt, domain.HashContent([]byte("A\n")), len("A\n"), stat.ModTime.UnixNano())
	}
}

func TestAChangeAfterARewriteHasTheLatestContent(t *testing.T) {
	v := hosttest.New("garden")
	v.WriteFile("a.md", "A\n")
	start := readFeed(t, v, host.FileChangeCursor{}).Next
	v.WriteFile("a.md", "A, longer\n")
	stat, err := v.StatFile(t.Context(), "a.md")
	if err != nil {
		t.Fatal(err)
	}

	changes := readFeed(t, v, start).Changes

	if len(changes) != 1 || changes[0].Event != host.EventFileChanged {
		t.Fatalf("the feed holds %+v, want one file.changed", changes)
	}
	if got := changes[0]; got.Hash != domain.HashContent([]byte("A, longer\n")) || got.Size != int64(len("A, longer\n")) || got.ModifiedAt != stat.ModTime.UnixNano() {
		t.Errorf("the change has hash %q, size %d, modified at %d, want the latest content", got.Hash, got.Size, got.ModifiedAt)
	}
}

func TestAnEarlierChangeHasTheLatestContentOfItsFile(t *testing.T) {
	v := hosttest.New("garden")
	start := readFeed(t, v, host.FileChangeCursor{}).Next
	v.WriteFile("a.md", "A\n")
	v.WriteFile("a.md", "A, longer\n")

	changes := readFeed(t, v, start).Changes

	if len(changes) != 2 {
		t.Fatalf("the feed holds %+v, want two changes", changes)
	}
	for _, one := range changes {
		if one.Hash != domain.HashContent([]byte("A, longer\n")) || one.Size != int64(len("A, longer\n")) {
			t.Errorf("%s of %s has hash %q, size %d, want the latest content", one.Event, one.Path, one.Hash, one.Size)
		}
	}
}

func TestARemovalHasEmptyHashSizeAndModificationTime(t *testing.T) {
	v := hosttest.New("garden")
	v.WriteFile("a.md", "A\n")
	start := readFeed(t, v, host.FileChangeCursor{}).Next
	v.Remove("a.md")

	changes := readFeed(t, v, start).Changes

	if len(changes) != 1 || changes[0].Event != host.EventFileRemoved || changes[0].Hash != "" || changes[0].Size != 0 || changes[0].ModifiedAt != 0 {
		t.Errorf("the feed holds %+v, want one removal with empty values", changes)
	}
}
