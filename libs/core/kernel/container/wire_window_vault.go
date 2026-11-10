package container

import (
	"context"
	"io"
	"sync/atomic"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	windowvault "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func mountVault(w *windowWiring) {
	state := w.inst.State
	handler := windowvault.NewHandler(state)
	path, served := windowvault.NewRoute(handler, state.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}

func getReadVault(cfg Config, db *vault.Index) indexing.OpenReading {
	return func(
		ctx context.Context, embedder embeddingport.Embedder, v domain.Vault, rebuild bool,
	) (indexing.ReadWholeVault, error) {
		request := cfg
		request.ShouldRebuildIndex = rebuild
		return request.ReadWholeVault(ctx, db, embedder, v)
	}
}

func newIndexWorker(w *windowWiring, state *editor.State, embedding *indexing.Embedding, out io.Writer) *indexing.Worker {
	return indexing.NewWorker(indexing.Config{
		Reading:  getReadVault(w.cfg, w.db),
		Embedder: embedding.GetEmbedder(),
		VectorCounts: func(ctx context.Context, v domain.Vault) (held, embedded int64, err error) {
			if !embedding.HasProgress() {
				return 0, 0, indexing.ErrNoProgress
			}
			name, recipe := embedding.Describe()
			state.RecordModel(name, recipe)
			return embedding.CountProgress(ctx, v.ID, recipe)
		},
		VectorsReady: state.EmitVectorsReady,
		Events:       windowsession.NewIndexEvents(&state.Runtime, state.Window, out),
		Quiet:        indexing.QuietPeriod,
		After:        time.After,
	})
}

func newIndexSession(worker *indexing.Worker) func() windowsession.IndexSession {
	return func() windowsession.IndexSession {
		session := worker.NewSession()
		return windowsession.IndexSession{
			Run: func(ctx context.Context, v domain.Vault, rebuild bool, read windowsession.ReadNotes) {
				session.Run(ctx, v, rebuild, read)
			},
			ReportVersionsChanged: session.ReportVersionsChanged,
		}
	}
}

func (c Config) getTextVersions() windowsession.TextVersions {
	if versions := c.vault().TextVersions; versions != nil {
		return versions.Changes
	}
	return nil
}

func (c Config) getTriggerSource() windowsession.TriggerSource {
	return windowsession.TriggerSource{
		Open: func() windowsession.TriggerRunner {
			if triggers := openTriggers(c.plugins()); triggers != nil {
				return triggers
			}
			return nil
		},
		Watch: c.WatchPluginChanges,
	}
}

func (c Config) startVault(db *vault.Index, refreshing *atomic.Pointer[vault.Refresh]) windowsession.VaultPassesFunc {
	return func(
		ctx context.Context, v domain.Vault, rebuild bool, report windowsession.Reporters,
	) windowsession.Started {
		request := c
		request.ShouldRebuildIndex = rebuild

		opening := request.VaultOpener(db)
		opening.ShouldRebuild = rebuild
		opening.ErrorHandler = report.Failed
		if report.Changed != nil {
			opening.Told = func(m vault.VaultChanges) { report.Changed(m.Paths, m.Assets, m.ShouldReload) }
		}

		open := opening.Begin(ctx, v)
		refresh := opening.GetRefresh()
		return windowsession.Started{
			Read:      open.Read,
			Run:       open.Run,
			Unwatched: open.GetUnwatchedReason(),
			Publish:   func() { refreshing.Store(&refresh) },
			Retire:    func() { refreshing.Store(nil) },
		}
	}
}
