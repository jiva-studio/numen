package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/vaulttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

var errPageGone = errors.New("the page is gone")

// fixedContext returns a function that always returns ctx.
func fixedContext(ctx context.Context) func() context.Context {
	return func() context.Context { return ctx }
}

// breaking serves a stream whose writes fail where `fails` returns true, and
// reports what the handler returned. The handler runs under the context that
// getContext returns.
type breaking struct {
	fails      func(n int, msg any) bool
	getContext func() context.Context
	returned   chan error
}

// breakingClient is the client of the review window over streams broken as b
// specifies.
type breakingClient struct {
	handler *API
	b       breaking
}

// newBreakingClient returns the client of the review window over streams broken
// as b specifies.
func newBreakingClient(_ *testing.T, handler *API, b breaking) breakingClient {
	return breakingClient{handler: handler, b: b}
}

// breakStream runs the stream method under the breaking context, counting each
// message sent and failing the ones b selects.
func breakStream[Req, Res any](
	c breakingClient, req *Req, method func(context.Context, *Req, func(*Res) error) error,
) (*messageStream[Res], error) {
	return openStream(context.WithoutCancel(c.b.getContext()), req, func(_ context.Context, req *Req, send func(*Res) error) error {
		count := 0
		err := method(c.b.getContext(), req, func(res *Res) error {
			count++
			if c.b.fails(count, res) {
				return errPageGone
			}
			return send(res)
		})
		c.b.returned <- err
		return err
	})
}

// WatchReloads opens the reload stream through the breaking writes.
func (c breakingClient) WatchReloads(
	_ context.Context, req *api.WatchReloadsRequest,
) (*messageStream[api.WatchReloadsResponse], error) {
	return breakStream(c, req, c.handler.WatchReloads)
}

// WatchCardsDue opens the count stream through the breaking writes.
func (c breakingClient) WatchCardsDue(
	_ context.Context, req *api.WatchCardsDueRequest,
) (*messageStream[api.WatchCardsDueResponse], error) {
	return breakStream(c, req, c.handler.WatchCardsDue)
}

// awaitReturn is what the handler returned.
func awaitReturn(t *testing.T, returned <-chan error) error {
	t.Helper()
	select {
	case err := <-returned:
		return err
	case <-time.After(vaulttest.Patience):
		t.Fatal("the page could not be written to and the stream is still standing")
		return nil
	}
}

// A stream whose page can no longer be written to ends with the write's error, at
// whichever message the failure was detected.
func TestAReloadStreamThatCannotBeSavedToEndsWithWhyItCannot(t *testing.T) {
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()

	for name, one := range map[string]struct {
		getContext func() context.Context
		fails      func(n int, msg any) bool
		moved      bool
	}{
		"opening": {fixedContext(cancelled), func(n int, _ any) bool { return n == 1 }, false},
		// 		// Nothing changed, and the stream repeats the message until the page is
		// 		// found gone.
		"repeating": {fixedContext(t.Context()), func(n int, _ any) bool { return n > 1 }, false},
		// 		// Something changed, and the page is gone by the time it is sent.
		"reloading": {fixedContext(t.Context()), func(_ int, msg any) bool {
			return msg.(*api.WatchReloadsResponse).Reload
		}, true},
	} {
		t.Run(name, func(t *testing.T) {
			handler, _, installation := newAPIOn(t)
			returned := make(chan error, 1)
			client := newBreakingClient(t, handler, breaking{fails: one.fails, getContext: one.getContext, returned: returned})

			stream, err := client.WatchReloads(t.Context(), &api.WatchReloadsRequest{})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { stream.Close() })
			if one.moved {
				if !stream.Receive() {
					t.Fatalf("the stream said nothing: %v", stream.Err())
				}
				installation.Host.Move()
			}

			if err := awaitReturn(t, returned); !errors.Is(err, errPageGone) {
				t.Errorf("the stream ended with %v, want the write's error", err)
			}
		})
	}
}

// written to, whichever message finds it out.
func TestWatchCardsDue_SaveFailureEndsStream(t *testing.T) {
	for name, fails := range map[string]func(n int, msg any) bool{
		// The vaults are listed before anything is counted.
		"listing": func(n int, _ any) bool { return n == 1 },
		// A count has come in.
		"counting": func(_ int, msg any) bool {
			return msg.(*api.WatchCardsDueResponse).Counted != nil
		},
	} {
		t.Run(name, func(t *testing.T) {
			handler, _ := newAPI(t, deck)
			front(t, handler)

			returned := make(chan error, 1)
			client := newBreakingClient(t, handler, breaking{fails: fails, getContext: fixedContext(t.Context()), returned: returned})
			stream, err := client.WatchCardsDue(t.Context(), &api.WatchCardsDueRequest{})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { stream.Close() })

			if err := awaitReturn(t, returned); !errors.Is(err, errPageGone) {
				t.Errorf("the stream ended with %v, want the write's error", err)
			}
		})
	}
}

