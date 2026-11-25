package windowsession

import (
	"context"

	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
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
	WriteGate
	// Seal refuses new writes and answers with what closes once the writes taken have finished.
	Seal() <-chan struct{}
	// Open accepts writes again.
	Open()
}

// Changes tells the views drawing the vault what changed.
type Changes interface {
	Publish(change window.Change)
}

// TabFront asks the clients to bring a tab to the front.
type TabFront interface {
	Publish(id string)
}

// Door is what refuses every question the window is asked once it closes.
type Door interface {
	Close()
}

// CloseFunc ends one resource of the window. The context bounds a close that waits.
type CloseFunc func(ctx context.Context) error

// ModelSource is the embedder of the window: the model it reads under and its announcements.
type ModelSource interface {
	Describe() (name, recipe string)
	SubscribeModelChange(fn func(isChunkSizeChanged bool))
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
	Run                   func(ctx context.Context, rebuild bool, read ReadNotes)
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
	NotifyChanged func(paths, assets []string, reload bool)
	Failed        window.ErrorHandler
}

// StartedPasses is the passes behind one vault. Publish makes the refresh of this run the one requests
// use, and Retire ends it.
type StartedPasses struct {
	Read      ReadNotes
	Run       func(ctx context.Context)
	Unwatched error
	Publish   func()
	Retire    func()
}

// VaultPasses starts the walk and the watch of one vault.
type VaultPasses interface {
	Start(ctx context.Context, v domain.Vault, rebuild bool, report Reporters) StartedPasses
}

type VaultPassesFunc func(ctx context.Context, v domain.Vault, rebuild bool, report Reporters) StartedPasses

// Start calls f.
func (f VaultPassesFunc) Start(ctx context.Context, v domain.Vault, rebuild bool, report Reporters) StartedPasses {
	return f(ctx, v, rebuild, report)
}

// TriggerRunner runs the triggers of the plugins over a vault.
type TriggerRunner interface {
	ExecuteAll(ctx context.Context, v domain.Vault) error
	Execute(ctx context.Context, v domain.Vault, paths []string) error
	ExecuteFrontmatter(ctx context.Context, v domain.Vault, paths []string) ([]string, error)
}

// TriggerSource is where the passes of the triggers come from. Open returns nil where the window runs
// no plugins. Watch reports each time the plugin host applied settings or forgot the failure of a
// plugin, and the function it returns stops the reports. WatchFrontmatter, which may be nil, hands report
// the paths of the notes of the vault whose frontmatter the index stored changed, until the function it
// returns is called.
type TriggerSource struct {
	Open             func() TriggerRunner
	Watch            func() (<-chan struct{}, func())
	WatchFrontmatter func(id domain.VaultID, report func(paths []string)) func()
}
