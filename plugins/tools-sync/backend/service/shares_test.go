package service_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/service"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
)

func TestSharing_AFolderReachesAReaderByALinkThroughTheMethodsOfTheView(t *testing.T) {
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	owner := synctest.NewFirstComputer(t, server, clock, "Test", synctest.NewWords(t))
	owner.Connect(t, "Test")
	owner.WriteNote("Test", "Food/Soup.md", "carrots")
	if err := owner.Sync(t.Context(), "Test"); err != nil {
		t.Fatal(err)
	}
	reader := synctest.NewReaderComputer(t, server, clock, "Reader")
	ownerSide := service.NewBackend(owner.SyncDeps()).WithInvites(owner.NewInvites(t))
	readerSide := service.NewBackend(reader.SyncDeps()).WithInvites(reader.NewInvites(t))
	ctx := t.Context()

	started, err := ownerSide.StartShareInvitation(ctx, api.StartShareInvitationRequest{VaultID: "Test", Folder: "Food"})
	if err != nil || started.Link == "" || started.ExpiresAt == "" {
		t.Fatalf("started = %+v, %v, want a link with its expiry", started, err)
	}
	preview, err := readerSide.PreviewShareInvitation(ctx, api.PreviewShareInvitationRequest{Link: started.Link})
	if err != nil || preview.Folder != "Food" || !preview.IsReadOnly || preview.OwnerFingerprint == "" {
		t.Fatalf("preview = %+v, %v, want the folder, the owner and read-only", preview, err)
	}
	sent, err := readerSide.AcceptShareInvitation(ctx, api.AcceptShareInvitationRequest{Link: started.Link, DeviceName: "Anna laptop"})
	if err != nil {
		t.Fatal(err)
	}
	var status api.GetShareInvitationStatusResponse
	synctest.Eventually(t, "the code on the reader", func() bool {
		status, err = readerSide.GetShareInvitationStatus(ctx, api.GetShareInvitationStatusRequest{RequestID: sent.RequestID})
		return err == nil && status.Phase == api.ShareRequestWaiting
	})
	var requests api.ListShareRequestsResponse
	synctest.Eventually(t, "the request on the owner", func() bool {
		requests, err = ownerSide.ListShareRequests(ctx, api.ListShareRequestsRequest{})
		return err == nil && len(requests.Requests) == 1
	})
	if requests.Requests[0].Code != status.Code || requests.Requests[0].DeviceName != "Anna laptop" || requests.Requests[0].Folder != "Food" {
		t.Fatalf("request = %+v, status = %+v, want one code on both sides", requests.Requests[0], status)
	}

	if _, err := ownerSide.ApproveShareRequest(ctx, api.ApproveShareRequestRequest{RequestID: requests.Requests[0].RequestID}); err != nil {
		t.Fatal(err)
	}

	synctest.Eventually(t, "the folder on the reader", func() bool {
		status, err = readerSide.GetShareInvitationStatus(ctx, api.GetShareInvitationStatusRequest{RequestID: sent.RequestID})
		return err == nil && status.Phase == api.ShareRequestReceived
	})
	received, err := readerSide.ListReceived(ctx, api.ListReceivedRequest{})
	if err != nil || len(received.Shares) != 1 || received.Shares[0].Name != "Food" || received.Shares[0].State != api.ReceivedReady {
		t.Fatalf("received = %+v, %v, want Food ready", received, err)
	}
	shared, err := ownerSide.GetShare(ctx, api.GetShareRequest{VaultID: "Test", Folder: "Food"})
	if err != nil || shared.State != api.ShareActive || len(shared.Readers) != 1 || shared.Readers[0].Label != "Anna laptop" ||
		shared.Readers[0].State != api.ReaderActive {
		t.Fatalf("share = %+v, %v, want one active reader named by its device", shared, err)
	}
	if _, err := ownerSide.RemoveReader(ctx, api.RemoveReaderRequest{VaultID: "Test", Folder: "Food", Fingerprint: shared.Readers[0].Fingerprint}); err != nil {
		t.Fatal(err)
	}
	if _, err := ownerSide.StopShare(ctx, api.StopShareRequest{VaultID: "Test", Folder: "Food"}); err != nil {
		t.Fatal(err)
	}
}

func TestSharing_RefusesWhatTheOwnerMayNotDo(t *testing.T) {
	w := newWorld(t)
	w.c.Connect(t, "Test")
	backend := service.NewBackend(w.c.SyncDeps()).WithInvites(w.c.NewInvites(t))

	_, wholeErr := backend.StartShareInvitation(t.Context(), api.StartShareInvitationRequest{VaultID: "Test", Folder: ""})
	_, badErr := backend.PreviewShareInvitation(t.Context(), api.PreviewShareInvitationRequest{Link: "numen-share:AAAA"})
	_, unknownErr := backend.ApproveShareRequest(t.Context(), api.ApproveShareRequestRequest{RequestID: "00"})

	for name, err := range map[string]error{"the whole vault": wholeErr, "a link that is none": badErr, "a request that is none": unknownErr} {
		if err == nil {
			t.Errorf("%s: the method did not refuse", name)
		}
	}
}
