package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/none"
	pluginvault "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostvault"
	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/tabkind"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/trigger"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/hold"
	vaultregistry "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// PluginVaultHooks is what a window adds to the vault it lets the plugins work.
type PluginVaultHooks struct {
	Written    func()
	IndexState func(ctx context.Context, v domain.Vault) (plugindomain.IndexState, error)
	Moved      func(ctx context.Context) (<-chan struct{}, error)
	Notes      *hostvault.NoteChanges
}

func getNoteChanges(notes NoteUseCases) hostvault.NoteChanges {
	return hostvault.NoteChanges{
		Rename: notes.Place.Rename, Move: notes.Place.Move, Remove: notes.Place.RemoveNote, Links: notes.Edit.Linking,
		AddLinks: note.NewAddLinks(notes.Edit.Linking),
	}
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
	useCases := vaultNotes(c.vault, db, index, noteViews{Writes: none.OpenGate{}})
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
			Stat:        file.NewHashFile(vaultReaders(c.vault), vaultVersions(c.vault)),
			ReadFile:    file.NewReadFile(vaultReaders(c.vault), vaultVersions(c.vault)),
			RangeReader: file.Read{Readers: vaultReaders(c.vault)},
		},
		hostvault.Writer{
			Write:  file.NewWriteFile(noteWriters(c.vault), index),
			Create: file.NewCreateFile(noteWriters(c.vault), index),
			Index:  file.NewScanFiles(index),
			Locks:  hold.NewLocks(writeLocks(c.vault), hold.LockHold),
		},
		getPluginLookups(db),
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
	useCases NoteUseCases, db *Index, window *hostvault.NoteChanges, current func() (domain.Vault, bool),
) hostvault.Arranger {
	arranger := hostvault.Arranger{
		Notes:       getNoteChanges(useCases),
		Frontmatter: useCases.Properties.Frontmatter,
		Checks:      getProblemChecks(getProblems(vaultWiring{}, db.NoteProblems())),
		Problems:    vault.NewGetParseProblems(db.NoteProblems()),
	}
	if window != nil {
		arranger.Window, arranger.WindowVault = *window, current
	}
	return arranger
}

func completeHooks(c pluginsWiring, db *Index, hooks PluginVaultHooks) PluginVaultHooks {
	if hooks.IndexState == nil {
		hooks.IndexState = getIndexState(db).Execute
	}
	if hooks.Moved == nil {
		hooks.Moved = func(ctx context.Context) (<-chan struct{}, error) { return vaultMoves(c.vault, ctx) }
	}
	return hooks
}

func setPluginWindow(c pluginsWiring, window hostendpoint.WindowControl) {
	c.Endpoints.Lender.Update(func(l *hostendpoint.Lent) { l.Window = window })
}

func emitPluginEvent(c pluginsWiring, event string) {
	c.Execution.Runtime.EmitEvent(event)
}

func signalPluginChanges(c pluginsWiring) {
	c.settings.PluginChanges.Signal()
}

func watchPluginChanges(c pluginsWiring) (<-chan struct{}, func()) {
	return c.settings.PluginChanges.Subscribe()
}

func openChooseTabKind(c pluginsWiring) *tabkind.Choose {
	return tabkind.NewChoose(getRuntimeManifests(c.Execution.Runtime), c.Execution.Watch, getWhenConfig(c))
}

func openVersionItems(c pluginsWiring) *textversion.VersionItems {
	if c.vault.TextVersions == nil {
		return nil
	}
	run := newPluginCommands(c, c.ActiveExecution, nil).Run
	return textversion.NewVersionItems(c.vault.TextVersions.Texts, c.vault.TextVersions.Versions, run, c.ActiveExecution.Watch)
}

func openTriggers(c pluginsWiring) *trigger.Triggers {
	if c.vault.TextVersions == nil {
		return nil
	}
	return trigger.NewTriggers(trigger.TriggersDeps{
		Plugins: getRuntimeManifests(c.ActiveExecution.Runtime), Run: newPluginCommands(c, c.ActiveExecution, nil).Run, Readers: vaultReaders(c.vault),
		Notes: pluginsNewNoteReader(c), Types: c.ActiveExecution.Watch, Versions: c.vault.TextVersions.Texts, Config: getWhenConfig(c),
	})
}

func pluginsNewNoteReader(c pluginsWiring) noteport.NoteReader {
	return note.NewReadFrontmatter(vaultReaders(c.vault))
}

func getWhenConfig(c pluginsWiring) pluginport.WhenConfig {
	return settingsadapter.WhenConfig{
		App:     func() []plugindomain.Setting { return listAppSettings(c.settings) },
		Plugins: c.Execution.Runtime.GetPluginInfos,
	}
}

func getProblemChecks(u *vault.GetProblems) func(ctx context.Context, v domain.Vault, named ...domain.Check) ([]domain.Check, []domain.VaultProblem, error) {
	return func(ctx context.Context, v domain.Vault, named ...domain.Check) ([]domain.Check, []domain.VaultProblem, error) {
		report, err := u.Execute(ctx, v, named...)
		return report.Ran, report.Found, err
	}
}

func GetFrontendPluginIDs(runtime PluginRuntime) []string {
	return getRuntimeManifests(runtime).ListFrontendPluginIDs()
}

// GetPluginDir returns the folder of a plugin and whether the plugin is installed.
func GetPluginDir(resolver pluginport.PluginResolver, id string) (string, bool) {
	p, ok := resolver.GetPlugin(id)
	return p.Dir, ok
}
