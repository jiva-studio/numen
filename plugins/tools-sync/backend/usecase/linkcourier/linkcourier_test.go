package linkcourier_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/linkcourier"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

type run struct {
	computer *synctest.Computer
	relay    *syncclient.LinkClient
	stop     context.CancelFunc
	done     chan error
	finished chan struct{}
}

func newRun(t *testing.T, serverURLOf func(*synctest.Server) string) *run {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	computer := synctest.NewFirstComputer(t, server, clock, "Test", synctest.NewWords(t))
	deps := computer.LinkDeps()
	relay := syncclient.NewLinkClient(server.URL, http.DefaultClient)
	if serverURLOf != nil {
		deps.Relay = syncclient.NewLinkClient(serverURLOf(server), http.DefaultClient)
	}
	ctx, stop := context.WithCancel(t.Context())
	done, finished := make(chan error, 1), make(chan struct{})
	go func() {
		done <- linkcourier.NewCourier(deps).Execute(ctx)
		close(finished)
	}()
	t.Cleanup(func() {
		stop()
		<-finished
	})
	return &run{computer: computer, relay: relay, stop: stop, done: done, finished: finished}
}

func TestExecute_CreatesChannelWhenCoreRequestsOneAndReturnsIt(t *testing.T) {
	t.Parallel()
	r := newRun(t, nil)
	if err := r.computer.Fake.ShowLink(); err != nil {
		t.Fatal(err)
	}

	id, err := r.computer.Fake.WaitLinkChannel(t.Context())

	if err != nil || len(id) != 16 {
		t.Fatalf("channel = %x, %v, want a 16 byte id from the relay", id, err)
	}
	if _, err := r.relay.SendMessage(t.Context(), id, syncclient.LinkSideScanner, []byte("hello")); err != nil {
		t.Errorf("the channel the core got is not one of the relay: %v", err)
	}
}

