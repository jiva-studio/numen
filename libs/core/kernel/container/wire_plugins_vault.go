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
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/hold"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"

	vaultregistry "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// pluginVaultHooks is what a window adds to the vault it lets the plugins work.
type pluginVaultHooks struct {
	WriteWake  indexing.WriteWake
	IndexState func(ctx context.Context, v domain.Vault) (plugindomain.IndexState, error)
	WatchMoves func(ctx context.Context) (<-chan struct{}, error)
	Notes      *hostvault.NoteChanges
}

func getNoteChanges(notes noteUseCases) hostvault.NoteChanges {
	return hostvault.NoteChanges{
		Rename: notes.Place.Rename, Move: notes.Place.Move, Remove: notes.Place.RemoveNote, Links: notes.Edit.Linking,
		AddLinks: note.NewAddLinks(notes.Edit.Linking),
	}
}

func installPluginVault(install pluginInstall, vaultDeps vaultParts,
	registry vaultregistry.Ports, db *vaultIndex, current func() (domain.Vault, bool), hooks pluginVaultHooks,
) error {
	opened, err := openPluginVault(install, vaultDeps, registry, db, current, completeHooks(vaultDeps, db, hooks))
	if err != nil {
		return err
	}
	setPluginVault(install.Endpoints.Lender, opened)
	return nil
}

func openPluginVault(install pluginInstall, vaultDeps vaultParts,
	registry vaultregistry.Ports, db *vaultIndex, current func() (domain.Vault, bool), hooks pluginVaultHooks,
) (*pluginvault.Access, error) {
	cache, err := appstate.OpenPluginCachesIn(install.PluginCachePath)
	if err != nil {
		return nil, err
	}
	refresh := createRefresh(vaultDeps, db, db.Notes())
	index := func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		return err
	}
	useCases := createNoteUseCases(vaultDeps, db, index, noteViews{})
	arranger := newArranger(useCases, db, hooks.Notes, current)
	stores := derivedfile.PluginStores{Options: vaultconfig.Options{ServiceDir: install.ServiceDir}}
	return hostvault.NewAccess(
		hostvault.Session{
			GetCurrent: current,
			ListVaults: registry.Vaults.List,
			Find:       vaultregistry.NewFindByID(registry.Lookup).Execute,
			IndexState: hooks.IndexState,
			WatchMoves: hooks.WatchMoves,
			WriteWake:  hooks.WriteWake,
		},
		hostvault.Files{
			Stat:        file.NewHashFile(vaultDeps.Readers, vaultDeps.Writers),
			ReadFile:    file.NewReadFile(vaultDeps.Readers, vaultDeps.Writers),
			RangeReader: file.Read{Readers: vaultDeps.Readers},
		},
		hostvault.Writer{
			Write:  file.NewWriteFile(vaultDeps.Writers, index),
			Create: file.NewCreateFile(vaultDeps.Writers, index),
			Index:  file.NewScanFiles(index),
			Locks:  hold.NewLocks(vaultDeps.Writers, hold.LockHold),
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
	useCases noteUseCases, db *vaultIndex, window *hostvault.NoteChanges, current func() (domain.Vault, bool),
) hostvault.Arranger {
	arranger := hostvault.Arranger{
		Notes:       getNoteChanges(useCases),
		Frontmatter: useCases.Properties.Frontmatter,
		RunChecks:   getProblemChecks(getProblems(db.NoteProblems())),
		Problems:    vault.NewListParseProblems(db.NoteProblems()),
	}
	if window != nil {
		arranger.Window, arranger.GetWindowVault = *window, current
	}
	return arranger
}

func completeHooks(v vaultParts, db *vaultIndex, hooks pluginVaultHooks) pluginVaultHooks {
	if hooks.IndexState == nil {
		hooks.IndexState = query.NewGetIndexState(db.NoteQueries()).Execute
	}
	if hooks.WatchMoves == nil {
		hooks.WatchMoves = func(ctx context.Context) (<-chan struct{}, error) { return watchMoves(ctx, v.IndexPath, v.Watcher) }
	}
	return hooks
}

func setPluginWindow(lender *hostendpoint.Lender, window hostendpoint.WindowControl) {
	lender.Update(func(l *hostendpoint.Lent) { l.Window = window })
}

func openVersionItems(install pluginInstall, exec *PluginExecution, settingsDeps settingsParts, vaultDeps vaultParts) *textversion.VersionItems {
	if vaultDeps.TextVersions == nil {
		return nil
	}
	running := getRunningExecution(exec, install.SafeMode)
	run := newPluginCommands(getWhenConfig(settingsDeps, exec.Runtime.Listing), vaultDeps, running, nil).Run
	return textversion.NewVersionItems(vaultDeps.TextVersions.Texts, vaultDeps.TextVersions.Versions, run, running.Watch.ContentTypeTable)
}

func openTriggers(install pluginInstall, exec *PluginExecution, settingsDeps settingsParts, vaultDeps vaultParts) *windowsession.Triggers {
	if vaultDeps.TextVersions == nil {
		return nil
	}
	running := getRunningExecution(exec, install.SafeMode)
	config := getWhenConfig(settingsDeps, exec.Runtime.Listing)
	return windowsession.NewTriggers(windowsession.TriggersDeps{
		Plugins: running.Runtime.Manifests, Run: newPluginCommands(config, vaultDeps, running, nil).Run, Readers: vaultDeps.Readers,
		Types: running.Watch.ContentTypeTable, Versions: vaultDeps.TextVersions.Texts, Config: config,
	})
}

func getWhenConfig(settingsDeps settingsParts, plugins pluginmanifest.PluginLister) pluginport.WhenConfig {
	return settingsadapter.WhenConfig{
		ListAppSettings: func() []plugindomain.Setting { return listAppSettings(settingsDeps) },
		ListPlugins:     plugins.GetPluginInfos,
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
