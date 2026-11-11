package container

import (
	"context"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"

	pluginvault "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginmanifest"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostvault"
	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/secret"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/tabkind"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/trigger"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/hold"
	vaultregistry "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// PluginVaultHooks is what a window adds to the vault it lets the plugins work.
type PluginVaultHooks struct {
	Written    func()
	IndexState func(ctx context.Context, v domain.Vault) (plugindomain.IndexState, error)
	Moved      func(ctx context.Context) (<-chan struct{}, error)
	Notes      note.Scenarios
}

func hasReaders(notes note.Scenarios) bool {
	return notes.Place.Move.IsConfigured()
}

func getNoteChanges(notes note.Scenarios) hostvault.NoteChanges {
	return hostvault.NoteChanges{
		Rename: notes.Place.Rename, Move: notes.Place.Move, Remove: notes.Place.RemoveNote, Links: notes.Edit.Linking, AddLinks: notes.Refer.AddLinks,
	}
}

func openNoteWriter(c pluginsWiring, v domain.Vault) (hostvault.NoteWriter, error) {
	writer, err := c.NoteWriters.OpenNoteWriter(v)
	if err != nil {
		return nil, err
	}
	return writer, nil
}

func installPluginVault(c pluginsWiring,
	registry vaultregistry.Ports, db *Index, current func() (domain.Vault, bool), hooks PluginVaultHooks,
) error {
	opened, err := pluginsOpenPluginVault(c, registry, db, current, completeHooks(c, db, hooks))
	if err != nil {
		return err
	}
	setPluginVault(c, opened)
	return nil
}

func pluginsOpenPluginVault(c pluginsWiring,
	registry vaultregistry.Ports, db *Index, current func() (domain.Vault, bool), hooks PluginVaultHooks,
) (*pluginvault.Access, error) {
	cache, err := openPluginCache(c)
	if err != nil {
		return nil, err
	}
	refresh := c.MakeRefresh(db, db.Notes())
	index := func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		return err
	}
	scenarios := c.BuildNotes(db, index)
	connections := hostvault.Connections{
		Reader:  note.NewConnectionReader(query.NewGetConnection(db.NoteQueries())),
		Writer:  note.NewConnectionWriter(scenarios),
		Joiner:  note.NewConnectionJoiner(scenarios),
		Creator: note.NewRelatedNoteCreator(scenarios),
	}
	arranger := newArranger(scenarios, db, hooks.Notes, current)
	stores := derivedfile.PluginStores{Options: vaultconfig.Options{ServiceDir: c.ServiceDir}}
	vaults := hostvault.Vault{
		Locks: hold.NewLocks(c.VaultLocks, clock.System{}),
		Session: hostvault.Session{
			Current:    current,
			Vaults:     registry.Vaults.List,
			Find:       vaultregistry.NewFindByID(registry.Lookup).Execute,
			IndexState: hooks.IndexState,
			Watcher:    vault.NewWatchChanges(c.VaultWatcher).Execute,
			Moved:      hooks.Moved,
			Written:    hooks.Written,
		},
		Files: hostvault.Files{
			Stat:        file.NewStatFile(c.VaultReaders),
			ReadFile:    file.NewReadFile(c.VaultReaders),
			RangeReader: file.Read{Readers: c.VaultReaders},
			OpenWriter:  func(v domain.Vault) (hostvault.NoteWriter, error) { return openNoteWriter(c, v) },
			Index:       index,
		},
		Lookups: hostvault.Lookups{
			Notes:      db.NoteQueries(),
			Types:      db.NoteQueries(),
			NamedPaths: db.NoteQueries(),
			TypedPaths: db.NoteQueries(),
			Links:      db.NoteQueries(),
		},
		Connections: connections,
		Arranger:    arranger,
		Stores: hostvault.Stores{
			StoreReaders: func(v domain.Vault, pluginID string) (pluginvault.StoreReader, error) {
				return stores.Open(v, pluginID)
			},
			StoreWriters: func(v domain.Vault, pluginID string) (pluginvault.StoreWriter, error) {
				return stores.OpenWriter(v, pluginID)
			},
			Cache: cache,
		},
	}
	return vaults.Access(), nil
}

func newArranger(
	scenarios note.Scenarios, db *Index, window note.Scenarios, current func() (domain.Vault, bool),
) hostvault.Arranger {
	arranger := hostvault.Arranger{
		Notes:    getNoteChanges(scenarios),
		Checks:   getProblemChecks(getProblems(vaultWiring{}, db.NoteProblems())),
		Problems: db.NoteProblems(),
	}
	if hasReaders(window) {
		arranger.Window, arranger.WindowVault = getNoteChanges(window), current
	}
	return arranger
}

