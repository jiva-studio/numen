package container

import (
	"context"

	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/absent"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	searchusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
)

// Settings reads the settings file; an unconfigured installation returns the default settings.
func (c Config) Settings() (settingsadapter.File, error) { return settingsSettings(settingsOf(c)) }

// SetSettings returns the configuration with the values of a settings file applied.
func (c Config) SetSettings(s settingsadapter.File) Config { return applySettingsFile(c, s) }

// DefaultSettings returns the settings of an unconfigured installation.
func DefaultSettings() settingsadapter.File { return defaultSettings() }

// Themes returns the themes service the window offers.
func (c Config) Themes(say func(string)) (*theme.Service, error) {
	return settingsThemes(settingsOf(c), Scales{}, say)
}

// The methods below are the wiring the tests of this package reach from outside it.

func (c Config) AgentProvider() agentport.AgentProvider { return pluginsAgentProvider(plugins(c)) }

func (c Config) CutNotes() chunking.CutNotes { return cutNotes(searchOf(c)) }

func (c Config) Embedder(ctx context.Context) (searchsourceport.Embedder, func() error, error) {
	return searchEmbedder(searchOf(c), ctx)
}

func (c Config) EmbedderReady() bool { return embedderReady(searchOf(c)) }

func (c Config) Embedders(ctx context.Context, tasks *run.Tasks) (indexing, asking searchsourceport.WatchedEmbedder, closer func() error, why error) {
	return searchEmbedders(searchOf(c), ctx, tasks)
}

func (c Config) GetEditorAgent() agentport.Agent { return getToolProvider(plugins(c)) }

func (c Config) GetSettingsAgent() agentport.Agent { return getSettingsAgent(plugins(c)) }

func (c Config) Models() func() []settingsdomain.Model { return models(c) }

func (c Config) NewSearchOver(index searchusecase.Index, asking searchsourceport.Embedder, errorHandler func(error)) searchusecase.Search {
	return newSearchOver(searchOf(c), index, asking, errorHandler)
}

func (c Config) OpenGetSettings(provider agentport.AgentProvider) *settings.Get {
	return openGetSettings(c, provider)
}

func (c Config) OpenWatchSettings() *settings.WatchSettingChanges {
	return openWatchSettings(settingsOf(c))
}

func (c Config) SubscribeSettingChanges() (<-chan []settingsdomain.Setting, func()) {
	return subscribeSettingChanges(settingsOf(c))
}

func (c Config) VaultOpenerWith(db *Index, walking vaultport.VaultReaders, watcher vaultport.VaultWatcher) *vault.VaultOpener {
	return vaultOpenerWith(vaultOf(c), db, walking, watcher)
}

func (c Config) VaultWatcher() vaultport.VaultWatcher { return vaultWatcher(vaultOf(c)) }

// AbsentBinding is the binding of a plugin host that runs nothing.
func AbsentBinding() pluginport.PluginBinding { return absent.Runtime{} }

// NotesShowingMoves is Notes where each move of a note in the vault the window shows also reaches view, so a
// view showing a note under its old name switches to the new name.
func NotesShowingMoves(
	c Config, db *Index, index edit.IndexUpdater, showing func() (domain.Vault, bool), view note.MoveView,
) NoteUseCases {
	return vaultNotes(vaultOf(c), db, index, noteViews{Moves: note.NewTellMoveInShownVault(showing, view), Writes: note.OpenGate{}})
}

func embedderReady(c searchWiring) bool {
	_, err := c.RunningHost.Runtime.ResolveCapability(bridge.CapabilitySearchSource)
	return err == nil
}

func notifyHost(c settingsWiring, batch []settingsdomain.Setting) { notifyPlugins(c).Execute(batch) }

func settingsGetSettingsAt(c settingsWiring, path string) (settingsadapter.File, error) {
	return settingsStore(c).ReadUncheckedAt(path)
}

func fileMovers(c vaultWiring) vaultport.FileMovers { return vaultWriters(c) }
