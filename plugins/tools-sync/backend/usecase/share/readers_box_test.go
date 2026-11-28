package share_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
)

func TestAcceptInvitation_AReaderRestoredFromTheWordsOpensTheSealOfItsEarlierBox(t *testing.T) {
	w := newWorld(t)
	words := synctest.NewWords(t)
	reader := synctest.NewFirstComputer(t, w.server, w.clock, "Reader", words)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	w.shareFolder(t)
	w.syncOwner(t)
	invitation := w.addReader(t, reader, "Anna")
	restored := synctest.NewRestoredComputer(t, w.server, w.clock, "Restored", words)

	received, err := restored.AcceptInvitation(t.Context(), invitation.Text)

	if err != nil {
		t.Fatalf("the restored device could not take the share: %v", err)
	}
	if got, ok := restored.ReadNote(received.VaultID, "Soup.md"); !ok || got != "carrots" {
		t.Errorf("the note on the restored device = %q (%v), want carrots", got, ok)
	}
}
