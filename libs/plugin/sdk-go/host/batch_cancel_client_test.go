package host_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

// canceledArranging is a host whose batches were canceled after the first note: the first entry
// is done and the others are returned NOT_ATTEMPTED with the detail naming the path.
type canceledArranging struct {
	fakeArrangeHost
}

func (a *canceledArranging) MoveNotes(
	context.Context, *connect.Request[v1.HostVaultServiceMoveNotesRequest],
) (*connect.Response[v1.HostVaultServiceMoveNotesResponse], error) {
	return connect.NewResponse(&v1.HostVaultServiceMoveNotesResponse{Moves: []*v1.HostNoteMove{
		{Move: &v1.MoveResult{From: "Danube.md", To: "Archive/Danube.md", Repaired: []string{"Deltas.md"}}},
		{
			Move: &v1.MoveResult{From: "Rivers/Rhine.md"}, Error: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_ATTEMPTED,
			Detail: "the call was canceled before Rivers/Rhine.md was moved",
		},
	}}), nil
}

func (a *canceledArranging) RemoveNotes(
	context.Context, *connect.Request[v1.HostVaultServiceRemoveNotesRequest],
) (*connect.Response[v1.HostVaultServiceRemoveNotesResponse], error) {
	return connect.NewResponse(&v1.HostVaultServiceRemoveNotesResponse{Removals: []*v1.HostNoteRemoval{
		{Path: "Danube.md", TrashPath: ".trash/Danube.md", BrokenLinks: []string{"Deltas.md"}},
		{
			Path: "Rivers/Rhine.md", Error: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_ATTEMPTED,
			Detail: "the call was canceled before Rivers/Rhine.md was removed",
		},
	}}), nil
}

func canceledClient(t *testing.T) *host.Client {
	t.Helper()
	server := httptest.NewServer(handler(&canceledArranging{}))
	t.Cleanup(server.Close)
	return host.NewClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})
}

func TestClientReturnsMoveSkippedByCancelAsNotAttempted(t *testing.T) {
	moves, err := canceledClient(t).MoveNotes(t.Context(), []string{"Danube.md", "Rivers/Rhine.md"}, "Archive")
	if err != nil {
		t.Fatalf("the move failed as a whole: %v", err)
	}

	if len(moves) != 2 {
		t.Fatalf("the moves came back as %+v, want 2", moves)
	}
	if m := moves[0]; m.Err != nil || !m.Move.IsMoved || m.Move.To != "Archive/Danube.md" {
		t.Errorf("the move done before the cancel came back as %+v, want Danube.md moved to Archive/Danube.md", m)
	}
	const detail = "the call was canceled before Rivers/Rhine.md was moved"
	if m := moves[1]; !errors.Is(m.Err, host.ErrNotAttempted) || !strings.Contains(m.Err.Error(), detail) || m.Move.IsMoved {
		t.Errorf("the move after the cancel came back as %+v, want ErrNotAttempted containing %q", m, detail)
	}
}

func TestClientReturnsRemovalSkippedByCancelAsNotAttempted(t *testing.T) {
	removals, err := canceledClient(t).RemoveNotes(t.Context(), []string{"Danube.md", "Rivers/Rhine.md"})
	if err != nil {
		t.Fatalf("the removal failed as a whole: %v", err)
	}

	if len(removals) != 2 {
		t.Fatalf("the removals came back as %+v, want 2", removals)
	}
	if r := removals[0]; r.Err != nil || r.TrashPath != ".trash/Danube.md" {
		t.Errorf("the removal done before the cancel came back as %+v, want Danube.md in .trash", r)
	}
	const detail = "the call was canceled before Rivers/Rhine.md was removed"
	if r := removals[1]; r.Path != "Rivers/Rhine.md" || !errors.Is(r.Err, host.ErrNotAttempted) || !strings.Contains(r.Err.Error(), detail) {
		t.Errorf("the removal after the cancel came back as %+v, want ErrNotAttempted containing %q", r, detail)
	}
}
