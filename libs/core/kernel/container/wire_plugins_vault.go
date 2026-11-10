package container

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/clock"
	"io/fs"

	pluginvault "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginmanifest"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostvault"
	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/commandbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/sharedmimeinfo"
	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/secret"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/tabkind"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/trigger"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
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
	registry vaultregistry.Ports, db *vault.Index, current func() (domain.Vault, bool), hooks PluginVaultHooks,
) error {
	opened, err := pluginsOpenPluginVault(c, registry, db, current, hooks)
	if err != nil {
		return err
	}
	setPluginVault(c, opened)
	return nil
}

func pluginsOpenPluginVault(c pluginsWiring,
	registry vaultregistry.Ports, db *vault.Index, current func() (domain.Vault, bool), hooks PluginVaultHooks,
) (*pluginvault.Access, error) {
	cache, err := openPluginCache(c)
	if err != nil {
		return nil, err
	}
	refresh := c.MakeRefresh(db, db.Stores.Notes)
	index := func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		return err
	}
	scenarios := c.BuildNotes(db, index)
	connections := hostvault.Connections{
		Reader:  note.NewConnectionReader(scenarios),
		Writer:  note.NewConnectionWriter(scenarios),
		Joiner:  note.NewConnectionJoiner(scenarios),
		Creator: note.NewRelatedNoteCreator(scenarios),
	}
	arranger := hostvault.Arranger{
		Notes:    getNoteChanges(scenarios),
		Checks:   getProblemChecks(vault.NewStandardGetProblems(db.NoteQueries.Problems)),
		Problems: db.NoteQueries.Problems,
	}
	if hasReaders(hooks.Notes) {
		arranger.Window, arranger.WindowVault = getNoteChanges(hooks.Notes), current
	}
	stores := derivedfile.PluginStores{Options: vaultconfig.Options{ServiceDir: c.ServiceDir}}
	vaults := hostvault.Vault{
		Locks: vault.NewHoldVault(c.VaultLocks, clock.System{}),
		Session: hostvault.Session{
			Current:    current,
			Vaults:     registry.Vaults.List,
			Find:       findVault(registry),
			IndexState: hooks.IndexState,
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
			Notes:      db.NoteQueries.NoteRefs,
			Types:      db.NoteQueries.NoteTypes,
			NamedPaths: db.NoteQueries.NamedPaths,
			TypedPaths: db.NoteQueries.TypedPaths,
			Links:      db.NoteQueries.Links,
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
	if c.VaultWatcher != nil {
		vaults.Session.Watcher = vault.NewWatchChanges(c.VaultWatcher).Execute
	}
	if vaults.Session.IndexState == nil {
		vaults.Session.IndexState = getIndexState(db)
	}
	if vaults.Session.Moved == nil {
		vaults.Session.Moved = c.Moves
	}
	return vaults.Access(), nil
}

func findVault(registry vaultregistry.Ports) func(id domain.VaultID) (domain.Vault, error) {
	return func(id domain.VaultID) (domain.Vault, error) {
		found, isFound, err := registry.Lookup.FindVault(string(id))
		if err != nil {
			return domain.Vault{}, err
		}
		if !isFound {
			return domain.Vault{}, fmt.Errorf("vault %q: %w", id, fs.ErrNotExist)
		}
		return found, nil
	}
}

func getIndexState(db *vault.Index) func(ctx context.Context, v domain.Vault) (plugindomain.IndexState, error) {
	return func(ctx context.Context, v domain.Vault) (plugindomain.IndexState, error) {
		has, err := db.NoteQueries.VaultHolds.Has(ctx, v.ID)
		if err != nil {
			return plugindomain.IndexState{}, err
		}
		if has {
			return plugindomain.IndexState{Status: plugindomain.Indexed}, nil
		}
		return plugindomain.IndexState{Status: plugindomain.Indexing}, nil
	}
}

func getPluginDir(c pluginsWiring, id string) (string, bool) {
	if c.PluginHost == nil {
		return "", false
	}
	p, ok := c.PluginHost.Runtime.GetPlugin(id)
	if !ok {
		return "", false
	}
	return p.Dir, true
}

func getFrontendPluginIDs(c pluginsWiring) []string {
	if c.PluginHost == nil {
		return nil
	}
	infos := c.PluginHost.Runtime.GetPluginInfos()
	ids := make([]string, 0, len(infos))
	for _, info := range infos {
		if !info.HasFrontend {
			continue
		}
		ids = append(ids, info.ID)
	}
	return ids
}

func setPluginWindow(c pluginsWiring, window hostendpoint.WindowControl) {
	if c.PluginHost != nil {
		c.PluginHost.Endpoint.SetPluginWindow(window)
	}
}

func emitPluginEvent(c pluginsWiring, event string) {
	if c.PluginHost != nil {
		c.PluginHost.Host.Events.EmitEvent(event)
	}
}

// credentialsLocator is a key store that keeps its values in a file when the machine has no keychain.
type credentialsLocator interface {
	// GetCredentialsPath returns the file the values are kept in, or the empty text while the
	// system keychain is in use.
	GetCredentialsPath(ctx context.Context) string
}

func openSecretLedger(c pluginsWiring) *settingsadapter.SecretLedger {
	return settingsadapter.NewSecretLedger(c.SettingsFiles, c.SettingsFile, c.ReadCheckedSettings, c.RefreshSettingEvents)
}

func openSecrets(c pluginsWiring) *secret.Secrets {
	var plugins pluginport.SecretPlugins = plugin.Absent{}
	if c.PluginHost != nil {
		plugins = c.PluginHost.Host.Config
	}
	return secret.NewSecrets(openSecretLedger(c), c.KeyStore, plugins, newSecretID, c.CredentialsPath, c.SecretGate)
}

func credentialsPathOf(store pluginport.KeyStore) func() string {
	return func() string {
		locator, isLocator := store.(credentialsLocator)
		if !isLocator {
			return ""
		}
		return locator.GetCredentialsPath(context.Background())
	}
}

func newSecretID() (string, error) {
	var random [4]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("read random bytes for the id of a secret: %w", err)
	}
	return "s" + hex.EncodeToString(random[:]), nil
}

