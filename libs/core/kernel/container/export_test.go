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
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"
)

// Settings reads the settings file; an unconfigured installation returns the default settings.
func (c TestInstallation) Settings() (settingsadapter.File, error) {
	return settingsSettings(c.buildSettingsWiring())
}

// SetSettings returns the configuration with the values of a settings file applied.
func (c TestInstallation) SetSettings(s settingsadapter.File) TestInstallation {
	c.Config = applySettingsFile(c.Config, c.Execution, c.Preferences, c.Resources, s)
	return c
}

// DefaultSettings returns the settings of an unconfigured installation.
func DefaultSettings() settingsadapter.File { return settingsadapter.DefaultFile() }

// Themes returns the themes service the window offers.
func (c TestInstallation) Themes(say func(string)) (*theme.Service, error) {
	return settingsThemes(c.buildSettingsWiring(), Scales{}, say)
}

// The methods below are the wiring the tests of this package reach from outside it.

func (c TestInstallation) CutNotes() chunking.CutNotes { return cutNotes(searchOf(c)) }

func (c TestInstallation) Embedder(ctx context.Context) (searchsourceport.WatchedSource, func() error, error) {
	return searchSource(searchOf(c), ctx)
}

func (c TestInstallation) EmbedderReady() bool { return embedderReady(searchOf(c)) }

func (c TestInstallation) Embedders(ctx context.Context) (indexing, asking searchsourceport.WatchedSource, closer func() error, why error) {
	return searchSources(searchOf(c), ctx)
}

func (c TestInstallation) NewSearchOver(queries sourceport.LexicalQueries, sources []searchusecase.Participant, errorHandler func(error)) searchusecase.Search {
	return newSearchOver(searchOf(c), queries, sources, errorHandler, nil)
}

func (c TestInstallation) OpenGetSettings() *settings.Get {
	return openGetSettings(c.buildSettingsWiring())
}

func (c TestInstallation) OpenWatchSettings() *settings.WatchSettingChanges {
	return openWatchSettings(c.buildSettingsWiring())
}

func (c TestInstallation) SubscribeSettingChanges() (<-chan []settingsdomain.Setting, func()) {
	return feed(c.buildSettingsWiring()).Subscribe()
}

func (c TestInstallation) VaultOpenerWith(db *Index, walking vaultport.VaultReaders, watcher vaultport.VaultWatcher) *vault.VaultOpener {
	return vaultOpenerWith(c.buildVaultWiring(), db, walking, watcher)
}

func (c TestInstallation) VaultWatcher() vaultport.VaultWatcher {
	return vaultWatcher(c.buildVaultWiring())
}

// AbsentBinding is the binding of a plugin host that runs nothing.
func AbsentBinding() pluginport.PluginBinding { return absent.Runtime{} }

// NotesShowingMoves is Notes where each move of a note in the vault the window shows also reaches view, so a
// view showing a note under its old name switches to the new name.
func NotesShowingMoves(
	c TestInstallation, db *Index, index edit.IndexUpdater, showing func() (domain.Vault, bool), view note.MoveView,
) NoteUseCases {
	return vaultNotes(c.buildVaultWiring(), db, index, noteViews{Moves: note.NewTellMoveInShownVault(showing, view), Writes: none.OpenGate{}})
}

func GetNoteChanges(notes NoteUseCases) hostvault.NoteChanges { return getNoteChanges(notes) }

func embedderReady(c searchWiring) bool {
	_, err := c.ActiveRuntime.ResolveCapability(bridge.CapabilitySearchSource)
	return err == nil
}

func notifyHost(c settingsWiring, batch []settingsdomain.Setting) { notifyPlugins(c).Execute(batch) }

func settingsGetSettingsAt(c settingsWiring, path string) (settingsadapter.File, error) {
	return settingsStore(c).ReadUncheckedAt(path)
}

func fileMovers(c vaultWiring) vaultport.FileMovers { return vaultWriters(c) }

func Scan(c TestInstallation, db *Index) vault.Scan { return searchScan(searchOf(c), db) }

func Embed(c TestInstallation, db *Index, source searchsourceport.Source, v domain.Vault) (vectorscase.Embed, error) {
	return searchEmbed(searchOf(c), db, source, v)
}

