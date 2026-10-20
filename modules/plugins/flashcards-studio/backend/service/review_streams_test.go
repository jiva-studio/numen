package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/plugins/flashcards-studio/backend/internal/vaulttest"
	"github.com/jiva-studio/numen/modules/plugins/flashcards-studio/backend/ports"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
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
		err := next(b.ctx, &breakingConn{StreamingHandlerConn: conn, fails: b.fails})
		b.returned <- err
		return err
	}
}

// breakingConn is the stream as the handler writes to it. The nth write is the
// nth message, counted from one.
type breakingConn struct {
	connect.StreamingHandlerConn
	fails func(n int, msg any) bool
	sent  int
}

func (c *breakingConn) Send(msg any) error {
	c.sent++
	if c.fails(c.sent, msg) {
		return errPageGone
	}
	return c.StreamingHandlerConn.Send(msg)
}

// newBreakingClient is the window's client over a server whose streams are
// broken as b says.
func newBreakingClient(t *testing.T, api *API, b breaking) numenv1connect.FlashcardsServiceClient {
	t.Helper()
	mux := http.NewServeMux()
	path, handler := numenv1connect.NewFlashcardsServiceHandler(api, connect.WithInterceptors(b))
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return numenv1connect.NewFlashcardsServiceClient(server.Client(), server.URL)
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
func TestAReloadStreamThatCannotBeWrittenToEndsWithWhyItCannot(t *testing.T) {
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()

	for name, one := range map[string]struct {
		ctx   context.Context
		fails func(n int, msg any) bool
		moved bool
	}{
		// The window is told the stream is open before it waits for anything.
		"opening": {cancelled, func(n int, _ any) bool { return n == 1 }, false},
		// Nothing moved, and the window says so again until the page is found
		// gone.
		"repeating": {t.Context(), func(n int, _ any) bool { return n > 1 }, false},
		// Something moved, and the page is gone by the time it is told.
		"reloading": {t.Context(), func(_ int, msg any) bool {
			return msg.(*v1.WatchReloadsResponse).GetReload()
		}, true},
	} {
		t.Run(name, func(t *testing.T) {
			api, _, installation := newAPIOn(t)
			returned := make(chan error, 1)
			client := newBreakingClient(t, api, breaking{fails: one.fails, ctx: one.ctx, returned: returned})

			stream, err := client.WatchReloads(t.Context(), connect.NewRequest(&v1.WatchReloadsRequest{}))
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

// The front door ends with the write's error when the page can no longer be
// written to, whichever message finds it out.
func TestTheFrontDoorThatCannotBeWrittenToEndsWithWhyItCannot(t *testing.T) {
	for name, fails := range map[string]func(n int, msg any) bool{
		// The vaults are listed before anything is counted.
		"listing": func(n int, _ any) bool { return n == 1 },
		// A count has come in.
		"counting": func(_ int, msg any) bool {
			return msg.(*v1.WatchCardsDueResponse).GetCounted() != nil
		},
	} {
		t.Run(name, func(t *testing.T) {
			api, _ := newAPI(t, deck)
			front(t, api)

			returned := make(chan error, 1)
			client := newBreakingClient(t, api, breaking{fails: fails, ctx: t.Context(), returned: returned})
			stream, err := client.WatchCardsDue(t.Context(), connect.NewRequest(&v1.WatchCardsDueRequest{}))
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
func TestTheFrontDoorFoundGoneWhileCountingEnds(t *testing.T) {
	api, _ := newAPI(t, deck)
	front(t, api)

	held := make(chan struct{})
	defer close(held)
	api.scenarios().flash.CardsDue = api.scenarios().flash.CardsDue.WithCardFaces(api.scenarios().flash.CardsDue.CardFaces().WithReaders(waiting{until: held}))

	returned := make(chan error, 1)
	client := newBreakingClient(t, api, breaking{
		fails:    func(n int, _ any) bool { return n > 1 },
		ctx:      t.Context(),
		returned: returned,
	})
	stream, err := client.WatchCardsDue(t.Context(), connect.NewRequest(&v1.WatchCardsDueRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })

	if err := awaitReturn(t, returned); !errors.Is(err, errPageGone) {
		t.Errorf("the stream ended with %v, want the write's error", err)
	}
}

// gated is vaults whose card faces come back when their own gate is closed.
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
func TestTheFrontDoorWaitsAfterEachCountBeforeSayingTheListAgain(t *testing.T) {
	api, held := newAPI(t, deck, other)
	front(t, api)

	first, second := make(chan struct{}), make(chan struct{})
	api.scenarios().flash.CardsDue = api.scenarios().flash.CardsDue.WithCardFaces(api.scenarios().flash.CardsDue.CardFaces().WithReaders(gated{
		until:   map[domain.VaultID]<-chan struct{}{held[0].ID: first, held[1].ID: second},
		entered: make(chan domain.VaultID, 16),
	}))
	openFirst := sync.OnceFunc(func() { close(first) })
	openSecond := sync.OnceFunc(func() { close(second) })
	t.Cleanup(func() { openFirst(); openSecond() })

	stream, err := newFlashcardsClient(t, api).WatchCardsDue(
		t.Context(), connect.NewRequest(&v1.WatchCardsDueRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })
	if !stream.Receive() {
		t.Fatalf("the front door never opened: %v", stream.Err())
	}

	// The first count lands half way to the list being said again, and the
	// second lands after the moment the list was due from the start.
	time.AfterFunc(500*time.Millisecond, openFirst)
	time.AfterFunc(1300*time.Millisecond, openSecond)

	var counted, listed int
	for stream.Receive() {
		if stream.Msg().GetCounted() != nil {
			counted++
		} else {
			listed++
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	if counted != 2 || listed != 0 {
		t.Errorf("%d counts arrived and the list was said again %d times, want 2 and 0", counted, listed)
	}
}

// Four vaults are counted alongside each other, and the rest wait for a place.
func TestFourVaultsAreCountedAtOnce(t *testing.T) {
	api, held := newAPI(t, deck, deck, deck, deck, deck, deck)
	front(t, api)

	until := make(chan struct{})
	gates := map[domain.VaultID]<-chan struct{}{}
	for _, v := range held {
		gates[v.ID] = until
	}
	entered := make(chan domain.VaultID, 64)
	api.scenarios().flash.CardsDue = api.scenarios().flash.CardsDue.WithCardFaces(api.scenarios().flash.CardsDue.CardFaces().WithReaders(
		gated{until: gates, entered: entered}))
	openAll := sync.OnceFunc(func() { close(until) })
	t.Cleanup(openAll)

	ctx, cancel := context.WithTimeout(t.Context(), vaulttest.Patience)
	defer cancel()
	stream, err := newFlashcardsClient(t, api).WatchCardsDue(
		ctx, connect.NewRequest(&v1.WatchCardsDueRequest{}))
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
func TestTheFrontDoorEndsOnlyWhenTheCountsRunningHaveEnded(t *testing.T) {
	api, held := newAPI(t, deck, deck, deck, deck, deck, deck)
	front(t, api)

	until := make(chan struct{})
	gates := map[domain.VaultID]<-chan struct{}{}
	for _, v := range held {
		gates[v.ID] = until
	}
	entered := make(chan domain.VaultID, 64)
	api.scenarios().flash.CardsDue = api.scenarios().flash.CardsDue.WithCardFaces(api.scenarios().flash.CardsDue.CardFaces().WithReaders(
		gated{until: gates, entered: entered}))
	openAll := sync.OnceFunc(func() { close(until) })
	t.Cleanup(openAll)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	returned := make(chan error, 1)
	client := newBreakingClient(t, api, breaking{
		fails: func(int, any) bool { return false }, ctx: ctx, returned: returned,
	})
	stream, err := client.WatchCardsDue(t.Context(), connect.NewRequest(&v1.WatchCardsDueRequest{}))
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