// A page found gone while a count is still running ends the stream, which is
// the only report there is.
func TestWatchCardsDue_MissingPageDuringCountEndsStream(t *testing.T) {
	handler, _ := newAPI(t, deck)
	front(t, handler)

	target := make(chan struct{})
	defer close(target)
	handler.scenarios().flash.CardsDue = handler.scenarios().flash.CardsDue.WithReviewCards(handler.scenarios().flash.CardsDue.ReviewCards().WithReaders(waiting{until: target}))

	returned := make(chan error, 1)
	client := newBreakingClient(t, handler, breaking{
		fails:      func(n int, _ any) bool { return n > 1 },
		getContext: fixedContext(t.Context()),
		returned:   returned,
	})
	stream, err := client.WatchCardsDue(t.Context(), &api.WatchCardsDueRequest{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })

	if err := awaitReturn(t, returned); !errors.Is(err, errPageGone) {
		t.Errorf("the stream ended with %v, want the write's error", err)
	}
}

// gated is vaults whose review cards come back when their own gate is closed.
type gated struct {
	until   map[domain.VaultID]<-chan struct{}
	entered chan domain.VaultID
}

func (g gated) Open(v domain.Vault) (ports.VaultReader, error) {
	select {
	case g.entered <- v.ID:
	default:
	}
	<-g.until[v.ID]
	return nil, errNothingRead
}