func completeHooks(c pluginsWiring, db *Index, hooks PluginVaultHooks) PluginVaultHooks {
	if hooks.IndexState == nil {
		hooks.IndexState = query.NewGetIndexState(db.NoteQueries()).Execute
	}
	if hooks.Moved == nil {
		hooks.Moved = c.Moves
	}
	return hooks
}

func getPluginDir(c pluginsWiring, id string) (string, bool) {
	p, ok := c.PluginHost.Runtime.GetPlugin(id)
	return p.Dir, ok
}

func getFrontendPluginIDs(c pluginsWiring) []string {
	return pluginmanifest.New(c.PluginHost.Runtime).ListFrontendPluginIDs()
}

func setPluginWindow(c pluginsWiring, window hostendpoint.WindowControl) {
	c.PluginHost.Endpoint.SetPluginWindow(window)
}

func emitPluginEvent(c pluginsWiring, event string) {
	c.PluginHost.Runtime.EmitEvent(event)
}

func openSecretLedger(c pluginsWiring) *settingsadapter.SecretLedger {
	return settingsadapter.NewSecretLedger(c.SettingsFiles, c.SettingsFile, c.ReadCheckedSettings, c.RefreshSettingEvents)
}

func openSecrets(c pluginsWiring) *secret.Secrets {
	return secret.NewSecrets(openSecretLedger(c), c.KeyStore, c.PluginHost.Configuration, settingsadapter.CreateSecretID, c.CredentialsPath, c.SecretGate)
}

func setSecretSource(c pluginsWiring) {
	ledger := openSecretLedger(c)
	c.PluginHost.Configuration.SetSecretSource(plugindomain.SecretSource{
		List:  ledger.GetSecrets,
		Prune: openSecrets(c).Prune.Execute,
	})
}

func signalPluginChanges(c pluginsWiring) {
	if c.PluginChanges != nil {
		c.PluginChanges.Signal()
	}
}

func watchPluginChanges(c pluginsWiring) (<-chan struct{}, func()) {
	if c.PluginChanges == nil {
		return make(chan struct{}), func() {}
	}
	return c.PluginChanges.Subscribe()
}

func openChooseTabKind(c pluginsWiring) *tabkind.Choose {
	choose, err := tabkind.NewChoose(pluginmanifest.New(c.PluginHost.Runtime), c.PluginHost.Watch, getWhenConfig(c))
	if err != nil {
		pluginsHandleError(c, err)
		return nil
	}
	return choose
}

func openVersionItems(c pluginsWiring) *textversion.VersionItems {
	if c.TextVersions == nil {
		return nil
	}
	run := newPluginCommands(c, c.RunningHost).Run
	items, err := textversion.NewVersionItems(c.TextVersions, run, c.RunningHost.Watch)
	if err != nil {
		pluginsHandleError(c, err)
		return nil
	}
	return items
}

func openTriggers(c pluginsWiring) *trigger.Triggers {
	if c.TextVersions == nil {
		return nil
	}
	return trigger.NewTriggers(trigger.TriggersDeps{
		Plugins: pluginmanifest.New(c.RunningHost.Runtime), Run: newPluginCommands(c, c.RunningHost).Run, Readers: c.VaultReaders,
		Notes: pluginsNewNoteReader(c), Types: c.RunningHost.Watch, Versions: c.TextVersions.Texts, Config: getWhenConfig(c),
	})
}

func pluginsNewNoteReader(c pluginsWiring) plugincommand.NoteReader {
	if c.VaultReaders == nil {
		return nil
	}
	return note.NewReadFrontmatter(c.VaultReaders)
}

func getWhenConfig(c pluginsWiring) pluginport.WhenConfig {
	return settingsadapter.WhenConfig{App: c.ListAppSettings, Plugins: c.PluginHost.Runtime.GetPluginInfos}
}

func getProblemChecks(u *vault.GetProblems) func(ctx context.Context, v domain.Vault, named ...domain.Check) ([]domain.Check, []domain.VaultProblem, error) {
	return func(ctx context.Context, v domain.Vault, named ...domain.Check) ([]domain.Check, []domain.VaultProblem, error) {
		report, err := u.Execute(ctx, v, named...)
		return report.Ran, report.Found, err
	}
}

// GetFrontendPluginIDs lists the plugins that have a frontend.
func GetFrontendPluginIDs(c Config) []string { return getFrontendPluginIDs(plugins(c)) }

// GetPluginDir returns the folder of a plugin and whether the plugin is installed.
func GetPluginDir(c Config, id string) (string, bool) { return getPluginDir(plugins(c), id) }
