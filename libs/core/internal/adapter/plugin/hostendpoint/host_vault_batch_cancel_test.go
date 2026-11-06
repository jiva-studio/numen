package hostendpoint

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// cancelMode is what the cancelling vault does on the call that cancels the batch.
type cancelMode int

const (
	// changeThenCancel changes the note, then cancels the call and answers success.
	changeThenCancel cancelMode = iota
	// cancelAndFail cancels the call and answers the error of the cancel, leaving the note as it was.
	cancelAndFail
	// changeThenCancelAndFail changes the note, then cancels the call and answers the error of the
	// cancel with the change it made.
	changeThenCancelAndFail
)

// cancellingVault is the vault of rivers that cancels the call it serves on its change number at
// (counted from one), as mode says.
type cancellingVault struct {
	*arrangingVault
	cancel context.CancelFunc
	at     int
	mode   cancelMode
	calls  int
}

func (c *cancellingVault) MoveNote(ctx context.Context, from, to string) (domain.MoveResult, error) {
	c.calls++
	if c.calls != c.at {
		return c.arrangingVault.MoveNote(ctx, from, to)
	}
	if c.mode == cancelAndFail {
		c.mu.Lock()
		c.moved = append(c.moved, domain.Move{From: from, To: to})
		c.mu.Unlock()
		c.cancel()
		return domain.MoveResult{From: from, To: to}, fmt.Errorf("move %s: %w", from, ctx.Err())
	}
	move, err := c.arrangingVault.MoveNote(ctx, from, to)
	c.cancel()
	if err == nil && c.mode == changeThenCancelAndFail {
		err = fmt.Errorf("repair the links to %s: %w", from, ctx.Err())
	}
	return move, err
}

func (c *cancellingVault) RemoveNote(ctx context.Context, target string) (string, []string, error) {
	c.calls++
	if c.calls != c.at {
		return c.arrangingVault.RemoveNote(ctx, target)
	}
	if c.mode == cancelAndFail {
		c.mu.Lock()
		c.removed = append(c.removed, target)
		c.mu.Unlock()
		c.cancel()
		return "", nil, fmt.Errorf("remove %s: %w", target, ctx.Err())
	}
	trashPath, broken, err := c.arrangingVault.RemoveNote(ctx, target)
	c.cancel()
	if err == nil && c.mode == changeThenCancelAndFail {
		err = fmt.Errorf("report the links to %s: %w", target, ctx.Err())
	}
	return trashPath, broken, err
}

// newCancellingEndpoint is the vault endpoint over the vault of rivers that cancels the context it
// returns on its change number at, with md.numen.a granted vault:write.
func newCancellingEndpoint(t *testing.T, at int, mode cancelMode) (*vaultEndpoint, *cancellingVault, context.Context) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	vault := &cancellingVault{arrangingVault: newArrangingVault(), cancel: cancel, at: at, mode: mode}
	hosted := newCallers(nil)
	hosted.vault = vault
	endpoint := newVaultEndpoint(&Endpoint{callers: listCallers{callers: hosted, granted: []string{pluginport.PermissionVaultWrite}}})
	return endpoint, vault, ctx
}

func authorizeAsA[T any](req *connect.Request[T]) *connect.Request[T] {
	req.Header().Set("Authorization", "Bearer token-a")
	return req
}

var cancelledBatch = []string{"Danube.md", "Rivers/Rhine.md", "Deltas.md"}

func requireNotAttemptedMove(t *testing.T, entry *v1.HostNoteMove, from string) {
	t.Helper()
	want := "the call was cancelled before " + from + " was moved"
	if entry.GetMove().GetFrom() != from || entry.GetMove().GetTo() != "" ||
		entry.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_ATTEMPTED || entry.GetDetail() != want {
		t.Errorf("the entry of %s is %v, want error NOT_ATTEMPTED with the detail %q", from, entry, want)
	}
}

func requireNotAttemptedRemoval(t *testing.T, entry *v1.HostNoteRemoval, target string) {
	t.Helper()
	want := "the call was cancelled before " + target + " was removed"
	if entry.GetPath() != target || entry.GetTrashPath() != "" ||
		entry.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_ATTEMPTED || entry.GetDetail() != want {
		t.Errorf("the entry of %s is %v, want error NOT_ATTEMPTED with the detail %q", target, entry, want)
	}
}

