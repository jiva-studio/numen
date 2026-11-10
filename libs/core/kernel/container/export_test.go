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
func (c Config) Settings() (settingsadapter.File, error) { return settingsSettings(c.settings()) }

// SetSettings returns the configuration with the values of a settings file applied.
func (c Config) SetSettings(s settingsadapter.File) Config { return applySettingsFile(c, s) }

// DefaultSettings returns the settings of an unconfigured installation.
func DefaultSettings() settingsadapter.File { return defaultSettings() }

// Themes returns the themes service the window offers.
func (c Config) Themes(say func(string)) (*theme.Service, error) {
	return settingsThemes(settingsOf(c), say)
}

// The methods below are the wiring the tests of this package reach from outside it.

func (c Config) AgentProvider() agentport.AgentProvider { return pluginsAgentProvider(c.plugins()) }

func (c Config) CutNotes() chunking.CutNotes { return cutNotes(c.search()) }

func (c Config) Embedder(ctx context.Context) (embeddingport.Embedder, func() error, error) {
	return searchEmbedder(c.search(), ctx)
}

func (c Config) EmbedderReady() bool { return embedderReady(c.search()) }

func (c Config) Embedders(ctx context.Context, tasks *run.Tasks) (indexing, asking embeddingport.Embedder, closer func() error, why error) {
	return searchEmbedders(c.search(), ctx, tasks)
}

func (c Config) GetEditorAgent() agentport.Agent { return getToolProvider(c.plugins()) }

func (c Config) GetSettingsAgent() agentport.Agent { return getSettingsAgent(c.plugins()) }

func (c Config) Models() func() []settingsdomain.Model { return models(c) }

func (c Config) NewSearchOver(passages sourceport.PassageQueries, asking embeddingport.Embedder, errorHandler func(error)) searchusecase.Search {
	return newSearchOver(c.search(), passages, asking, errorHandler)
}

func (c Config) OpenGetSettings(provider agentport.AgentProvider) *settings.Get {
	return openGetSettings(c, provider)
}

func (c Config) OpenWatchSettings() *settings.WatchSettingChanges {
	return openWatchSettings(c.settings())
}

func (c Config) SubscribeSettingChanges() (<-chan []settingsdomain.Setting, func()) {
	return subscribeSettingChanges(c.settings())
}

func (c Config) VaultOpenerWith(db *vault.Index, walking vaultport.VaultReaders, watcher vaultport.VaultWatcher) *vault.VaultOpener {
	return vaultOpenerWith(c.vault(), db, walking, watcher)
}

func (c Config) VaultWatcher() vaultport.VaultWatcher { return vaultWatcher(c.vault()) }
