package shareinvite_test

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	domainshare "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/shareinvite"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

const (
	owned  domain.VaultID = "Test"
	folder string         = "Food"
)

type world struct {
	clock  *synctest.Clock
	server *synctest.Server
	owner  *synctest.Computer
	sharer shareinvite.Deps
}

func newWorld(t *testing.T) *world {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	owner := synctest.NewFirstComputer(t, server, clock, string(owned), synctest.NewWords(t))
	owner.Connect(t, owned)
	owner.WriteNote(owned, "Food/Soup.md", "carrots")
	if err := owner.Sync(t.Context(), owned); err != nil {
		t.Fatal(err)
	}
	return &world{clock: clock, server: server, owner: owner, sharer: owner.StartInviteCourier(t)}
}

func (w *world) newReader(t *testing.T, vault string) (*synctest.Computer, shareinvite.Deps) {
	t.Helper()
	reader := synctest.NewReaderComputer(t, w.server, w.clock, vault)
	return reader, reader.StartInviteCourier(t)
}

func (w *world) start(t *testing.T) shareinvite.Started {
	t.Helper()
	started, err := shareinvite.NewStart(w.sharer).Execute(t.Context(), owned, folder)
	if err != nil {
		t.Fatal(err)
	}
	return started
}

func join(t *testing.T, deps shareinvite.Deps, link, device string) string {
	t.Helper()
	joined, err := shareinvite.NewJoin(deps).Execute(t.Context(), link, device)
	if err != nil {
		t.Fatal(err)
	}
	return joined.ID
}

func waitPhase(t *testing.T, deps shareinvite.Deps, id string, want shareinvite.Phase) shareinvite.Status {
	t.Helper()
	var status shareinvite.Status
	synctest.AwaitCondition(t, "the request to be "+string(want), func() bool {
		var err error
		status, err = shareinvite.NewGetStatus(deps).Execute(id)
		return err == nil && status.Phase == want
	})
	return status
}

func (w *world) waitRequest(t *testing.T, count int) []shareinvite.Request {
	t.Helper()
	var requests []shareinvite.Request
	synctest.AwaitCondition(t, "the requests to arrive", func() bool {
		requests = shareinvite.NewListRequests(w.sharer).Execute(t.Context())
		return len(requests) == count
	})
	return requests
}

func TestApprove_AddsTheDeviceAsReaderOfTheFolder(t *testing.T) {
	w := newWorld(t)
	reader, readerDeps := w.newReader(t, "Reader")
	started := w.start(t)
	id := join(t, readerDeps, started.Link, "Anna laptop")
	waiting := waitPhase(t, readerDeps, id, shareinvite.PhaseWaiting)
	requests := w.waitRequest(t, 1)

	if err := shareinvite.NewApprove(w.sharer).Execute(t.Context(), requests[0].ID); err != nil {
		t.Fatal(err)
	}

	waitPhase(t, readerDeps, id, shareinvite.PhaseReceived)
	received, err := reader.ListReceived(t.Context())
	if err != nil || len(received) != 1 || received[0].Name != folder {
		t.Fatalf("received = %+v, %v, want the folder %s", received, err, folder)
	}
	if text, ok := reader.ReadNote(received[0].VaultID, "Soup.md"); !ok || text != "carrots" {
		t.Errorf("note = %q (%v), want carrots", text, ok)
	}
	overview, err := w.owner.GetShare(t.Context(), owned, folder)
	if err != nil || len(overview.Readers) != 1 || overview.Readers[0].Label != "Anna laptop" {
		t.Errorf("readers = %+v, %v, want the device name as the label", overview.Readers, err)
	}
	if requests[0].Code != waiting.Code || requests[0].DeviceName != "Anna laptop" {
		t.Errorf("request = %+v, status = %+v, want the same code and the announced device name", requests[0], waiting)
	}
}

func TestApprove_ShowsTheRequestAsAHealthItemUntilAnswered(t *testing.T) {
	w := newWorld(t)
	_, readerDeps := w.newReader(t, "Reader")
	id := join(t, readerDeps, w.start(t).Link, "Anna laptop")
	waitPhase(t, readerDeps, id, shareinvite.PhaseWaiting)
	requests := w.waitRequest(t, 1)
	hasItem := func() bool {
		for _, item := range w.owner.Health.List() {
			if item.Key == shareinvite.HealthKey && strings.Contains(item.Text, "Anna laptop") && strings.Contains(item.Detail, requests[0].Code) {
				return true
			}
		}
		return false
	}
	synctest.AwaitCondition(t, "the health item", hasItem)

	if err := shareinvite.NewDecline(w.sharer).Execute(t.Context(), requests[0].ID); err != nil {
		t.Fatal(err)
	}

	synctest.AwaitCondition(t, "the health item to go", func() bool { return !hasItem() })
}

