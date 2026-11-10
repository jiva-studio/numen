// Package editor serves the vault queries and commands of the editor window.
//
// The request and response types are defined in libs/core/protocol. The client
// connects to the handlers over HTTP in both the desktop window and a browser
// during development.
package editor

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/broadcast"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/tablinks"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"

	indexport "github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing/indexcontract"
	windowport "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/windowcontract"
)

// State is the vault the window is looking at and what every handler of it
// reads: the vault shown, the door on every question, the writes and tasks in
// the air, and who is drawing what changes.
type State struct {
	vault atomic.Pointer[domain.Vault]

	closed atomic.Bool

	questions inflight

	session atomic.Pointer[vaultSession]

	ChangeEvents broadcaster[change]

	OpenVault func(context.Context, domain.Vault, *PluginOpen) error

	NotifyNoteWritten func()

	Edits broadcaster[domain.Edit]

	Places broadcaster[domain.Place]

	TabIDs broadcaster[string]

	openTabs atomic.Pointer[domain.OpenTabs]

	openTabsReports reportSignal

	TabActivationBound time.Duration

	NotifyOpenTabs func(domain.OpenTabs)

	OpenTabsStore windowport.OpenTabsStore

	DetachTab func(context.Context, DetachedTab) error

	TabLinks *tablinks.Registry

	Writing inflight

	Ready       atomic.Bool
	Error       wire.Reason
	WatchError  wire.Reason
	Unreachable wire.Reason

	Window *wire.Window

	EmitEvent func(event string)

	Indexing IndexState

	// Playing is the socket recordings are served over, and nothing until it is open.
	Playing atomic.Pointer[Loopback]
}

// IndexState is how far the vault has been read for meaning.
type IndexState struct {
	Progress indexport.IndexProgress
	Model    atomic.Value
	Recipe   atomic.Value
}

// GetCurrentVault is the vault the window has open. A window standing on nothing
// answers with no vault at all.
func (s *State) GetCurrentVault() domain.Vault {
	if v := s.vault.Load(); v != nil {
		return *v
	}
	return domain.Vault{}
}

func (s *State) writeOpenTabs(ctx context.Context, open domain.OpenTabs) error {
	if v := s.GetCurrentVault(); s.OpenTabsStore != nil && v.Path != "" {
		return vaults.WriteOpenTabs(ctx, s.OpenTabsStore, v, open)
	}
	return nil
}

func (s *State) readOpenTabs(ctx context.Context) (domain.OpenTabs, error) {
	v := s.GetCurrentVault()
	if s.OpenTabsStore == nil || v.Path == "" {
		return domain.OpenTabs{}, nil
	}
	return vaults.ReadOpenTabs(ctx, s.OpenTabsStore, v)
}

func (s *State) setVault(v domain.Vault) { s.vault.Store(&v) }

func (s *State) setSession(on *vaultSession) { s.session.Store(on) }

func (s *State) reportTask(pos vaultreading.Task) { s.Window.AddTask(pos) }

func (s *State) finishTask(id string) { s.Window.RemoveTask(id) }

func (s *State) isIndexOutdated(err error) bool {
	if err == nil {
		s.finishTask(updatingIndex)
		return false
	}
	if !errors.Is(err, note.ErrIndexOutdated) {
		return false
	}
	s.reportTask(vaultreading.Task{ID: updatingIndex, Title: "Update index", Error: err.Error()})
	return true
}

// Close rejects all new requests permanently. It returns after the requests
// already accepted have completed, so the index and embedder stay available
// for their whole duration.
func (s *State) Close() {
	s.closed.Store(true)
	<-s.questions.seal()
}

func (s *State) isClosed() bool { return s.closed.Load() }

func (s *State) getCurrentVault() (domain.Vault, error) {
	v := s.GetCurrentVault()
	if v.ID == "" {
		return domain.Vault{}, connect.NewError(connect.CodeFailedPrecondition, errNoVault)
	}
	return v, nil
}

func text(v *atomic.Value) string {
	s, isString := v.Load().(string)
	if !isString {
		return ""
	}
	return s
}

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
	// the state is answered currently. A window standing on nothing holds no
	// chunks and counts none.
	if h.state.Indexing.Progress != nil && showing.ID != "" {
		if target, embedded, err := indexing.GetProgress(ctx, h.state.Indexing.Progress, showing.ID, text(&h.state.Indexing.Recipe)); err == nil {
			out.Coverage.ChunkCount, out.Coverage.EmbeddedCount = target, embedded
		}
	}
	return connect.NewResponse(out), nil
}

