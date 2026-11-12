package editor

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/theme"
	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// wiring is the parts of an installation that read a vault into a cache.
type wiring struct {
	reading readingParts
}

func newWiring(indexPath string, shouldRebuild bool, texts textdomain.Readers) wiring {
	return wiring{reading: newReadingParts(indexPath, shouldRebuild, texts)}
}

func (w wiring) openIndex(t *testing.T) *testIndex {
	t.Helper()
	db, err := w.reading.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	return db
}

func (w wiring) readVault(db *testIndex) indexing.OpenReading {
	return testPasses{wiring: w, db: db}.ReadWholeVault
}

// sessionOptions is what a test changes about the session it starts.
type sessionOptions struct {
	embedding *indexing.Embedding
	idle      time.Duration
	rebuild   bool
	out       io.Writer
}

type noRecord struct{}

func (noRecord) Execute(domain.VaultID) error { return nil }

type openDoor struct{}

func (openDoor) Close() {}

func (w wiring) startSession(
	t *testing.T, ctx context.Context, db *testIndex, state *State, v domain.Vault,
	readers vaultport.VaultReaders, watcher vaultport.VaultWatcher, opts sessionOptions,
) (end func()) {
	t.Helper()
	embedding := opts.embedding
	worker := indexing.NewWorker(indexing.Config{
		Reading:  w.readVault(db),
		Embedder: embedding.GetEmbedder(),
		VectorCounts: func(ctx context.Context, v domain.Vault) (held, embedded int64, err error) {
			if !embedding.HasProgress() {
				return 0, 0, indexing.ErrNoProgress
			}
			name, recipe := embedding.Describe()
			state.Runtime.RecordModel(name, recipe)
			return embedding.CountProgress(ctx, v.ID, recipe)
		},
		SearchSourceReady: state.Events.EmitSearchSourceReady,
		Events:       windowsession.NewIndexEvents(&state.Runtime, state.Window, opts.out),
		Quiet:        opts.idle,
		After:        time.After,
	})
	state.Writes.NotifyNoteWritten = worker.WakeNotes
	passes := testPasses{wiring: w, db: db, readers: readers, watcher: watcher}
	diag := windowsession.NewDiagnostics(opts.out, state.Window)
	runner := windowsession.NewPassRunner(windowsession.PassRunnerConfig{
		Failures: &state.Runtime, Diag: diag,
		NewSession: func() windowsession.IndexSession {
			indexed := worker.NewSession()
			return windowsession.IndexSession{
				Run: func(ctx context.Context, v domain.Vault, rebuild bool, read windowsession.ReadNotes) {
					indexed.Run(ctx, v, rebuild, read)
				},
				ReportVersionsChanged: indexed.ReportVersionsChanged,
			}
		},
		Reporter: windowsession.NewChangeReporter(&state.Notices.ChangeEvents, worker, nil, diag),
		Passes:   windowsession.VaultPassesFunc(passes.StartVault),
	})
	session := windowsession.NewSession(&state.Runtime, noRecord{}, diag, nil, runner)
	guard := &windowsession.Guard{}
	rebuild := windowsession.NewRebuildSession(guard, session)
	start := windowsession.NewStartWindow(diag, nil, windowsession.NewModelFollow(nil, worker, &state.Runtime, rebuild), session, opts.rebuild)
	closeWindow := windowsession.NewCloseWindow(openDoor{}, session)
	start.Execute(ctx, v)
	return func() {
		if err := closeWindow.Execute(ctx); err != nil {
			t.Error(err)
		}
	}
}

type testPasses struct {
	wiring
	db      *testIndex
	readers vaultport.VaultReaders
	watcher vaultport.VaultWatcher
}

func (p testPasses) ReadWholeVault(
	ctx context.Context, embedder embeddingport.Embedder, v domain.Vault, rebuild bool,
) (indexing.ReadWholeVault, error) {
	return p.reading.ReadWholeVault(ctx, p.db, embedder, v, rebuild)
}

func (p testPasses) StartVault(
	ctx context.Context, v domain.Vault, rebuild bool, report windowsession.Reporters,
) windowsession.Started {
	opening := p.reading.OpenVaultWith(p.db, p.readers, p.watcher)
	opening.ShouldRebuild = rebuild
	var changed func(vaults.VaultChanges)
	if report.Changed != nil {
		changed = func(m vaults.VaultChanges) { report.Changed(m.Paths, m.Assets, m.ShouldReload) }
	}

	open := opening.Begin(ctx, v, changed, report.Failed)
	return windowsession.Started{Read: open.Read, Run: open.Run, Unwatched: open.GetUnwatchedReason()}
}

// themeConfig is where the themes and the settings of an installation are kept.
type themeConfig struct {
	RegistryPath              string
	InterfaceScale, TextScale float64
}

func (c themeConfig) themes(say func(string)) (*theme.Service, error) {
	return openThemes(c.RegistryPath, c.InterfaceScale, c.TextScale, say)
}