func TestDecline_TellsTheDeviceAndAddsNoReader(t *testing.T) {
	w := newWorld(t)
	_, readerDeps := w.newReader(t, "Reader")
	id := join(t, readerDeps, w.start(t).Link, "Anna laptop")
	waitPhase(t, readerDeps, id, shareinvite.PhaseWaiting)
	requests := w.waitRequest(t, 1)

	if err := shareinvite.NewDecline(w.sharer).Execute(t.Context(), requests[0].ID); err != nil {
		t.Fatal(err)
	}

	waitPhase(t, readerDeps, id, shareinvite.PhaseDeclined)
	overview, err := w.owner.GetShare(t.Context(), owned, folder)
	if err != nil || len(overview.Readers) != 0 {
		t.Errorf("readers = %+v, %v, want none", overview.Readers, err)
	}
	if left := shareinvite.NewListRequests(w.sharer).Execute(t.Context()); len(left) != 0 {
		t.Errorf("requests = %+v, want none after the answer", left)
	}
}

func TestJoin_RefusesALinkAfterItExpired(t *testing.T) {
	w := newWorld(t)
	_, readerDeps := w.newReader(t, "Reader")
	started := w.start(t)
	w.clock.Advance(started.ExpiresAt.Sub(w.clock.Now()) + time.Second)

	_, err := shareinvite.NewJoin(readerDeps).Execute(t.Context(), started.Link, "Anna laptop")

	if !errors.Is(err, domainshare.ErrLinkExpired) {
		t.Errorf("error = %v, want ErrLinkExpired", err)
	}
}

func TestStart_ExpiresInTheLifetimeOfTheChannelAtMostADay(t *testing.T) {
	w := newWorld(t)

	started := w.start(t)

	if limit := w.clock.Now().Add(shareinvite.MaxLifetime); started.ExpiresAt.After(limit) || !started.ExpiresAt.After(w.clock.Now()) {
		t.Errorf("expires at %v, want a moment after now and within %v", started.ExpiresAt, shareinvite.MaxLifetime)
	}
}

func TestJoin_RefusesASecondDeviceOnTheSameLink(t *testing.T) {
	w := newWorld(t)
	_, firstDeps := w.newReader(t, "First")
	_, secondDeps := w.newReader(t, "Second")
	link := w.start(t).Link
	first := join(t, firstDeps, link, "First laptop")
	waitPhase(t, firstDeps, first, shareinvite.PhaseWaiting)

	second := join(t, secondDeps, link, "Second laptop")

	waitPhase(t, secondDeps, second, shareinvite.PhaseUsed)
	requests := w.waitRequest(t, 1)
	if requests[0].DeviceName != "First laptop" {
		t.Errorf("request = %+v, want the first device only", requests[0])
	}
	if status, err := shareinvite.NewGetStatus(firstDeps).Execute(first); err != nil || status.Phase != shareinvite.PhaseWaiting {
		t.Errorf("first status = %+v, %v, want it still waiting", status, err)
	}
}

func TestReadLink_RefusesATamperedLink(t *testing.T) {
	w := newWorld(t)
	_, readerDeps := w.newReader(t, "Reader")
	link := w.start(t).Link
	middle := len(link) / 2
	flipped := string(link[middle]%26 + 'A')
	if flipped == string(link[middle]) {
		flipped = "B"
	}
	tampered := link[:middle] + flipped + link[middle+1:]
	for name, text := range map[string]string{"changed": tampered, "cut": link[:len(link)-6], "another scheme": strings.Replace(link, "numen-share:", "numen-link:", 1)} {
		if _, err := shareinvite.NewReadLink(readerDeps).Execute(text); !errors.Is(err, domainshare.ErrBadLink) {
			t.Errorf("%s: error = %v, want ErrBadLink", name, err)
		}
		if _, err := shareinvite.NewJoin(readerDeps).Execute(t.Context(), text, ""); !errors.Is(err, domainshare.ErrBadLink) {
			t.Errorf("%s: join error = %v, want ErrBadLink", name, err)
		}
	}
}

func TestReadLink_RefusesALinkForAnotherServer(t *testing.T) {
	w := newWorld(t)
	_, readerDeps := w.newReader(t, "Reader")
	readerDeps.Server = "https://other.example.org"
	link := w.start(t).Link

	_, err := shareinvite.NewReadLink(readerDeps).Execute(link)

	if !errors.Is(err, domainshare.ErrLinkOtherServer) {
		t.Errorf("error = %v, want ErrLinkOtherServer", err)
	}
}

