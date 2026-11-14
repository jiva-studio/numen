package windowsession

import (
	"context"

	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Pages are the clients of the window that hold unsaved work.
type Pages interface {
	// RequestFlush has every page write what only it holds and waits for the request to end. It
	// returns false when a page holds work awaiting a decision of the user.
	RequestFlush(ctx context.Context) bool
	CancelFlush()
	IsQuestionOpen() bool
}

// Writes is the writes taken and not yet finished.
type Writes interface {
	// BeginWrite registers a write and reports false once the window is closing, when nothing may be written.
	BeginWrite() bool
	// FinishWrite ends a write that BeginWrite registered.
	FinishWrite()
	// Seal refuses new writes and answers with what closes once the writes taken have finished.
	Seal() <-chan struct{}
	// Open accepts writes again.
	Open()
}

// Tasks is the list of what the window is doing.
type Tasks interface {
	AddTask(task work.Task)
	RemoveTask(id string)
}

// Changes tells the views drawing the vault what changed.
type Changes interface {
	Publish(change window.Change)
}

// IdentityCheck refuses a vault whose folder does not carry its identity.
type IdentityCheck interface {
	Execute(v domain.Vault) error
}

// OpenRecord remembers the vault the window opened last.
type OpenRecord interface {
	Execute(id domain.VaultID) error
}

// TabFront asks the clients to bring a tab to the front.
type TabFront interface {
	Publish(id string)
}

// Door is what refuses every question the window is asked once it closes.
type Door interface {
	Close()
}

// Closer ends one resource of the window. The context bounds a close that waits.
type Closer interface {
	Close(ctx context.Context) error
}

type CloseFunc func(ctx context.Context) error

// Close calls f.
func (f CloseFunc) Close(ctx context.Context) error { return f(ctx) }

// ModelSource is the embedder of the window: the model it reads under and its announcements.
type ModelSource interface {
	Describe() (name, recipe string)
	OnModelChange(fn func(isChunkSizeChanged bool))
}

// IndexWake wakes the worker that indexes the vaults of the window.
type IndexWake interface {
	WakeNotes()
	WakeSources()
	ClearSkips(paths []string)
}

// ReadNotes reads the notes of a vault once and returns what the walk did.
type ReadNotes = func(ctx context.Context, during func()) (vaultreading.ScanResult, error)

// IndexSession is the pass that reads one vault into the index and cuts again what it is asked to.
type IndexSession struct {
	Run                   func(ctx context.Context, v domain.Vault, rebuild bool, read ReadNotes)
	ReportVersionsChanged func(v domain.Vault, path string)
}

// TextVersions keeps the live versions of the texts of files.
type TextVersions interface {
	SetListener(listener textdomain.VersionListener)
	// ReportFilesChanged reports to the listener each changed file of the vault whose live text
	// version changed.
	ReportFilesChanged(ctx context.Context, v domain.Vault, paths []string) error
}

// Reporters are what the passes behind a vault call back.
type Reporters struct {
	Changed func(paths, assets []string, reload bool)
	Failed  window.ErrorHandler
}

// Started is the passes behind one vault. Publish makes the refresh of this run the one requests
// use, and Retire ends it.
type Started struct {
	Read      ReadNotes
	Run       func(ctx context.Context)
	Unwatched error
	Publish   func()
	Retire    func()
}

// VaultPasses starts the walk and the watch of one vault.
type VaultPasses interface {
	Start(ctx context.Context, v domain.Vault, rebuild bool, report Reporters) Started
}

type VaultPassesFunc func(ctx context.Context, v domain.Vault, rebuild bool, report Reporters) Started

// Start calls f.
func (f VaultPassesFunc) Start(ctx context.Context, v domain.Vault, rebuild bool, report Reporters) Started {
	return f(ctx, v, rebuild, report)
}

// TriggerRunner runs the triggers of the plugins over a vault.
type TriggerRunner interface {
	ExecuteAll(ctx context.Context, v domain.Vault) error
	Execute(ctx context.Context, v domain.Vault, paths []string) error
}

// TriggerSource is where the passes of the triggers come from. Open returns nil where the window runs
// no plugins. Watch reports each time the plugin host applied settings or forgot the failure of a
// plugin, and the function it returns stops the reports.
type TriggerSource struct {
	Open  func() TriggerRunner
	Watch func() (<-chan struct{}, func())
}

type currentVault interface {
	GetCurrentVault() domain.Vault
}
