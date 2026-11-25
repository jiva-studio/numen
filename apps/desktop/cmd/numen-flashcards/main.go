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
)

const appName = "numen-flashcards"

func main() {
	cfg := platform.GetConfig()
	services := makeServices(os.Stderr)
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

	if err := run(cfg, services, debug); err != nil {
		fmt.Fprintln(os.Stderr, appName+":", err)
		os.Exit(1)
	}
}

func makeServices(out io.Writer) platform.Installation {
	services := platform.NewPluginServices(platform.TargetFlashcards)
	services.Resources.ErrorHandler = func(err error) {
		// gomutants:disable-next-line BRANCH_IF,CONDITIONALS_NEGATION reason="the branch returns where the function ends, so skipping it or inverting its condition ends the function at the same point"
		if _, writeErr := fmt.Fprintln(out, appName+":", err); writeErr != nil {
			return
		}
	}
	services.Execution.Runtime.SetSafeMode(false)
	return services
}

func run(cfg container.Config, services platform.Installation, debug debugOptions) error {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	cfg, _, err := cfg.ApplySettings(services.Execution, services.Preferences, services.Resources)
	if err != nil {
		return err
	}
	if err := debug.registerVault(cfg, services.Resources); err != nil {
		return err
	}

	opening := container.ShellRequest{
		Target: platform.TargetFlashcards,
		Log:    container.ShellLog{Out: os.Stderr, Label: appName, Notify: notifyUser},
	}
	started, err := cfg.OpenShell(services.Execution, services.Preferences, services.Endpoints, services.Resources, ctx, opening)
	if err != nil {
		return err
	}

	target := shutdown.NewSequence(
		stop,
		shutdown.Report(appName, os.Stderr, "plugins", func() error {
			return container.StopPlugins(services.Execution.Watch, context.Background())
		}),
		started.CloseVaults,
		shutdown.Report(appName, os.Stderr, "", started.CloseIndex),
	)
	defer target.Run()
	themes.Report(started.Themes, os.Stderr)

	pages, err := shell.CreatePagesHandler()
	if err != nil {
		return err
	}
	handler := started.API.NewHandler(pages)
	if debug.isServingHTTP() {
		return debug.serveHTTP(ctx, services.Resources, handler)
	}
	return runApplication(handler, target)
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
	app.Window.NewWithOptions(frame.ApplyPlatformFrame(application.WebviewWindowOptions{
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
