package hostendpoint

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/usecase/note"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// cancelMode is what the cancelling vault does on the call that cancels the batch.
type cancelMode int

const (
	// changeThenCancel changes the note, then cancels the call and returns success.
	changeThenCancel cancelMode = iota
	// cancelAndFail cancels the call and returns the error of the cancel, leaving the note as it was.
	cancelAndFail
	// changeThenCancelAndFail changes the note, then cancels the call and returns the error of the
	// cancel with the change it made.
	changeThenCancelAndFail
)

// cancellingVault is the vault of rivers that cancels the call it serves on its change number cancelIndex
// (counted from one), as mode sets.
type cancellingVault struct {
	*fakeNoteVault
	cancel      context.CancelFunc
	cancelIndex int
	mode        cancelMode
	calls       int
}

func (c *cancellingVault) MoveNote(ctx context.Context, from, to string) (domain.MoveResult, error) {
	c.calls++
	if c.calls != c.cancelIndex {
		return c.fakeNoteVault.MoveNote(ctx, from, to)
	}
	if c.mode == cancelAndFail {
		c.mu.Lock()
		c.moved = append(c.moved, domain.Move{From: from, To: to})
		c.mu.Unlock()
		c.cancel()
		return domain.MoveResult{From: from, To: to}, fmt.Errorf("move %s: %w", from, ctx.Err())
	}
	move, err := c.fakeNoteVault.MoveNote(ctx, from, to)
	c.cancel()
	if err == nil && c.mode == changeThenCancelAndFail {
		err = fmt.Errorf("repair the links to %s: %w", from, ctx.Err())
	}
	return move, err
}

func (c *cancellingVault) RemoveNote(ctx context.Context, target string) (string, []string, error) {
	c.calls++
	if c.calls != c.cancelIndex {
		return c.fakeNoteVault.RemoveNote(ctx, target)
	}
	if c.mode == cancelAndFail {
		c.mu.Lock()
		c.removed = append(c.removed, target)
		c.mu.Unlock()
		c.cancel()
		return "", nil, fmt.Errorf("remove %s: %w", target, ctx.Err())
	}
	trashPath, broken, err := c.fakeNoteVault.RemoveNote(ctx, target)
	c.cancel()
	if err == nil && c.mode == changeThenCancelAndFail {
		err = fmt.Errorf("report the links to %s: %w", target, ctx.Err())
	}
	return trashPath, broken, err
}

func newCancellingEndpoint(t *testing.T, cancelIndex int, mode cancelMode) (*vaultEndpoint, *cancellingVault, context.Context) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	vault := &cancellingVault{fakeNoteVault: newFakeNoteVault(), cancel: cancel, cancelIndex: cancelIndex, mode: mode}
	hostCallers := newCallers(nil)
	hostCallers.vault = vault
	endpoint := newVaultEndpoint(&Endpoint{callers: vaultListCallers{callers: hostCallers, granted: []string{pluginport.PermissionVaultWrite}}})
	return endpoint, vault, ctx
}

func authorizeAsA[T any](req *connect.Request[T]) *connect.Request[T] {
	req.Header().Set("Authorization", "Bearer token-a")
	return req
}

func getCancelledBatch() []string { return []string{"Danube.md", "Rivers/Rhine.md", "Deltas.md"} }

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

func TestMoveCancelledAfterTwoNotesReturnsRestAsNotAttempted(t *testing.T) {
	endpoint, vault, ctx := newCancellingEndpoint(t, 2, changeThenCancel)

	res, err := endpoint.MoveNotes(ctx, authorizeAsA(connect.NewRequest(&v1.HostVaultServiceMoveNotesRequest{
		Paths: getCancelledBatch(), Folder: "Archive",
	})))
	if err != nil {
		t.Fatalf("the cancelled move failed as a whole: %v", err)
	}

	moves := res.Msg.GetMoves()
	if res.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_UNSPECIFIED || len(moves) != 3 {
		t.Fatalf("the move returned error %v with %d entries, want no call error and 3 entries", res.Msg.GetError(), len(moves))
	}
	requireMoved(t, moves[0], "Danube.md", "Archive/Danube.md")
	requireMoved(t, moves[1], "Rivers/Rhine.md", "Archive/Rhine.md")
	requireNotAttemptedMove(t, moves[2], "Deltas.md")
	if _, found := vault.files["Deltas.md"]; !found {
		t.Error("Deltas.md was moved after the cancel")
	}
	if len(vault.moved) != 2 {
		t.Errorf("the vault received a request for the moves %v, want only the two before the cancel", vault.moved)
	}
}

