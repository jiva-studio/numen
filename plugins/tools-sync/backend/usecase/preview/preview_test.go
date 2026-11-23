package preview_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/commit"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/preview"
)

const (
	source domain.VaultID = "Test"
	target domain.VaultID = "Copy"
)

type world struct {
	a, b   *synctest.Computer
	remote string
}

func newWorld(t *testing.T) *world {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	words := synctest.NewWords(t)
	a := synctest.NewFirstComputer(t, server, clock, string(source), words)
	a.Connect(t, source)
	b := synctest.NewSecondComputer(t, server, clock, string(target), words)
	return &world{a: a, b: b, remote: a.Link(t, source).RemoteVaultID}
}

func (w *world) syncedNote(t *testing.T, path, text string) {
	t.Helper()
	w.a.WriteNote(source, path, text)
	if err := w.a.Sync(t.Context(), source); err != nil {
		t.Fatal(err)
	}
}

func (w *world) preview(t *testing.T) preview.Preview {
	t.Helper()
	got, err := preview.NewPreviewConnect(w.b.PreviewDeps()).Execute(t.Context(), preview.Input{VaultID: target, RemoteID: w.remote})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestPreviewConnect_SortsEveryNoteOfBothSidesByHowTheyMeet(t *testing.T) {
	w := newWorld(t)
	w.syncedNote(t, "Shopping list.md", "milk")
	w.syncedNote(t, "Plan.md", "eggs")
	w.syncedNote(t, "Food/Soup.md", "leek")
	w.b.WriteNote(target, "Shopping list.md", "milk")
	w.b.WriteNote(target, "Plan.md", "tea")
	w.b.WriteNote(target, "Ideas.md", "walk")

	got := w.preview(t)

	want := preview.Preview{
		Same: []string{"Shopping list.md"}, LocalOnly: []string{"Ideas.md"}, RemoteOnly: []string{"Food/Soup.md"}, Differ: []string{"Plan.md"},
	}
	if !slices.Equal(got.Same, want.Same) || !slices.Equal(got.LocalOnly, want.LocalOnly) ||
		!slices.Equal(got.RemoteOnly, want.RemoteOnly) || !slices.Equal(got.Differ, want.Differ) {
		t.Errorf("preview = %+v, want %+v", got, want)
	}
}

func TestPreviewConnect_ListsEachGroupInPathOrder(t *testing.T) {
	w := newWorld(t)
	w.syncedNote(t, "Zebra.md", "one")
	w.syncedNote(t, "Apple.md", "two")
	w.syncedNote(t, "Mango.md", "three")

	got := w.preview(t)

	if !slices.Equal(got.RemoteOnly, []string{"Apple.md", "Mango.md", "Zebra.md"}) {
		t.Errorf("remote only = %v, want the paths in order", got.RemoteOnly)
	}
}

func TestPreviewConnect_FindsNoDifferenceInAFolderWithEqualNotesOnly(t *testing.T) {
	w := newWorld(t)
	w.syncedNote(t, "Shopping list.md", "milk")
	w.b.WriteNote(target, "Shopping list.md", "milk")

	got := w.preview(t)

	if len(got.Differ) != 0 || len(got.LocalOnly) != 0 || len(got.RemoteOnly) != 0 || !slices.Equal(got.Same, []string{"Shopping list.md"}) {
		t.Errorf("preview = %+v, want one equal note and no difference", got)
	}
}

func TestPreviewConnect_MatchesPathsThatDifferInUnicodeCompositionOnly(t *testing.T) {
	w := newWorld(t)
	w.syncedNote(t, "Café.md", "latte")
	w.b.WriteNote(target, "Café.md", "latte")

	got := w.preview(t)

	if len(got.Same) != 1 || len(got.LocalOnly) != 0 || len(got.RemoteOnly) != 0 {
		t.Errorf("preview = %+v, want the two spellings as one equal note", got)
	}
}

func TestPreviewConnect_LeavesAFileThatIsNotSyncedOutOfBothSides(t *testing.T) {
	w := newWorld(t)
	w.b.WriteNote(target, "draft.tmp", "bytes")

	got := w.preview(t)

	if len(got.LocalOnly) != 0 {
		t.Errorf("local only = %v, want no file that is not synced", got.LocalOnly)
	}
}

func TestPreviewConnect_WritesNoLinkNoRecordAndNoFile(t *testing.T) {
	w := newWorld(t)
	w.syncedNote(t, "Shopping list.md", "milk")
	w.b.WriteNote(target, "Plan.md", "tea")

	w.preview(t)

	if _, err := w.b.State.ReadLink(t.Context(), target); !errors.Is(err, vaultlink.ErrNotConnected) {
		t.Errorf("link error = %v, want %v", err, vaultlink.ErrNotConnected)
	}
	stored, err := w.b.Host.Store(target).List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 0 {
		t.Errorf("store folder holds %v, want nothing written", stored)
	}
	if _, isThere := w.b.ReadNote(target, "Shopping list.md"); isThere {
		t.Error("the note of the synced vault was written to the folder")
	}
	if text, _ := w.b.ReadNote(target, "Plan.md"); text != "tea" {
		t.Errorf("note Plan = %q, want tea as it was", text)
	}
}

func TestPreviewConnect_RefusesAnIdThatNoSyncedVaultHas(t *testing.T) {
	w := newWorld(t)
	w.remote = "00112233445566778899aabbccddeeff"

	_, err := preview.NewPreviewConnect(w.b.PreviewDeps()).Execute(t.Context(), preview.Input{VaultID: target, RemoteID: w.remote})

	if err == nil {
		t.Error("error = nil, want a refusal for an unknown synced vault")
	}
}

func TestPreviewConnect_IgnoresANoteThatTheServerHoldsAsRemoved(t *testing.T) {
	w := newWorld(t)
	vault := transfer.Vault{ID: w.a.GetRemoteID(t, source), Key: w.a.GetVaultKey(t, source)}
	outcomes, err := commit.NewCommitRevisions(w.a.CommitDeps()).Execute(t.Context(), transfer.CommitInput{
		Vault: vault, Files: []transfer.File{{Path: "Shopping list.md", Content: []byte("milk")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	w.a.CommitRemoval(t, vault, "Shopping list.md", outcomes[0])
	w.b.WriteNote(target, "Shopping list.md", "milk")

	got := w.preview(t)

	if len(got.RemoteOnly) != 0 || len(got.Differ) != 0 || len(got.Same) != 0 || !slices.Equal(got.LocalOnly, []string{"Shopping list.md"}) {
		t.Errorf("preview = %+v, want the note of the folder alone, as the removal is no head", got)
	}
}