func IndexTexts(c TestInstallation, db *Index, v domain.Vault) (filetext.Index, error) {
	return indexTexts(searchOf(c), db, v, nil)
}

func NewSearch(c TestInstallation, db *Index, asking searchsourceport.Source, errorHandler func(error)) searchusecase.Search {
	return newSearch(searchOf(c), db, asking, errorHandler, nil)
}

func SyncSetting(c TestInstallation) naming.SyncSetting {
	return syncSetting(c.buildSettingsWiring())
}

func GetSettingsWriter(c TestInstallation) func(entries []settingsdomain.Setting) error {
	return getSettingsWriter(c.buildSettingsWiring())
}

// UnsubscribeHost stops the settings subscription of the plugin host before a test ends.
func UnsubscribeHost(c TestInstallation) {
	wiring := c.buildSettingsWiring()
	if wiring.SubscribedHosts == nil {
		return
	}
	wiring.SubscribedHosts.Unsubscribe(wiring.Configuration)
}

func VaultReaders(c TestInstallation) vaultport.VaultReaders {
	return vaultReaders(c.buildVaultWiring())
}

func UpdateIndex(c TestInstallation, db *Index) edit.IndexUpdater {
	refresh := vaultMakeRefresh(c.buildVaultWiring(), db, db.Notes())
	return func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		return err
	}
}

func Moves(c TestInstallation, ctx context.Context) (<-chan struct{}, error) {
	return vaultMoves(c.buildVaultWiring(), ctx)
}

func Notes(c TestInstallation, db *Index, index edit.IndexUpdater) NoteUseCases {
	return vaultNotes(c.buildVaultWiring(), db, index, noteViews{Writes: none.OpenGate{}})
}

func VaultOpener(c TestInstallation, db *Index) *vault.VaultOpener {
	return vaultOpener(c.buildVaultWiring(), db)
}

func InstallPluginVault(c TestInstallation, db *Index, current func() (domain.Vault, bool), hooks PluginVaultHooks) error {
	vaults, err := c.registry()
	if err != nil {
		return err
	}
	return installPluginVault(c.buildPluginInstall(), c.buildVaultWiring(), vaults, db, current, hooks)
}

func searchOf(c TestInstallation) searchWiring {
	_, search, _ := buildVaultSearch(c.Config, c.Execution, c.Preferences, c.Endpoints, c.Resources, nil)
	return search
}

func ReadWholeVault(c TestInstallation,
	ctx context.Context, db *Index, source searchsourceport.Source, v domain.Vault,
) (indexing.ReadWholeVault, error) {
	return readWholeVault(searchOf(c), ctx, db, source, v, GetRefresh(c, db))
}

type TestInstallation struct {
	Config
	*TestPlugins
	*Resources
}

func NewTestInstallation(c Config, host *TestPlugins) TestInstallation {
	if host == nil {
		host = &TestPlugins{}
	}
	execution, preferences, endpoints := PreparePluginHost(host.Execution, host.Preferences, host.Endpoints, c.SafeMode)
	return TestInstallation{Config: c, TestPlugins: &TestPlugins{Execution: execution, Preferences: preferences, Endpoints: endpoints}, Resources: NewResources()}
}

func (c TestInstallation) buildSettingsWiring() settingsWiring {
	return settingsOf(c.Config, c.Execution, c.Preferences, c.Resources)
}

func (c TestInstallation) buildPluginInstall() pluginInstall {
	install := newPluginInstall(c.Config, c.Endpoints, c.Resources)
	return install
}

func (c TestInstallation) buildVaultWiring() vaultWiring {
	_, _, vault := buildVaultSearch(c.Config, c.Execution, c.Preferences, c.Endpoints, c.Resources, nil)
	return vault
}

type TestPlugins struct {
	Execution   *PluginExecution
	Preferences *PluginPreferences
	Endpoints   *PluginEndpoints
}

func AdaptTestPlugins(management *PluginManagement, supervision *PluginSupervision) *TestPlugins {
	execution, preferences, endpoints := AdaptPluginHost(management, supervision)
	return &TestPlugins{Execution: execution, Preferences: preferences, Endpoints: endpoints}
}

func GetRefresh(c TestInstallation, db *Index) vault.Refresh {
	return vaultMakeRefresh(c.buildVaultWiring(), db, db.Notes())
}
