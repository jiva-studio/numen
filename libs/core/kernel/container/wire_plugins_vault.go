package container

import (
	"context"
	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"

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

func installPluginVault(install pluginInstall, vaultDeps vaultWiring,
	registry vaultregistry.Ports, db *Index, current func() (domain.Vault, bool), hooks PluginVaultHooks,
) error {
	opened, err := pluginsOpenPluginVault(install, vaultDeps, registry, db, current, completeHooks(vaultDeps, db, hooks))
	if err != nil {
		return err
	}
	setPluginVault(install.Endpoints.Lender, opened)
	return nil
}

func pluginsOpenPluginVault(install pluginInstall, vaultDeps vaultWiring,
	registry vaultregistry.Ports, db *Index, current func() (domain.Vault, bool), hooks PluginVaultHooks,
) (*pluginvault.Access, error) {
	cache, err := openPluginCache(install.PluginCachePath)
	if err != nil {
		return nil, err
	}
	refresh := vaultMakeRefresh(vaultDeps, db, db.Notes())
	index := func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		return err
	}
	useCases := vaultNotes(vaultDeps, db, index, noteViews{Writes: none.OpenGate{}})
	arranger := newArranger(useCases, db, hooks.Notes, current)
	stores := derivedfile.PluginStores{Options: vaultconfig.Options{ServiceDir: install.ServiceDir}}
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
			Stat:        file.NewHashFile(vaultReaders(vaultDeps), vaultVersions(vaultDeps)),
			ReadFile:    file.NewReadFile(vaultReaders(vaultDeps), vaultVersions(vaultDeps)),
			RangeReader: file.Read{Readers: vaultReaders(vaultDeps)},
		},
		hostvault.Writer{
			Write:  file.NewWriteFile(noteWriters(vaultDeps), index),
			Create: file.NewCreateFile(noteWriters(vaultDeps), index),
			Index:  file.NewScanFiles(index),
			Locks:  hold.NewLocks(writeLocks(vaultDeps), hold.LockHold),
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

func completeHooks(v vaultWiring, db *Index, hooks PluginVaultHooks) PluginVaultHooks {
	if hooks.IndexState == nil {
		hooks.IndexState = getIndexState(db).Execute
	}
	if hooks.Moved == nil {
		hooks.Moved = func(ctx context.Context) (<-chan struct{}, error) { return vaultMoves(v, ctx) }
	}
	return hooks
}

func setPluginWindow(lender *hostendpoint.Lender, window hostendpoint.WindowControl) {
	lender.Update(func(l *hostendpoint.Lent) { l.Window = window })
}

func emitPluginEvent(events pluginport.PluginEventEmitter, event string) {
	events.EmitEvent(event)
}

func signalPluginChanges(changes *settingsusecase.PluginChanges) {
	changes.Signal()
}

func watchPluginChanges(changes *settingsusecase.PluginChanges) (<-chan struct{}, func()) {
	return changes.Subscribe()
}

func openChooseTabKind(install pluginInstall, settingsDeps settingsWiring) *tabkind.Choose {
	return tabkind.NewChoose(getRuntimeManifests(install.Execution.Runtime), install.Execution.Watch, getWhenConfig(install, settingsDeps))
}

func openVersionItems(install pluginInstall, settingsDeps settingsWiring, vaultDeps vaultWiring) *textversion.VersionItems {
	if vaultDeps.TextVersions == nil {
		return nil
	}
	run := newPluginCommands(install, settingsDeps, vaultDeps, install.ActiveExecution, nil).Run
	return textversion.NewVersionItems(vaultDeps.TextVersions.Texts, vaultDeps.TextVersions.Versions, run, install.ActiveExecution.Watch)
}

func openTriggers(install pluginInstall, settingsDeps settingsWiring, vaultDeps vaultWiring) *trigger.Triggers {
	if vaultDeps.TextVersions == nil {
		return nil
	}
	return trigger.NewTriggers(trigger.TriggersDeps{
		Plugins: getRuntimeManifests(install.ActiveExecution.Runtime), Run: newPluginCommands(install, settingsDeps, vaultDeps, install.ActiveExecution, nil).Run, Readers: vaultReaders(vaultDeps),
		Notes: pluginsNewNoteReader(vaultDeps), Types: install.ActiveExecution.Watch, Versions: vaultDeps.TextVersions.Texts, Config: getWhenConfig(install, settingsDeps),
	})
}

func pluginsNewNoteReader(v vaultWiring) noteport.NoteReader {
	return note.NewReadFrontmatter(vaultReaders(v))
}

func getWhenConfig(install pluginInstall, settingsDeps settingsWiring) pluginport.WhenConfig {
	return settingsadapter.WhenConfig{
		App:     func() []plugindomain.Setting { return listAppSettings(settingsDeps) },
		Plugins: install.Execution.Runtime.GetPluginInfos,
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
