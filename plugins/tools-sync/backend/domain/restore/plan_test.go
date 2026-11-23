package restore_test

import (
	"bytes"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/restore"
)

func TestPlanRecovery_AdoptsAHeadWhoseContentIsTheLocalNoteInsteadOfSendingItAgain(t *testing.T) {
	t.Parallel()
	bread := note.Hash([]byte("bread"))
	records := []restore.Known{record(1, "a.md", bread, 8)}
	same := head(1, "a.md", 9)
	same.Hash = bread

	got := restore.PlanRecovery(records, heads(same), map[string]string{"a.md": bread})

	action := wantAction(t, got, restore.Adopt, 1)
	if !bytes.Equal(action.ParentRev, bytes.Repeat([]byte{9}, 32)) {
		t.Errorf("head to adopt = %x, want the head the server holds", action.ParentRev)
	}
}

func TestPlanRecovery_LeavesAHeadAnotherComputerCommittedOnTopOfTheRecordedOneToThePull(t *testing.T) {
	t.Parallel()
	milk := note.Hash([]byte("milk"))
	records := []restore.Known{record(1, "a.md", milk, 7)}
	newer := head(1, "a.md", 9)
	newer.ParentRev = bytes.Repeat([]byte{7}, 32)
	newer.Hash = note.Hash([]byte("tea"))

	got := restore.PlanRecovery(records, heads(newer), map[string]string{"a.md": milk})

	wantAction(t, got, restore.Keep, 1)
}

func TestPlanRecovery_SendsOverAHeadOnTopOfTheRecordedOneWhenItsBodyIsGone(t *testing.T) {
	t.Parallel()
	milk := note.Hash([]byte("milk"))
	records := []restore.Known{record(1, "a.md", milk, 7)}
	lost := head(1, "a.md", 9)
	lost.ParentRev, lost.IsBlobMissing = bytes.Repeat([]byte{7}, 32), true

	got := restore.PlanRecovery(records, heads(lost), map[string]string{"a.md": milk})

	wantAction(t, got, restore.Send, 1)
}
