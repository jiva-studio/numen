package service_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/service"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
)

func TestLeaveShare_ForgetsTheShareOnThisDeviceAndTheOwnerStillListsTheReader(t *testing.T) {
	w := newWorld(t)
	owner, reader := w.c, synctest.NewReaderComputer(t, w.server, w.clock, "Reader")
	owner.Connect(t, "Test")
	received := receiveFood(t, owner, reader)
	readerSide := service.NewBackend(reader.SyncDeps())

	_, err := readerSide.LeaveShare(t.Context(), api.LeaveShareRequest{VaultID: string(received)})

	if err != nil {
		t.Fatal(err)
	}
	if listed, err := readerSide.ListReceived(t.Context(), api.ListReceivedRequest{}); err != nil || len(listed.Shares) != 0 {
		t.Errorf("received = %+v, %v, want none", listed, err)
	}
	if vaults := listVaults(t, readerSide); len(vaults) != 1 || vaults[0].ID != "Reader" {
		t.Errorf("vaults = %+v, want the own vault of the reader only: the folder is an ordinary folder now", vaults)
	}
	state, err := owner.GetShare(t.Context(), "Test", "Food")
	if err != nil || len(state.Readers) != 1 || state.Readers[0].State != "active" {
		t.Errorf("share = %+v, %v, want the reader listed until the owner removes it", state, err)
	}
}

func TestLeaveShare_RefusesAVaultThatIsNoSharedFolder(t *testing.T) {
	w := newWorld(t)
	w.c.Connect(t, "Test")
	backend := service.NewBackend(w.c.SyncDeps())

	_, err := backend.LeaveShare(t.Context(), api.LeaveShareRequest{VaultID: "Test"})

	if !errors.Is(err, share.ErrNotReceived) {
		t.Errorf("error = %v, want the folder not shared with you", err)
	}
	if vaults := listVaults(t, backend); len(vaults) != 1 || vaults[0].State != api.StateConnected {
		t.Errorf("vaults = %+v, want the vault untouched", vaults)
	}
}

func receiveFood(t *testing.T, owner, reader *synctest.Computer) domain.VaultID {
	t.Helper()
	owner.WriteNote("Test", "Food/Soup.md", "carrots")
	if err := owner.Sync(t.Context(), "Test"); err != nil {
		t.Fatal(err)
	}
	if err := owner.CreateShare(t.Context(), "Test", "Food"); err != nil {
		t.Fatal(err)
	}
	code, err := reader.GetReaderCode(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	invitation, err := owner.AddReader(t.Context(), "Test", "Food", code.Code, "Anna laptop", false)
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := reader.AcceptInvitation(t.Context(), invitation.Text)
	if err != nil {
		t.Fatal(err)
	}
	return accepted.VaultID
}
