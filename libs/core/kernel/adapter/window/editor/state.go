// Package editor serves the vault queries and commands of the editor window.
//
// The request and response types are defined in libs/core/protocol. The client
// connects to the handlers over HTTP in both the desktop window and a browser
// during development.
package editor

import (
	"context"
	"sync/atomic"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// State is what the handlers of the window share: the vault shown and how far it is read, the door on
// every question, the writes taken, and who is drawing what changes. Each part answers for one role.
type State struct {
	Runtime windowsession.Runtime
	Tabs    windowsession.TabReports

	Gate    *wire.Gate
	Writes  *wire.Writes
	Notices *Notices
	Events  *Events

	NotifyOpenTabs func(window.OpenTabs)

	DetachTab func(context.Context, window.DetachedTab) error

	Window *wire.Window

	Indexing IndexState

	Playing atomic.Pointer[Loopback]
}

// IndexState is the search source of the window and how far it has indexed the vault.
type IndexState struct {
	Embedding *indexing.Embedding
}

// Frame is the application window around the editor.
type Frame interface {
	// ShowOpenTabs tells the frame which tabs are open, so that its title follows them.
	ShowOpenTabs(open window.OpenTabs)
	// DetachTab opens a window of its own for a tab.
	DetachTab(ctx context.Context, tab window.DetachedTab) error
}

// NewState makes the state of a window that shows no vault. frame is the application window around the
// editor; a nil frame is a window with no application around it, which opens no window for a tab. emit
// tells the plugins an event of the window; a nil emit tells no one.
func NewState(tasks *run.Tasks, frame Frame, emit func(event string)) *State {
	tasked := &wire.Window{Name: wire.Editor, Tasks: tasks}
	state := &State{
		Gate:    &wire.Gate{},
		Writes:  wire.NewWrites(tasked),
		Notices: newNotices(),
		Events:  &Events{Emit: emit},
		Window:  tasked,
	}
	if frame != nil {
		state.NotifyOpenTabs, state.DetachTab = frame.ShowOpenTabs, frame.DetachTab
	}
	// The editor is open on one vault, and answers which as itself.
	state.Window.Vault = func() string { return string(state.GetCurrentVault().ID) }
	return state
}

// GetCurrentVault is the vault the window shows.
func (s *State) GetCurrentVault() domain.Vault { return s.Runtime.GetCurrentVault() }

// GetShownVault is the vault the window shows, and false while it shows none.
func (s *State) GetShownVault() (domain.Vault, bool) {
	current := s.GetCurrentVault()
	return current, current.Path != ""
}

// SetIndexing sets the search source of the window and records its name and recipe.
// It is set before any chunk is read: the recipe determines whether a chunk already
// carries a vector.
func (s *State) SetIndexing(embedding *indexing.Embedding) {
	s.Indexing.Embedding = embedding
	s.Runtime.RecordModel(embedding.Describe())
}

// ReportWordsAlone records that the search runs on words only, with the reason. It is a task because
// a window opened from a desktop entry has no terminal.
func (s *State) ReportWordsAlone(err error) {
	s.Window.AddTask(work.Task{ID: work.WordsAlone, Title: "Search by words only", Error: err.Error()})
}

// ClearWordsAlone removes the words-only task once a search source answers again.
func (s *State) ClearWordsAlone() {
	s.Window.RemoveTask(work.WordsAlone)
}

// ReportFloorError records a similarity floor that is stored and cannot be read.
func (s *State) ReportFloorError(err error) {
	s.Window.AddTask(work.Task{ID: readingTheFloor, Title: "Read search floor", Error: err.Error()})
}

// GetVaultState is what the window knows about the vault it shows: whether the first scan is done and
// what stopped it.
func (s *State) GetVaultState() window.VaultState {
	return windowsession.GetVaultState(&s.Runtime)
}

// ReportOpenTabs keeps the tabs the window reported, wakes whoever waits for a report, and tells
// the application.
func (s *State) ReportOpenTabs(open window.OpenTabs) {
	s.Tabs.ReportOpenTabs(open)
	if s.NotifyOpenTabs != nil {
		s.NotifyOpenTabs(open)
	}
}

// ShowOpenTabs tells the application the tabs that are open now.
func (s *State) ShowOpenTabs() {
	if s.NotifyOpenTabs != nil {
		s.NotifyOpenTabs(s.Tabs.GetOpenTabs())
	}
}

func (s *State) CanOpenTabWindow() bool { return s.DetachTab != nil }

func (s *State) OpenTabWindow(ctx context.Context, tab window.DetachedTab) error {
	return s.DetachTab(ctx, tab)
}

// ClosePlaying stops the socket recordings are played over.
func (s *State) ClosePlaying(ctx context.Context) error { return s.Playing.Load().Close(ctx) }

// GetMediaAddress is where the file with this stat is loaded from, and nowhere while the socket
// does not answer.
func (s *State) GetMediaAddress(v domain.Vault, stat domain.FileStat) string {
	return s.Playing.Load().Address(v, stat)
}

// GetPluginFileAddress is where a file of a plugin's store of the vault is loaded from, and nowhere
// while the socket does not answer.
func (s *State) GetPluginFileAddress(v domain.Vault, address file.Address, size int64) string {
	return s.Playing.Load().PluginFileAddress(v, address, size)
}
