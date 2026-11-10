package editor

import (
	"context"
	"testing"

	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	searchwiring "github.com/jiva-studio/numen/libs/core/kernel/container/search"
	settingswiring "github.com/jiva-studio/numen/libs/core/kernel/container/settings"
	vaultwiring "github.com/jiva-studio/numen/libs/core/kernel/container/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/pluginchanges"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/settingevents"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/secret"
	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// These tests put a window together over parts of their own, because the composition root imports
// this package and cannot be imported by its tests.

// wiring is the parts of an installation that read a vault into a cache.
type wiring struct {
	vaults vaultwiring.Config
	search searchwiring.Config
}

func newWiring(indexPath string, shouldRebuild bool, texts textport.TextReaders) wiring {
	timer := clock.System{}
	readers := vaultreader.Readers{}
	search := searchwiring.Config{
		ShouldRebuildIndex: shouldRebuild,
		Chunks:             &note.ChunkTokens{},
		Clock:              timer,
		VaultReaders:       readers,
		TextReaders:        texts,
	}
	return wiring{
		vaults: vaultwiring.Config{
			IndexPath:   indexPath,
			Clock:       timer,
			TextReaders: texts,
			VaultLocks:  vaultwriter.NewLocks(),
			Watches:     vaultwatch.NewHub(),
			CutNotes:    search.CutNotes(),
			SetNoteCut:  search.SetNoteCut,
		},
		search: search,
	}
}

func (w wiring) openIndex(t *testing.T) *vaults.Index {
	t.Helper()
	db, err := w.vaults.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	return db
}

func (w wiring) readVault(db *vaults.Index) ReadVaultFunc {
	return testPasses{wiring: w, db: db}.ReadWholeVault
}

func (w wiring) session(db *vaults.Index, readers vaultport.VaultReaders, watcher vaultport.VaultWatcher) SessionSource {
	passes := testPasses{wiring: w, db: db, readers: readers, watcher: watcher}
	return SessionSource{
		StartVault:             passes.StartVault,
		SetTextVersionListener: func(textport.VersionListener) {},
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
) (vaultreading.ReadWholeVault, error) {
	request := p.search
	request.ShouldRebuildIndex = rebuild
	return request.ReadWholeVault(ctx, p.db, embedder, v, p.vaults.MakeRefresh(p.db, p.db.Notes()))
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
	opening := p.vaults.VaultOpenerWith(p.db, p.readers, p.watcher)
	opening.ShouldRebuild = rebuild
	opening.ErrorHandler = handleError
	if notified != nil {
		opening.Told = func(m vaults.VaultChanges) { notified(m.Paths, m.Assets, m.ShouldReload) }
	}

	open := opening.Begin(ctx, v)
	return open.Read, open.Run, opening.GetRefresh(), open.GetUnwatchedReason()
}

func (p testPasses) GetRefresh() vaults.Refresh { return p.vaults.MakeRefresh(p.db, p.db.Notes()) }

// themeConfig is where the themes and the settings of an installation are kept.
type themeConfig struct {
	RegistryPath              string
	InterfaceScale, TextScale float64
}

func (c themeConfig) themes(say func(string)) (*theme.Service, error) {
	return settingswiring.Config{
		RegistryPath:    c.RegistryPath,
		InterfaceScale:  c.InterfaceScale,
		TextScale:       c.TextScale,
		Watches:         vaultwatch.NewHub(),
		SettingsFiles:   settingsadapter.NewFiles(),
		SettingEvents:   settingevents.New(),
		SecretGate:      secret.NewGate(),
		SubscribedHosts: &settingsusecase.HostSubscriptions{},
		PluginChanges:   &pluginchanges.Signals{},
		Programs:        &settingsadapter.ProgramList{},
	}.Themes(say)
}
