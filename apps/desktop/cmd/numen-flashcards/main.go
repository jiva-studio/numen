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
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/connectioncall"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/shell"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
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
		// gomutants:disable-next-line reason="the error branch runs only when the call fails, and no test makes it fail"
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

	api := makeWindowAPI(cfg)
	started, err := container.OpenShell(cfg, ctx, platform.TargetFlashcards, api, container.ShellLog{Out: os.Stderr, Label: appName, Notify: notifyUser})
	if err != nil {
		return err
	}

	target := shutdown.NewSequence(
		stop,
		shutdown.Report(appName, os.Stderr, "plugins", func() error {
			return cfg.StopPlugins(context.Background())
		}),
		started.CloseVaults,
		shutdown.Report(appName, os.Stderr, "", started.CloseIndex),
	)
	defer target.Run()
	themes.Report(started.Themes, os.Stderr)

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
		GetFrontendPluginIDs: func() []string { return container.GetFrontendPluginIDs(cfg) },
		GetPluginDir:         func(id string) (string, bool) { return container.GetPluginDir(cfg, id) },
		PluginServices:       container.GetPluginServices(cfg),
		ConnectionCalls:      connectioncall.NewService(container.GetConnectionCalls(cfg), container.GetConnectionRows(cfg)),
	}
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

func notifyUser(message string) {
	fmt.Fprintln(os.Stderr, message)
}