func TestExecute_PostsCoreOutputToOtherSideOfChannel(t *testing.T) {
	t.Parallel()
	r := newRun(t, nil)
	if err := r.computer.Fake.ShowLink(); err != nil {
		t.Fatal(err)
	}
	id, err := r.computer.Fake.WaitLinkChannel(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := r.computer.Fake.QueueLinkOutbound([]byte("sealed blob")); err != nil {
		t.Fatal(err)
	}

	got, err := r.relay.WaitMessages(t.Context(), id, syncclient.LinkSideScanner, 0, 10*time.Second)

	if err != nil || len(got.Messages) != 1 || !bytes.Equal(got.Messages[0].Body, []byte("sealed blob")) {
		t.Errorf("the scanner reads %+v, %v, want the blob of the core", got, err)
	}
}

func TestExecute_PassesWhatTheOtherSidePostsToTheCore(t *testing.T) {
	t.Parallel()
	r := newRun(t, nil)
	if err := r.computer.Fake.ShowLink(); err != nil {
		t.Fatal(err)
	}
	id, err := r.computer.Fake.WaitLinkChannel(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.relay.SendMessage(t.Context(), id, syncclient.LinkSideDisplayer, []byte("from the scanner")); err != nil {
		t.Fatal(err)
	}

	got, err := r.computer.Fake.WaitLinkInbound(t.Context())

	if err != nil || !bytes.Equal(got, []byte("from the scanner")) {
		t.Errorf("the core got %q, %v, want the blob of the scanner", got, err)
	}
}

func TestExecute_ClosesTheChannelWhenTheCoreEndsTheLink(t *testing.T) {
	t.Parallel()
	r := newRun(t, nil)
	if err := r.computer.Fake.ShowLink(); err != nil {
		t.Fatal(err)
	}
	id, err := r.computer.Fake.WaitLinkChannel(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := r.computer.Fake.CloseLink(); err != nil {
		t.Fatal(err)
	}

	for range 100 {
		if _, err := r.relay.SendMessage(t.Context(), id, syncclient.LinkSideScanner, []byte("x")); err != nil {
			return
		}
		<-time.After(50 * time.Millisecond)
	}
	t.Error("the channel of an ended link is still open on the relay")
}

func TestExecute_ReportsLinkNeedsSyncServerWhenRelayIsMissing(t *testing.T) {
	t.Parallel()
	r := newRun(t, func(*synctest.Server) string { return "http://127.0.0.1:1" })
	if err := r.computer.Fake.ShowLink(); err != nil {
		t.Fatal(err)
	}

	for range 100 {
		if r.computer.HasItem("Linking needs the sync server") {
			return
		}
		<-time.After(50 * time.Millisecond)
	}
	t.Errorf("health = %+v, want the item \"Linking needs the sync server\"", r.computer.Health.List())
}

func TestExecute_StopsWhenTheContextEnds(t *testing.T) {
	t.Parallel()
	r := newRun(t, nil)
	r.stop()
	if err := <-r.done; err != nil && !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want nil or the cancellation", err)
	}
}

func waitHealthKey(t *testing.T, r *run, key string) bool {
	t.Helper()
	for range 100 {
		if r.computer.HasItemKey(key) {
			return true
		}
		<-time.After(50 * time.Millisecond)
	}
	return false
}

func TestExecute_SendsNothingOfALinkWhoseQRCodeNamesAnotherServer(t *testing.T) {
	t.Parallel()
	r := newRun(t, nil)
	channel, err := r.relay.CreateChannel(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := r.computer.Fake.ScanLink(channel.ID, "https://other.example.org"); err != nil {
		t.Fatal(err)
	}
	if err := r.computer.Fake.QueueLinkOutbound([]byte("sealed blob")); err != nil {
		t.Fatal(err)
	}

	if !waitHealthKey(t, r, "link-other-server") {
		t.Fatalf("health = %+v, want the item that names another server", r.computer.Health.List())
	}
	got, err := r.relay.WaitMessages(t.Context(), channel.ID, syncclient.LinkSideDisplayer, 0, 200*time.Millisecond)
	if err != nil || len(got.Messages) != 0 {
		t.Errorf("the relay holds %+v, %v, want nothing posted for another server", got, err)
	}
}

func TestExecute_SendsALinkWhoseQRCodeNamesTheServerOfTheDevice(t *testing.T) {
	t.Parallel()
	r := newRun(t, nil)
	channel, err := r.relay.CreateChannel(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := r.computer.Fake.ScanLink(channel.ID, r.computer.Server.URL); err != nil {
		t.Fatal(err)
	}
	if err := r.computer.Fake.QueueLinkOutbound([]byte("sealed blob")); err != nil {
		t.Fatal(err)
	}

	got, err := r.relay.WaitMessages(t.Context(), channel.ID, syncclient.LinkSideDisplayer, 0, 10*time.Second)

	if err != nil || len(got.Messages) != 1 || !bytes.Equal(got.Messages[0].Body, []byte("sealed blob")) {
		t.Errorf("the displayer reads %+v, %v, want the blob of the core", got, err)
	}
}

type lingeringRelay struct {
	ports.Relay
	waiting chan struct{}
	release chan struct{}
}

func (l lingeringRelay) WaitMessages(ctx context.Context, id []byte, side syncclient.LinkSide, after uint32, wait time.Duration) (syncclient.LinkMessages, error) {
	close(l.waiting)
	<-ctx.Done()
	<-l.release
	return syncclient.LinkMessages{}, ctx.Err()
}

func TestExecute_ReturnsOnlyAfterTheWaitForMessagesEnded(t *testing.T) {
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	computer := synctest.NewFirstComputer(t, server, clock, "Test", synctest.NewWords(t))
	deps := computer.LinkDeps()
	relay := lingeringRelay{Relay: deps.Relay, waiting: make(chan struct{}), release: make(chan struct{})}
	deps.Relay = relay
	ctx, stop := context.WithCancel(t.Context())
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		if err := linkcourier.NewCourier(deps).Execute(ctx); err != nil && !errors.Is(err, context.Canceled) {
			t.Error(err)
		}
	}()
	if err := computer.Fake.ShowLink(); err != nil {
		t.Fatal(err)
	}
	if _, err := computer.Fake.WaitLinkChannel(t.Context()); err != nil {
		t.Fatal(err)
	}
	<-relay.waiting

	stop()

	select {
	case <-finished:
		t.Fatal("Execute returned while the wait for messages still ran")
	case <-time.After(200 * time.Millisecond):
	}
	close(relay.release)
	<-finished
}
