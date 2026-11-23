package hosttest_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
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

	changes := readFeed(t, v, cursor).Changes
	if len(changes) != 2 || changes[0] != (host.FileChange{SourceID: id, Path: "lakes/a.md", Event: host.EventFileMoved, Seq: cursor.Seq + 1}) ||
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
