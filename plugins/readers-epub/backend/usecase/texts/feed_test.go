package texts

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/changes"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/internal/testsupport"
)

type followedVault struct {
	vault  *hosttest.Vault
	texts  *hosttest.Texts
	writer *Writer
}

func newFollowedVault(t *testing.T, files map[string][]byte) followedVault {
	t.Helper()
	vault := hosttest.New("vault")
	texts := hosttest.NewTexts("vault", pluginID)
	texts.SetStore(pluginID, vault.Store())
	f := followedVault{vault: vault, texts: texts, writer: NewWriter(filesOf(vault), textsOf(texts), storeOf(vault), hosttest.NewQueue(), noReport)}
	t.Cleanup(f.writer.Close)
	for name, raw := range files {
		f.add(name, raw)
	}
	return f
}

func (f followedVault) add(name string, raw []byte) {
	f.texts.WriteFile(name, raw)
	f.vault.WriteFile(name, string(raw))
}

func (f followedVault) newFollower(t *testing.T) *Follower {
	t.Helper()
	follower := NewFollower(f.writer, func(id domain.VaultID) Feed {
		return Feed{Changes: f.vault.With(id), Store: f.vault.Store().With(id)}
	}, func(what string, err error) { t.Errorf("%s: %v", what, err) })
	t.Cleanup(follower.Close)
	return follower
}

func follow(t *testing.T, follower *Follower) {
	t.Helper()
	if err := follower.Follow(t.Context(), "vault"); err != nil {
		t.Fatalf("Follow: %v", err)
	}
}

func TestANewBookGetsItsTextOnce(t *testing.T) {
	f := newFollowedVault(t, map[string][]byte{"notes/a.md": []byte("# A\n")})
	follower := f.newFollower(t)
	follow(t, follower)
	f.texts.TakeChanges()

	f.add("books/pictured.epub", testsupport.CreateImageEpub(t))
	follow(t, follower)
	follow(t, follower)

	if got := f.texts.TakeChanges(); !slices.Equal(got, []string{"books/pictured.epub"}) {
		t.Errorf("the versions of %v changed, want books/pictured.epub once", got)
	}
	if text := readLiveText(t, f.texts, "books/pictured.epub"); text.Version().Kind != Kind {
		t.Errorf("the live text is %+v", text.Version())
	}
}

func TestAMovedBookIsNotReadAgain(t *testing.T) {
	f := newFollowedVault(t, map[string][]byte{"pictured.epub": testsupport.CreateImageEpub(t)})
	follower := f.newFollower(t)
	follow(t, follower)
	f.texts.TakeChanges()

	if moves, err := f.vault.MoveNotes(t.Context(), []string{"pictured.epub"}, "books"); err != nil || moves[0].Err != nil {
		t.Fatalf("move: %v %+v", err, moves)
	}
	follow(t, follower)

	if got := f.texts.TakeChanges(); len(got) != 0 {
		t.Errorf("a move made the texts of %v", got)
	}
}

func TestABatchOfBooksAppliedTwiceMakesOneText(t *testing.T) {
	f := newFollowedVault(t, map[string][]byte{"notes/a.md": []byte("# A\n")})
	follower := f.newFollower(t)
	follow(t, follower)
	from := changes.Source{Feed: f.vault, Store: f.vault.Store(), CursorFile: CursorFile}
	before, err := changes.ReadCursor(t.Context(), from)
	if err != nil {
		t.Fatal(err)
	}
	f.add("pictured.epub", testsupport.CreateImageEpub(t))
	follow(t, follower)
	f.texts.TakeChanges()

	content, err := json.Marshal(before)
	if err != nil {
		t.Fatal(err)
	}
	f.vault.Store().WriteFile(CursorFile, content)
	follow(t, follower)

	if got := f.texts.TakeChanges(); len(got) != 0 {
		t.Errorf("the batch read again made the texts of %v", got)
	}
}

func TestAResetMakesOnlyTheMissingTextsOfBooks(t *testing.T) {
	f := newFollowedVault(t, map[string][]byte{"pictured.epub": testsupport.CreateImageEpub(t), "notes/a.md": []byte("# A\n")})
	follower := f.newFollower(t)
	follow(t, follower)
	f.texts.TakeChanges()
	f.add("guided.epub", testsupport.CreateGuidedEpub(t))
	f.vault.Remove("notes/a.md")
	f.vault.TrimRemovals()
	f.vault.ResetFeed()

	follow(t, follower)

	if got := f.texts.TakeChanges(); !slices.Equal(got, []string{"guided.epub"}) {
		t.Errorf("the reset made the texts of %v, want guided.epub alone", got)
	}
}

func TestABookFollowerStartedAgainReadsOnlyTheChangesSinceItsCursor(t *testing.T) {
	f := newFollowedVault(t, map[string][]byte{"pictured.epub": testsupport.CreateImageEpub(t)})
	follow(t, f.newFollower(t))
	f.texts.TakeChanges()
	f.add("guided.epub", testsupport.CreateGuidedEpub(t))
	f.vault.Refuse("pictured.epub", errors.New("pictured.epub is read again"))

	follow(t, f.newFollower(t))

	if got := f.texts.TakeChanges(); !slices.Equal(got, []string{"guided.epub"}) {
		t.Errorf("the follower started again made the texts of %v, want guided.epub alone", got)
	}
}
