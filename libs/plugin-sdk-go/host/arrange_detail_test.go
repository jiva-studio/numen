package host_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

const diskFull = "write Broken.md: the disk is full"

// brokenArranging is a host whose change of Broken.md fails with an error that has no code of its
// own, and whose other refusals carry the core's text.
type brokenArranging struct {
	arranging
}

func (a *brokenArranging) RenameNote(
	context.Context, *connect.Request[v1.HostVaultServiceRenameNoteRequest],
) (*connect.Response[v1.HostVaultServiceRenameNoteResponse], error) {
	return connect.NewResponse(&v1.HostVaultServiceRenameNoteResponse{
		Error: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INTERNAL, Path: "Broken.md", Detail: diskFull,
	}), nil
}

func (a *brokenArranging) MoveNotes(
	context.Context, *connect.Request[v1.HostVaultServiceMoveNotesRequest],
) (*connect.Response[v1.HostVaultServiceMoveNotesResponse], error) {
	return connect.NewResponse(&v1.HostVaultServiceMoveNotesResponse{Moves: []*v1.HostNoteMove{
		{Move: &v1.MoveResult{From: "Danube.md", To: "Archive/Danube.md"}},
		{Move: &v1.MoveResult{From: "Broken.md"}, Error: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INTERNAL, Detail: diskFull},
		{Move: &v1.MoveResult{From: "Nowhere.md"}, Error: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_FOUND, Detail: "move Nowhere.md: no such note"},
	}}), nil
}

func (a *brokenArranging) RemoveNotes(
	context.Context, *connect.Request[v1.HostVaultServiceRemoveNotesRequest],
) (*connect.Response[v1.HostVaultServiceRemoveNotesResponse], error) {
	return connect.NewResponse(&v1.HostVaultServiceRemoveNotesResponse{Removals: []*v1.HostNoteRemoval{
		{Path: "Danube.md", TrashPath: ".trash/Danube.md"},
		{Path: "Broken.md", Error: v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_INTERNAL, Detail: diskFull},
	}}), nil
}

func brokenClient(t *testing.T) *host.Client {
	t.Helper()
	server := httptest.NewServer(handler(&brokenArranging{}))
	t.Cleanup(server.Close)
	return host.NewClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})
}

func TestTheClientKeepsTheDetailOfARenameThatFailedWithNoCode(t *testing.T) {
	_, err := brokenClient(t).RenameNote(t.Context(), "Broken.md", "Mended")

	if err == nil || !strings.Contains(err.Error(), diskFull) {
		t.Errorf("the rename returned %v, want an error carrying %q", err, diskFull)
	}
}

func TestTheClientKeepsTheDetailOfEachMoveEntryInItsError(t *testing.T) {
	moves, err := brokenClient(t).MoveNotes(t.Context(), []string{"Danube.md", "Broken.md", "Nowhere.md"}, "Archive")
	if err != nil {
		t.Fatalf("the move failed as a whole: %v", err)
	}

	if len(moves) != 3 || moves[0].Err != nil || !moves[0].Move.IsMoved {
		t.Fatalf("the moves came back as %+v", moves)
	}
	if m := moves[1]; m.Err == nil || !strings.Contains(m.Err.Error(), diskFull) {
		t.Errorf("the move of Broken.md came back as %+v, want an error carrying %q", m, diskFull)
	}
	if m := moves[2]; !errors.Is(m.Err, host.ErrNotExist) || !strings.Contains(m.Err.Error(), "move Nowhere.md: no such note") {
		t.Errorf("the move of Nowhere.md came back as %+v, want ErrNotExist carrying the core's text", m)
	}
}

func TestTheClientKeepsTheDetailOfARemovalEntryInItsError(t *testing.T) {
	removals, err := brokenClient(t).RemoveNotes(t.Context(), []string{"Danube.md", "Broken.md"})
	if err != nil {
		t.Fatalf("the removal failed as a whole: %v", err)
	}

	if len(removals) != 2 || removals[0].Err != nil {
		t.Fatalf("the removals came back as %+v", removals)
	}
	if r := removals[1]; r.Err == nil || !strings.Contains(r.Err.Error(), diskFull) {
		t.Errorf("the removal of Broken.md came back as %+v, want an error carrying %q", r, diskFull)
	}
}
