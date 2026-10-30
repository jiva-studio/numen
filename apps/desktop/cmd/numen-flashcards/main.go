// Command numen-flashcards is the window in which the user reviews cards.
//
// It is a shell over the plugins whose manifests name the desktop.flashcards target: it loads their
// frontends over HTTP, forwards the calls for the services they serve, and exposes every vault of
// the installation to them. A vault the index does not carry is read into it when a plugin queries it.
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

	agentport "github.com/jiva-studio/numen/libs/core/usecase/completion/agentcontract"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	vaultport "github.com/jiva-studio/numen/libs/core/usecase/vault/vaultcontract"
)

const appName = "numen-flashcards"

func main() {
	cfg := makeConfig(os.Stderr)
	var showVersion bool
	var debug debugOptions
	flag.StringVar(&cfg.IndexPath, "index", cfg.IndexPath, "path to the index database")
	flag.StringVar(&cfg.RegistryPath, "registry", "", "path to the vault list")
	flag.BoolVar(&showVersion, "version", false, "say what this build is and stop")
	registerDebugFlags(flag.CommandLine, &debug)
	flag.Parse()

	if showVersion {
		if _, err := fmt.Fprintln(os.Stdout, version.GetVersionLine(appName)); err != nil {
			fmt.Fprintln(os.Stderr, appName+":", err)
			os.Exit(1)
		}
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
	cfg.ErrorHandler = func(err error) {
		if _, writeErr := fmt.Fprintln(out, appName+":", err); writeErr != nil {
			return
		}
	}
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
		Config: cfg, Index: db, Out: os.Stderr, Label: appName,
	}

	api := makeWindowAPI(cfg, makeNotes(cfg, db, vaults))
	api.ListVaults = registry.List
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
// the agent the settings choose, whether the index carries a vault, and where links land.
func makeWindowAPI(cfg container.Config, notes container.Notes) *shell.API {
	agent := cfg.GetSettingsAgent()
	return &shell.API{
		GetAgent:             func() agentport.Agent { return agent },
		IsVaultIndexed:       notes.IsVaultIndexed,
		ResolveLinks:         notes.ResolveLinks,
		Window:               shell.NewWindow(platform.TargetFlashcards, nil),
		GetFrontendPluginIDs: cfg.GetFrontendPluginIDs,
		GetPluginDir:         cfg.GetPluginDir,
		PluginServices:       cfg.GetPluginServices(),
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

// followChanges notifies the plugins that something moved each time the index
// changes, including when another window writes it.
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
