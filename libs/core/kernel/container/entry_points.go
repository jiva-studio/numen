package container

import (
	"context"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/ensureindexed"
	"io"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/connectioncall"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/shell"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/connectionlog"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/servicebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchcancel"
	vaultregistry "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// OpenEditor opens the registry and the cache, builds the editor window over them, opens the vault named, or
// the one shown last when none is named, and connects the plugins to the vault and the window. frame is the application
// window around the editor, or nil.
func (c Config) OpenEditor(
	exec *PluginExecution, prefs *PluginPreferences, endpoints *PluginEndpoints, res *Resources, ctx context.Context, opening WindowOpening,
) (*editor.Installation, error) {
	ports, db, err := c.openRegistryIndex(res, ctx)
	if err != nil {
		return nil, err
	}
	return c.openWindow(exec, prefs, endpoints, res, ctx, windowStartup{opening: opening, ports: ports, db: db, isEditor: true})
}

func (c Config) openRegistryIndex(res *Resources, ctx context.Context) (vaultregistry.Ports, *vaultIndex, error) {
	ports, err := c.registry()
	if err != nil {
		return ports, nil, err
	}
	db, err := c.openIndex(res, ctx)
	return ports, db, err
}

func connectWindow(install pluginInstall, exec *PluginExecution, vaultDeps vaultWiring, opened *windowRoutes, startup windowStartup, access windowPluginAccess) (*editor.Installation, error) {
	if !startup.isEditor {
		return opened.Installation, nil
	}
	state := opened.session
	exec.Runtime.pluginBinding.SetVaultPath(state.GetCurrentVault().Path)
	hooks := pluginVaultHooks{WriteWake: access.writeWake, Notes: &access.notes}
	if err := installPluginVault(install, vaultDeps, startup.ports, startup.db, state.GetShownVault, hooks); err != nil {
		return nil, err
	}
	setPluginVaultList(install.Endpoints.Lender, access.vaults)
	setPluginTexts(install.Endpoints.Lender, vaultDeps.TextVersions)
	shareEditorWindow(startup.db, opened, access.showTab, install, vaultDeps)
	return opened.Installation, nil
}

// Shell is what a shell window closes when it ends: CloseVaults waits until nothing is still writing to the
// index, CloseIndex closes it, and Themes is why the themes could not be loaded, or nil.
type Shell struct {
	API         *shell.API
	CloseVaults func()
	CloseIndex  func() error
	Themes      error
}

// ShellOpening identifies the target window, its services and its output.
type ShellOpening struct {
	Target string
	Log    ShellLog
}

// OpenShell opens the registry and the cache, creates the window of target, binds the notes, the vault list,
// the vault indexer, the plugin vault, the change feed and the themes of the shell window to api, and tells
// log.Notify what it cannot follow.
func (c Config) OpenShell(exec *PluginExecution, prefs *PluginPreferences, endpoints *PluginEndpoints, res *Resources, ctx context.Context, opening ShellOpening) (Shell, error) {
	exec = getPluginExecution(exec)
	ports, db, err := c.openRegistryIndex(res, ctx)
	if err != nil {
		return Shell{}, err
	}
	install := newPluginInstall(c, prefs, endpoints, res)
	settingsDeps, _, vaultDeps := buildVaultSearch(c, exec, prefs, res, nil)

	log := opening.Log
	api := &shell.API{
		Window:          shell.NewWindow(opening.Target, nil),
		Frontends:       shell.NewFrontendPlugins(exec.Runtime.Manifests.ListFrontendPluginIDs, func(id string) (string, bool) { return getPluginDir(exec.Runtime.Resolver, id) }),
		PluginServices:  servicebridge.Open(exec.Runtime.Activator, exec.Runtime.Resolver, exec.Runtime.Listing),
		ConnectionCalls: connectioncall.NewService(getConnectionCalls(prefs, endpoints), openRowLists(install, exec, settingsDeps, vaultDeps), searchcancel.NewSearches(GetClock(res))),
	}
	setModelCalls(install, exec, settingsDeps, vaultDeps, connectionlog.NewLogNotices().ReportSetAside)
	vaults := openVaults(vaultDeps, db, log.Out, log.Label, ensureindexed.NewReportRead(&api.Runs, &api.Changes).Execute)
	closers := Shell{API: api, CloseVaults: vaults.CloseAndWait, CloseIndex: db.Close}
	setShellNotes(api, db)
	vaultList := vaultregistry.NewList(ports.Vaults)
	api.Vaults = &vaultList
	api.ResolveLinks = query.NewResolveVaultLinks(ports.Vaults, query.NewResolveLinks(db.NoteQueries()))
	api.SetVaultIndexer(ctx, vaults.IndexVault)
	hooks := pluginVaultHooks{IndexState: api.EnsureIndexed, Moved: api.WatchIndexChanges}
	if err := installPluginVault(install, vaultDeps, ports, db, getNoShownVault, hooks); err != nil {
		return closers, err
	}
	closers.Themes = followShell(settingsDeps, vaultDeps, ctx, api, log.Label, log.Notify)
	return closers, nil
}

// ShellLog is where a shell window writes what it reports: Out under Label, and Notify for what the user
// must see.
type ShellLog struct {
	Out    io.Writer
	Label  string
	Notify func(message string)
}

func followShell(settingsDeps settingsWiring, vaultDeps vaultWiring, ctx context.Context, api *shell.API, label string, notify func(message string)) error {
	changes, err := vaultMoves(ctx, vaultDeps.IndexPath, vaultDeps.Watcher)
	if err != nil {
		notify(label + ": the index is not being followed: " + err.Error())
	} else {
		api.ReportChangesFrom(ctx, changes)
	}
	service, themesErr := settingsThemes(settingsDeps, Scales{}, func(response string) {
		if response != "" {
			notify("themes: " + response)
		}
	})
	api.Themes = service
	return themesErr
}

func getNoShownVault() (domain.Vault, bool) { return domain.Vault{}, false }

// FlushTimeout is how long a page may take to write its unsaved data when the vault closes.
func FlushTimeout() time.Duration { return windowsession.FlushTimeout }
