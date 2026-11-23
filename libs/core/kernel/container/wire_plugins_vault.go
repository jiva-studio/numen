package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginmanifest"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	pluginvault "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostvault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"

	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/trigger"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/hold"

	vaultregistry "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// pluginVaultHooks is what a window adds to the vault it lets the plugins work.
type pluginVaultHooks struct {
	WriteWake  indexing.WriteWake
	IndexState func(ctx context.Context, v domain.Vault) (plugindomain.IndexState, error)
	Moved      func(ctx context.Context) (<-chan struct{}, error)
	Notes      *hostvault.NoteChanges
}

func getNoteChanges(notes noteUseCases) hostvault.NoteChanges {
	return hostvault.NoteChanges{
		Rename: notes.Place.Rename, Move: notes.Place.Move, Remove: notes.Place.RemoveNote, Links: notes.Edit.Linking,
		AddLinks: note.NewAddLinks(notes.Edit.Linking),
	}
}

func installPluginVault(install pluginInstall, vaultDeps vaultWiring,
	registry vaultregistry.Ports, db *vaultIndex, current func() (domain.Vault, bool), hooks pluginVaultHooks,
) error {
	opened, err := pluginsOpenPluginVault(install, vaultDeps, registry, db, current, completeHooks(vaultDeps, db, hooks))
	if err != nil {
		return err
	}
	install.Endpoints.Lender.Update(func(l *hostendpoint.Lent) { l.Vault = opened })
	return nil
}

func pluginsOpenPluginVault(install pluginInstall, vaultDeps vaultWiring,
	registry vaultregistry.Ports, db *vaultIndex, current func() (domain.Vault, bool), hooks pluginVaultHooks,
) (*pluginvault.Access, error) {
	cache, err := appstate.OpenPluginCachesIn(install.PluginCachePath)
	if err != nil {
		return nil, err
	}
	refresh := vaultMakeRefresh(vaultDeps, db, db.Notes())
	index := func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		return err
	}
	useCases := vaultNotes(vaultDeps, db, index, noteViews{})
	arranger := newArranger(useCases, db, hooks.Notes, current)
	stores := derivedfile.PluginStores{Options: vaultconfig.Options{ServiceDir: install.ServiceDir}}
	storeReaders := func(v domain.Vault, pluginID string) (pluginvault.StoreReader, error) {
		return stores.Open(v, pluginID)
	}
	storeWriters := func(v domain.Vault, pluginID string) (pluginvault.StoreWriter, error) {
		return stores.OpenWriter(v, pluginID)
	}
	stat := file.NewHashFile(vaultDeps.Readers, vaultDeps.Writers)
	follow := hostvault.IndexFollow{}
	return hostvault.NewAccess(
		hostvault.Session{
			Current:    current,
			Vaults:     registry.Vaults.List,
			Find:       vaultregistry.NewFindByID(registry.Lookup).Execute,
			IndexState: hooks.IndexState,
			Moved:      hooks.Moved,
			WriteWake:  hooks.WriteWake,
		},
		hostvault.Files{
			Stat:        stat,
			Bytes:       file.NewReadBytes(vaultDeps.Readers, stat),
			ReadFile:    file.NewReadFile(vaultDeps.Readers, vaultDeps.Writers),
			RangeReader: file.Read{Readers: vaultDeps.Readers},
		},
		hostvault.Writer{
			Write:  file.NewWriteFile(vaultDeps.Writers, index),
			Create: file.NewCreateFile(vaultDeps.Writers, index),
			Index:  file.NewScanFiles(index),
			Locks:  hold.NewLocks(vaultDeps.Writers, hold.LockHold),
			Move:   note.NewMoveFile(useCases.Place.Move, db.Sources(), nil, follow),
			Remove: note.NewRemoveFile(&useCases.Place.Remove, follow),
			Stat:   stat,
			Count:  file.NewCountFiles(vaultDeps.Readers), Import: file.NewImportFile(vaultDeps.Writers),
			StoreReaders: storeReaders, StoreWriters: storeWriters,
		},
		getPluginLookups(db),
		arranger,
		hostvault.Stores{
			StoreReaders: storeReaders,
			StoreWriters: storeWriters,
			Cache:        cache,
		},
	), nil
}

func newArranger(
	useCases noteUseCases, db *vaultIndex, window *hostvault.NoteChanges, current func() (domain.Vault, bool),
) hostvault.Arranger {
	arranger := hostvault.Arranger{
		Notes:       getNoteChanges(useCases),
		Frontmatter: useCases.Properties.Frontmatter,
		Checks:      getProblemChecks(getProblems(db.NoteProblems())),
		Problems:    vault.NewGetParseProblems(db.NoteProblems()),
	}
	if window != nil {
		arranger.Window, arranger.WindowVault = *window, current
	}
	return arranger
}

func completeHooks(v vaultWiring, db *vaultIndex, hooks pluginVaultHooks) pluginVaultHooks {
	if hooks.IndexState == nil {
		hooks.IndexState = query.NewGetIndexState(db.NoteQueries()).Execute
	}
	if hooks.Moved == nil {
		hooks.Moved = func(ctx context.Context) (<-chan struct{}, error) { return vaultMoves(ctx, v.IndexPath, v.Watcher) }
	}
	return hooks
}

func openVersionItems(install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring) *textversion.VersionItems {
	if vaultDeps.TextVersions == nil {
		return nil
	}
	running := getRunningExecution(exec, install.SafeMode)
	run := newPluginCommands(getWhenConfig(settingsDeps, exec.Runtime.Listing), vaultDeps, running, nil).Run
	return textversion.NewVersionItems(vaultDeps.TextVersions.Texts, vaultDeps.TextVersions.Versions, run, running.Watch.ContentTypeTable)
}

func openTriggers(install pluginInstall, exec *PluginExecution, settingsDeps settingsWiring, vaultDeps vaultWiring) *trigger.Triggers {
	if vaultDeps.TextVersions == nil {
		return nil
	}
	running := getRunningExecution(exec, install.SafeMode)
	config := getWhenConfig(settingsDeps, exec.Runtime.Listing)
	return trigger.NewTriggers(trigger.TriggersDeps{
		Plugins: running.Runtime.Manifests, Run: newPluginCommands(config, vaultDeps, running, nil).Run, Readers: vaultDeps.Readers,
		Types: running.Watch.ContentTypeTable, Versions: vaultDeps.TextVersions.Texts, Config: config,
	})
}

func getWhenConfig(settingsDeps settingsWiring, plugins pluginmanifest.PluginLister) pluginport.WhenConfig {
	return settingsadapter.WhenConfig{
		App:     func() []plugindomain.Setting { return listAppSettings(settingsDeps) },
		Plugins: plugins.GetPluginInfos,
	}
}

func getProblemChecks(u *vault.GetProblems) func(ctx context.Context, v domain.Vault, named ...domain.Check) ([]domain.Check, []domain.VaultProblem, error) {
	return func(ctx context.Context, v domain.Vault, named ...domain.Check) ([]domain.Check, []domain.VaultProblem, error) {
		report, err := u.Execute(ctx, v, named...)
		return report.Ran, report.Found, err
	}
}

func getPluginDir(resolver pluginResolver, id string) (string, bool) {
	p, ok := resolver.GetPlugin(id)
	return p.Dir, ok
}
