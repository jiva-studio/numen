// Package editor serves the vault queries and commands of the editor window.
//
// The request and response types are defined in libs/protocol. The client
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

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/broadcast"
	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/tablinks"
	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"

	indexport "github.com/jiva-studio/numen/libs/core/usecase/indexing/indexcontract"
	windowport "github.com/jiva-studio/numen/libs/core/usecase/vault/windowcontract"
)

// State is the vault the window is looking at and what every handler of it
// reads: the vault shown, the door on every question, the writes and tasks in
// the air, and who is drawing what changes.
type State struct {
	// vault is the vault the window has open. It is replaced while requests are
	// being served, so every reader takes it through GetCurrentVault.
	vault atomic.Pointer[domain.Vault]

	// closed rejects new requests. It is set before the index and the embedder
	// that handlers use are closed.
	closed atomic.Bool

	// questions are the questions taken and not yet answered. A search, a note
	// and a link are read straight from the index, and Close waits here until
	// the last of them is off it.
	//
	// A stream is not counted: it lives as long as the page that opened it.
	questions inflight

	// published as one after the vault, so a run, a cut and a drop reach the
	// vault the request was answered over. Nothing while the vault is being
	// changed.
	session atomic.Pointer[vaultSession]

	ChangeEvents broadcaster[change]

	// OpenVault puts another vault in the window: the agents are stopped, the vault
	// is swapped, and the agents are started again against the one that
	// arrived. It is bound by every build that serves the list of vaults. The open is the
	// plugin open whose lock the switch shares, nil for a user vault switch.
	OpenVault func(context.Context, domain.Vault, *PluginOpen) error

	// NotifyNoteWritten is called after a note save. The embedding pipeline then
	// queries the index for chunks without vectors once the vault is idle. Nil
	// when no embedding pipeline runs, in which case a save changes no vectors.
	NotifyNoteWritten func()

	// Edits broadcasts note edits to the subscribers of this vault.
	Edits broadcaster[domain.Edit]

	// Places broadcasts place focus requests to the subscribers of this vault.
	Places broadcaster[domain.Place]

	// TabIDs broadcasts the ids of open tabs to make active to the subscribers of this vault.
	TabIDs broadcaster[string]

	// openTabs is the open-tab report most recently received from the window.
	// It is replaced while requests are served, so readers use GetOpenTabs.
	openTabs atomic.Pointer[domain.OpenTabs]

	// openTabsReports is notified after each open-tab report is stored.
	openTabsReports reportSignal

	// TabActivationBound is the maximum time ActivateTab waits for the window to report the tab active.
	// Zero means 2 seconds.
	TabActivationBound time.Duration

	// NotifyOpenTabs is called with each accepted open-tab report. Nil when
	// nothing consumes the report.
	NotifyOpenTabs func(domain.OpenTabs)

	// OpenTabsStore keeps the open tabs of the vault across restarts. Nil keeps
	// them for as long as the core runs.
	OpenTabsStore windowport.OpenTabsStore

	// DetachTab opens a tab in a window of its own.
	DetachTab func(context.Context, DetachedTab) error

	// TabLinks stores the link groups of the tabs of all windows. When nil, the
	// tab links RPCs return Unimplemented.
	TabLinks *tablinks.Registry

	// Writing is the writes taken and not yet finished.
	Writing inflight

	// Ready is set when the scan has finished. Error holds the scan failure, if any.
	Ready atomic.Bool
	Error wire.Reason
	// WatchError is why the vault is not being followed, when it is not.
	WatchError wire.Reason
	// Unreachable is why an agent cannot be reached, when one cannot.
	Unreachable wire.Reason

	// Window is this window itself: everything being done behind it, which
	// whatever does work puts itself into, and everyone drawing it for the
	// moment it goes.
	Window *wire.Window

	// EmitEvent queues an event into the plugin host event bus.
	EmitEvent func(event string)

	Indexing IndexState
}

// IndexState is how far the vault has been read for meaning.
type IndexState struct {
	// Progress reports chunking and embedding progress. Nil when the vault is
	// not indexed for semantic search; the window then reports no coverage.
	Progress indexport.IndexProgress
	// Model is the embedding model identity that new vectors are created with.
	// It decides whether a chunk already has a vector. Empty when no model is
	// installed, in which case nothing is embedded. The indexing goroutine
	// writes it and every request reads it.
	Model atomic.Value
	// Recipe is everything that decides what a vector is, which is what a
	// vector is found by.
	Recipe atomic.Value
}

