package pull_test

import (
	"bytes"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/commit"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/deletions"
)

const trash = ".trash/"

// away is a pair whose second computer pulled two notes, then the first computer removed "Shopping list" and
// the server forgot the removal.
type away struct {
	*pair
	shopping transfer.Outcome
	plan     transfer.Outcome
}

func newAway(t *testing.T) *away {
	t.Helper()
	p := newPair(t)
	w := &away{pair: p, shopping: p.push(t, "Shopping list.md", "milk", nil), plan: p.push(t, "Plan.md", "tea", nil)}
	p.pull(t, trustAll)
	return w
}

func (w *away) forgetRemoval(t *testing.T, removed string, outcome transfer.Outcome) {
	t.Helper()
	w.a.CommitRemoval(t, w.vault, removed, outcome)
	w.server.ForgetRemovals()
}

func (w *away) sync(t *testing.T) {
	t.Helper()
	if err := w.b.Sync(t.Context(), target); err != nil {
		t.Fatal(err)
	}
}

func (w *away) countLive(t *testing.T) int {
	t.Helper()
	live := 0
	for _, stored := range w.a.ReadChanges(t, w.vault.ID) {
		if !stored.GetRevision().GetDeleted() {
			live++
		}
	}
	return live
}

func TestPullChanges_RemovesAnUnchangedNoteTheServerNoLongerHoldsToTheTrashOnAReset(t *testing.T) {
	w := newAway(t)
	w.forgetRemoval(t, "Shopping list.md", w.shopping)

	w.pull(t, trustAll)

	if _, isThere := w.b.ReadNote(target, "Shopping list.md"); isThere {
		t.Error("the removed note is still in the folder")
	}
	if text, isThere := w.b.ReadNote(target, trash+"Shopping list.md"); !isThere || text != "milk" {
		t.Errorf("trash note = %q (there: %v), want milk in the trash", text, isThere)
	}
	if _, isThere := w.b.GetFileRow(t, target, "Shopping list.md"); isThere {
		t.Error("the record of the removed note is kept")
	}
}

func TestPullChanges_KeepsANoteTheServerStillHoldsOnAReset(t *testing.T) {
	w := newAway(t)
	w.forgetRemoval(t, "Shopping list.md", w.shopping)
	before, _ := w.b.GetFileRow(t, target, "Plan.md")

	result := w.pull(t, trustAll)

	if text, _ := w.b.ReadNote(target, "Plan.md"); text != "tea" {
		t.Errorf("note Plan = %q, want tea", text)
	}
	after, isThere := w.b.GetFileRow(t, target, "Plan.md")
	if !isThere || !bytes.Equal(after.FileID, before.FileID) || after.BaseHash != before.BaseHash {
		t.Errorf("row = %+v (there: %v), want the row it had, %+v", after, isThere, before)
	}
	if len(result.Conflicts) != 0 {
		t.Errorf("conflicts = %v, want none", result.Conflicts)
	}
}

func TestPullChanges_QueuesAChangedNoteTheServerNoLongerHoldsAsANewFileOnAReset(t *testing.T) {
	w := newAway(t)
	w.b.WriteNote(target, "Shopping list.md", "bread")
	w.forgetRemoval(t, "Shopping list.md", w.shopping)

	w.pull(t, trustAll)

	if text, _ := w.b.ReadNote(target, "Shopping list.md"); text != "bread" {
		t.Errorf("note = %q, want bread kept", text)
	}
	row, isThere := w.b.GetFileRow(t, target, "Shopping list.md")
	if isThere && (len(row.FileID) != 0 && string(row.FileID) != "null" || row.BaseHash != "") {
		t.Errorf("row = %+v, want no file id and no base, so the pass sends it as a new file", row)
	}
}

func TestSyncPass_SendsAChangedNoteTheServerForgotAsANewNoteOnAReset(t *testing.T) {
	w := newAway(t)
	w.b.WriteNote(target, "Shopping list.md", "bread")
	w.forgetRemoval(t, "Shopping list.md", w.shopping)

	w.sync(t)

	if err := w.a.Sync(t.Context(), source); err != nil {
		t.Fatal(err)
	}
	if text, _ := w.a.ReadNote(source, "Shopping list.md"); text != "bread" {
		t.Errorf("note on the first computer = %q, want bread", text)
	}
	if w.countLive(t) != 2 {
		t.Errorf("server holds %d live notes, want 2", w.countLive(t))
	}
}

func TestSyncPass_SendsANewNoteOfTheAwayComputerAfterAReset(t *testing.T) {
	w := newAway(t)
	w.b.WriteNote(target, "Ideas.md", "eggs")
	w.forgetRemoval(t, "Shopping list.md", w.shopping)

	w.sync(t)

	if err := w.a.Sync(t.Context(), source); err != nil {
		t.Fatal(err)
	}
	if text, _ := w.a.ReadNote(source, "Ideas.md"); text != "eggs" {
		t.Errorf("note on the first computer = %q, want eggs", text)
	}
	if text, isThere := w.b.ReadNote(target, "Ideas.md"); !isThere || text != "eggs" {
		t.Errorf("note on the away computer = %q (there: %v), want eggs kept", text, isThere)
	}
}