func TestReadLink_TellsFolderOwnerAndReadOnly(t *testing.T) {
	w := newWorld(t)
	_, readerDeps := w.newReader(t, "Reader")
	started := w.start(t)

	preview, err := shareinvite.NewReadLink(readerDeps).Execute(started.Link)

	if err != nil || preview.Folder != folder || preview.OwnerFingerprint == "" || !preview.IsReadOnly || !preview.ExpiresAt.Equal(started.ExpiresAt.Truncate(time.Second)) {
		t.Errorf("preview = %+v, %v, want the folder, the owner, read-only and the expiry", preview, err)
	}
}

func TestJoin_RefusesTheLinkOfTheSameAccount(t *testing.T) {
	w := newWorld(t)
	link := w.start(t).Link

	_, err := shareinvite.NewJoin(w.sharer).Execute(t.Context(), link, "")

	if !errors.Is(err, domainshare.ErrOwnIdentity) {
		t.Errorf("error = %v, want ErrOwnIdentity", err)
	}
}

func TestRevoke_StopsALinkThatWasNotUsed(t *testing.T) {
	w := newWorld(t)
	_, readerDeps := w.newReader(t, "Reader")
	started := w.start(t)
	listed := shareinvite.NewListInvitations(w.sharer).Execute(t.Context())
	if len(listed) != 1 || !listed[0].ExpiresAt.Equal(started.ExpiresAt) || listed[0].IsRequested {
		t.Fatalf("invitations = %+v, want the link with its expiry and no request", listed)
	}

	if err := shareinvite.NewRevoke(w.sharer).Execute(t.Context(), listed[0].ID); err != nil {
		t.Fatal(err)
	}

	if left := shareinvite.NewListInvitations(w.sharer).Execute(t.Context()); len(left) != 0 {
		t.Errorf("invitations = %+v, want none", left)
	}
	if _, err := shareinvite.NewJoin(readerDeps).Execute(t.Context(), started.Link, ""); !errors.Is(err, domainshare.ErrLinkExpired) {
		t.Errorf("join error = %v, want ErrLinkExpired", err)
	}
}

func TestApprove_RefusesARequestThatDoesNotExist(t *testing.T) {
	w := newWorld(t)

	err := shareinvite.NewApprove(w.sharer).Execute(t.Context(), "00")

	if !errors.Is(err, domainshare.ErrUnknownRequest) {
		t.Errorf("error = %v, want ErrUnknownRequest", err)
	}
}

func TestApprove_ShareTheWholeVaultWhenTheFolderIsEmpty(t *testing.T) {
	w := newWorld(t)
	reader, readerDeps := w.newReader(t, "Reader")
	started, err := shareinvite.NewStart(w.sharer).Execute(t.Context(), owned, "")
	if err != nil {
		t.Fatal(err)
	}
	id := join(t, readerDeps, started.Link, "Anna laptop")
	waitPhase(t, readerDeps, id, shareinvite.PhaseWaiting)
	requests := w.waitRequest(t, 1)
	if requests[0].Label != string(owned) {
		t.Errorf("request label = %q, want the vault name", requests[0].Label)
	}

	if err := shareinvite.NewApprove(w.sharer).Execute(t.Context(), requests[0].ID); err != nil {
		t.Fatal(err)
	}

	waitPhase(t, readerDeps, id, shareinvite.PhaseReceived)
	received, err := reader.ListReceived(t.Context())
	if err != nil || len(received) != 1 {
		t.Fatalf("received = %+v, %v, want one vault", received, err)
	}
	if text, ok := reader.ReadNote(received[0].VaultID, "Food/Soup.md"); !ok || text != "carrots" {
		t.Errorf("note = %q (%v), want carrots", text, ok)
	}
}

type markedRelay struct {
	ports.Relay

	isMarked atomic.Bool
}

func (m *markedRelay) WaitMessages(ctx context.Context, channelID []byte, reader syncclient.LinkSide, afterSeq uint32, wait time.Duration) (syncclient.LinkMessages, error) {
	if reader == ports.LinkSideScanner && m.isMarked.CompareAndSwap(false, true) {
		return syncclient.LinkMessages{Messages: []syncclient.LinkMessage{{Seq: 0, Body: []byte("numen-share-used")}}, IsClosed: false}, nil
	}
	return m.Relay.WaitMessages(ctx, channelID, reader, afterSeq, wait)
}

func TestJoin_IgnoresAnUnauthenticatedUsedMarkerBeforeAuthentication(t *testing.T) {
	w := newWorld(t)
	_, readerDeps := w.newReader(t, "Reader")
	readerDeps.Relay = &markedRelay{Relay: readerDeps.Relay}
	link := w.start(t).Link

	id := join(t, readerDeps, link, "Anna laptop")

	waitPhase(t, readerDeps, id, shareinvite.PhaseWaiting)
}
