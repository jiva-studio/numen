package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/none"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostvault"
	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/naming"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/absent"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/filetext"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	searchusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	vectorscase "github.com/jiva-studio/numen/libs/core/kernel/usecase/vectors"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
)

// Settings reads the settings file; an unconfigured installation returns the default settings.
func (c Config) Settings() (settingsadapter.File, error) { return settingsSettings(settingsOf(c)) }

// SetSettings returns the configuration with the values of a settings file applied.
func (c Config) SetSettings(s settingsadapter.File) Config { return applySettingsFile(c, s) }

// DefaultSettings returns the settings of an unconfigured installation.
func DefaultSettings() settingsadapter.File { return settingsadapter.DefaultFile() }

// Themes returns the themes service the window offers.
func (c Config) Themes(say func(string)) (*theme.Service, error) {
	return settingsThemes(settingsOf(c), Scales{}, say)
}

// The methods below are the wiring the tests of this package reach from outside it.

func (c Config) CutNotes() chunking.CutNotes { return cutNotes(searchOf(c)) }

func (c Config) Embedder(ctx context.Context) (searchsourceport.WatchedSource, func() error, error) {
	return searchSource(searchOf(c), ctx)
}

func (c Config) EmbedderReady() bool { return embedderReady(searchOf(c)) }

func (c Config) Embedders(ctx context.Context) (indexing, asking searchsourceport.WatchedSource, closer func() error, why error) {
	return searchSources(searchOf(c), ctx)
}

func (c Config) NewSearchOver(queries sourceport.LexicalQueries, sources []searchusecase.Participant, errorHandler func(error)) searchusecase.Search {
	return newSearchOver(searchOf(c), queries, sources, errorHandler, nil)
}

func (c Config) OpenGetSettings() *settings.Get {
	return openGetSettings(c)
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
	return vaultNotes(vaultOf(c), db, index, noteViews{Moves: note.NewTellMoveInShownVault(showing, view), Writes: none.OpenGate{}})
}

func GetNoteChanges(notes NoteUseCases) hostvault.NoteChanges { return getNoteChanges(notes) }

func embedderReady(c searchWiring) bool {
	_, err := c.RunningHost.Runtime.ResolveCapability(bridge.CapabilitySearchSource)
	return err == nil
}

func notifyHost(c settingsWiring, batch []settingsdomain.Setting) { notifyPlugins(c).Execute(batch) }

func settingsGetSettingsAt(c settingsWiring, path string) (settingsadapter.File, error) {
	return settingsStore(c).ReadUncheckedAt(path)
}

func fileMovers(c vaultWiring) vaultport.FileMovers { return vaultWriters(c) }

func Scan(c Config, db *Index) vault.Scan { return searchScan(searchOf(c), db) }

func Embed(c Config, db *Index, source searchsourceport.Source, v domain.Vault) (vectorscase.Embed, error) {
	return searchEmbed(searchOf(c), db, source, v)
}

func IndexTexts(c Config, db *Index, v domain.Vault) (filetext.Index, error) {
	return indexTexts(searchOf(c), db, v, nil)
}

func NewSearch(c Config, db *Index, asking searchsourceport.Source, errorHandler func(error)) searchusecase.Search {
	return newSearch(searchOf(c), db, asking, errorHandler, nil)
}

func SyncSetting(c Config) naming.SyncSetting { return syncSetting(settingsOf(c)) }

func GetSettingsWriter(c Config) func(entries []settingsdomain.Setting) error {
	return getSettingsWriter(settingsOf(c))
}

// UnsubscribeHost stops the settings subscription of the plugin host before a test ends.
func UnsubscribeHost(c Config) {
	wiring := settingsOf(c)
	if wiring.SubscribedHosts == nil {
		return
	}
	wiring.SubscribedHosts.Unsubscribe(wiring.PluginHost.Configuration)
}

func VaultReaders(c Config) vaultport.VaultReaders { return vaultReaders(vaultOf(c)) }

func UpdateIndex(c Config, db *Index) edit.IndexUpdater {
	refresh := vaultMakeRefresh(vaultOf(c), db, db.Notes())
	return func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		return err
	}
}

func Moves(c Config, ctx context.Context) (<-chan struct{}, error) {
	return vaultMoves(vaultOf(c), ctx)
}

func Notes(c Config, db *Index, index edit.IndexUpdater) NoteUseCases {
	return vaultNotes(vaultOf(c), db, index, noteViews{Writes: none.OpenGate{}})
}

func VaultOpener(c Config, db *Index) *vault.VaultOpener { return vaultOpener(vaultOf(c), db) }

func InstallPluginVault(c Config, db *Index, current func() (domain.Vault, bool), hooks PluginVaultHooks) error {
	vaults, err := c.registry()
	if err != nil {
		return err
	}
	return installPluginVault(plugins(c), vaults, db, current, hooks)
}

func searchOf(c Config) searchWiring { return plugins(c).search }

func ReadWholeVault(c Config,
	ctx context.Context, db *Index, source searchsourceport.Source, v domain.Vault,
) (indexing.ReadWholeVault, error) {
	return readWholeVault(searchOf(c), ctx, db, source, v, makeRefresh(c, db, db.Notes()))
}