func TestSyncPass_DoesNotSendARemovedNoteBackOnTheNextPassAfterAReset(t *testing.T) {
	w := newAway(t)
	w.forgetRemoval(t, "Shopping list.md", w.shopping)

	w.sync(t)
	w.sync(t)

	if live := w.countLive(t); live != 1 {
		t.Errorf("server holds %d live notes, want 1 (the removed note did not come back)", live)
	}
}

func TestPullChanges_DropsTheRecordOfANoteThatIsNoLongerInTheFolderOnAReset(t *testing.T) {
	w := newAway(t)
	w.b.Vaults.With(target).Remove("Shopping list.md")
	w.forgetRemoval(t, "Shopping list.md", w.shopping)

	w.pull(t, trustAll)

	if _, isThere := w.b.GetFileRow(t, target, "Shopping list.md"); isThere {
		t.Error("the record of a note that is gone from both sides is kept")
	}
	if _, isThere := w.b.ReadNote(target, trash+"Shopping list.md"); isThere {
		t.Error("a note that was not there was moved to the trash")
	}
}

func TestPullChanges_RemovesNothingWhenAResetListsNoNote(t *testing.T) {
	w := newAway(t)
	w.a.CommitRemoval(t, w.vault, "Shopping list.md", w.shopping)
	w.a.CommitRemoval(t, w.vault, "Plan.md", w.plan)
	w.server.ForgetRemovals()

	w.pull(t, trustAll)

	for path, text := range map[string]string{"Shopping list.md": "milk", "Plan.md": "tea"} {
		if got, isThere := w.b.ReadNote(target, path); !isThere || got != text {
			t.Errorf("note %s = %q (there: %v), want %q kept: an empty list is no proof of a removal", path, got, isThere, text)
		}
	}
}

func TestSyncPass_SendsEveryNoteBackWhenAResetListsNoNote(t *testing.T) {
	w := newAway(t)
	w.a.CommitRemoval(t, w.vault, "Shopping list.md", w.shopping)
	w.a.CommitRemoval(t, w.vault, "Plan.md", w.plan)
	w.server.ForgetRemovals()

	w.sync(t)

	if live := w.countLive(t); live != 2 {
		t.Errorf("server holds %d live notes, want both notes of the away computer sent again", live)
	}
}

func TestPullChanges_NeverRemovesANoteThatHasNoRecordOnAReset(t *testing.T) {
	w := newAway(t)
	w.b.WriteNote(target, "Ideas.md", "eggs")
	w.forgetRemoval(t, "Shopping list.md", w.shopping)

	w.pull(t, trustAll)

	if text, isThere := w.b.ReadNote(target, "Ideas.md"); !isThere || text != "eggs" {
		t.Errorf("note Ideas = %q (there: %v), want eggs kept", text, isThere)
	}
}

func TestPullChanges_KeepsANoteWhoseHeadDoesNotVerifyOnAReset(t *testing.T) {
	w := newAway(t)
	old := commitAsAway(t, w, "Old.md", "oldtext")
	commitAsAway(t, w, "Kept.md", "keptext")
	w.pull(t, trustAll)
	w.forgetRemoval(t, "Old.md", old)
	awayKey := w.b.GetDeviceSignKey(t)

	result := w.pull(t, func(key []byte) bool { return bytes.Equal(key, awayKey) })

	if result.Unverified != 2 {
		t.Errorf("unverified = %d, want 2 for the two heads of the first computer", result.Unverified)
	}
	for _, path := range []string{"Shopping list.md", "Plan.md"} {
		if _, isThere := w.b.ReadNote(target, path); !isThere {
			t.Errorf("note %s was removed although its head did not verify", path)
		}
	}
	if _, isThere := w.b.ReadNote(target, trash+"Old.md"); !isThere {
		t.Error("the removal of Old, which has a verified list to stand on, was not applied")
	}
}

func commitAsAway(t *testing.T, w *away, path, text string) transfer.Outcome {
	t.Helper()
	outcomes, err := commit.NewCommitRevisions(w.b.CommitDeps()).Execute(t.Context(), transfer.CommitInput{
		Vault: w.vault, Files: []transfer.File{{Path: path, Content: []byte(text)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return outcomes[0]
}

func TestPullChanges_PausesWhenAResetWouldRemoveMostOfTheNotesAndKeepsThemUntilTheAnswer(t *testing.T) {
	p := newPair(t)
	paths := []string{"A.md", "B.md", "C.md", "D.md", "E.md", "F.md"}
	outcomes := map[string]transfer.Outcome{}
	for _, path := range paths {
		outcomes[path] = p.push(t, path, "text of "+path, nil)
	}
	p.pull(t, trustAll)
	before := p.b.Link(t, target)
	for _, path := range paths[:5] {
		p.a.CommitRemoval(t, p.vault, path, outcomes[path])
	}
	p.server.ForgetRemovals()

	p.pull(t, trustAll)

	for _, path := range paths {
		if _, isThere := p.b.ReadNote(target, path); !isThere {
			t.Errorf("note %s was removed although five of six removals wait for an answer", path)
		}
	}
	if !p.b.HasItemKey(deletions.HealthKey) {
		t.Error("no paused removals were reported")
	}
	if after := p.b.Link(t, target); after.ServerSeq != before.ServerSeq {
		t.Errorf("position = %d, want %d kept, so the next pass reads the vault again", after.ServerSeq, before.ServerSeq)
	}
}
