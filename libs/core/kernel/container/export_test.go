package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostvault"
	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/chunking"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/naming"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/sourcebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	searchusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"
)

// Settings reads the settings file; an unconfigured installation returns the default settings.
func (c TestInstallation) Settings() (settingsadapter.File, error) {
	return createSettings(c.buildSettingsWiring())
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
	return createThemeService(c.buildSettingsWiring(), Scales{}, say)
}

// The methods below are the wiring the tests of this package reach from outside it.

func (c TestInstallation) CutNotes() chunking.CutNotes { return cutNotes(indexPlanOf(c).ChunkPolicy) }

func (c TestInstallation) Embedder(ctx context.Context) (searchsourceport.Source, func() error, error) {
	source, closeSource, err := sourceOf(c).Open(ctx)
	return getSourceAccess(source).searchSource, closeSource, err
}

func (c TestInstallation) EmbedderReady() bool { return embedderReady(sourceOf(c)) }

func (c TestInstallation) Embedders(ctx context.Context) (indexing, asking searchsourceport.Source, closer func() error, why error) {
	source, closer, err := sourceOf(c).OpenWatching(ctx)
	access := getSourceAccess(source)
	return access.searchSource, access.searchSource, closer, err
}

func (c TestInstallation) NewSearchOver(queries searchusecase.LexicalQueries, sources []searchusecase.Participant, errorHandler func(error)) searchusecase.Search {
	return newSearchOver(newTextSearch(indexPlanOf(c), c.machine.ResolvedCompute.Workers), queries, sources, errorHandler, nil)
}

func (c TestInstallation) OpenGetSettings() *settings.Get {
	return openGetSettings(c.buildSettingsWiring())
}

func (c TestInstallation) OpenWatchSettings() *settings.WatchSettingChanges {
	return openWatchSettings(c.buildSettingsWiring())
}

func (c TestInstallation) SubscribeSettingChanges() (<-chan []settingsdomain.Setting, func()) {
	return getFeed(c.buildSettingsWiring()).Subscribe()
}

// SubscribeAppliedPluginSettings reports when the host finishes applying a settings batch.
func (c TestInstallation) SubscribeAppliedPluginSettings() (<-chan struct{}, func()) {
	return c.Resources.settings.subscriptions.Changes.Subscribe()
}

func (c TestInstallation) VaultOpenerWith(db *Index, walking vaultport.VaultReaders, watcher indexing.VaultWatcher) *indexing.VaultOpener {
	return createVaultOpenerWith(c.buildVaultWiring(), db, walking, watcher)
}

func (c TestInstallation) VaultWatcher() indexing.VaultWatcher {
	return c.buildVaultWiring().Watcher
}

// NotesShowingMoves is Notes where each move of a note in the vault the window shows also reaches view, so a
// view showing a note under its old name switches to the new name.
func NotesShowingMoves(
	c TestInstallation, db *Index, index edit.IndexUpdater, showing func() (domain.Vault, bool), view note.MoveView,
) NoteUseCases {
	return createNoteUseCases(c.buildVaultWiring(), db, index, noteViews{Moves: note.NewTellMoveInShownVault(showing, view)})
}

func GetNoteChanges(notes NoteUseCases) hostvault.NoteChanges { return getNoteChanges(notes) }

func sourceOf(c TestInstallation) sourcebridge.Binding {
	return newSourceBinding(c.Execution, c.SafeMode, c.Endpoints, c.Resources)
}

func embedderReady(c sourcebridge.Binding) bool {
	_, err := c.Plugins.ResolveCapability(bridge.CapabilitySearchSource)
	return err == nil
}

func notifyHost(c TestInstallation, batch []settingsdomain.Setting) {
	notifyPlugins(c.buildSettingsWiring(), c.Preferences).Execute(batch)
}

func settingsGetSettingsAt(c settingsWiring, path string) (settingsadapter.File, error) {
	return createSettingsStore(c).ReadUncheckedAt(path)
}

func Scan(c TestInstallation, db *Index) indexing.Scan { return searchScan(indexPlanOf(c), db) }

func Embed(c TestInstallation, db *Index, source searchsourceport.Source, v domain.Vault) (indexing.Embed, error) {
	return searchEmbed(indexPlanOf(c), db, source, v)
}

