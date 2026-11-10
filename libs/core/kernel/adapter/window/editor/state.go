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

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/broadcast"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// State is the vault the window is looking at and what every handler of it
// reads: the vault shown, the door on every question, the writes and tasks in
// the air, and who is drawing what changes.
type State struct {
	vault atomic.Pointer[domain.Vault]

	closed atomic.Bool

	questions inflight

	session atomic.Pointer[vaultSession]

	ChangeEvents broadcaster[window.Change]

	OpenVault func(context.Context, domain.Vault, *PluginOpen) error

	NotifyNoteWritten func()

	Edits broadcaster[domain.Edit]

	Places broadcaster[domain.Place]

	TabIDs broadcaster[string]

	openTabs atomic.Pointer[domain.OpenTabs]

	openTabsReports reportSignal

	TabActivationBound time.Duration

	NotifyOpenTabs func(domain.OpenTabs)

	DetachTab func(context.Context, window.DetachedTab) error

	Writing inflight

	Ready       atomic.Bool
	Error       wire.Reason
	WatchError  wire.Reason
	Unreachable wire.Reason

	Window *wire.Window

	EmitEvent func(event string)

	Indexing IndexState

	Playing atomic.Pointer[Loopback]
}

// IndexState is how far the vault has been read for meaning.
type IndexState struct {
	Embedding *indexing.Embedding
	Model     atomic.Value
	Recipe    atomic.Value
}

// storeModel records the name and the recipe of the model the vault is embedded under.
func (s *State) storeModel(embedding *indexing.Embedding) {
	name, recipe := embedding.Describe()
	s.Indexing.Model.Store(name)
	s.Indexing.Recipe.Store(recipe)
}

// GetCurrentVault is the vault the window has open. A window standing on nothing
// answers with no vault at all.
func (s *State) GetCurrentVault() domain.Vault {
	if v := s.vault.Load(); v != nil {
		return *v
	}
	return domain.Vault{}
}

// GetVaultState is what the window knows about the vault it shows: whether the first scan is done,
// what stopped it, and how far the vault has been read for meaning.
func (s *State) GetVaultState(ctx context.Context) window.VaultState {
	showing := s.GetCurrentVault()
	out := window.VaultState{
		Vault:       showing,
		IsReady:     s.Ready.Load(),
		Error:       s.Error.Why(),
		WatchError:  s.WatchError.Why(),
		IsEmbedding: text(&s.Indexing.Model) != "",
	}
	// A count that cannot be taken leaves the pair at nothing, and the rest of
	// the state is answered currently. A window standing on nothing holds no
	// chunks and counts none.
	if s.Indexing.Embedding.HasProgress() && showing.ID != "" {
		if target, embedded, err := s.Indexing.Embedding.CountProgress(ctx, showing.ID, text(&s.Indexing.Recipe)); err == nil {
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
	s.openTabs.Store(&open)
	s.openTabsReports.notify()
	if s.NotifyOpenTabs != nil {
		s.NotifyOpenTabs(open)
	}
}

// GetOpenTabs returns the open tabs as last reported by the window, or the
// empty set when the window has not reported any.
func (s *State) GetOpenTabs() domain.OpenTabs {
	if open := s.openTabs.Load(); open != nil {
		return *open
	}
	return domain.OpenTabs{}
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

func (s *State) setVault(v domain.Vault) { s.vault.Store(&v) }

func (s *State) setSession(on *vaultSession) { s.session.Store(on) }

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

func text(v *atomic.Value) string {
	s, isString := v.Load().(string)
	if !isString {
		return ""
	}
	return s
}

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
func (w *inflight) open()                 { w.Inflight.Open() }

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