// A stream that has just sent a count does not send the list again immediately:
// the wait before the list restarts after every count.
func TestWatchCardsDue_ListWaitRestartsAfterEachCount(t *testing.T) {
	handler, target := newAPI(t, deck, other)
	front(t, handler)

	first, second := make(chan struct{}), make(chan struct{})
	handler.scenarios().flash.CardsDue = handler.scenarios().flash.CardsDue.WithReviewCards(handler.scenarios().flash.CardsDue.ReviewCards().WithReaders(gated{
		until:   map[domain.VaultID]<-chan struct{}{target[0].ID: first, target[1].ID: second},
		entered: make(chan domain.VaultID, 16),
	}))
	openFirst := sync.OnceFunc(func() { close(first) })
	openSecond := sync.OnceFunc(func() { close(second) })
	t.Cleanup(func() { openFirst(); openSecond() })

	stream, err := newFlashcardsClient(t, handler).WatchCardsDue(
		t.Context(), &api.WatchCardsDueRequest{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })
	if !stream.Receive() {
		t.Fatalf("overview stream never opened: %v", stream.Err())
	}

	// 	// The first count arrives halfway to the next list message, and the second
	// 	// arrives after the point at which the list was due from the start.
	time.AfterFunc(500*time.Millisecond, openFirst)
	time.AfterFunc(1300*time.Millisecond, openSecond)

	var counted, items int
	for stream.Receive() {
		if stream.Msg().Counted != nil {
			counted++
		} else {
			items++
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	if counted != 2 || items != 0 {
		t.Errorf("%d counts arrived and the list was said again %d times, want 2 and 0", counted, items)
	}
}

// Four vaults are counted alongside each other, and the rest wait for a place.
func TestFourVaultsAreCountedAtOnce(t *testing.T) {
	handler, target := newAPI(t, deck, deck, deck, deck, deck, deck)
	front(t, handler)

	until := make(chan struct{})
	gates := map[domain.VaultID]<-chan struct{}{}
	for _, v := range target {
		gates[v.ID] = until
	}
	entered := make(chan domain.VaultID, 64)
	handler.scenarios().flash.CardsDue = handler.scenarios().flash.CardsDue.WithReviewCards(handler.scenarios().flash.CardsDue.ReviewCards().WithReaders(
		gated{until: gates, entered: entered}))
	openAll := sync.OnceFunc(func() { close(until) })
	t.Cleanup(openAll)

	ctx, cancel := context.WithTimeout(t.Context(), vaulttest.Patience)
	defer cancel()
	stream, err := newFlashcardsClient(t, handler).WatchCardsDue(
		ctx, &api.WatchCardsDueRequest{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })

	running := map[domain.VaultID]bool{}
	for len(running) < 4 {
		select {
		case id := <-entered:
			running[id] = true
		case <-time.After(vaulttest.Patience):
			t.Fatalf("%d vaults are being counted, want 4", len(running))
		}
	}
	select {
	case id := <-entered:
		running[id] = true
		t.Errorf("%d vaults are being counted at once, want 4", len(running))
	case <-time.After(300 * time.Millisecond):
	}

	openAll()
	counted := 0
	for stream.Receive() {
		if stream.Msg().Counted != nil {
			counted++
		}
	}
	if counted != 6 {
		t.Errorf("%d vaults were counted, want 6: %v", counted, stream.Err())
	}
}

// A stream whose client has disconnected waits for the counts already running
// before it ends, so no count outlives the call that started it.
func TestWatchCardsDue_EndsAfterRunningCountsFinish(t *testing.T) {
	handler, target := newAPI(t, deck, deck, deck, deck, deck, deck)
	front(t, handler)

	until := make(chan struct{})
	gates := map[domain.VaultID]<-chan struct{}{}
	for _, v := range target {
		gates[v.ID] = until
	}
	entered := make(chan domain.VaultID, 64)
	handler.scenarios().flash.CardsDue = handler.scenarios().flash.CardsDue.WithReviewCards(handler.scenarios().flash.CardsDue.ReviewCards().WithReaders(
		gated{until: gates, entered: entered}))
	openAll := sync.OnceFunc(func() { close(until) })
	t.Cleanup(openAll)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	returned := make(chan error, 1)
	client := newBreakingClient(t, handler, breaking{
		fails: func(int, any) bool { return false }, getContext: fixedContext(ctx), returned: returned,
	})
	stream, err := client.WatchCardsDue(t.Context(), &api.WatchCardsDueRequest{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })

	running := map[domain.VaultID]bool{}
	for len(running) < 4 {
		select {
		case id := <-entered:
			running[id] = true
		case <-time.After(vaulttest.Patience):
			t.Fatalf("%d vaults are being counted, want 4", len(running))
		}
	}

	cancel()
	select {
	case <-returned:
		t.Fatal("the stream ended while four counts were still running")
	case <-time.After(300 * time.Millisecond):
	}

	openAll()
	if err := awaitReturn(t, returned); err != nil {
		t.Errorf("the stream ended with %v", err)
	}
}

// TestAStreamWhoseClientWentAwayEnds. A stream rooted at the process receives no
// notice that the page it was opened from is gone. A failed write is the only
// report available, so the stream sends one.
func TestAStreamWhoseClientWentAwayEnds(t *testing.T) {
	handler, _ := newAPI(t)
	returned := make(chan error, 1)
	firstMessage := make(chan struct{}, 1)
	go func() {
		returned <- handler.WatchReloads(context.Background(), &api.WatchReloadsRequest{},
			func(*api.WatchReloadsResponse) error {
				select {
				case firstMessage <- struct{}{}:
					return nil
				default:
					return errPageGone
				}
			})
	}()

	select {
	case <-firstMessage:
	case <-time.After(vaulttest.Patience):
		t.Fatal("the stream never sent its opening message")
	}

	// The page is gone, and nothing cancelled the context the handler holds.
	if err := awaitReturn(t, returned); !errors.Is(err, errPageGone) {
		t.Errorf("the stream ended with %v, want the write's error", err)
	}
}

// TestTheCountsSayAgainWhileAVaultIsBeingCounted. Counting reads whole vaults,
// four of them at once, and the request context belongs to the process. A
// stream saying nothing until a count lands leaves those counters reading for a
// window that has gone, and reloading the page stacks another four on them.
func TestTheCountsSayAgainWhileAVaultIsBeingCounted(t *testing.T) {
	handler, _ := newAPI(t, deck)
	// The vault is read when the window opens it, and what a count that will not
	// come back does to the stream is what is under test here.
	front(t, handler)

	target := make(chan struct{})
	defer close(target)
	handler.scenarios().flash.CardsDue = handler.scenarios().flash.CardsDue.WithReviewCards(handler.scenarios().flash.CardsDue.ReviewCards().WithReaders(waiting{until: target}))

	stream, err := newFlashcardsClient(t, handler).WatchCardsDue(
		t.Context(), &api.WatchCardsDueRequest{})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	if !stream.Receive() {
		t.Fatalf("overview stream never opened: %v", stream.Err())
	}
	if len(stream.Msg().Vaults) != 1 {
		t.Fatalf("overview stream received %+v", stream.Msg().Vaults)
	}

	// Nothing has been counted and nothing will be, so what arrives next is the
	// stream reaching for its client.
	if !stream.Receive() {
		t.Fatalf("the stream said nothing while the count ran: %v", stream.Err())
	}
	response := stream.Msg()
	if response.Counted != nil {
		t.Fatalf("a count arrived for a vault nothing could read: %+v", response.Counted)
	}
	if len(response.Vaults) != 1 {
		t.Errorf("the stream said again and listed %+v", response.Vaults)
	}
}

// waiting is a vault whose review cards never come back, which is what counting a
// vault of many thousands looks like from the stream's side.
type waiting struct{ until <-chan struct{} }

func (w waiting) Open(domain.Vault) (ports.VaultReader, error) {
	<-w.until
	return nil, errNothingRead
}

var errNothingRead = errors.New("the count was let go")
