package webui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// TestAStreamWhoseClientWentAwayEnds. The window roots every request at the
// process, so a handler is never told the page it was opened from is gone: the
// editor reloads on a vault swap and leaves one of each stream behind. A write
// that fails is the one report there is, so each stream makes one.
func TestAStreamWhoseClientWentAwayEnds(t *testing.T) {
	streams := map[string]func(context.Context, connect.HTTPClient, string) error{
		"changes": func(ctx context.Context, http connect.HTTPClient, at string) error {
			_, err := numenv1connect.NewVaultServiceClient(http, at).
				Changes(ctx, connect.NewRequest(&v1.ChangesRequest{}))
			return err
		},
		"editing": func(ctx context.Context, http connect.HTTPClient, at string) error {
			_, err := numenv1connect.NewVaultServiceClient(http, at).
				Editing(ctx, connect.NewRequest(&v1.EditingRequest{}))
			return err
		},
		"focus": func(ctx context.Context, http connect.HTTPClient, at string) error {
			_, err := numenv1connect.NewVaultServiceClient(http, at).
				Focus(ctx, connect.NewRequest(&v1.FocusRequest{}))
			return err
		},
		"quitting": func(ctx context.Context, http connect.HTTPClient, at string) error {
			_, err := numenv1connect.NewVaultServiceClient(http, at).
				Quitting(ctx, connect.NewRequest(&v1.QuittingRequest{}))
			return err
		},
		"tasks": func(ctx context.Context, http connect.HTTPClient, at string) error {
			_, err := numenv1connect.NewVaultServiceClient(http, at).
				Tasks(ctx, connect.NewRequest(&v1.TasksRequest{}))
			return err
		},
		"ask": func(ctx context.Context, http connect.HTTPClient, at string) error {
			_, err := numenv1connect.NewAgentServiceClient(http, at).
				Ask(ctx, connect.NewRequest(&v1.AskRequest{Asked: "what is here"}))
			return err
		},
	}

	for name, open := range streams {
		t.Run(name, func(t *testing.T) {
			api := &API{Tasking: task.New()}
			api.Answers(silentAgent{})

			entered, returned := make(chan struct{}, 1), make(chan struct{}, 1)
			mux := http.NewServeMux()
			vault, vaults := numenv1connect.NewVaultServiceHandler(api)
			mux.Handle(vault, rooted(vaults, entered, returned))
			agent, agents := numenv1connect.NewAgentServiceHandler(api)
			mux.Handle(agent, rooted(agents, entered, returned))
			server := httptest.NewServer(mux)
			t.Cleanup(server.Close)

			if err := open(t.Context(), server.Client(), server.URL); err != nil {
				t.Fatal(err)
			}

			select {
			case <-entered:
			case <-time.After(10 * time.Second):
				t.Fatal("the stream never reached the handler")
			}

			// The page is gone: the editor reloaded, and nothing cancelled the
			// context the handler holds.
			server.CloseClientConnections()

			select {
			case <-returned:
			case <-time.After(10 * time.Second):
				t.Fatal("the client went away and the stream is still standing")
			}
		})
	}
}

// rooted serves the handler the way the window does: the request context is the
// process's own, and ends when the application ends and at no other moment.
func rooted(handler http.Handler, entered, returned chan<- struct{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entered <- struct{}{}
		defer func() { returned <- struct{}{} }()
		handler.ServeHTTP(w, r.WithContext(context.Background()))
	})
}

// silentAgent takes a task and says nothing about it, which is what an agent
// waiting on a model looks like.
type silentAgent struct{}

func (silentAgent) Take(context.Context, port.Task) (port.Work, error) {
	return silentWork{steps: make(chan port.Step)}, nil
}

func (silentAgent) Finish(context.Context, string) error { return nil }

type silentWork struct{ steps chan port.Step }

func (w silentWork) Steps() <-chan port.Step { return w.steps }
func (w silentWork) Stop() error             { return nil }
