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

	"github.com/jiva-studio/numen/apps/desktop/internal/frame"
	"github.com/jiva-studio/numen/apps/desktop/internal/platform"
	"github.com/jiva-studio/numen/apps/desktop/internal/shutdown"
	"github.com/jiva-studio/numen/apps/desktop/internal/themes"
	"github.com/jiva-studio/numen/apps/desktop/internal/version"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/shell"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/note/domain"
)

const appName = "numen-flashcards"

func main() {
	cfg := makeConfig(os.Stderr)
	var showVersion bool
	var debug debugOptions
	flag.StringVar(&cfg.IndexPath, "index", cfg.IndexPath, "path to the index database")
	flag.StringVar(&cfg.RegistryPath, "registry", "", "path to the vault list")
	flag.BoolVar(&showVersion, "version", false, "print the version and exit")
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

func makeConfig(out io.Writer) container.Config {
	cfg := platform.GetConfigFor(platform.TargetFlashcards)
	cfg.ErrorHandler = func(err error) {
		// gomutants:disable-next-line
		if _, writeErr := fmt.Fprintln(out, appName+":", err); writeErr != nil {
			return
		}
	}
	return cfg
}

func run(cfg container.Config, debug debugOptions) error {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	cfg, _, err := cfg.ApplySettings()
	if err != nil {
		return err
	}
	if err := debug.registerVault(cfg); err != nil {
		return err
	}

	list, err := cfg.Registry()
	if err != nil {
		return err
	}

	// The editor's window may write the same index at the same time.
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return err
	}

	vaults := cfg.OpenVaults(db, os.Stderr, appName)

	api := makeWindowAPI(cfg)
	cfg.SetShellNotes(api, db, vaults.UpdateIndex)
	api.ListVaults = cfg.GetVaultList(list).Execute
	cfg.BindVaultIndexer(ctx, api, vaults)
	registry, err := cfg.Registry()
	if err != nil {
		return err
	}
	if err := cfg.InstallPluginVault(registry, db, getNoCurrentVault, getPluginVaultHooks(api)); err != nil {
		return err
	}

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
	themes.Report(cfg.SetShellThemes(api, func(response string) {
		if response != "" {
			fmt.Fprintln(os.Stderr, "themes:", response)
		}
	}), os.Stderr)

	pages, err := shell.Pages()
	if err != nil {
		return err
	}
	handler := api.NewHandler(pages)
	if debug.isServingHTTP() {
		return debug.serveHTTP(ctx, cfg, handler)
	}
	return runApplication(handler, target)
}

func makeWindowAPI(cfg container.Config) *shell.API {
	return &shell.API{
		Agent:                cfg.GetSettingsAgentHandler(),
		Window:               shell.NewWindow(platform.TargetFlashcards, nil),
		GetFrontendPluginIDs: cfg.GetFrontendPluginIDs,
		GetPluginDir:         cfg.GetPluginDir,
		PluginServices:       cfg.GetPluginServices(),
	}
}

func getNoCurrentVault() (domain.Vault, bool) { return domain.Vault{}, false }

func getPluginVaultHooks(api *shell.API) container.PluginVaultHooks {
	return container.PluginVaultHooks{IndexState: api.EnsureIndexed, Moved: api.WatchMoves}
}

func followChanges(ctx context.Context, cfg container.Config, api *shell.API) {
	moves, err := cfg.Moves(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, appName+": the index is not being followed:", err)
		return
	}
	api.ReportChangesFrom(ctx, moves)
}

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
	app.Window.NewWithOptions(frame.Window(application.WebviewWindowOptions{
		Title:  "numen — flashcards",
		Width:  760,
		Height: 720,
		URL:    "/",
	}))
	return app.Run()
}