func requireMoved(t *testing.T, entry *v1.HostNoteMove, from, to string) {
	t.Helper()
	if entry.GetMove().GetFrom() != from || entry.GetMove().GetTo() != to || entry.GetError() != 0 || entry.GetDetail() != "" {
		t.Errorf("the entry of %s is %v, want it moved to %s with no error", from, entry, to)
	}
}

func requireRemoved(t *testing.T, entry *v1.HostNoteRemoval, target string) {
	t.Helper()
	if entry.GetPath() != target || entry.GetTrashPath() != ".trash/"+target || entry.GetError() != 0 || entry.GetDetail() != "" {
		t.Errorf("the entry of %s is %v, want it in .trash with no error", target, entry)
	}
}

func TestAMoveCancelledAfterTwoNotesAnswersTheTwoMovesAndTheRestAsNotAttempted(t *testing.T) {
	endpoint, vault, ctx := newCancellingEndpoint(t, 2, changeThenCancel)

	res, err := endpoint.MoveNotes(ctx, authorizeAsA(connect.NewRequest(&v1.HostVaultServiceMoveNotesRequest{
		Paths: cancelledBatch, Folder: "Archive",
	})))
	if err != nil {
		t.Fatalf("the cancelled move failed as a whole: %v", err)
	}

	moves := res.Msg.GetMoves()
	if res.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_UNSPECIFIED || len(moves) != 3 {
		t.Fatalf("the move answered error %v with %d entries, want no call error and 3 entries", res.Msg.GetError(), len(moves))
	}
	requireMoved(t, moves[0], "Danube.md", "Archive/Danube.md")
	requireMoved(t, moves[1], "Rivers/Rhine.md", "Archive/Rhine.md")
	requireNotAttemptedMove(t, moves[2], "Deltas.md")
	if _, found := vault.files["Deltas.md"]; !found {
		t.Error("Deltas.md was moved after the cancel")
	}
	if len(vault.moved) != 2 {
		t.Errorf("the vault was asked for the moves %v, want only the two before the cancel", vault.moved)
	}
}

func TestAMoveWhoseNoteFailsOnTheCancelAnswersThatNoteAndTheRestAsNotAttempted(t *testing.T) {
	endpoint, vault, ctx := newCancellingEndpoint(t, 2, cancelAndFail)

	res, err := endpoint.MoveNotes(ctx, authorizeAsA(connect.NewRequest(&v1.HostVaultServiceMoveNotesRequest{
		Paths: cancelledBatch, Folder: "Archive",
	})))
	if err != nil {
		t.Fatalf("the cancelled move failed as a whole: %v", err)
	}

	moves := res.Msg.GetMoves()
	if res.Msg.GetError() != 0 || len(moves) != 3 {
		t.Fatalf("the move answered error %v with %d entries, want no call error and 3 entries", res.Msg.GetError(), len(moves))
	}
	requireMoved(t, moves[0], "Danube.md", "Archive/Danube.md")
	requireNotAttemptedMove(t, moves[1], "Rivers/Rhine.md")
	requireNotAttemptedMove(t, moves[2], "Deltas.md")
	for _, still := range []string{"Rivers/Rhine.md", "Deltas.md"} {
		if _, found := vault.files[still]; !found {
			t.Errorf("%s is not where it was", still)
		}
	}
}

func TestAMoveWhoseNoteMovedBeforeTheCancelAnswersThatNoteAsMoved(t *testing.T) {
	endpoint, _, ctx := newCancellingEndpoint(t, 2, changeThenCancelAndFail)

	res, err := endpoint.MoveNotes(ctx, authorizeAsA(connect.NewRequest(&v1.HostVaultServiceMoveNotesRequest{
		Paths: cancelledBatch, Folder: "Archive",
	})))
	if err != nil {
		t.Fatalf("the cancelled move failed as a whole: %v", err)
	}

	moves := res.Msg.GetMoves()
	if res.Msg.GetError() != 0 || len(moves) != 3 {
		t.Fatalf("the move answered error %v with %d entries, want no call error and 3 entries", res.Msg.GetError(), len(moves))
	}
	requireMoved(t, moves[0], "Danube.md", "Archive/Danube.md")
	requireMoved(t, moves[1], "Rivers/Rhine.md", "Archive/Rhine.md")
	requireNotAttemptedMove(t, moves[2], "Deltas.md")
}

