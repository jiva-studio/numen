// Package editor answers what a client may ask about a vault.
//
// The questions and their answers are the schema in modules/libs/protocol; this
// is the half that answers them. The interface asks over HTTP either way, so
// the same handler serves the window the application opens and a browser during
// development.
package editor

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/broadcast"
	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/indexport"
	"github.com/jiva-studio/numen/libs/core/port/windowport"
)

// State is the vault the window is looking at and what every handler of it
// reads: the vault shown, the door on every question, the writes and tasks in
// the air, and who is drawing what changes.
type State struct {
	// vault is the vault the window has open. It is replaced while requests are
	// being served, so every reader takes it through GetCurrentVault.
	vault atomic.Pointer[domain.Vault]

	// closed is the door on every question. It is closed before the index and the
	// embedder an answer reaches into are taken away.
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
	// arrived. It is bound by every build that serves the list of vaults.
	OpenVault func(context.Context, domain.Vault) error

	// NotifyNoteWritten is what a save raises: the reading behind the window asks the index
	// what owes a vector, once the vault has been still. Nil for a build with
	// nothing reading behind it, and then a save changes no vectors.
	NotifyNoteWritten func()

	// Edits is everyone drawing this vault, for a change to a note being made
	// while they may be showing it.
	Edits broadcaster[domain.Edit]

	// Places is everyone drawing this vault, for when something asks that a
	// place be put in front of the person.
	Places broadcaster[domain.Place]

	// openTabs is what currently open, as the window last said. It is
	// replaced while requests are being served, so every reader takes it
	// through GetOpenTabs.
	openTabs atomic.Pointer[domain.OpenTabs]

	// NotifyOpenTabs hears what currently open each time the window says it,
	// once what it said stands. A build without one takes the report and tells
	// nobody.
	NotifyOpenTabs func(domain.OpenTabs)

	// OpenTabsStore keeps the open tabs of the vault across restarts. Nil keeps
	// them for as long as the core runs.
	OpenTabsStore windowport.OpenTabsStore

	// DetachTab opens a tab in its own window at the given screen coordinates.
	DetachTab func(context.Context, string, int, int) error

	// Writing is the writes taken and not yet finished.
	Writing inflight

	// Ready is set when the scan finished, and Error says why it could not.
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
	// Progress answers how far cutting and embedding have got. Nil for a vault
	// nothing is reading for meaning, and the window then says nothing about it.
	Progress indexport.IndexProgress
	// Model is the identity vectors are being made under, which is what decides
	// whether a chunk already carries one. Empty for an installation with no
	// model, and then nothing is going to embed anything. It is named by the
	// goroutine reading the vault and asked for by every request.
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
		return s.OpenTabsStore.WriteOpenTabs(ctx, v, open)
	}
	return nil
}

// readOpenTabs reads the open tabs kept for the vault in the window.
func (s *State) readOpenTabs(ctx context.Context) (domain.OpenTabs, error) {
	v := s.GetCurrentVault()
	if s.OpenTabsStore == nil || v.Path == "" {
		return domain.OpenTabs{}, nil
	}
	return s.OpenTabsStore.ReadOpenTabs(ctx, v)
}

// setVault puts a vault in front of whoever asks from now on.
func (s *State) setVault(v domain.Vault) { s.vault.Store(&v) }

// setSession sets the session a run, a rechunk and a drop are taken through from now
// on. They arrive together, after the vault they belong to.
func (s *State) setSession(on *vaultSession) { s.session.Store(on) }

// reportTask puts one piece of work in the list of what is being done behind the
// window, for a build that keeps one.
func (s *State) reportTask(pos run.Task) { s.Window.AddTask(pos) }

// finishTask takes one piece of work out of that list.
func (s *State) finishTask(id string) { s.Window.RemoveTask(id) }

// isUnlevelled says whether a write reached the vault and the index did not
// follow.
// The answer goes on the wire, so the client that saved knows search has not
// until a write levels the index again, so a person who is not looking at that
// note is told too.
func (s *State) isUnlevelled(err error) bool {
	if err == nil {
		s.finishTask(levellingTheIndex)
		return false
	}
	if !errors.Is(err, note.ErrUnlevelled) {
		return false
	}
	s.reportTask(run.Task{ID: levellingTheIndex, Title: "Bringing the index level", Error: err.Error()})
	return true
}

// Close refuses every question from now on, and there is no opening it again. It
// answers once the questions already taken have been answered, so everything an
// answer reaches into is still there for the whole of it.
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
	s, _ := v.Load().(string)
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
		if target, embedded, err := h.state.Indexing.Progress.GetProgress(ctx, showing.ID, text(&h.state.Indexing.Recipe)); err == nil {
			out.Coverage.ChunkCount, out.Coverage.EmbeddedCount = target, embedded
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
	line, isDone := h.state.ChangeEvents.subscribe()
	defer isDone()

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
	renamedMoves := make([]*v1.Move, 0, len(what.renamed))
	for _, went := range what.renamed {
		renamedMoves = append(renamedMoves, &v1.Move{From: went.From, To: went.To})
	}
	return &v1.WatchVaultChangesResponse{Paths: what.paths, Reload: what.shouldReload, Renamed: renamedMoves}
}

// GetWindow is this window, for whatever asks for a place to be put in front of
// the person.
func (s *State) GetWindow() windowport.Window { return view{s} }

// view tells the clients and nothing more. What travelling there looks
// like is theirs, and a place asked for while nobody is drawing is a place
// nobody sees.
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

// change is what a client is told: the files that are different now, notes
// and assets alike, or that the vault has to be read again.
type change struct {
	paths        []string
	shouldReload bool
	// renamed is the notes that are no longer where they were, each by where it
	// was and where it now is.
	renamed []domain.Move
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

// newPlaceBroadcaster delivers to every subscriber, for the places something asks to be
// put in front of the person.
func newPlaceBroadcaster() broadcaster[domain.Place] {
	return broadcaster[domain.Place]{
		Broadcaster: broadcast.Broadcaster[domain.Place]{IsLatest: true, Capacity: 1},
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