func setSecretSource(c pluginsWiring) {
	if c.PluginHost == nil {
		return
	}
	ledger := openSecretLedger(c)
	c.PluginHost.Host.Config.SetSecretSource(plugindomain.SecretSource{
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
	if c.PluginHost == nil {
		return nil
	}
	choose, err := tabkind.NewChoose(pluginmanifest.New(c.PluginHost.Runtime), c.PluginHost.Host.ContentTypes, getWhenConfig(c))
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
	var commands pluginport.PluginCommands = plugin.Absent{}
	var runs pluginport.PluginCommandRuns = commandbridge.NoCommands{}
	var types contenttypeport.ContentTypeTable = sharedmimeinfo.Open()
	if c.PluginHost != nil && !c.SafeMode {
		commands, runs, types = pluginmanifest.New(c.PluginHost.Runtime), commandbridge.Open(c.PluginHost.Runtime), c.PluginHost.Host.ContentTypes
	}
	run := plugincommand.NewCommands(commands, runs, types, pluginsNewNoteReader(c), getWhenConfig(c)).Run
	items, err := textversion.NewVersionItems(c.TextVersions, run, types)
	if err != nil {
		pluginsHandleError(c, err)
		return nil
	}
	return items
}

func openTriggers(c pluginsWiring) *trigger.Triggers {
	if c.PluginHost == nil || c.SafeMode || c.TextVersions == nil {
		return nil
	}
	return trigger.NewTriggers(trigger.TriggersDeps{
		Plugins: pluginmanifest.New(c.PluginHost.Runtime), Run: getPluginCommands(c).Run, Readers: c.VaultReaders,
		Notes: pluginsNewNoteReader(c), Types: c.PluginHost.Host.ContentTypes, Versions: c.TextVersions.Texts, Config: getWhenConfig(c),
	})
}

func pluginsNewNoteReader(c pluginsWiring) plugincommand.NoteReader {
	if c.VaultReaders == nil {
		return nil
	}
	return note.NewReadFrontmatter(c.VaultReaders)
}

func getWhenConfig(c pluginsWiring) pluginport.WhenConfig {
	settings := settingsadapter.WhenConfig{App: c.ListAppSettings}
	if c.PluginHost != nil {
		settings.Plugins = c.PluginHost.Runtime.GetPluginInfos
	}
	return settings
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
