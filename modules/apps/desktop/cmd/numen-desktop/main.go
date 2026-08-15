// numen-desktop is the window: one vault, one note, and the plex around it.
//
// It is a separate binary from the command line one because a webview links
// against the system's own browser, and a binary carrying one can no longer be
// built for another machine from this one. Both are entry points onto the same
// core.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/webui"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
)

func main() {
	var cfg container.Config
	flag.StringVar(&cfg.IndexPath, "index", "", "path to the index database")
	flag.StringVar(&cfg.RegistryPath, "registry", "", "path to the vault list")
	flag.Parse()

	if err := run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "numen-desktop:", err)
		os.Exit(1)
	}
}

func run(cfg container.Config) error {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	api, closeIndex, err := webui.Open(ctx, cfg, os.Stdout)
	if err != nil {
		return err
	}
	defer closeIndex()

	app := application.New(application.Options{
		Name: "numen",
		Assets: application.AssetOptions{
			Handler: api.Serving(webui.Pages()),
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "numen — " + api.Showing().Name,
		Width:  1280,
		Height: 860,
		URL:    "/",
	})

	return app.Run()
}