func TestMoveFailingOnCancelReturnsNoteErrorAndRestNotAttempted(t *testing.T) {
	endpoint, vault, ctx := newCancellingEndpoint(t, 2, cancelAndFail)

	res, err := endpoint.MoveNotes(ctx, authorizeAsA(connect.NewRequest(&v1.HostVaultServiceMoveNotesRequest{
		Paths: getCancelledBatch(), Folder: "Archive",
	})))
	if err != nil {
		t.Fatalf("the cancelled move failed as a whole: %v", err)
	}

	moves := res.Msg.GetMoves()
	if res.Msg.GetError() != 0 || len(moves) != 3 {
		t.Fatalf("the move returned error %v with %d entries, want no call error and 3 entries", res.Msg.GetError(), len(moves))
	}
	requireMoved(t, moves[0], "Danube.md", "Archive/Danube.md")
	requireNotAttemptedMove(t, moves[1], "Rivers/Rhine.md")
	requireNotAttemptedMove(t, moves[2], "Deltas.md")
	for _, unchanged := range []string{"Rivers/Rhine.md", "Deltas.md"} {
		if _, found := vault.files[unchanged]; !found {
			t.Errorf("%s is not where it was", unchanged)
		}
	}
}

func TestMoveBeforeCancelReturnsThatNoteAsMoved(t *testing.T) {
	endpoint, _, ctx := newCancellingEndpoint(t, 2, changeThenCancelAndFail)

	res, err := endpoint.MoveNotes(ctx, authorizeAsA(connect.NewRequest(&v1.HostVaultServiceMoveNotesRequest{
		Paths: getCancelledBatch(), Folder: "Archive",
	})))
	if err != nil {
		t.Fatalf("the cancelled move failed as a whole: %v", err)
	}

	moves := res.Msg.GetMoves()
	if res.Msg.GetError() != 0 || len(moves) != 3 {
		t.Fatalf("the move returned error %v with %d entries, want no call error and 3 entries", res.Msg.GetError(), len(moves))
	}
	requireMoved(t, moves[0], "Danube.md", "Archive/Danube.md")
	requireMoved(t, moves[1], "Rivers/Rhine.md", "Archive/Rhine.md")
	requireNotAttemptedMove(t, moves[2], "Deltas.md")
}

func TestMoveCancelledBeforeFirstNoteReturnsAllNotAttempted(t *testing.T) {
	endpoint, vault, ctx := newCancellingEndpoint(t, 0, changeThenCancel)
	vault.cancel()

	res, err := endpoint.MoveNotes(ctx, authorizeAsA(connect.NewRequest(&v1.HostVaultServiceMoveNotesRequest{
		Paths: getCancelledBatch(), Folder: "Archive",
	})))
	if err != nil {
		t.Fatalf("the cancelled move failed as a whole: %v", err)
	}

	moves := res.Msg.GetMoves()
	if res.Msg.GetError() != 0 || len(moves) != 3 {
		t.Fatalf("the move returned error %v with %d entries, want no call error and 3 entries", res.Msg.GetError(), len(moves))
	}
	for i, from := range getCancelledBatch() {
		requireNotAttemptedMove(t, moves[i], from)
	}
	if len(vault.moved) != 0 {
		t.Errorf("the vault received a request for the moves %v after the cancel", vault.moved)
	}
}

func TestRemovalCancelledAfterOneNoteReturnsRestAsNotAttempted(t *testing.T) {
	endpoint, vault, ctx := newCancellingEndpoint(t, 1, changeThenCancel)

	res, err := endpoint.RemoveNotes(ctx, authorizeAsA(connect.NewRequest(&v1.HostVaultServiceRemoveNotesRequest{
		Paths: getCancelledBatch(),
	})))
	if err != nil {
		t.Fatalf("the cancelled removal failed as a whole: %v", err)
	}

	removals := res.Msg.GetRemovals()
	if res.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_UNSPECIFIED || len(removals) != 3 {
		t.Fatalf("the removal returned error %v with %d entries, want no call error and 3 entries", res.Msg.GetError(), len(removals))
	}
	requireRemoved(t, removals[0], "Danube.md")
	requireNotAttemptedRemoval(t, removals[1], "Rivers/Rhine.md")
	requireNotAttemptedRemoval(t, removals[2], "Deltas.md")
	if !slices.Equal(vault.removed, []string{"Danube.md"}) {
		t.Errorf("the vault received a call to remove %v, want only Danube.md", vault.removed)
	}
}

