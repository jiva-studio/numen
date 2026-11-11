package hostendpoint

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/note/domain"

	vaultapi "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// cancelMode is what the canceling vault does on the call that cancels the batch.
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

// cancelingVault is the vault of rivers that cancels the call it serves on its change number cancelIndex
// (counted from one), as mode sets.
type cancelingVault struct {
	*fakeNoteVault
	cancel      context.CancelFunc
	cancelIndex int
	mode        cancelMode
	calls       int
}

func (c *cancelingVault) MoveNote(ctx context.Context, from, to string) (domain.MoveResult, error) {
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

func (c *cancelingVault) RemoveNote(ctx context.Context, target string) (string, []string, error) {
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

func newCancelingEndpoint(t *testing.T, cancelIndex int, mode cancelMode) (numenv1connect.HostVaultServiceClient, *cancelingVault, context.Context) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	vault := &cancelingVault{fakeNoteVault: newFakeNoteVault(), cancel: cancel, cancelIndex: cancelIndex, mode: mode}
	hostCallers := newCallers(nil)
	hostCallers.vault = vaultOf(vault)
	endpoint := serveVaultCalls(t, ctx, hostCallers.over(nil, manifest.PermissionVaultWrite))
	return endpoint, vault, ctx
}

func authorizeAsA[T any](req *connect.Request[T]) *connect.Request[T] {
	req.Header().Set("Authorization", "Bearer token-a")
	return req
}

func getCanceledBatch() []string { return []string{"Danube.md", "Rivers/Rhine.md", "Deltas.md"} }

func requireNotAttemptedMove(t *testing.T, entry *v1.HostNoteMove, from string) {
	t.Helper()
	want := "the call was canceled before " + from + " was moved"
	if entry.GetMove().GetFrom() != from || entry.GetMove().GetTo() != "" ||
		entry.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_ATTEMPTED || entry.GetDetail() != want {
		t.Errorf("the entry of %s is %v, want error NOT_ATTEMPTED with the detail %q", from, entry, want)
	}
}

func requireNotAttemptedRemoval(t *testing.T, entry *v1.HostNoteRemoval, target string) {
	t.Helper()
	want := "the call was canceled before " + target + " was removed"
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

func TestMoveCanceledAfterTwoNotesReturnsRestAsNotAttempted(t *testing.T) {
	endpoint, vault, _ := newCancelingEndpoint(t, 2, changeThenCancel)

	res, err := endpoint.MoveNotes(t.Context(), authorizeAsA(connect.NewRequest(&v1.HostVaultServiceMoveNotesRequest{
		Paths: getCanceledBatch(), Folder: "Archive",
	})))
	if err != nil {
		t.Fatalf("the canceled move failed as a whole: %v", err)
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
	endpoint, vault, _ := newCancelingEndpoint(t, 2, cancelAndFail)

	res, err := endpoint.MoveNotes(t.Context(), authorizeAsA(connect.NewRequest(&v1.HostVaultServiceMoveNotesRequest{
		Paths: getCanceledBatch(), Folder: "Archive",
	})))
	if err != nil {
		t.Fatalf("the canceled move failed as a whole: %v", err)
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
	endpoint, _, _ := newCancelingEndpoint(t, 2, changeThenCancelAndFail)

	res, err := endpoint.MoveNotes(t.Context(), authorizeAsA(connect.NewRequest(&v1.HostVaultServiceMoveNotesRequest{
		Paths: getCanceledBatch(), Folder: "Archive",
	})))
	if err != nil {
		t.Fatalf("the canceled move failed as a whole: %v", err)
	}

	moves := res.Msg.GetMoves()
	if res.Msg.GetError() != 0 || len(moves) != 3 {
		t.Fatalf("the move returned error %v with %d entries, want no call error and 3 entries", res.Msg.GetError(), len(moves))
	}
	requireMoved(t, moves[0], "Danube.md", "Archive/Danube.md")
	requireMoved(t, moves[1], "Rivers/Rhine.md", "Archive/Rhine.md")
	requireNotAttemptedMove(t, moves[2], "Deltas.md")
}

func TestRemovalCanceledAfterOneNoteReturnsRestAsNotAttempted(t *testing.T) {
	endpoint, vault, _ := newCancelingEndpoint(t, 1, changeThenCancel)

	res, err := endpoint.RemoveNotes(t.Context(), authorizeAsA(connect.NewRequest(&v1.HostVaultServiceRemoveNotesRequest{
		Paths: getCanceledBatch(),
	})))
	if err != nil {
		t.Fatalf("the canceled removal failed as a whole: %v", err)
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
	endpoint, vault, _ := newCancelingEndpoint(t, 2, cancelAndFail)

	res, err := endpoint.RemoveNotes(t.Context(), authorizeAsA(connect.NewRequest(&v1.HostVaultServiceRemoveNotesRequest{
		Paths: getCanceledBatch(),
	})))
	if err != nil {
		t.Fatalf("the canceled removal failed as a whole: %v", err)
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
	endpoint, _, _ := newCancelingEndpoint(t, 2, changeThenCancelAndFail)

	res, err := endpoint.RemoveNotes(t.Context(), authorizeAsA(connect.NewRequest(&v1.HostVaultServiceRemoveNotesRequest{
		Paths: getCanceledBatch(),
	})))
	if err != nil {
		t.Fatalf("the canceled removal failed as a whole: %v", err)
	}

	removals := res.Msg.GetRemovals()
	if res.Msg.GetError() != 0 || len(removals) != 3 {
		t.Fatalf("the removal returned error %v with %d entries, want no call error and 3 entries", res.Msg.GetError(), len(removals))
	}
	requireRemoved(t, removals[0], "Danube.md")
	requireRemoved(t, removals[1], "Rivers/Rhine.md")
	requireNotAttemptedRemoval(t, removals[2], "Deltas.md")
}

// cancelingLinkVault is the linking vault that writes the links of the first note of a call,
// cancels the call, and returns plugin.ErrNotAttempted for the links of every later note, as the
// core vault does.
type cancelingLinkVault struct {
	*linkingVault
	cancel context.CancelFunc
}

func (c *cancelingLinkVault) AddLinks(ctx context.Context, additions []plugin.LinkAddition) ([]plugin.LinkAdded, error) {
	first := additions[0].From
	var written []plugin.LinkAddition
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
	out := make([]plugin.LinkAdded, len(additions))
	next := 0
	for i, one := range additions {
		if one.From == first {
			out[i] = added[next]
			next++
			continue
		}
		out[i] = plugin.LinkAdded{
			Err: fmt.Errorf("the call was canceled before the links were added to %s: %w", one.From, plugin.ErrNotAttempted),
		}
	}
	return out, nil
}

func TestAddLinksCanceledAfterTheFirstNoteReturnsRestAsNotAttempted(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	vault := &cancelingLinkVault{linkingVault: newLinkingVault(), cancel: cancel}
	hostCallers := newCallers(nil)
	hostCallers.vault = vaultOf(vault)
	endpoint := serveVaultCalls(t, ctx, hostCallers.over(nil, manifest.PermissionVaultWrite))
	batch := getCanceledBatch()
	links := make([]*v1.HostLinkAddition, len(batch))
	for i, from := range batch {
		links[i] = &v1.HostLinkAddition{From: from, Baseline: vault.readBaseline(t, from), To: "Estuary", Role: v1.Role_ROLE_REF}
	}

	res, err := endpoint.AddLinks(t.Context(), authorizeAsA(connect.NewRequest(&v1.HostVaultServiceAddLinksRequest{Links: links})))
	if err != nil {
		t.Fatalf("the canceled link addition failed as a whole: %v", err)
	}

	results := res.Msg.GetResults()
	if res.Msg.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_UNSPECIFIED || len(results) != 3 {
		t.Fatalf("AddLinks returned error %v with %d results, want no call error and 3 results", res.Msg.GetError(), len(results))
	}
	if r := results[0]; r.GetError() != 0 || r.GetDetail() != "" || r.GetStat().GetPath() != "Danube.md" || r.GetStat().GetSize() == 0 {
		t.Errorf("result 0 is %v, want the file stat of Danube.md after the write and no error", r)
	}
	for i, from := range batch[1:] {
		want := "the call was canceled before the links were added to " + from + ": " + plugin.ErrNotAttempted.Error()
		if r := results[i+1]; r.GetError() != v1.HostVaultErrorCode_HOST_VAULT_ERROR_CODE_NOT_ATTEMPTED || r.GetStat() != nil || r.GetDetail() != want {
			t.Errorf("result %d is %v, want NOT_ATTEMPTED with the detail %q and no file stat", i+1, r, want)
		}
	}
}

func serveVaultCalls(t *testing.T, ctx context.Context, callers Callers) numenv1connect.HostVaultServiceClient {
	t.Helper()
	// Every call is served under ctx, so that a test ends the call from inside the vault the way a stopped plugin does.
	mux := http.NewServeMux()
	mux.Handle(vaultapi.NewHandler(&Endpoint{callers: callers}, vaultSource{callers: callers}))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r.WithContext(ctx))
	}))
	t.Cleanup(server.Close)
	return numenv1connect.NewHostVaultServiceClient(server.Client(), server.URL)
}
