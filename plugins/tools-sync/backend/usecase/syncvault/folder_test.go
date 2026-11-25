package syncvault_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

func newFolderJoinPair(t *testing.T) *pair {
	t.Helper()
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Shopping list.md", "milk")
	sync(t, p.a, source)
	return p
}

func TestSyncVault_AdoptsANoteOfAFolderThatHasTheTextOfTheSyncedNote(t *testing.T) {
	p := newFolderJoinPair(t)
	p.b.WriteNote(target, "Shopping list.md", "milk")
	revisions := p.a.ReadChanges(t, p.a.GetRemoteID(t, source))

	sync(t, p.b, target)

	wantNote(t, p.b, target, "Shopping list.md", "milk")
	row, ok := p.b.GetFileRow(t, target, "Shopping list.md")
	if !ok || row.BaseHash != note.Hash([]byte("milk")) || len(row.HeadRev) == 0 {
		t.Errorf("row = %+v (there: %v), want the hash of milk as base and the head recorded", row, ok)
	}
	if now := p.a.ReadChanges(t, p.a.GetRemoteID(t, source)); len(now) != len(revisions) || now[0].GetSeq() != revisions[0].GetSeq() {
		t.Errorf("server holds %v, want the one revision it held before: equal notes are not sent again", now)
	}
}

func TestSyncVault_SendsANoteOnlyTheFolderHasAndReceivesANoteOnlyTheSyncedVaultHas(t *testing.T) {
	p := newFolderJoinPair(t)
	p.b.WriteNote(target, "Plan.md", "tea")

	sync(t, p.b, target)
	sync(t, p.a, source)

	wantNote(t, p.b, target, "Shopping list.md", "milk")
	wantNote(t, p.a, source, "Plan.md", "tea")
}

// The next two tests wait for the conflict resolver of #2570: a note with another text on each side gets the text
// of the synced vault and the text of the folder in a conflict note named with the day.
func TestSyncVault_KeepsBothTextsOfANoteThatDiffersWhenAFolderJoins(t *testing.T) {
	p := newFolderJoinPair(t)
	p.b.WriteNote(target, "Shopping list.md", "bread")

	sync(t, p.b, target)

	wantNote(t, p.b, target, "Shopping list.md", "milk")
	wantNote(t, p.b, target, "Shopping list (conflict "+synctest.GetStart().Format("2006-01-02")+").md", "bread")
}

func TestSyncVault_SendsTheConflictNoteOfAJoinedFolderToTheOtherComputer(t *testing.T) {
	p := newFolderJoinPair(t)
	p.b.WriteNote(target, "Shopping list.md", "bread")

	sync(t, p.b, target)
	sync(t, p.a, source)

	wantNote(t, p.a, source, "Shopping list.md", "milk")
	wantNote(t, p.a, source, "Shopping list (conflict "+synctest.GetStart().Format("2006-01-02")+").md", "bread")
}
