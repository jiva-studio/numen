package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/internal/vaulttest"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/ports"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

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
	path, handler := numenv1connect.NewFlashcardsServiceHandler(api)
	mux.Handle(path, rooted(handler, entered, returned))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	_, err := numenv1connect.NewFlashcardsServiceClient(server.Client(), server.URL).
		WatchReloads(t.Context(), connect.NewRequest(&v1.WatchReloadsRequest{}))
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
	api.scenarios().flash.CardsDue = api.scenarios().flash.CardsDue.WithCardFaces(api.scenarios().flash.CardsDue.CardFaces().WithReaders(waiting{until: target}))

	stream, err := newFlashcardsClient(t, api).WatchCardsDue(
		t.Context(), connect.NewRequest(&v1.WatchCardsDueRequest{}))
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

// waiting is a vault whose card faces never come back, which is what counting a
// vault of many thousands looks like from the stream's side.
type waiting struct{ until <-chan struct{} }

func (w waiting) Open(domain.Vault) (ports.VaultReader, error) {
	<-w.until
	return nil, errNothingRead
}

var errNothingRead = errors.New("the count was let go")
