package vault

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// recordingArranger records the notes it is asked to move.
type recordingArranger struct {
	Arranger
	moved []string
}

func (a *recordingArranger) MoveNote(_ context.Context, from, to string) (domain.MoveResult, error) {
	a.moved = append(a.moved, from)
	return domain.MoveResult{From: from, To: to, IsMoved: true}, nil
}

type anyCaller struct{}

func (anyCaller) Authorize(string, string) (string, error) { return "md.numen.a", nil }

type shownAccess struct{ access *Access }

func (s shownAccess) GetPluginVault() *Access { return s.access }

func (shownAccess) GetPluginVaultList() List { return nil }

func TestMoveCanceledBeforeFirstNoteReturnsAllNotAttempted(t *testing.T) {
	arranger := &recordingArranger{}
	handler := &arrangeCalls{calls: &calls{auth: anyCaller{}, source: shownAccess{access: &Access{Arranger: arranger}}}}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	paths := []string{"Danube.md", "Rivers/Rhine.md", "Deltas.md"}

	res, err := handler.MoveNotes(ctx, connect.NewRequest(&v1.HostVaultServiceMoveNotesRequest{Paths: paths, Folder: "Archive"}))

	if err != nil {
		t.Fatalf("the canceled move failed as a whole: %v", err)
	}
	moves := res.Msg.GetMoves()
	if res.Msg.GetError() != 0 || len(moves) != len(paths) {
		t.Fatalf("the move returned error %v with %d entries, want no call error and %d entries", res.Msg.GetError(), len(moves), len(paths))
	}
	for i, entry := range moves {
		if entry.GetMove().GetFrom() != paths[i] || entry.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_ATTEMPTED {
			t.Errorf("entry %d is %v, want %s as not attempted", i, entry, paths[i])
		}
	}
	if len(arranger.moved) != 0 {
		t.Errorf("a canceled move reached the arranger for %v", arranger.moved)
	}
}
