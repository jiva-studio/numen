package flashcards

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// TestAStreamWhoseClientWentAwayEnds. The window roots every request at the
// process, so a handler is never told the page it was opened from is gone. A
// write that fails is the one report there is, so each stream makes one.
func TestAStreamWhoseClientWentAwayEnds(t *testing.T) {
	streams := map[string]func(context.Context, connect.HTTPClient, string) error{
		"moving": func(ctx context.Context, http connect.HTTPClient, at string) error {
			_, err := numenv1connect.NewFlashcardsServiceClient(http, at).
				WatchReloads(ctx, connect.NewRequest(&v1.WatchReloadsRequest{}))
			return err
		},
		"tasks": func(ctx context.Context, http connect.HTTPClient, at string) error {
			_, err := numenv1connect.NewWindowServiceClient(http, at).
				WatchTasks(ctx, connect.NewRequest(&v1.WatchTasksRequest{Window: wire.Review}))
			return err
		},
		"quitting": func(ctx context.Context, http connect.HTTPClient, at string) error {
			_, err := numenv1connect.NewWindowServiceClient(http, at).
				WatchQuit(ctx, connect.NewRequest(&v1.WatchQuitRequest{Window: wire.Review}))
			return err
		},
		"ask": func(ctx context.Context, http connect.HTTPClient, at string) error {
			_, err := numenv1connect.NewAgentServiceClient(http, at).
				AskAgent(ctx, connect.NewRequest(&v1.AskAgentRequest{Asked: "what is this card"}))
			return err
		},
	}

	for name, open := range streams {
		t.Run(name, func(t *testing.T) {
			api := &API{
				Registry: registry{},
				Now:      time.Now,
				Window:   NewWindow(task.New()),
			}
			api.Answers(testsupport.SilentAgent{})

			entered, returned := make(chan struct{}, 1), make(chan struct{}, 1)
			mux := http.NewServeMux()
			cards, decks := numenv1connect.NewFlashcardsServiceHandler(api)
			mux.Handle(cards, testsupport.NewRootedHandler(decks, entered, returned))
			agent, agents := numenv1connect.NewAgentServiceHandler(api)
			mux.Handle(agent, testsupport.NewRootedHandler(agents, entered, returned))
			drawn, itself := numenv1connect.NewWindowServiceHandler(api.Window)
			mux.Handle(drawn, testsupport.NewRootedHandler(itself, entered, returned))
			server := httptest.NewServer(mux)
			t.Cleanup(server.Close)

			if err := open(t.Context(), server.Client(), server.URL); err != nil {
				t.Fatal(err)
			}

			select {
			case <-entered:
			case <-time.After(testsupport.Patience):
				t.Fatal("the stream never reached the handler")
			}

			// The page is gone, and nothing cancelled the context the handler
			// holds.
			server.CloseClientConnections()

			select {
			case <-returned:
			case <-time.After(testsupport.Patience):
				t.Fatal("the client went away and the stream is still standing")
			}
		})
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

	held := make(chan struct{})
	defer close(held)
	api.CardsDue.CardFaces.Readers = waiting{until: held}

	stream, err := newFlashcardsClient(t, api).WatchCardsDue(
		t.Context(), connect.NewRequest(&v1.WatchCardsDueRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	if !stream.Receive() {
		t.Fatalf("the front door never opened: %v", stream.Err())
	}
	if len(stream.Msg().GetVaults()) != 1 {
		t.Fatalf("the front door opened on %+v", stream.Msg().GetVaults())
	}

	// Nothing has been counted and nothing will be, so what arrives next is the
	// stream reaching for its client.
	if !stream.Receive() {
		t.Fatalf("the stream said nothing while the count ran: %v", stream.Err())
	}
	said := stream.Msg()
	if said.GetCounted() != nil {
		t.Fatalf("a count arrived for a vault nothing could read: %+v", said.GetCounted())
	}
	if len(said.GetVaults()) != 1 {
		t.Errorf("the stream said again and listed %+v", said.GetVaults())
	}
}

// waiting is a vault whose card faces never come back, which is what counting a
// vault of many thousands looks like from the stream's side.
type waiting struct{ until <-chan struct{} }

func (w waiting) Open(domain.Vault) (port.VaultReader, error) {
	<-w.until
	return nil, errNothingRead
}

var errNothingRead = errors.New("the count was let go")
