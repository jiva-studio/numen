package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// mountVault mounts the service that answers what the vault in the window is doing.
func mountVault(w *windowWiring) { w.inst.Mount(editor.NewVaultRoute(w.inst.State)) }

// getWindowSources are the passes behind a vault of the window, over this installation's walk and
// watch, text versions, plugins and cache.
func getWindowSources(cfg Config, db *Index) editor.Sources {
	return editor.Sources{
		ReadVault: func(
			ctx context.Context, embedder embeddingport.Embedder, v domain.Vault, rebuild bool,
		) (vault.ReadWholeVault, error) {
			request := cfg
			request.ShouldRebuildIndex = rebuild
			return request.ReadWholeVault(ctx, db, embedder, v)
		},
		StartVault:             cfg.startVault(db),
		SetTextVersionListener: cfg.vault().SetTextVersionListener,
		ReportTextFilesChanged: cfg.vault().ReportTextFilesChanged,
		OpenTriggers:           cfg.plugins().OpenTriggers,
		WatchPluginChanges:     cfg.WatchPluginChanges,
		GetRefresh:             func() vault.Refresh { return cfg.GetRefresh(db) },
		CloseIndex:             db.Close,
		GetProblemQueries:      db.Problems,
		GetPassageQueries:      db.Passages,
	}
}

func (c Config) startVault(db *Index) editor.StartVaultFunc {
	return func(
		ctx context.Context, v domain.Vault, rebuild bool,
		notified func(paths, assets []string, reload bool),
		handleError window.ErrorHandler,
	) (
		read func(ctx context.Context, during func()) (notes int, err error),
		run func(ctx context.Context),
		refresh vault.Refresh,
		unwatched error,
	) {
		request := c
		request.ShouldRebuildIndex = rebuild

		opening := request.VaultOpener(db)
		opening.ShouldRebuild = rebuild
		opening.ErrorHandler = handleError
		if notified != nil {
			opening.Told = func(m vault.VaultChanges) { notified(m.Paths, m.Assets, m.ShouldReload) }
		}

		open := opening.Begin(ctx, v)
		return open.Read, open.Run, opening.GetRefresh(), open.GetUnwatchedReason()
	}
}
