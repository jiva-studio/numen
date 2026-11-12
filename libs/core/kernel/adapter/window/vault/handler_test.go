package vault

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/broadcast"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// showing is a window with a vault, the state it reports, and the changes it publishes.
type showing struct {
	state   window.VaultState
	changes broadcast.Broadcaster[window.Change]
}

func (s *showing) GetVaultState(context.Context) (window.VaultState, error) { return s.state, nil }

func (s *showing) SubscribeChanges() (<-chan window.Change, func()) { return s.changes.Subscribe() }

func serve(t *testing.T, current *showing) numenv1connect.VaultServiceClient {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewVaultServiceHandler(NewHandler(current, current)))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return numenv1connect.NewVaultServiceClient(server.Client(), server.URL)
}

func TestTheVaultStateAnswersWhatTheWindowKnows(t *testing.T) {
	client := serve(t, &showing{state: window.VaultState{
		Vault:         domain.Vault{ID: "v-heron", Name: "Herons", Path: "/vaults/herons"},
		IsReady:       true,
		Error:         "the index is locked",
		WatchError:    "the watch stopped",
		IsEmbedding:   true,
		ChunkCount:    40,
		EmbeddedCount: 12,
	}})

	out, err := client.GetVaultState(t.Context(), connect.NewRequest(&v1.GetVaultStateRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	got := out.Msg
	if got.GetId() != "v-heron" || got.GetName() != "Herons" || got.GetPath() != "/vaults/herons" {
		t.Errorf("the vault is %v", got)
	}
	scan := got.GetScan()
	if !scan.GetIsReady() || scan.GetError() != "the index is locked" || scan.GetWatchError() != "the watch stopped" {
		t.Errorf("the scan is %v", scan)
	}
	coverage := got.GetCoverage()
	if !coverage.GetIsEmbedding() || coverage.GetChunkCount() != 40 || coverage.GetEmbeddedCount() != 12 {
		t.Errorf("the coverage is %v", coverage)
	}
}

func TestAChangeOfTheVaultReachesAWatchingClient(t *testing.T) {
	current := &showing{changes: broadcast.Broadcaster[window.Change]{Capacity: 8}}
	client := serve(t, current)
	stream, err := client.WatchVaultChanges(t.Context(), connect.NewRequest(&v1.WatchVaultChangesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, stream)
	// The watch subscribes before it confirms, so what is published after the confirmation arrives.
	if !stream.Receive() {
		t.Fatalf("the watch confirmed nothing: %v", stream.Err())
	}
	current.changes.Publish(window.Change{
		Paths:        []string{"notes/Heron.md"},
		ShouldReload: true,
		Renamed:      []domain.Move{{From: "notes/Egret.md", To: "notes/Heron.md"}},
		VersionPaths: []string{"scans/page.png"},
	})

	var msg *v1.WatchVaultChangesResponse
	for msg == nil && stream.Receive() {
		if len(stream.Msg().GetPaths()) > 0 {
			msg = stream.Msg()
		}
	}
	if msg == nil {
		t.Fatalf("the change never arrived: %v", stream.Err())
	}
	if !slices.Equal(msg.GetPaths(), []string{"notes/Heron.md"}) || !msg.GetReload() ||
		!slices.Equal(msg.GetVersionPaths(), []string{"scans/page.png"}) {
		t.Errorf("the change is %v", msg)
	}
	if moves := msg.GetRenamed(); len(moves) != 1 || moves[0].GetFrom() != "notes/Egret.md" || moves[0].GetTo() != "notes/Heron.md" {
		t.Errorf("the moves are %v", moves)
	}
}
