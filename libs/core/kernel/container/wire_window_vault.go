package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	windowvault "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func mountVault(w *windowWiring) {
	state := w.inst.State
	handler := windowvault.NewHandler(state)
	path, served := windowvault.NewRoute(handler, state.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}

func getReadVault(cfg Config, db *Index) editor.ReadVaultFunc {
	return func(
		ctx context.Context, embedder embeddingport.Embedder, v domain.Vault, rebuild bool,
	) (vault.ReadWholeVault, error) {
		request := cfg
		request.ShouldRebuildIndex = rebuild
		return request.ReadWholeVault(ctx, db, embedder, v)
	}
}

func getWindowSession(cfg Config, db *Index) editor.SessionSource {
	return editor.SessionSource{
		StartVault:             cfg.startVault(db),
		SetTextVersionListener: cfg.vault().SetTextVersionListener,
		ReportTextFilesChanged: cfg.vault().ReportTextFilesChanged,
		GetRefresh:             func() vault.Refresh { return cfg.GetRefresh(db) },
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
