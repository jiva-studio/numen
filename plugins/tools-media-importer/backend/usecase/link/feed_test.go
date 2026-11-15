package link_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/link"
)

const linkNote = "---\ntype: link\nurl: " + articleAddress + "\n---\n"

func (w *world) addLinkNote(t *testing.T, path, content string) {
	t.Helper()
	w.vault.File(path, content)
	w.texts.File(path, []byte(content))
}

func (w *world) fetchMissing(t *testing.T) {
	t.Helper()
	if err := w.links.FetchMissingTexts(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func (w *world) countVersions(t *testing.T, path string) int {
	t.Helper()
	listing, err := w.texts.ListVersions(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	return len(listing.Versions)
}

func newAutomaticWorld(t *testing.T) *world {
	t.Helper()
	w := newWorld(t)
	w.isTextAutomatic.Store(true)
	return w
}

func TestALinkNoteAddedToTheVaultGetsItsTextOnce(t *testing.T) {
	w := newAutomaticWorld(t)
	w.fetchMissing(t)
	w.addLinkNote(t, "links/mill.md", linkNote)
	w.vault.File("notes/plain.md", "---\ntype: deck\nurl: "+videoAddress+"\n---\n")

	w.fetchMissing(t)
	w.fetchMissing(t)

	if got := w.sites.texts.Load(); got != 1 {
		t.Errorf("the text was fetched %d times, want once", got)
	}
	if got := w.countVersions(t, "links/mill.md"); got != 1 {
		t.Errorf("the link note has %d versions, want 1", got)
	}
}

func TestTheLinkNotesTheVaultHoldsGetTheirTextOnTheFirstRun(t *testing.T) {
	w := newAutomaticWorld(t)
	w.addLinkNote(t, "links/mill.md", linkNote)

	w.fetchMissing(t)

	if got := w.countVersions(t, "links/mill.md"); got != 1 {
		t.Errorf("the link note has %d versions, want 1", got)
	}
}

func TestAMovedLinkNoteIsNotFetchedAgain(t *testing.T) {
	w := newAutomaticWorld(t)
	w.fetchMissing(t)
	w.addLinkNote(t, "links/mill.md", linkNote)
	w.fetchMissing(t)

	if _, err := w.vault.MoveNotes(t.Context(), []string{"links/mill.md"}, "archive"); err != nil {
		t.Fatal(err)
	}
	w.fetchMissing(t)

	if got := w.sites.texts.Load(); got != 1 {
		t.Errorf("the text was fetched %d times, want once", got)
	}
}

func TestABatchReadTwiceFetchesNothingMore(t *testing.T) {
	w := newAutomaticWorld(t)
	w.fetchMissing(t)
	before, err := w.store.Read(t.Context(), link.CursorFile)
	if err != nil {
		t.Fatal(err)
	}
	w.addLinkNote(t, "links/mill.md", linkNote)
	w.fetchMissing(t)

	if err := w.store.Write(t.Context(), link.CursorFile, before); err != nil {
		t.Fatal(err)
	}
	w.fetchMissing(t)

	if got := w.sites.texts.Load(); got != 1 {
		t.Errorf("the text was fetched %d times, want once", got)
	}
	if got := w.countVersions(t, "links/mill.md"); got != 1 {
		t.Errorf("the link note has %d versions, want 1", got)
	}
}

func TestARestartReadsOnlyTheChangesSinceTheCursor(t *testing.T) {
	w := newAutomaticWorld(t)
	w.addLinkNote(t, "links/mill.md", linkNote)
	w.fetchMissing(t)
	made, err := w.links.Read(t.Context(), "links/mill.md")
	if err != nil {
		t.Fatal(err)
	}
	if err := w.texts.RemoveVersion(t.Context(), "links/mill.md", made.Key); err != nil {
		t.Fatal(err)
	}
	w.addLinkNote(t, "links/video.md", "---\ntype: link\nurl: "+videoAddress+"\n---\n")

	restarted := link.New(link.Host{PluginID: pluginID, Vaults: vaultsOf(w.vault), Store: foldersOf(w.vault), Texts: textsOf(w.texts), Queue: w.queue, Clock: newFakeClock(time.Unix(1_700_000_000, 0))},
		func() link.Fetcher { return w.sites },
		func() link.Settings { return link.Settings{IsTextDownloadAutomatic: true} })
	t.Cleanup(restarted.Close)
	if err := restarted.FetchMissingTexts(t.Context()); err != nil {
		t.Fatal(err)
	}

	if got := w.countVersions(t, "links/mill.md"); got != 0 {
		t.Errorf("the removed text of the earlier note was fetched again: %d versions", got)
	}
	if got := w.countVersions(t, "links/video.md"); got != 1 {
		t.Errorf("the note added since has %d versions, want 1", got)
	}
}

func TestAResetFetchesOnlyTheMissingTexts(t *testing.T) {
	w := newAutomaticWorld(t)
	w.addLinkNote(t, "links/mill.md", linkNote)
	w.fetchMissing(t)
	w.addLinkNote(t, "links/video.md", "---\ntype: link\nurl: "+videoAddress+"\n---\n")

	w.vault.ResetFeed()
	w.fetchMissing(t)

	if got := w.sites.texts.Load(); got != 2 {
		t.Errorf("the texts were fetched %d times, want twice", got)
	}
}

func TestAChangedAddressIsFetchedAgain(t *testing.T) {
	w := newAutomaticWorld(t)
	w.fetchMissing(t)
	w.addLinkNote(t, "links/mill.md", linkNote)
	w.fetchMissing(t)

	w.addLinkNote(t, "links/mill.md", "---\ntype: link\nurl: "+videoAddress+"\n---\n")
	w.fetchMissing(t)

	if got := w.sites.texts.Load(); got != 2 {
		t.Errorf("the texts were fetched %d times, want twice", got)
	}
}

func TestNothingIsFetchedWhileTheSettingIsOff(t *testing.T) {
	w := newWorld(t)
	w.addLinkNote(t, "links/mill.md", linkNote)

	w.fetchMissing(t)

	if got := w.sites.texts.Load(); got != 0 {
		t.Errorf("the text was fetched %d times with the setting off", got)
	}
	if _, err := w.store.Read(t.Context(), link.CursorFile); err == nil {
		t.Error("the cursor moved with the setting off")
	}
	w.isTextAutomatic.Store(true)
	w.fetchMissing(t)
	if got := w.countVersions(t, "links/mill.md"); got != 1 {
		t.Errorf("after the setting is on the link note has %d versions, want 1", got)
	}
}

func TestAStartWithReviewFetchesEveryMissingTextInTheBackground(t *testing.T) {
	w := newAutomaticWorld(t)
	w.addLinkNote(t, "links/mill.md", linkNote)
	w.fetchMissing(t)
	made, err := w.links.Read(t.Context(), "links/mill.md")
	if err != nil {
		t.Fatal(err)
	}
	if err := w.texts.RemoveVersion(t.Context(), "links/mill.md", made.Key); err != nil {
		t.Fatal(err)
	}

	w.links.StartMissingTextFetch(t.Context(), false)
	w.links.StartMissingTextFetch(t.Context(), true)
	task := "text:" + string(w.vault.Vault().ID) + ":links/mill.md"
	w.queue.WaitForEnd(t, task)
	w.queue.WaitForEnd(t, task)
	w.links.Close()

	if got := w.countVersions(t, "links/mill.md"); got != 1 {
		t.Errorf("after a review the link note has %d versions, want 1", got)
	}
}
