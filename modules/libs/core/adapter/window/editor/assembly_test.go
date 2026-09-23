package editor

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// NewAssembly is the window's assembly over a configuration, bound the way an
// application binds one. The index it opens is the window's to close.
func NewAssembly(t *testing.T, cfg container.Config) Assembly {
	t.Helper()

	made, err := cfg.GetEditorAssembly(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// watched is the window's assembly with the walk and the watch a test drives,
// in place of the ones an installation is bound to. Everything else is the
// installation's own.
type watched struct {
	container.EditorAssembly

	cfg     container.Config
	db      *container.Index
	readers port.VaultReaders
	watcher port.VaultWatcher
}

// newWatched is that assembly over an index and the pieces a test drives.
func newWatched(
	cfg container.Config, db *container.Index,
	readers port.VaultReaders, watcher port.VaultWatcher,
) watched {
	return watched{
		EditorAssembly: container.NewEditorAssembly(cfg, nil, db),
		cfg:            cfg,
		db:             db,
		readers:        readers,
		watcher:        watcher,
	}
}

func (w watched) StartVault(
	ctx context.Context, v domain.Vault, rebuild bool,
	told func(paths, assets []string, reload bool),
	handleError port.ErrorHandler,
) (
	read func(ctx context.Context, during func()) (notes int, err error),
	run func(ctx context.Context),
	refresh vaults.Refresh,
	unwatched error,
) {
	asked := w.cfg
	asked.ShouldRebuildIndex = rebuild

	opening := asked.VaultOpenerWith(w.db, w.readers, w.watcher)
	opening.ShouldRebuild = rebuild
	opening.ErrorHandler = handleError
	if told != nil {
		opening.Told = func(m container.VaultChanges) { told(m.Paths, m.Assets, m.ShouldReload) }
	}

	open := opening.Begin(ctx, v)
	return open.Read, open.Run, opening.GetRefresh(), open.GetUnwatchedReason()
}
