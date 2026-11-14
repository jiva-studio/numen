package container

import (
	"context"

	pluginvault "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"

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
	Notes      *NoteUseCases
}

func getNoteChanges(notes NoteUseCases) hostvault.NoteChanges {
	return hostvault.NoteChanges{
		Rename: notes.Place.Rename, Move: notes.Place.Move, Remove: notes.Place.RemoveNote, Links: notes.Edit.Linking, AddLinks: notes.Refer.AddLinks,
	}
}

func openNoteWriter(c pluginsWiring, v domain.Vault) (hostvault.NoteWriter, error) {
	writer, err := noteWriters(c.vault).OpenNoteWriter(v)
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
	refresh := vaultMakeRefresh(c.vault, db, db.Notes())
	index := func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		return err
	}
	useCases := vaultNotes(c.vault, db, index, noteViews{Writes: note.OpenGate{}})
	arranger := newArranger(useCases, db, hooks.Notes, current)
	stores := derivedfile.PluginStores{Options: vaultconfig.Options{ServiceDir: c.ServiceDir}}
	return hostvault.NewAccess(
		hostvault.Session{
			Current:    current,
			Vaults:     registry.Vaults.List,
			Find:       vaultregistry.NewFindByID(registry.Lookup).Execute,
			IndexState: hooks.IndexState,
			Moved:      hooks.Moved,
			Written:    hooks.Written,
		},
		hostvault.Files{
			Stat:        file.NewStatFile(vaultReaders(c.vault)),
			ReadFile:    file.NewReadFile(vaultReaders(c.vault)),
			RangeReader: file.Read{Readers: vaultReaders(c.vault)},
			OpenWriter:  func(v domain.Vault) (hostvault.NoteWriter, error) { return openNoteWriter(c, v) },
			Index:       index,
			Locks:       hold.NewLocks(writeLocks(c.vault), hold.LockHold),
		},
		hostvault.Lookups{
			Notes:      db.NoteQueries(),
			Types:      db.NoteQueries(),
			NamedPaths: db.NoteQueries(),
			TypedPaths: db.NoteQueries(),
			Links:      db.NoteQueries(),
		},
		arranger,
		hostvault.Stores{
			StoreReaders: func(v domain.Vault, pluginID string) (pluginvault.StoreReader, error) {
				return stores.Open(v, pluginID)
			},
			StoreWriters: func(v domain.Vault, pluginID string) (pluginvault.StoreWriter, error) {
				return stores.OpenWriter(v, pluginID)
			},
			Cache: cache,
		},
	), nil
}

func newArranger(
	useCases NoteUseCases, db *Index, window *NoteUseCases, current func() (domain.Vault, bool),
) hostvault.Arranger {
	arranger := hostvault.Arranger{
		Notes:    getNoteChanges(useCases),
		Checks:   getProblemChecks(getProblems(vaultWiring{}, db.NoteProblems())),
		Problems: db.NoteProblems(),
	}
	if window != nil {
		arranger.Window, arranger.WindowVault = getNoteChanges(*window), current
	}
	return arranger
}

func completeHooks(c pluginsWiring, db *Index, hooks PluginVaultHooks) PluginVaultHooks {
	if hooks.IndexState == nil {
		hooks.IndexState = query.NewGetIndexState(db.NoteQueries()).Execute
	}
	if hooks.Moved == nil {
		hooks.Moved = func(ctx context.Context) (<-chan struct{}, error) { return vaultMoves(c.vault, ctx) }
	}
	return hooks
}

func getPluginDir(c pluginsWiring, id string) (string, bool) {
	p, ok := c.PluginHost.Runtime.GetPlugin(id)
	return p.Dir, ok
}

func getFrontendPluginIDs(c pluginsWiring) []string {
	return getRuntimeManifests(c.PluginHost.Runtime).ListFrontendPluginIDs()
}

func setPluginWindow(c pluginsWiring, window hostendpoint.WindowControl) {
	c.PluginHost.Endpoint.SetPluginWindow(window)
}

func emitPluginEvent(c pluginsWiring, event string) {
	c.PluginHost.Runtime.EmitEvent(event)
}

func signalPluginChanges(c pluginsWiring) {
	c.settings.PluginChanges.Signal()
}

func watchPluginChanges(c pluginsWiring) (<-chan struct{}, func()) {
	return c.settings.PluginChanges.Subscribe()
}

func openChooseTabKind(c pluginsWiring) *tabkind.Choose {
	return tabkind.NewChoose(getRuntimeManifests(c.PluginHost.Runtime), c.PluginHost.Watch, getWhenConfig(c))
}

func openVersionItems(c pluginsWiring) *textversion.VersionItems {
	if c.vault.TextVersions == nil {
		return nil
	}
	run := newPluginCommands(c, c.RunningHost, nil).Run
	return textversion.NewVersionItems(c.vault.TextVersions.Texts, c.vault.TextVersions.Versions, run, c.RunningHost.Watch)
}

func openTriggers(c pluginsWiring) *trigger.Triggers {
	if c.vault.TextVersions == nil {
		return nil
	}
	return trigger.NewTriggers(trigger.TriggersDeps{
		Plugins: getRuntimeManifests(c.RunningHost.Runtime), Run: newPluginCommands(c, c.RunningHost, nil).Run, Readers: vaultReaders(c.vault),
		Notes: pluginsNewNoteReader(c), Types: c.RunningHost.Watch, Versions: c.vault.TextVersions.Texts, Config: getWhenConfig(c),
	})
}

func pluginsNewNoteReader(c pluginsWiring) plugincommand.NoteReader {
	return note.NewReadFrontmatter(vaultReaders(c.vault))
}

func getWhenConfig(c pluginsWiring) pluginport.WhenConfig {
	return settingsadapter.WhenConfig{
		App:     func() []plugindomain.Setting { return listAppSettings(c.settings) },
		Plugins: c.PluginHost.Runtime.GetPluginInfos,
	}
}

func getProblemChecks(u *vault.GetProblems) func(ctx context.Context, v domain.Vault, named ...domain.Check) ([]domain.Check, []domain.VaultProblem, error) {
	return func(ctx context.Context, v domain.Vault, named ...domain.Check) ([]domain.Check, []domain.VaultProblem, error) {
		report, err := u.Execute(ctx, v, named...)
		return report.Ran, report.Found, err
	}
}

func GetFrontendPluginIDs(c Config) []string { return getFrontendPluginIDs(plugins(c)) }

// GetPluginDir returns the folder of a plugin and whether the plugin is installed.
func GetPluginDir(c Config, id string) (string, bool) { return getPluginDir(plugins(c), id) }
