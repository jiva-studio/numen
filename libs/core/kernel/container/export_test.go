package container

import (
	"context"

	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	searchusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
)

// Settings reads the settings file; an unconfigured installation returns the default settings.
func (c Config) Settings() (settingsadapter.File, error) { return c.readSettingsFile() }

// SetSettings returns the configuration with the values of a settings file applied.
func (c Config) SetSettings(s settingsadapter.File) Config { return c.applySettingsFile(s) }

// DefaultSettings returns the settings of an unconfigured installation.
func DefaultSettings() settingsadapter.File { return defaultSettings() }

// Themes returns the themes service the window offers.
func (c Config) Themes(say func(string)) (*theme.Service, error) {
	return settingsThemes(c.settings(), say)
}

// The methods below are the wiring the tests of this package reach from outside it.

func (c Config) AgentProvider() agentport.AgentProvider { return c.providerOfAgents() }

func (c Config) CutNotes() chunking.CutNotes { return c.cutNotes() }

func (c Config) Embedder(ctx context.Context) (embeddingport.Embedder, func() error, error) {
	return c.embedder(ctx)
}

func (c Config) EmbedderReady() bool { return c.embedderReady() }

func (c Config) Embedders(ctx context.Context, tasks *run.Tasks) (indexing, asking embeddingport.Embedder, closer func() error, why error) {
	return c.embedders(ctx, tasks)
}

func (c Config) GetEditorAgent() agentport.Agent { return c.getEditorAgent() }

func (c Config) GetSettingsAgent() agentport.Agent { return c.getSettingsAgent() }

func (c Config) Models() func() []settingsdomain.Model { return c.models() }

func (c Config) NewSearchOver(passages sourceport.PassageQueries, asking embeddingport.Embedder, errorHandler func(error)) searchusecase.Search {
	return c.newSearchOver(passages, asking, errorHandler)
}

func (c Config) OpenGetSettings(provider agentport.AgentProvider) *settings.Get {
	return c.openGetSettings(provider)
}

func (c Config) OpenWatchSettings() *settings.WatchSettingChanges { return c.openWatchSettings() }

func (c Config) SubscribeSettingChanges() (<-chan []settingsdomain.Setting, func()) {
	return c.subscribeSettingChanges()
}

func (c Config) VaultOpenerWith(db *vault.Index, walking vaultport.VaultReaders, watcher vaultport.VaultWatcher) *vault.VaultOpener {
	return c.vaultOpenerWith(db, walking, watcher)
}

func (c Config) VaultWatcher() vaultport.VaultWatcher { return c.vaultWatcher() }
