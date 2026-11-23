package extract_test

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/changes"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/internal/testsupport"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/usecase/extract"
)

func follow(t *testing.T, follower *extract.Follower) {
	t.Helper()
	if err := follower.Follow(t.Context(), "vault"); err != nil {
		t.Fatalf("Follow: %v", err)
	}
}

func readCursor(t *testing.T, f testsupport.FollowedVault) host.FileChangeCursor {
	t.Helper()
	cursor, err := changes.ReadCursor(t.Context(), changes.Source{Feed: f.Vault, Store: f.Vault.Store(), CursorFile: extract.CursorFile})
	if err != nil {
		t.Fatal(err)
	}
	return cursor
}

func TestANewPDFGetsItsTextLayerOnce(t *testing.T) {
	f := testsupport.NewFollowedVault(t, map[string][]byte{"notes/a.md": []byte("# A\n")})
	follower := f.NewFollower(t)
	follow(t, follower)
	f.Texts.TakeChanges()

	f.Add("books/tiny.pdf", testsupport.ReadFixture(t, "tiny.pdf"))
	follow(t, follower)
	follow(t, follower)

	if got := f.Texts.TakeChanges(); !slices.Equal(got, []string{"books/tiny.pdf"}) {
		t.Errorf("the versions of %v changed, want books/tiny.pdf once", got)
	}
	if text := testsupport.ReadLive(t, f.Texts, "books/tiny.pdf"); text.Version().Kind != extract.TextKind {
		t.Errorf("the live text is %+v", text.Version())
	}
}

func TestAMovedPDFIsNotReadAgain(t *testing.T) {
	f := testsupport.NewFollowedVault(t, map[string][]byte{"tiny.pdf": testsupport.ReadFixture(t, "tiny.pdf")})
	follower := f.NewFollower(t)
	follow(t, follower)
	f.Texts.TakeChanges()

	if moves, err := f.Vault.MoveNotes(t.Context(), []string{"tiny.pdf"}, "books"); err != nil || moves[0].Err != nil {
		t.Fatalf("move: %v %+v", err, moves)
	}
	follow(t, follower)

	if got := f.Texts.TakeChanges(); len(got) != 0 {
		t.Errorf("a move made the text layers of %v", got)
	}
}

func TestABatchAppliedTwiceMakesOneTextLayer(t *testing.T) {
	f := testsupport.NewFollowedVault(t, map[string][]byte{"notes/a.md": []byte("# A\n")})
	follower := f.NewFollower(t)
	follow(t, follower)
	before := readCursor(t, f)
	f.Add("tiny.pdf", testsupport.ReadFixture(t, "tiny.pdf"))
	follow(t, follower)
	f.Texts.TakeChanges()

	content, err := json.Marshal(before)
	if err != nil {
		t.Fatal(err)
	}
	f.Vault.Store().WriteFile(extract.CursorFile, content)
	follow(t, follower)

	if got := f.Texts.TakeChanges(); len(got) != 0 {
		t.Errorf("the batch read again made the text layers of %v", got)
	}
	if after := readCursor(t, f); after.Seq <= before.Seq {
		t.Errorf("the cursor is %+v after the batch, was %+v", after, before)
	}
}

func TestAResetMakesOnlyTheMissingTextLayers(t *testing.T) {
	tiny := testsupport.ReadFixture(t, "tiny.pdf")
	f := testsupport.NewFollowedVault(t, map[string][]byte{"tiny.pdf": tiny, "notes/a.md": []byte("# A\n")})
	follower := f.NewFollower(t)
	follow(t, follower)
	f.Texts.TakeChanges()
	f.Add("labels.pdf", testsupport.ReadFixture(t, "labels.pdf"))
	f.Vault.Remove("notes/a.md")
	f.Vault.TrimRemovals()
	f.Vault.ResetFeed()

	follow(t, follower)

	if got := f.Texts.TakeChanges(); !slices.Equal(got, []string{"labels.pdf"}) {
		t.Errorf("the reset made the text layers of %v, want labels.pdf alone", got)
	}
}

func TestAFollowerStartedAgainReadsOnlyTheChangesSinceItsCursor(t *testing.T) {
	f := testsupport.NewFollowedVault(t, map[string][]byte{"tiny.pdf": testsupport.ReadFixture(t, "tiny.pdf")})
	follow(t, f.NewFollower(t))
	f.Texts.TakeChanges()
	f.Add("labels.pdf", testsupport.ReadFixture(t, "labels.pdf"))
	f.Vault.Refuse("tiny.pdf", errors.New("tiny.pdf is read again"))

	follow(t, f.NewFollower(t))

	if got := f.Texts.TakeChanges(); !slices.Equal(got, []string{"labels.pdf"}) {
		t.Errorf("the follower started again made the text layers of %v, want labels.pdf alone", got)
	}
}
