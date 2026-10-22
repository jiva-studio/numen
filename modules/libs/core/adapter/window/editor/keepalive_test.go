package editor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
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
				AskAgent(ctx, connect.NewRequest(&v1.AskAgentRequest{Prompt: "what is here"}))
			return err
		},
	}

	for name, open := range streams {
		t.Run(name, func(t *testing.T) {
			state := &State{Window: &wire.Window{Named: wire.Editor, Tasking: run.NewTasks()}}
			services := createServices(state)
			state.SetAgent(testsupport.SilentAgent{})

			entered, returned := make(chan struct{}, 1), make(chan struct{}, 1)
			mux := http.NewServeMux()
			vault, vaults := numenv1connect.NewVaultServiceHandler(services.Vault)
			mux.Handle(vault, testsupport.NewRootedHandler(vaults, entered, returned))
			standing, workspace := numenv1connect.NewWorkspaceServiceHandler(services.Workspace)
			mux.Handle(standing, testsupport.NewRootedHandler(workspace, entered, returned))
			filed, notes := numenv1connect.NewNoteServiceHandler(services.Note)
			mux.Handle(filed, testsupport.NewRootedHandler(notes, entered, returned))
			agent, agents := numenv1connect.NewAgentServiceHandler(services.Agent)
			mux.Handle(agent, testsupport.NewRootedHandler(agents, entered, returned))
			rendered, itself := numenv1connect.NewWindowServiceHandler(state.Window)
			mux.Handle(rendered, testsupport.NewRootedHandler(itself, entered, returned))
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
