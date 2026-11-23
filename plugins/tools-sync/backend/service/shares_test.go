package service_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/service"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
)

func TestSharing_AFolderReachesAReaderThroughTheMethodsOfTheView(t *testing.T) {
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	owner := synctest.NewFirstComputer(t, server, clock, "Test", synctest.NewWords(t))
	owner.Connect(t, "Test")
	owner.WriteNote("Test", "Food/Soup.md", "carrots")
	if err := owner.Sync(t.Context(), "Test"); err != nil {
		t.Fatal(err)
	}
	reader := synctest.NewReaderComputer(t, server, clock, "Reader")
	ownerSide, readerSide := service.NewBackend(owner.SyncDeps()), service.NewBackend(reader.SyncDeps())
	ctx := t.Context()

	code, err := readerSide.GetReaderCode(ctx, api.GetReaderCodeRequest{})
	if err != nil {
		t.Fatal(err)
	}
	read, err := ownerSide.ReadReaderCode(ctx, api.ReadReaderCodeRequest{Code: code.Code})
	if err != nil || read.Fingerprint != code.Fingerprint {
		t.Fatalf("fingerprint = %q (%v), want %q, the one the reader shows", read.Fingerprint, err, code.Fingerprint)
	}
	before, err := ownerSide.GetShare(ctx, api.GetShareRequest{VaultID: "Test", Folder: "Food"})
	if err != nil || before.State != api.ShareNotShared || len(before.Readers) != 0 {
		t.Fatalf("share before = %+v (%v), want a folder that is not shared", before, err)
	}
	added, err := ownerSide.AddReader(ctx, api.AddReaderRequest{VaultID: "Test", Folder: "Food", Code: code.Code, Label: "Anna"})
	if err != nil || added.Fingerprint != code.Fingerprint || added.Invitation == "" {
		t.Fatalf("added = %+v (%v), want the invitation for the reader", added, err)
	}

	preview, err := readerSide.ReadInvitation(ctx, api.ReadInvitationRequest{Text: added.Invitation})
	if err != nil || preview.ShareName != "Food" || !preview.IsForThisAccount || preview.AlreadyReceived {
		t.Fatalf("preview = %+v (%v), want the share Food for this account", preview, err)
	}
	if _, err := readerSide.AcceptInvitation(ctx, api.AcceptInvitationRequest{Text: added.Invitation}); err != nil {
		t.Fatal(err)
	}

	assertSharing(t, ownerSide, readerSide, code.Fingerprint)
}

func assertSharing(t *testing.T, ownerSide, readerSide *service.Backend, fingerprint string) {
	t.Helper()
	ctx := t.Context()
	received, err := readerSide.ListReceived(ctx, api.ListReceivedRequest{})
	if err != nil || len(received.Shares) != 1 || received.Shares[0].Name != "Food" || received.Shares[0].State != api.ReceivedReady ||
		received.Shares[0].OwnerFingerprint == "" || received.Shares[0].LastSyncedAt == "" {
		t.Errorf("received = %+v (%v), want Food, ready, with its owner and the time of the first pass", received, err)
	}
	vaults := listVaults(t, readerSide)
	isReceived := false
	for _, vault := range vaults {
		isReceived = isReceived || (vault.Name == "Food" && vault.State == api.StateReceived)
	}
	if !isReceived {
		t.Errorf("vaults = %+v, want the vault Food received", vaults)
	}
	shared, err := ownerSide.GetShare(ctx, api.GetShareRequest{VaultID: "Test", Folder: "Food"})
	if err != nil || shared.State != api.ShareActive || len(shared.Readers) != 1 || shared.Readers[0].Fingerprint != fingerprint ||
		shared.Readers[0].Label != "Anna" || shared.Readers[0].State != api.ReaderActive {
		t.Errorf("share = %+v (%v), want one active reader Anna with the fingerprint of the code", shared, err)
	}
	if _, err := ownerSide.RemoveReader(ctx, api.RemoveReaderRequest{VaultID: "Test", Folder: "Food", Fingerprint: fingerprint}); err != nil {
		t.Fatal(err)
	}
	if _, err := ownerSide.StopShare(ctx, api.StopShareRequest{VaultID: "Test", Folder: "Food"}); err != nil {
		t.Fatal(err)
	}
	stopped, err := ownerSide.GetShare(ctx, api.GetShareRequest{VaultID: "Test", Folder: "Food"})
	if err != nil || stopped.State != api.ShareNotShared {
		t.Errorf("share after it stopped = %+v (%v), want a folder that is not shared", stopped, err)
	}
}

func TestSharing_RefusesWhatTheOwnerMayNotDo(t *testing.T) {
	w := newWorld(t)
	w.c.Connect(t, "Test")
	backend := service.NewBackend(w.c.SyncDeps())
	code, err := backend.GetReaderCode(t.Context(), api.GetReaderCodeRequest{})
	if err != nil {
		t.Fatal(err)
	}

	_, ownErr := backend.AddReader(t.Context(), api.AddReaderRequest{VaultID: "Test", Folder: "Food", Code: code.Code, Label: "Me"})
	_, wholeErr := backend.AddReader(t.Context(), api.AddReaderRequest{VaultID: "Test", Folder: "", Code: code.Code, Label: "Me"})
	_, badErr := backend.ReadReaderCode(t.Context(), api.ReadReaderCodeRequest{Code: "numen-reader-1.AAAA"})

	for name, err := range map[string]error{"the own identity": ownErr, "the whole vault": wholeErr, "a code that is none": badErr} {
		if err == nil {
			t.Errorf("%s: the method did not refuse", name)
		}
	}
}
