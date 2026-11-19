package container

import (
	"context"
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
	ports, err := c.registry()
	if err != nil {
		return nil, err
	}
	db, err := OpenIndex(c, res, ctx)
	if err != nil {
		return nil, err
	}
	a := openIndexedPlugins(c, exec, prefs, endpoints, res, db)
	return openEditorWindow(ctx, a, res, ports, db, opening)
}

func openEditorWindow(ctx context.Context, a pluginsWiring, res *Resources, ports vaultregistry.Ports, db *Index, opening WindowOpening) (*OpenedWindow, error) {
	opened, err := openWindowWith(ctx, a, getLogEntries(res), ports, db, opening)
	if err != nil {
		return nil, err
	}
	return connectEditor(ports, db, opened)
}

func connectEditor(ports vaultregistry.Ports, db *Index, opened *windowWiring) (*OpenedWindow, error) {
	state, a := opened.state, opened.a
	setPluginVaultPath(a, state.GetCurrentVault().Path)
	pluginNotes := getNoteChanges(*opened.cases.PluginNotes)
	hooks := PluginVaultHooks{Written: state.Writes.NoteWritten, Notes: &pluginNotes}
	if err := installPluginVault(a, ports, db, state.GetShownVault, hooks); err != nil {
		return nil, err
	}
	shareVaultList(opened)
	setPluginTexts(a.Endpoints.Lender, a.vault.TextVersions)
	shareEditorWindow(db, opened)
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
	ports, err := c.registry()
	if err != nil {
		return Shell{}, err
	}
	db, err := OpenIndex(c, res, ctx)
	if err != nil {
		return Shell{}, err
	}
	opening.API.Window = shell.NewWindow(opening.Target, nil)
	return bindShell(plugins(c, exec, prefs, endpoints, res), ctx, ports, db, opening.API, opening.Log)
}

// ShellLog is where a shell window writes what it reports: Out under Label, and Notify for what the user
// must see.
type ShellLog struct {
	Out    io.Writer
	Label  string
	Notify func(message string)
}

func bindShell(
	a pluginsWiring, ctx context.Context, ports vaultregistry.Ports, db *Index, api *shell.API, log ShellLog,
) (Shell, error) {
	setModelCalls(a, connectionlog.NewLogNotices().ReportSetAside)
	vaults := openVaults(a.vault, db, log.Out, log.Label, api.ReportVaultRead)
	closers := Shell{CloseVaults: vaults.CloseAndWait, CloseIndex: db.Close}
	SetShellNotes(api, db)
	vaultList := vaultregistry.NewList(ports.Vaults)
	api.Vaults = &vaultList
	api.ResolveLinks = query.NewResolveVaultLinks(ports.Vaults, query.NewResolveLinks(db.NoteQueries()))
	api.SetVaultIndexer(ctx, vaults.IndexVault)
	hooks := PluginVaultHooks{IndexState: api.EnsureIndexed, Moved: api.WatchMoves}
	if err := installPluginVault(a, ports, db, getNoShownVault, hooks); err != nil {
		return closers, err
	}
	closers.Themes = followShell(a, ctx, api, log.Label, log.Notify)
	return closers, nil
}

func followShell(a pluginsWiring, ctx context.Context, api *shell.API, label string, notify func(message string)) error {
	changes, err := vaultMoves(ctx, a.vault.IndexPath, a.vault.Watcher)
	if err != nil {
		notify(label + ": the index is not being followed: " + err.Error())
	} else {
		api.ReportChangesFrom(ctx, changes)
	}
	service, themesErr := settingsThemes(a.settings, Scales{}, func(response string) {
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
