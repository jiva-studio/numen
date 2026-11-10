package editor

import (
	"context"
	"testing"

	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// wiring is the parts of an installation that read a vault into a cache.
type wiring struct {
	reading readingParts
}

func newWiring(indexPath string, shouldRebuild bool, texts textdomain.Readers) wiring {
	return wiring{reading: newReadingParts(indexPath, shouldRebuild, texts)}
}

func (w wiring) openIndex(t *testing.T) *vaults.Index {
	t.Helper()
	db, err := w.reading.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	return db
}

func (w wiring) readVault(db *vaults.Index) indexing.OpenReading {
	return testPasses{wiring: w, db: db}.ReadWholeVault
}

func (w wiring) session(db *vaults.Index, readers vaultport.VaultReaders, watcher vaultport.VaultWatcher) SessionSource {
	passes := testPasses{wiring: w, db: db, readers: readers, watcher: watcher}
	return SessionSource{
		StartVault:             passes.StartVault,
		SetTextVersionListener: func(textdomain.VersionListener) {},
		ReportTextFilesChanged: func(context.Context, domain.Vault, []string) error { return nil },
		GetRefresh:             passes.GetRefresh,
	}
}

type testPasses struct {
	wiring
	db      *vaults.Index
	readers vaultport.VaultReaders
	watcher vaultport.VaultWatcher
}

func (p testPasses) ReadWholeVault(
	ctx context.Context, embedder embeddingport.Embedder, v domain.Vault, rebuild bool,
) (indexing.ReadWholeVault, error) {
	return p.reading.ReadWholeVault(ctx, p.db, embedder, v, rebuild)
}

func (p testPasses) StartVault(
	ctx context.Context, v domain.Vault, rebuild bool,
	notified func(paths, assets []string, reload bool),
	handleError window.ErrorHandler,
) (
	read func(ctx context.Context, during func()) (notes int, err error),
	run func(ctx context.Context),
	refresh vaults.Refresh,
	unwatched error,
) {
	opening := p.reading.OpenVaultWith(p.db, p.readers, p.watcher)
	opening.ShouldRebuild = rebuild
	opening.ErrorHandler = handleError
	if notified != nil {
		opening.Told = func(m vaults.VaultChanges) { notified(m.Paths, m.Assets, m.ShouldReload) }
	}

	open := opening.Begin(ctx, v)
	return open.Read, open.Run, opening.GetRefresh(), open.GetUnwatchedReason()
}

func (p testPasses) GetRefresh() vaults.Refresh { return p.reading.GetRefresh(p.db) }

// themeConfig is where the themes and the settings of an installation are kept.
type themeConfig struct {
	RegistryPath              string
	InterfaceScale, TextScale float64
}

func (c themeConfig) themes(say func(string)) (*theme.Service, error) {
	return openThemes(c.RegistryPath, c.InterfaceScale, c.TextScale, say)
}
