// Package editor serves the vault queries and commands of the editor window.
//
// The request and response types are defined in libs/core/protocol. The client
// connects to the handlers over HTTP in both the desktop window and a browser
// during development.
package editor

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/broadcast"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// State is what every handler of the window reads: the vault shown and how far it is read (the
// embedded runtime of the window session), the door on every question, the writes and tasks in the
// air, and who is drawing what changes.
type State struct {
	windowsession.Runtime

	closed atomic.Bool

	questions inflight

	ChangeEvents broadcaster[window.Change]

	OpenVault func(context.Context, domain.Vault, *windowsession.PluginOpen) error

	NotifyNoteWritten func()

	Edits broadcaster[domain.Edit]

	Places broadcaster[domain.Place]

	TabIDs broadcaster[string]

	NotifyOpenTabs func(domain.OpenTabs)

	DetachTab func(context.Context, window.DetachedTab) error

	Writing inflight

	Unreachable wire.Reason

	Window *wire.Window

	EmitEvent func(event string)

	Indexing IndexState

	Playing atomic.Pointer[Loopback]
}

// IndexState is how far the vault has been read for meaning.
type IndexState struct {
	Embedding *indexing.Embedding
}

// NewState makes the state of a window that shows no vault.
func NewState(tasks *run.Tasks) *State {
	state := &State{
		ChangeEvents: newChangeBroadcaster(),
		Places:       newPlaceBroadcaster(),
		TabIDs:       newTabIDBroadcaster(),
		Edits:        newEditBroadcaster(),
		Window:       &wire.Window{Name: wire.Editor, Tasks: tasks},
	}
	// The editor is open on one vault, and answers which as itself.
	state.Window.Vault = func() string { return string(state.GetCurrentVault().ID) }
	return state
}

// SetIndexing sets how far the vault has been read for meaning, and the model it is read under.
// It is set before any chunk is read: the model name and recipe determine whether
// a chunk already carries a vector, and tell the window that chunks will be
// embedded.
func (s *State) SetIndexing(embedding *indexing.Embedding) {
	s.Indexing.Embedding = embedding
	s.RecordModel(embedding.Describe())
}

// ReportWordsAlone records that the search runs on words only, with the reason. It is a task because
// a window opened from a desktop entry has no terminal.
func (s *State) ReportWordsAlone(err error) {
	s.reportTask(work.Task{ID: work.WordsAlone, Title: "Search by words only", Error: err.Error()})
}

// ReportFloorError records a similarity floor that is stored and cannot be read.
func (s *State) ReportFloorError(err error) {
	s.reportTask(work.Task{ID: readingTheFloor, Title: "Read search floor", Error: err.Error()})
}

// GetVaultState is what the window knows about the vault it shows: whether the first scan is done,
// what stopped it, and how far the vault has been read for meaning.
func (s *State) GetVaultState(ctx context.Context) window.VaultState {
	showing := s.GetCurrentVault()
	name, recipe := s.GetModel()
	out := window.VaultState{
		Vault:       showing,
		IsReady:     s.IsReady(),
		Error:       s.GetError(),
		WatchError:  s.GetWatchError(),
		IsEmbedding: name != "",
	}
	// A count that cannot be taken leaves the pair at nothing, and the rest of
	// the state is answered currently. A window standing on nothing holds no
	// chunks and counts none.
	if s.Indexing.Embedding.HasProgress() && showing.ID != "" {
		if target, embedded, err := s.Indexing.Embedding.CountProgress(ctx, showing.ID, recipe); err == nil {
			out.ChunkCount, out.EmbeddedCount = target, embedded
		}
	}
	return out
}

// SubscribeChanges delivers each change of the vault until the returned function is called.
func (s *State) SubscribeChanges() (<-chan window.Change, func()) { return s.ChangeEvents.subscribe() }

// SubscribePlaces delivers each place the window is asked to show until the returned function is called.
func (s *State) SubscribePlaces() (<-chan domain.Place, func()) { return s.Places.subscribe() }

// SubscribeTabIDs delivers the id of each tab the window is asked to bring to the front until the
// returned function is called.
func (s *State) SubscribeTabIDs() (<-chan string, func()) { return s.TabIDs.subscribe() }

// ReportOpenTabs keeps the tabs the window reported, wakes whoever waits for a report, and tells
// the application.
func (s *State) ReportOpenTabs(open domain.OpenTabs) {
	s.Runtime.ReportOpenTabs(open)
	if s.NotifyOpenTabs != nil {
		s.NotifyOpenTabs(open)
	}
}

// ShowVault shows another vault in the window, as a user switch.
func (s *State) ShowVault(ctx context.Context, v domain.Vault) error {
	return s.OpenVault(ctx, v, nil)
}

// CanOpenTabWindow is whether the application opens a window for a tab.
func (s *State) CanOpenTabWindow() bool { return s.DetachTab != nil }

// OpenTabWindow opens a window of its own for a tab.
func (s *State) OpenTabWindow(ctx context.Context, tab window.DetachedTab) error {
	return s.DetachTab(ctx, tab)
}

// ClosePlaying stops the socket recordings are played over.
func (s *State) ClosePlaying(ctx context.Context) error { return s.Playing.Load().Close(ctx) }

func (s *State) reportTask(pos work.Task) { s.Window.AddTask(pos) }

func (s *State) finishTask(id string) { s.Window.RemoveTask(id) }

func (s *State) isIndexOutdated(err error) bool {
	if err == nil {
		s.finishTask(updatingIndex)
		return false
	}
	if !errors.Is(err, note.ErrIndexOutdated) {
		return false
	}
	s.reportTask(work.Task{ID: updatingIndex, Title: "Update index", Error: err.Error()})
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

// PublishPlace asks the subscribed clients to show the place.
func (s *State) PublishPlace(pos domain.Place) { s.Places.publish(pos) }

// PublishMove tells the subscribed clients that a note moved.
func (s *State) PublishMove(went domain.Move) {
	s.ChangeEvents.publish(window.Change{Renamed: []domain.Move{went}})
}

// PublishEdit tells the subscribed clients that a change to a note is in progress.
func (s *State) PublishEdit(edit domain.Edit) { s.Edits.publish(edit) }

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

func newChangeBroadcaster() broadcaster[window.Change] {
	return broadcaster[window.Change]{
		Broadcaster: broadcast.Broadcaster[window.Change]{
			Fallback: func(window.Change) window.Change { return window.Change{ShouldReload: true} },
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

// PublishTab asks the subscribed clients to bring the tab to the front.
func (s *State) PublishTab(id string) { s.TabIDs.publish(id) }
