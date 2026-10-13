package editor

import (
	"context"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
)

// vaultHandler answers what the vault in the window is doing: whether it has
// been scanned, and what moved in it.
type vaultHandler struct{ state *State }

func (h *vaultHandler) GetVaultState(
	ctx context.Context, _ *connect.Request[v1.GetVaultStateRequest],
) (*connect.Response[v1.GetVaultStateResponse], error) {
	showing := h.state.GetCurrentVault()
	out := &v1.GetVaultStateResponse{
		Id:   string(showing.ID),
		Name: showing.Name,
		Path: showing.Path,
		Scan: &v1.Scan{
			IsReady:    h.state.Ready.Load(),
			Error:      h.state.Error.Why(),
			WatchError: h.state.WatchError.Why(),
		},
		Coverage: &v1.IndexCoverage{IsEmbedding: text(&h.state.Indexing.Model) != ""},
	}
	// A count that cannot be taken leaves the pair at nothing, and the rest of
	// the state is answered as it stands. A window standing on nothing holds no
	// chunks and counts none.
	if h.state.Indexing.Progress != nil && showing.ID != "" {
		if held, embedded, err := h.state.Indexing.Progress.GetProgress(ctx, showing.ID, text(&h.state.Indexing.Recipe)); err == nil {
			out.Coverage.ChunkCount, out.Coverage.EmbeddedCount = held, embedded
		}
	}
	return connect.NewResponse(out), nil
}

// WatchVaultChanges reports what moved, for as long as the caller listens.
func (h *vaultHandler) WatchVaultChanges(
	ctx context.Context,
	_ *connect.Request[v1.WatchVaultChangesRequest],
	out *connect.ServerStream[v1.WatchVaultChangesResponse],
) error {
	line, done := h.state.ChangeEvents.subscribe()
	defer done()

	// Named as following before anything has changed. A stream that says
	// nothing until the vault moves is indistinguishable from one that never
	// opened, and a caller waiting on its first message waits for an edit.
	if err := out.Send(&v1.WatchVaultChangesResponse{}); err != nil {
		return err
	}

	repeat := time.NewTicker(wire.Again)
	defer repeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-repeat.C:
			if err := out.Send(&v1.WatchVaultChangesResponse{}); err != nil {
				return err
			}
		case what, open := <-line:
			if !open {
				return nil
			}
			if err := out.Send(newChangeResponse(what)); err != nil {
				return err
			}
		}
	}
}

// newChangeResponse is the wire message that reports one change.
func newChangeResponse(what change) *v1.WatchVaultChangesResponse {
	renamed := make([]*v1.Move, 0, len(what.renamed))
	for _, went := range what.renamed {
		renamed = append(renamed, &v1.Move{From: went.From, To: went.To})
	}
	return &v1.WatchVaultChangesResponse{Paths: what.paths, Reload: what.shouldReload, Renamed: renamed}
}