// GetCurrentVault is the vault the window has open. A window standing on nothing
// answers with no vault at all.
func (s *State) GetCurrentVault() domain.Vault {
	if v := s.vault.Load(); v != nil {
		return *v
	}
	return domain.Vault{}
}

// writeOpenTabs records the open tabs of the vault in the window.
func (s *State) writeOpenTabs(ctx context.Context, open domain.OpenTabs) error {
	if v := s.GetCurrentVault(); s.OpenTabsStore != nil && v.Path != "" {
		return vaults.WriteOpenTabs(ctx, s.OpenTabsStore, v, open)
	}
	return nil
}

// readOpenTabs reads the open tabs kept for the vault in the window.
func (s *State) readOpenTabs(ctx context.Context) (domain.OpenTabs, error) {
	v := s.GetCurrentVault()
	if s.OpenTabsStore == nil || v.Path == "" {
		return domain.OpenTabs{}, nil
	}
	return vaults.ReadOpenTabs(ctx, s.OpenTabsStore, v)
}

// setVault sets the vault that subsequent requests are served over.
func (s *State) setVault(v domain.Vault) { s.vault.Store(&v) }

// setSession sets the session a run, a rechunk and a drop are taken through from now
// on. They arrive together, after the vault they belong to.
func (s *State) setSession(on *vaultSession) { s.session.Store(on) }

// reportTask puts one piece of work in the list of what is being done behind the
// window, for a build that keeps one.
func (s *State) reportTask(pos run.Task) { s.Window.AddTask(pos) }

// finishTask takes one piece of work out of that list.
func (s *State) finishTask(id string) { s.Window.RemoveTask(id) }

// isIndexOutdated reports whether a write reached the vault but the index did not
// follow. The result is returned to the client that saved, and the pending
// task is shown until a later write updates the index again.
func (s *State) isIndexOutdated(err error) bool {
	if err == nil {
		s.finishTask(updatingIndex)
		return false
	}
	if !errors.Is(err, note.ErrIndexOutdated) {
		return false
	}
	s.reportTask(run.Task{ID: updatingIndex, Title: "Update index", Error: err.Error()})
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

// getCurrentVault is the vault a question is answered over. A window standing on nothing
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

// newChangeResponse is the wire message that reports one change.
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
	// renamed lists notes that moved, each with its old and new path.
	renamed []domain.Move
	// versionPaths lists the files whose text versions changed: one was made or deleted, or became
	// live or no longer live.
	versionPaths []string
}

// newChangeBroadcaster delivers what moved to its subscribers.
func newChangeBroadcaster() broadcaster[change] {
	return broadcaster[change]{
		Broadcaster: broadcast.Broadcaster[change]{
			Fallback: func(change) change { return change{shouldReload: true} },
			Capacity: 8,
		},
	}
}

// newPlaceBroadcaster delivers place focus requests to every subscriber.
func newPlaceBroadcaster() broadcaster[domain.Place] {
	return broadcaster[domain.Place]{
		Broadcaster: broadcast.Broadcaster[domain.Place]{IsLatest: true, Capacity: 1},
	}
}

// newTabIDBroadcaster delivers the id of a tab to make active to every subscriber. A newer id
// replaces a pending one.
func newTabIDBroadcaster() broadcaster[string] {
	return broadcaster[string]{
		Broadcaster: broadcast.Broadcaster[string]{IsLatest: true, Capacity: 1},
	}
}

// newEditBroadcaster delivers to every subscriber, for a change to a note being made
// while they are showing it.
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
	ID string
	// X and Y are the screen position of the new window.
	X, Y int
	// Width and Height are the size of the new window; 0 keeps the default size.
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

// wait returns a channel that the next notify closes.
func (r *reportSignal) wait() <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.next == nil {
		r.next = make(chan struct{})
	}
	return r.next
}

// notify closes the channel that wait returned.
func (r *reportSignal) notify() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.next != nil {
		close(r.next)
		r.next = nil
	}
}