func TestAMoveCancelledBeforeTheFirstNoteAnswersEveryPathAsNotAttempted(t *testing.T) {
	endpoint, vault, ctx := newCancellingEndpoint(t, 0, changeThenCancel)
	vault.cancel()

	res, err := endpoint.MoveNotes(ctx, authorizeAsA(connect.NewRequest(&v1.HostVaultServiceMoveNotesRequest{
		Paths: cancelledBatch, Folder: "Archive",
	})))
	if err != nil {
		t.Fatalf("the cancelled move failed as a whole: %v", err)
	}

	moves := res.Msg.GetMoves()
	if res.Msg.GetError() != 0 || len(moves) != 3 {
		t.Fatalf("the move answered error %v with %d entries, want no call error and 3 entries", res.Msg.GetError(), len(moves))
	}
	for i, from := range cancelledBatch {
		requireNotAttemptedMove(t, moves[i], from)
	}
	if len(vault.moved) != 0 {
		t.Errorf("the vault was asked for the moves %v after the cancel", vault.moved)
	}
}

func TestARemovalCancelledAfterOneNoteAnswersTheRemovalAndTheRestAsNotAttempted(t *testing.T) {
	endpoint, vault, ctx := newCancellingEndpoint(t, 1, changeThenCancel)

	res, err := endpoint.RemoveNotes(ctx, authorizeAsA(connect.NewRequest(&v1.HostVaultServiceRemoveNotesRequest{
		Paths: cancelledBatch,
	})))
	if err != nil {
		t.Fatalf("the cancelled removal failed as a whole: %v", err)
	}

	removals := res.Msg.GetRemovals()
	if res.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_UNSPECIFIED || len(removals) != 3 {
		t.Fatalf("the removal answered error %v with %d entries, want no call error and 3 entries", res.Msg.GetError(), len(removals))
	}
	requireRemoved(t, removals[0], "Danube.md")
	requireNotAttemptedRemoval(t, removals[1], "Rivers/Rhine.md")
	requireNotAttemptedRemoval(t, removals[2], "Deltas.md")
	if !slices.Equal(vault.removed, []string{"Danube.md"}) {
		t.Errorf("the vault was asked to remove %v, want only Danube.md", vault.removed)
	}
}

func TestARemovalWhoseNoteFailsOnTheCancelAnswersThatNoteAndTheRestAsNotAttempted(t *testing.T) {
	endpoint, vault, ctx := newCancellingEndpoint(t, 2, cancelAndFail)

	res, err := endpoint.RemoveNotes(ctx, authorizeAsA(connect.NewRequest(&v1.HostVaultServiceRemoveNotesRequest{
		Paths: cancelledBatch,
	})))
	if err != nil {
		t.Fatalf("the cancelled removal failed as a whole: %v", err)
	}

	removals := res.Msg.GetRemovals()
	if res.Msg.GetError() != 0 || len(removals) != 3 {
		t.Fatalf("the removal answered error %v with %d entries, want no call error and 3 entries", res.Msg.GetError(), len(removals))
	}
	requireRemoved(t, removals[0], "Danube.md")
	requireNotAttemptedRemoval(t, removals[1], "Rivers/Rhine.md")
	requireNotAttemptedRemoval(t, removals[2], "Deltas.md")
	for _, still := range []string{"Rivers/Rhine.md", "Deltas.md"} {
		if _, found := vault.files[still]; !found {
			t.Errorf("%s was removed", still)
		}
	}
}

func TestARemovalWhoseNoteWentToTheTrashBeforeTheCancelAnswersThatNoteAsRemoved(t *testing.T) {
	endpoint, _, ctx := newCancellingEndpoint(t, 2, changeThenCancelAndFail)

	res, err := endpoint.RemoveNotes(ctx, authorizeAsA(connect.NewRequest(&v1.HostVaultServiceRemoveNotesRequest{
		Paths: cancelledBatch,
	})))
	if err != nil {
		t.Fatalf("the cancelled removal failed as a whole: %v", err)
	}

	removals := res.Msg.GetRemovals()
	if res.Msg.GetError() != 0 || len(removals) != 3 {
		t.Fatalf("the removal answered error %v with %d entries, want no call error and 3 entries", res.Msg.GetError(), len(removals))
	}
	requireRemoved(t, removals[0], "Danube.md")
	requireRemoved(t, removals[1], "Rivers/Rhine.md")
	requireNotAttemptedRemoval(t, removals[2], "Deltas.md")
}
