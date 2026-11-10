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
	windowagent "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/shell"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	agentuse "github.com/jiva-studio/numen/libs/core/kernel/usecase/agent"
	"github.com/jiva-studio/numen/libs/note/domain"

	pluginport "github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand/plugincontract"
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

func makeNotes(cfg container.Config, db *container.Index, vaults *container.OpenVaults) container.Notes {
	return cfg.Notes(db, vaults.UpdateIndex)
}

func run(cfg container.Config, debug debugOptions) error {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	selected, err := cfg.Settings()
	if err != nil {
		return err
	}
	cfg = cfg.SetSettings(selected)
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

	vaults := &container.OpenVaults{
		Config: cfg, Index: db, Out: os.Stderr, Label: appName,
	}

	api := makeWindowAPI(cfg, makeNotes(cfg, db, vaults))
	api.ListVaults = list.Vaults.List
	bindVaultIndexer(ctx, api, vaults)
	lent, err := openPluginVault(cfg, db, api)
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
	api.Themes = themes.Load(cfg.Themes, func(response string) {
		if response != "" {
			fmt.Fprintln(os.Stderr, "themes:", response)
		}
	}, os.Stderr)

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

func makeWindowAPI(cfg container.Config, notes container.Notes) *shell.API {
	agent := &windowagent.Handler{}
	if taking := cfg.GetSettingsAgent(); taking != nil {
		agent.Start, agent.End = agentuse.NewStartTask(taking), agentuse.NewEndConversation(taking)
	}
	return &shell.API{
		Agent:                agent,
		IsVaultIndexed:       notes.IsVaultIndexed,
		ResolveLinks:         notes.ResolveLinks,
		Window:               shell.NewWindow(platform.TargetFlashcards, nil),
		GetFrontendPluginIDs: cfg.GetFrontendPluginIDs,
		GetPluginDir:         cfg.GetPluginDir,
		PluginServices:       cfg.GetPluginServices(),
	}
}

func openPluginVault(
	cfg container.Config, db *container.Index, api *shell.API,
) (pluginport.PluginVault, error) {
	vaults, err := cfg.Registry()
	if err != nil {
		return nil, err
	}
	return cfg.OpenPluginVault(vaults, db,
		func() (domain.Vault, bool) { return domain.Vault{}, false },
		container.PluginVaultHooks{
			IndexState: api.EnsureIndexed,
			Moved:      api.WatchMoves,
		})
}

func bindVaultIndexer(ctx context.Context, api *shell.API, vaults *container.OpenVaults) {
	vaults.RecordVault = func(v domain.Vault) {
		api.ClearIndexFailure(v.ID)
		api.ReportChange()
	}
	api.SetVaultIndexer(ctx, vaults.IndexVault)
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
