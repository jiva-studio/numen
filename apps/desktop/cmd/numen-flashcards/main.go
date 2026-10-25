// Command numen-flashcards is the window a person runs their cards in.
//
// It is a shell over the plugins whose manifests name the desktop.flashcards target: it loads their
// frontends over HTTP, forwards the calls for the services they serve, and lends them every vault of
// the installation. A vault the index does not carry is read into it when a plugin asks about it.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jiva-studio/numen/apps/desktop/internal/platform"
	"github.com/jiva-studio/numen/apps/desktop/internal/shutdown"
	"github.com/jiva-studio/numen/apps/desktop/internal/themes"
	"github.com/jiva-studio/numen/apps/desktop/internal/version"
	"github.com/jiva-studio/numen/libs/core/adapter/window/shell"
	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

const appName = "numen-flashcards"

func main() {
	cfg := makeConfig(os.Stderr)
	var showVersion bool
	var debug debugOptions
	flag.StringVar(&cfg.IndexPath, "index", "", "path to the index database")
	flag.StringVar(&cfg.RegistryPath, "registry", "", "path to the vault list")
	flag.BoolVar(&showVersion, "version", false, "say what this build is and stop")
	registerDebugFlags(flag.CommandLine, &debug)
	flag.Parse()

	if showVersion {
		fmt.Println(version.GetVersionLine(appName))
		return
	}

	if err := run(cfg, debug); err != nil {
		fmt.Fprintln(os.Stderr, appName+":", err)
		os.Exit(1)
	}
}

// makeConfig is the configuration of the window of the desktop.flashcards target, which reports what
// the core carried on past to out.
func makeConfig(out io.Writer) container.Config {
	cfg := platform.GetConfigFor(platform.TargetFlashcards)
	cfg.ErrorHandler = func(err error) { fmt.Fprintln(out, appName+":", err) }
	return cfg
}

// makeNotes is everything that acts on the vault's notes. What a write touched is brought up to date
// in the index through the opening the vault was opened with.
func makeNotes(cfg container.Config, db *container.Index, vaults *container.OpenVaults) container.Notes {
	return cfg.Notes(db.Queries(), db.Links(), db.Sources(), db.SourcesKnown(), vaults.UpdateIndex)
}

func run(cfg container.Config, debug debugOptions) error {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	if err := cfg.MigratePluginSettings(); err != nil {
		return err
	}
	selected, err := cfg.Settings()
	if err != nil {
		return err
	}
	cfg = cfg.SetSettings(selected)
	if err := debug.registerVault(cfg); err != nil {
		return err
	}

	registry, err := cfg.Registry()
	if err != nil {
		return err
	}

	// The editor's window may write the same index at the same time.
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return err
	}

	vaults := &container.OpenVaults{
		Config: cfg, Index: db, Lifetime: ctx, Out: os.Stderr, Label: appName,
	}

	api := makeWindowAPI(cfg, makeNotes(cfg, db, vaults))
	bindVaultIndexer(ctx, api, vaults)
	lent, err := openPluginVault(cfg, registry, db, api)
	if err != nil {
		return err
	}
	cfg.SetPluginVault(lent)

	target := shutdown.NewSequence(
		stop,
		shutdown.Report(appName, os.Stderr, "plugins", func() error {
			return cfg.StopPlugins(context.Background())
		}),
		vaults.CloseAndWait,
		shutdown.Report(appName, os.Stderr, "", db.Close),
	)
	defer target.Run()

	followChanges(ctx, cfg, api)
	api.Themes = themes.Load(cfg.Themes, func(response string) { fmt.Fprintln(os.Stderr, "themes:", response) }, os.Stderr)

	pages, err := shell.Pages()
	if err != nil {
		return err
	}
	handler := api.NewHandler(pages)
	if debug.isServingHTTP() {
		return debug.serveHTTP(ctx, handler)
	}
	return runApplication(handler, target)
}

// makeWindowAPI is the window of the desktop.flashcards target: its plugins, the services they serve,
// and whether the index carries a vault.
func makeWindowAPI(cfg container.Config, notes container.Notes) *shell.API {
	return &shell.API{
		IsVaultIndexed: notes.IsVaultIndexed,
		Window:         shell.NewWindow(platform.TargetFlashcards, nil),
		GetPluginIDs:   cfg.GetPluginIDs,
		GetPluginDir:   cfg.GetPluginDir,
		PluginServices: cfg.GetPluginServices(),
	}
}

// openPluginVault is every vault of the installation, lent to the plugins. The window shows none of
// its own, so a plugin names the vault of each call.
func openPluginVault(
	cfg container.Config, registry vaultport.VaultRegistry, db *container.Index, api *shell.API,
) (pluginport.PluginVault, error) {
	return cfg.OpenPluginVault(registry, db,
		func() (domain.Vault, bool) { return domain.Vault{}, false },
		container.PluginVaultHooks{
			IndexState: api.EnsureIndexed,
			Moved:      api.WatchMoves,
		})
}

// bindVaultIndexer connects the walk of a vault into the index to the API. A vault that moved is
// walked again where the last walk failed.
func bindVaultIndexer(ctx context.Context, api *shell.API, vaults *container.OpenVaults) {
	vaults.RecordVault = func(v domain.Vault) {
		api.ClearIndexFailure(v.ID)
		api.ReportChange()
	}
	api.SetVaultIndexer(ctx, vaults.IndexVault)
}

// followChanges tells the plugins that something moved each time the index changes, which another
// window writing it also does.
func followChanges(ctx context.Context, cfg container.Config, api *shell.API) {
	moves, err := cfg.Moves(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, appName+": the index is not being followed:", err)
		return
	}
	api.ReportChangesFrom(ctx, moves)
}

// runApplication opens the window and runs until it closes.
func runApplication(handler http.Handler, target *shutdown.Sequence) error {
	app := application.New(application.Options{
		Name: appName,
		Assets: application.AssetOptions{
			Handler: handler,
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		// Runs on the thread the window is drawn on, once the application has stopped dispatching.
		PostShutdown: target.Run,
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "numen — flashcards",
		Width:  760,
		Height: 720,
		URL:    "/",
	})
	return app.Run()
}