func TestRemovalFailingOnCancelReturnsErrorAndRestNotAttempted(t *testing.T) {
	endpoint, vault, ctx := newCancellingEndpoint(t, 2, cancelAndFail)

	res, err := endpoint.RemoveNotes(ctx, authorizeAsA(connect.NewRequest(&v1.HostVaultServiceRemoveNotesRequest{
		Paths: getCancelledBatch(),
	})))
	if err != nil {
		t.Fatalf("the cancelled removal failed as a whole: %v", err)
	}

	removals := res.Msg.GetRemovals()
	if res.Msg.GetError() != 0 || len(removals) != 3 {
		t.Fatalf("the removal returned error %v with %d entries, want no call error and 3 entries", res.Msg.GetError(), len(removals))
	}
	requireRemoved(t, removals[0], "Danube.md")
	requireNotAttemptedRemoval(t, removals[1], "Rivers/Rhine.md")
	requireNotAttemptedRemoval(t, removals[2], "Deltas.md")
	for _, unchanged := range []string{"Rivers/Rhine.md", "Deltas.md"} {
		if _, found := vault.files[unchanged]; !found {
			t.Errorf("%s was removed", unchanged)
		}
	}
}

func TestRemovalTrashedBeforeCancelReturnsThatNoteAsRemoved(t *testing.T) {
	endpoint, _, ctx := newCancellingEndpoint(t, 2, changeThenCancelAndFail)

	res, err := endpoint.RemoveNotes(ctx, authorizeAsA(connect.NewRequest(&v1.HostVaultServiceRemoveNotesRequest{
		Paths: getCancelledBatch(),
	})))
	if err != nil {
		t.Fatalf("the cancelled removal failed as a whole: %v", err)
	}

	removals := res.Msg.GetRemovals()
	if res.Msg.GetError() != 0 || len(removals) != 3 {
		t.Fatalf("the removal returned error %v with %d entries, want no call error and 3 entries", res.Msg.GetError(), len(removals))
	}
	requireRemoved(t, removals[0], "Danube.md")
	requireRemoved(t, removals[1], "Rivers/Rhine.md")
	requireNotAttemptedRemoval(t, removals[2], "Deltas.md")
}

// cancellingLinkVault is the linking vault that writes the links of the first note of a call,
// cancels the call, and returns note.ErrNotAttempted for the links of every later note, as the
// core vault does.
type cancellingLinkVault struct {
	*linkingVault
	cancel context.CancelFunc
}

func (c *cancellingLinkVault) AddLinks(ctx context.Context, additions []pluginport.PluginLinkAddition) ([]pluginport.PluginLinkAdded, error) {
	first := additions[0].From
	var written []pluginport.PluginLinkAddition
	for _, one := range additions {
		if one.From == first {
			written = append(written, one)
		}
	}
	added, err := c.linkingVault.AddLinks(ctx, written)
	if err != nil {
		return nil, err
	}
	c.cancel()
	out := make([]pluginport.PluginLinkAdded, len(additions))
	next := 0
	for i, one := range additions {
		if one.From == first {
			out[i] = added[next]
			next++
			continue
		}
		out[i] = pluginport.PluginLinkAdded{
			Err: fmt.Errorf("the call was cancelled before the links were added to %s: %w", one.From, note.ErrNotAttempted),
		}
	}
	return out, nil
}

func TestAddLinksCancelledAfterTheFirstNoteReturnsRestAsNotAttempted(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	vault := &cancellingLinkVault{linkingVault: newLinkingVault(), cancel: cancel}
	hostCallers := newCallers(nil)
	hostCallers.vault = vault
	endpoint := newVaultEndpoint(&Endpoint{callers: vaultListCallers{callers: hostCallers, granted: []string{pluginport.PermissionVaultWrite}}})
	batch := getCancelledBatch()
	links := make([]*v1.HostLinkAddition, len(batch))
	for i, from := range batch {
		links[i] = &v1.HostLinkAddition{From: from, Baseline: vault.readBaseline(t, from), To: "Estuary", Role: v1.Role_ROLE_REF}
	}

	res, err := endpoint.AddLinks(ctx, authorizeAsA(connect.NewRequest(&v1.HostVaultServiceAddLinksRequest{Links: links})))
	if err != nil {
		t.Fatalf("the cancelled link addition failed as a whole: %v", err)
	}

	results := res.Msg.GetResults()
	if res.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_UNSPECIFIED || len(results) != 3 {
		t.Fatalf("AddLinks returned error %v with %d results, want no call error and 3 results", res.Msg.GetError(), len(results))
	}
	if r := results[0]; r.GetError() != 0 || r.GetDetail() != "" || r.GetStat().GetPath() != "Danube.md" || r.GetStat().GetSize() == 0 {
		t.Errorf("result 0 is %v, want the file stat of Danube.md after the write and no error", r)
	}
	for i, from := range batch[1:] {
		want := "the call was cancelled before the links were added to " + from + ": " + note.ErrNotAttempted.Error()
		if r := results[i+1]; r.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_ATTEMPTED || r.GetStat() != nil || r.GetDetail() != want {
			t.Errorf("result %d is %v, want NOT_ATTEMPTED with the detail %q and no file stat", i+1, r, want)
		}
	}
}