// WatchVaultChanges streams vault changes until the caller disconnects.
func (h *vaultHandler) WatchVaultChanges(
	ctx context.Context,
	_ *connect.Request[v1.WatchVaultChangesRequest],
	out *connect.ServerStream[v1.WatchVaultChangesResponse],
) error {
	line, isDone := h.state.ChangeEvents.subscribe()
	defer isDone()

	// An initial empty message confirms the subscription. A stream that sends
	// nothing until the vault changes is indistinguishable from one that never
	// opened.
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

func newChangeResponse(what change) *v1.WatchVaultChangesResponse {
	renamedMoves := make([]*v1.Move, 0, len(what.renamed))
	for _, went := range what.renamed {
		renamedMoves = append(renamedMoves, &v1.Move{From: went.From, To: went.To})
	}
	return &v1.WatchVaultChangesResponse{
		Paths: what.paths, Reload: what.shouldReload, Renamed: renamedMoves, VersionPaths: what.versionPaths,
	}
}

// GetWindow returns the window handle for requesting place focus.
func (s *State) GetWindow() windowport.Window { return view{s} }

// view forwards window requests to the subscribed clients. A place requested
// while no client is subscribed is not shown.
type view struct{ *State }

func (v view) FocusPlace(_ context.Context, pos domain.Place) error {
	v.Places.publish(pos)
	return nil
}

func (v view) ShowMove(_ context.Context, went domain.Move) error {
	v.ChangeEvents.publish(change{renamed: []domain.Move{went}})
	return nil
}

func (v view) ShowEdit(_ context.Context, edit domain.Edit) error {
	v.Edits.publish(edit)
	return nil
}

type broadcaster[T any] struct {
	broadcast.Broadcaster[T]
}

func (a *broadcaster[T]) subscribe() (<-chan T, func()) { return a.Broadcaster.Subscribe() }
func (a *broadcaster[T]) publish(what T)                { a.Broadcaster.Publish(what) }

type inflight struct {
	broadcast.Inflight
}

func (w *inflight) begin() bool           { return w.Inflight.Begin() }
func (w *inflight) finish()               { w.Inflight.Finish() }
func (w *inflight) seal() <-chan struct{} { return w.Inflight.Seal() }
func (w *inflight) open()                 { w.Inflight.Open() }

var errClosing = broadcast.ErrClosing

// change reports the changed file paths, notes and assets alike, or that the
// vault must be reloaded.
type change struct {
	paths        []string
	shouldReload bool
	renamed      []domain.Move
	versionPaths []string
}

func newChangeBroadcaster() broadcaster[change] {
	return broadcaster[change]{
		Broadcaster: broadcast.Broadcaster[change]{
			Fallback: func(change) change { return change{shouldReload: true} },
			Capacity: 8,
		},
	}
}

func newPlaceBroadcaster() broadcaster[domain.Place] {
	return broadcaster[domain.Place]{
		Broadcaster: broadcast.Broadcaster[domain.Place]{IsLatest: true, Capacity: 1},
	}
}

func newTabIDBroadcaster() broadcaster[string] {
	return broadcaster[string]{
		Broadcaster: broadcast.Broadcaster[string]{IsLatest: true, Capacity: 1},
	}
}

func newEditBroadcaster() broadcaster[domain.Edit] {
	return broadcaster[domain.Edit]{
		Broadcaster: broadcast.Broadcaster[domain.Edit]{
			IsLatest:   true,
			GetTopic:   func(response domain.Edit) string { return response.Change },
			IsRequired: func(reply domain.Edit) bool { return reply.IsDone },
			Capacity:   8,
		},
	}
}

// DetachedTab is a tab to open in a window of its own. The new window reads
// the tab by its id with ReadDetachedTab.
type DetachedTab struct {
	ID            string
	X, Y          int
	Width, Height int
}

func (v view) ActivateTab(_ context.Context, id string) error {
	v.TabIDs.publish(id)
	return nil
}

// reportSignal closes a channel when a report arrives. The zero value is ready to use.
type reportSignal struct {
	mu   sync.Mutex
	next chan struct{}
}

func (r *reportSignal) wait() <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.next == nil {
		r.next = make(chan struct{})
	}
	return r.next
}

func (r *reportSignal) notify() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.next != nil {
		close(r.next)
		r.next = nil
	}
}
