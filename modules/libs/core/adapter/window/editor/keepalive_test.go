package editor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/task"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
)

// TestAStreamWhoseClientWentAwayEnds. The window roots every request at the
// process, so a handler is never told the page it was opened from is gone: the
// editor reloads on a vault swap and leaves one of each stream behind. A write
// that fails is the one report there is, so each stream makes one.
func TestAStreamWhoseClientWentAwayEnds(t *testing.T) {
	streams := map[string]func(context.Context, connect.HTTPClient, string) error{
		"changes": func(ctx context.Context, http connect.HTTPClient, at string) error {
			_, err := numenv1connect.NewVaultServiceClient(http, at).
				WatchVaultChanges(ctx, connect.NewRequest(&v1.WatchVaultChangesRequest{}))
			return err
		},
		"editing": func(ctx context.Context, http connect.HTTPClient, at string) error {
			_, err := numenv1connect.NewNoteServiceClient(http, at).
				WatchEdits(ctx, connect.NewRequest(&v1.WatchEditsRequest{}))
			return err
		},
		"focus": func(ctx context.Context, http connect.HTTPClient, at string) error {
			_, err := numenv1connect.NewWorkspaceServiceClient(http, at).
				WatchFocus(ctx, connect.NewRequest(&v1.WatchFocusRequest{}))
			return err
		},
		"quitting": func(ctx context.Context, http connect.HTTPClient, at string) error {
			_, err := numenv1connect.NewWindowServiceClient(http, at).
				WatchQuit(ctx, connect.NewRequest(&v1.WatchQuitRequest{Window: wire.Editor}))
			return err
		},
		"tasks": func(ctx context.Context, http connect.HTTPClient, at string) error {
			_, err := numenv1connect.NewWindowServiceClient(http, at).
				WatchTasks(ctx, connect.NewRequest(&v1.WatchTasksRequest{Window: wire.Editor}))
			return err
		},
		"ask": func(ctx context.Context, http connect.HTTPClient, at string) error {
			_, err := numenv1connect.NewAgentServiceClient(http, at).
				AskAgent(ctx, connect.NewRequest(&v1.AskAgentRequest{Asked: "what is here"}))
			return err
		},
	}

	for name, open := range streams {
		t.Run(name, func(t *testing.T) {
			api := &API{Window: &wire.Window{Named: wire.Editor, Tasking: task.New()}}
			api.Answers(testsupport.SilentAgent{})

			entered, returned := make(chan struct{}, 1), make(chan struct{}, 1)
			mux := http.NewServeMux()
			vault, vaults := numenv1connect.NewVaultServiceHandler(api)
			mux.Handle(vault, testsupport.Rooted(vaults, entered, returned))
			standing, workspace := numenv1connect.NewWorkspaceServiceHandler(api)
			mux.Handle(standing, testsupport.Rooted(workspace, entered, returned))
			filed, notes := numenv1connect.NewNoteServiceHandler(api)
			mux.Handle(filed, testsupport.Rooted(notes, entered, returned))
			agent, agents := numenv1connect.NewAgentServiceHandler(api)
			mux.Handle(agent, testsupport.Rooted(agents, entered, returned))
			drawn, itself := numenv1connect.NewWindowServiceHandler(api.Window)
			mux.Handle(drawn, testsupport.Rooted(itself, entered, returned))
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

			// The page is gone: the editor reloaded, and nothing cancelled the
			// context the handler holds.
			server.CloseClientConnections()

			select {
			case <-returned:
			case <-time.After(testsupport.Patience):
				t.Fatal("the client went away and the stream is still standing")
			}
		})
	}
}