func IndexTexts(c TestInstallation, db *Index, v domain.Vault) (indexing.Index, error) {
	return indexTexts(indexPlanOf(c), db, v, nil)
}

func NewSearch(c TestInstallation, db *Index, asking searchsourceport.Source, errorHandler func(error)) searchusecase.Search {
	return newSearch(newTextSearch(indexPlanOf(c), c.machine.ResolvedCompute.Workers), db, asking, errorHandler, nil)
}

func SyncSetting(c TestInstallation) naming.SyncSetting {
	return syncSetting(c.buildSettingsWiring())
}

func GetSettingsWriter(c TestInstallation) func(entries []settingsdomain.Setting) error {
	return getSettingsWriter(c.buildSettingsWiring())
}

// UnsubscribeHost stops the settings subscription of the plugin host before a test ends.
func UnsubscribeHost(c TestInstallation) {
	c.Resources.settings.subscriptions.Hosts.Unsubscribe(c.Preferences.Configuration)
}

func VaultReaders(c TestInstallation) vaultport.VaultReaders {
	return c.buildVaultWiring().Readers
}

func UpdateIndex(c TestInstallation, db *Index) edit.IndexUpdater {
	refresh := createRefresh(c.buildVaultWiring(), db, db.Notes())
	return func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		return err
	}
}

func Moves(c TestInstallation, ctx context.Context) (<-chan struct{}, error) {
	access := c.buildVaultWiring()
	return watchMoves(ctx, access.IndexPath, access.Watcher)
}

func Notes(c TestInstallation, db *Index, index edit.IndexUpdater) NoteUseCases {
	return createNoteUseCases(c.buildVaultWiring(), db, index, noteViews{})
}

func VaultOpener(c TestInstallation, db *Index) *indexing.VaultOpener {
	return createVaultOpener(c.buildVaultWiring(), db)
}

func InstallPluginVault(c TestInstallation, db *Index, current func() (domain.Vault, bool), hooks PluginVaultHooks) error {
	vaults, err := openVaultRegistry(c.RegistryPath)
	if err != nil {
		return err
	}
	return installPluginVault(c.buildPluginInstall(), c.buildVaultWiring(), vaults, db, current, hooks)
}

func indexPlanOf(c TestInstallation) vaultIndexPlan {
	_, search, _ := buildVaultSearch(c.Config, c.Execution, c.Preferences, c.Resources, nil)
	return search
}

func ReadWholeVault(c TestInstallation,
	ctx context.Context, db *Index, source searchsourceport.Source, v domain.Vault,
) (indexing.ReadWholeVault, error) {
	return readWholeVault(indexPlanOf(c), ctx, db, source, v, GetRefresh(c, db))
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
	return createSettingsWiring(c.Config, c.Execution, c.Preferences, c.Resources)
}

func (c TestInstallation) buildPluginInstall() pluginInstall {
	install := newPluginInstall(c.Config, c.Preferences, c.Endpoints, c.Resources)
	return install
}

func (c TestInstallation) buildVaultWiring() vaultWiring {
	_, _, vault := buildVaultSearch(c.Config, c.Execution, c.Preferences, c.Resources, nil)
	return vault
}

type TestPlugins struct {
	Execution   *PluginExecution
	Preferences *PluginPreferences
	Endpoints   *PluginEndpoints
}

func AdaptTestPlugins(management *PluginManagement, supervision *PluginSupervision) *TestPlugins {
	execution, preferences, endpoints := CreatePluginHost(management, supervision)
	return &TestPlugins{Execution: execution, Preferences: preferences, Endpoints: endpoints}
}

func GetRefresh(c TestInstallation, db *Index) indexing.Refresh {
	return createRefresh(c.buildVaultWiring(), db, db.Notes())
}

func (c TestInstallation) openTextVersions(db *Index) *textVersions {
	return newTextVersions(vaultconfig.Options{ServiceDir: c.ServiceDir}, getRunningExecution(c.Execution, c.SafeMode).Runtime.Listing, GetClock(c.Resources), db)
}

// These aliases let external container tests inspect the composition they exercise.
type Index = vaultIndex
type PluginVaultHooks = pluginVaultHooks
type NoteUseCases = noteUseCases

func OpenIndex(c Config, res *Resources, ctx context.Context) (*Index, error) {
	return c.openIndex(res, ctx)
}
