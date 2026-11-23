package receive_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
)

func TestAcceptInvitation_ReceivesAReaderTheOwnerAddedAgainIntoItsVaultWithANewInvitation(t *testing.T) {
	w := newWorld(t)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	first := w.shareWith(t, w.reader, "Anna")
	received := w.receive(t, w.reader, first)
	if err := w.owner.RemoveReader(t.Context(), owned, folder, first.ReaderFingerprint); err != nil {
		t.Fatal(err)
	}
	if err := w.reader.Sync(t.Context(), received.VaultID); !errors.Is(err, share.ErrEnded) {
		t.Fatalf("error = %v, want the share ended", err)
	}
	if _, err := w.reader.AcceptInvitation(t.Context(), first.Text); !errors.Is(err, share.ErrEnded) {
		t.Errorf("the old invitation after the removal: error = %v, want ErrEnded", err)
	}
	code, err := w.reader.GetReaderCode(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	again, err := w.owner.AddReader(t.Context(), owned, folder, code.Code, "Anna", false)
	if err != nil {
		t.Fatal(err)
	}
	w.owner.WriteNote(owned, "Food/Bread.md", "flour")
	w.syncOwner(t)

	rejoined, err := w.reader.AcceptInvitation(t.Context(), again.Text)

	if err != nil {
		t.Fatal(err)
	}
	if rejoined.VaultID != received.VaultID {
		t.Errorf("vault = %q, want the vault the share had, %q", rejoined.VaultID, received.VaultID)
	}
	wantNote(t, w.reader, received.VaultID, "Soup.md", "carrots")
	wantNote(t, w.reader, received.VaultID, "Bread.md", "flour")
	listed, err := w.reader.ListReceived(t.Context())
	if err != nil || len(listed) != 1 || listed[0].State != "ready" {
		t.Errorf("received = %+v, error = %v, want one share, ready", listed, err)
	}
}

func TestAcceptInvitation_RefusesTheNewInvitationWhileTheShareStillRuns(t *testing.T) {
	w := newWorld(t)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	invitation := w.shareWith(t, w.reader, "Anna")
	w.receive(t, w.reader, invitation)

	_, err := w.reader.AcceptInvitation(t.Context(), invitation.Text)

	if !errors.Is(err, share.ErrAlreadyReceived) {
		t.Errorf("error = %v, want ErrAlreadyReceived", err)
	}
}

func TestAcceptInvitation_TheReaderAddedAgainStillOpensANoteOfTheEpochBeforeItsRemoval(t *testing.T) {
	w := newWorld(t)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	first := w.shareWith(t, w.reader, "Anna")
	received := w.receive(t, w.reader, first)
	if err := w.owner.RemoveReader(t.Context(), owned, folder, first.ReaderFingerprint); err != nil {
		t.Fatal(err)
	}
	w.reader.Vaults.With(received.VaultID).Remove("Soup.md")
	code, err := w.reader.GetReaderCode(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	again, err := w.owner.AddReader(t.Context(), owned, folder, code.Code, "Anna", false)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := w.reader.AcceptInvitation(t.Context(), again.Text); err != nil {
		t.Fatal(err)
	}

	if got := w.reader.Link(t, received.VaultID).Received.Epoch; got != 1 {
		t.Errorf("epoch = %d, want 1: the removal started it", got)
	}
	wantNote(t, w.reader, received.VaultID, "Soup.md", "carrots")
}
