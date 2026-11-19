package container

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/shell"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/connectionlog"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	vaultregistry "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// OpenEditor opens the registry and the cache, builds the editor window over them, opens the vault named, or
// the one shown last when none is named, and connects the plugins to the vault and the window. frame is the application
// window around the editor, or nil.
func (c Config) OpenEditor(
	exec *PluginExecution, prefs *PluginPreferences, endpoints *PluginEndpoints, res *Resources, ctx context.Context, opening WindowOpening,
) (*OpenedWindow, error) {
	exec = getPluginExecution(exec)
	ports, err := c.registry()
	if err != nil {
		return nil, err
	}
	db, err := OpenIndex(c, res, ctx)
	if err != nil {
		return nil, err
	}
	install := newPluginInstall(c, endpoints, res)
	settingsDeps, searchDeps, vaultDeps := openIndexedVaultSearch(c, exec, prefs, endpoints, res, db)
	startup := windowStartup{opening: opening, ports: ports, db: db, entries: getLogEntries(res)}
	first, err := getFirstWindowVault(startup)
	if err != nil {
		return nil, err
	}
	plugins := openWindowPlugins(install, exec, settingsDeps, vaultDeps, db)
	opened, closeEmbedder := buildWindow(startup, ctx, plugins, searchDeps, vaultDeps)
	mountServices(opened, install, exec, settingsDeps, searchDeps, vaultDeps)
	if err := startWindow(ctx, opened, first); err != nil {
		return nil, errors.Join(err, closeEmbedder(), db.Close())
	}

	return connectEditor(install, exec, vaultDeps, ports, db, opened)
}

func connectEditor(install pluginInstall, exec *PluginExecution, vaultDeps vaultWiring, ports vaultregistry.Ports, db *Index, opened *windowWiring) (*OpenedWindow, error) {
	state := opened.state
	setPluginVaultPath(exec.Runtime.PluginBinding, state.GetCurrentVault().Path)
	pluginNotes := getNoteChanges(*opened.cases.PluginNotes)
	hooks := PluginVaultHooks{Written: state.Writes.NoteWritten, Notes: &pluginNotes}
	if err := installPluginVault(install, vaultDeps, ports, db, state.GetShownVault, hooks); err != nil {
		return nil, err
	}
	shareVaultList(opened, install)
	setPluginTexts(vaultDeps)
	shareEditorWindow(db, opened, install, vaultDeps)
	return getOpenedWindow(opened), nil
}

// Shell is what a shell window closes when it ends: CloseVaults waits until nothing is still writing to the
// index, CloseIndex closes it, and Themes is why the themes could not be loaded, or nil.
type Shell struct {
	CloseVaults func()
	CloseIndex  func() error
	Themes      error
}

// ShellOpening identifies the target window, its services and its output.
type ShellOpening struct {
	Target string
	API    *shell.API
	Log    ShellLog
}

// OpenShell opens the registry and the cache, gives api the window of target, binds the notes, the vault list,
// the vault indexer, the plugin vault, the change feed and the themes of the shell window to api, and tells
// log.Notify what it cannot follow.
func (c Config) OpenShell(exec *PluginExecution, prefs *PluginPreferences, endpoints *PluginEndpoints, res *Resources, ctx context.Context, opening ShellOpening) (Shell, error) {
	exec = getPluginExecution(exec)
	ports, err := c.registry()
	if err != nil {
		return Shell{}, err
	}
	db, err := OpenIndex(c, res, ctx)
	if err != nil {
		return Shell{}, err
	}
	opening.API.Window = shell.NewWindow(opening.Target, nil)
	install := newPluginInstall(c, endpoints, res)
	settingsDeps, _, vaultDeps := openVaultSearch(c, exec, prefs, endpoints, res)

	api, log := opening.API, opening.Log
	setModelCalls(install, exec, settingsDeps, vaultDeps, connectionlog.NewLogNotices().ReportSetAside)
	vaults := openVaults(vaultDeps, db, log.Out, log.Label, api.ReportVaultRead)
	closers := Shell{CloseVaults: vaults.CloseAndWait, CloseIndex: db.Close}
	SetShellNotes(api, db)
	vaultList := vaultregistry.NewList(ports.Vaults)
	api.Vaults = &vaultList
	api.ResolveLinks = query.NewResolveVaultLinks(ports.Vaults, query.NewResolveLinks(db.NoteQueries()))
	api.SetVaultIndexer(ctx, vaults.IndexVault)
	hooks := PluginVaultHooks{IndexState: api.EnsureIndexed, Moved: api.WatchMoves}
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
	changes, err := vaultMoves(vaultDeps, ctx)
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
