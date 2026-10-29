package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	flashcardsv1 "github.com/jiva-studio/numen/plugins/views-flashcards-studio/gen/numen/flashcards/v1"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/gen/numen/flashcards/v1/flashcardsv1connect"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/vaulttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

var errPageGone = errors.New("the page is gone")

// breaking serves a stream whose writes fail where fails says so, and hands
// back what the handler returned. The handler runs under ctx.
type breaking struct {
	fails    func(n int, msg any) bool
	ctx      context.Context
	returned chan error
}

func (breaking) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc { return next }

func (breaking) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (b breaking) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(_ context.Context, conn connect.StreamingHandlerConn) error {
		err := next(b.ctx, &breakingConn{StreamingHandlerConn: conn, fails: b.fails}) //nolint:contextcheck // handler runs under ctx
		b.returned <- err
		return err
	}
}

// breakingConn is the stream as the handler writes to it. The nth write is the
// nth message, counted from one.
type breakingConn struct {
	connect.StreamingHandlerConn
	fails     func(n int, msg any) bool
	sentCount int
}

func (c *breakingConn) Send(msg any) error {
	c.sentCount++
	if c.fails(c.sentCount, msg) {
		return errPageGone
	}
	return c.StreamingHandlerConn.Send(msg)
}

// newBreakingClient is the window's client over a server whose streams are
// broken as b says.
func newBreakingClient(t *testing.T, api *API, b breaking) flashcardsv1connect.FlashcardsServiceClient {
	t.Helper()
	mux := http.NewServeMux()
	path, handler := flashcardsv1connect.NewFlashcardsServiceHandler(api, connect.WithInterceptors(b))
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return flashcardsv1connect.NewFlashcardsServiceClient(server.Client(), server.URL)
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

// A stream whose page can no longer be written to ends with the write's error,
// at whichever message it was found out.
func TestAReloadStreamThatCannotBeSavedToEndsWithWhyItCannot(t *testing.T) {
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()

	for name, one := range map[string]struct {
		ctx   context.Context
		fails func(n int, msg any) bool
		moved bool
	}{
		"opening": {cancelled, func(n int, _ any) bool { return n == 1 }, false},
		// Nothing moved, and the window says so again until the page is found
		// gone.
		"repeating": {t.Context(), func(n int, _ any) bool { return n > 1 }, false},
		// Something moved, and the page is gone by the time it is told.
		"reloading": {t.Context(), func(_ int, msg any) bool {
			return msg.(*flashcardsv1.WatchReloadsResponse).GetReload()
		}, true},
	} {
		t.Run(name, func(t *testing.T) {
			api, _, installation := newAPIOn(t)
			returned := make(chan error, 1)
			client := newBreakingClient(t, api, breaking{fails: one.fails, ctx: one.ctx, returned: returned})

			stream, err := client.WatchReloads(t.Context(), connect.NewRequest(&flashcardsv1.WatchReloadsRequest{}))
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
			return msg.(*flashcardsv1.WatchCardsDueResponse).GetCounted() != nil
		},
	} {
		t.Run(name, func(t *testing.T) {
			api, _ := newAPI(t, deck)
			front(t, api)

			returned := make(chan error, 1)
			client := newBreakingClient(t, api, breaking{fails: fails, ctx: t.Context(), returned: returned})
			stream, err := client.WatchCardsDue(t.Context(), connect.NewRequest(&flashcardsv1.WatchCardsDueRequest{}))
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
	api, _ := newAPI(t, deck)
	front(t, api)

	target := make(chan struct{})
	defer close(target)
	api.scenarios().flash.CardsDue = api.scenarios().flash.CardsDue.WithReviewCards(api.scenarios().flash.CardsDue.ReviewCards().WithReaders(waiting{until: target}))

	returned := make(chan error, 1)
	client := newBreakingClient(t, api, breaking{
		fails:    func(n int, _ any) bool { return n > 1 },
		ctx:      t.Context(),
		returned: returned,
	})
	stream, err := client.WatchCardsDue(t.Context(), connect.NewRequest(&flashcardsv1.WatchCardsDueRequest{}))
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

// A stream that has just said a count is not about to say the list again: the
// time it waits starts over with every count.
func TestWatchCardsDue_ListWaitRestartsAfterEachCount(t *testing.T) {
	api, target := newAPI(t, deck, other)
	front(t, api)

	first, second := make(chan struct{}), make(chan struct{})
	api.scenarios().flash.CardsDue = api.scenarios().flash.CardsDue.WithReviewCards(api.scenarios().flash.CardsDue.ReviewCards().WithReaders(gated{
		until:   map[domain.VaultID]<-chan struct{}{target[0].ID: first, target[1].ID: second},
		entered: make(chan domain.VaultID, 16),
	}))
	openFirst := sync.OnceFunc(func() { close(first) })
	openSecond := sync.OnceFunc(func() { close(second) })
	t.Cleanup(func() { openFirst(); openSecond() })

	stream, err := newFlashcardsClient(t, api).WatchCardsDue(
		t.Context(), connect.NewRequest(&flashcardsv1.WatchCardsDueRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })
	if !stream.Receive() {
		t.Fatalf("overview stream never opened: %v", stream.Err())
	}

	// The first count lands half way to the list being said again, and the
	// second lands after the moment the list was due from the start.
	time.AfterFunc(500*time.Millisecond, openFirst)
	time.AfterFunc(1300*time.Millisecond, openSecond)

	var counted, items int
	for stream.Receive() {
		if stream.Msg().GetCounted() != nil {
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
	api, target := newAPI(t, deck, deck, deck, deck, deck, deck)
	front(t, api)

	until := make(chan struct{})
	gates := map[domain.VaultID]<-chan struct{}{}
	for _, v := range target {
		gates[v.ID] = until
	}
	entered := make(chan domain.VaultID, 64)
	api.scenarios().flash.CardsDue = api.scenarios().flash.CardsDue.WithReviewCards(api.scenarios().flash.CardsDue.ReviewCards().WithReaders(
		gated{until: gates, entered: entered}))
	openAll := sync.OnceFunc(func() { close(until) })
	t.Cleanup(openAll)

	ctx, cancel := context.WithTimeout(t.Context(), vaulttest.Patience)
	defer cancel()
	stream, err := newFlashcardsClient(t, api).WatchCardsDue(
		ctx, connect.NewRequest(&flashcardsv1.WatchCardsDueRequest{}))
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
		if stream.Msg().GetCounted() != nil {
			counted++
		}
	}
	if counted != 6 {
		t.Errorf("%d vaults were counted, want 6: %v", counted, stream.Err())
	}
}

// A stream whose client has gone waits for the counts already running before
// it ends, so no count outlives the call that asked for it.
func TestWatchCardsDue_EndsAfterRunningCountsFinish(t *testing.T) {
	api, target := newAPI(t, deck, deck, deck, deck, deck, deck)
	front(t, api)

	until := make(chan struct{})
	gates := map[domain.VaultID]<-chan struct{}{}
	for _, v := range target {
		gates[v.ID] = until
	}
	entered := make(chan domain.VaultID, 64)
	api.scenarios().flash.CardsDue = api.scenarios().flash.CardsDue.WithReviewCards(api.scenarios().flash.CardsDue.ReviewCards().WithReaders(
		gated{until: gates, entered: entered}))
	openAll := sync.OnceFunc(func() { close(until) })
	t.Cleanup(openAll)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	returned := make(chan error, 1)
	client := newBreakingClient(t, api, breaking{
		fails: func(int, any) bool { return false }, ctx: ctx, returned: returned,
	})
	stream, err := client.WatchCardsDue(t.Context(), connect.NewRequest(&flashcardsv1.WatchCardsDueRequest{}))
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

// rooted serves the handler the way a window does: the request context is the process's own, and
// ends when the application ends and at no other moment.
func rooted(handler http.Handler, entered, returned chan<- struct{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entered <- struct{}{}
		defer func() { returned <- struct{}{} }()
		handler.ServeHTTP(w, r.WithContext(context.Background())) //nolint:contextcheck // request context ends with the process
	})
}

// TestAStreamWhoseClientWentAwayEnds. A handler rooted at the process is never told the page it was
// opened from is gone. A write that fails is the one report there is, so the stream makes one.
func TestAStreamWhoseClientWentAwayEnds(t *testing.T) {
	api, _ := newAPI(t)
	entered, returned := make(chan struct{}, 1), make(chan struct{}, 1)
	mux := http.NewServeMux()
	path, handler := flashcardsv1connect.NewFlashcardsServiceHandler(api)
	mux.Handle(path, rooted(handler, entered, returned))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	_, err := flashcardsv1connect.NewFlashcardsServiceClient(server.Client(), server.URL).
		WatchReloads(t.Context(), connect.NewRequest(&flashcardsv1.WatchReloadsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-entered:
	case <-time.After(vaulttest.Patience):
		t.Fatal("the stream never reached the handler")
	}

	// The page is gone, and nothing cancelled the context the handler holds.
	server.CloseClientConnections()

	select {
	case <-returned:
	case <-time.After(vaulttest.Patience):
		t.Fatal("the client went away and the stream is still standing")
	}
}

// TestTheCountsSayAgainWhileAVaultIsBeingCounted. Counting reads whole vaults,
// four of them at once, and the request context belongs to the process. A
// stream saying nothing until a count lands leaves those counters reading for a
// window that has gone, and reloading the page stacks another four on them.
func TestTheCountsSayAgainWhileAVaultIsBeingCounted(t *testing.T) {
	api, _ := newAPI(t, deck)
	// The vault is read when the window opens it, and what a count that will not
	// come back does to the stream is what is under test here.
	front(t, api)

	target := make(chan struct{})
	defer close(target)
	api.scenarios().flash.CardsDue = api.scenarios().flash.CardsDue.WithReviewCards(api.scenarios().flash.CardsDue.ReviewCards().WithReaders(waiting{until: target}))

	stream, err := newFlashcardsClient(t, api).WatchCardsDue(
		t.Context(), connect.NewRequest(&flashcardsv1.WatchCardsDueRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	if !stream.Receive() {
		t.Fatalf("overview stream never opened: %v", stream.Err())
	}
	if len(stream.Msg().GetVaults()) != 1 {
		t.Fatalf("overview stream received %+v", stream.Msg().GetVaults())
	}

	// Nothing has been counted and nothing will be, so what arrives next is the
	// stream reaching for its client.
	if !stream.Receive() {
		t.Fatalf("the stream said nothing while the count ran: %v", stream.Err())
	}
	response := stream.Msg()
	if response.GetCounted() != nil {
		t.Fatalf("a count arrived for a vault nothing could read: %+v", response.GetCounted())
	}
	if len(response.GetVaults()) != 1 {
		t.Errorf("the stream said again and listed %+v", response.GetVaults())
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
