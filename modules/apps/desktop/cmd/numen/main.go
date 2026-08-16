// Command numen is the window: one vault, one note, and the plex around it.
//
// It is a separate binary from the command line one because a webview links
// against the system's own browser, and a binary carrying one can no longer be
// built for another machine from this one. Both are entry points onto the same
// core.
//
// It also carries the tools an agent works the vault through, unless it is
// built with the `nomcp` tag.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/webui"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
)

func main() {
	var cfg container.Config
	var agents agentOptions
	var zoom float64
	flag.StringVar(&cfg.IndexPath, "index", "", "path to the index database")
	flag.StringVar(&cfg.RegistryPath, "registry", "", "path to the vault list")
	flag.StringVar(&agents.addr, "mcp-addr", defaultAgentAddr,
		"where agents reach this vault; anything but a loopback address opens it to the network")
	flag.BoolVar(&agents.off, "no-mcp", false, "do not let agents reach this vault")
	flag.Float64Var(&zoom, "zoom", 0, "how large everything is drawn, 1 being as designed")
	flag.Parse()

	if err := run(cfg, agents, zoom); err != nil {
		fmt.Fprintln(os.Stderr, "numen:", err)
		os.Exit(1)
	}
}

func run(cfg container.Config, agents agentOptions, zoom float64) error {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	opened, err := webui.Open(ctx, cfg, os.Stdout)
	if err != nil {
		return err
	}
	defer opened.Close()

	closeAgents, err := serveAgents(ctx, cfg, opened, agents, os.Stdout)
	if err != nil {
		return err
	}
	defer closeAgents()

	pages, err := webui.Pages()
	if err != nil {
		return err
	}

	app := application.New(application.Options{
		Name: "numen",
		Assets: application.AssetOptions{
			Handler: opened.API.Serving(pages),
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "numen — " + opened.API.Showing().Name,
		Width:  1280,
		Height: 860,
		URL:    "/",
		Zoom:   drawnAt(zoom),
	})

	return app.Run()
}

// drawnAt is how large everything is drawn.
//
// A screen says how many pixels it has and not how large they are, so the
// desktop is asked: GDK_DPI_SCALE is what the person told their session text
// should be scaled by, and this window is drawn to match. The flag says it
// outright for a person who wants this one window larger.
func drawnAt(asked float64) float64 {
	if asked > 0 {
		return asked
	}
	if scale, err := strconv.ParseFloat(os.Getenv("GDK_DPI_SCALE"), 64); err == nil && scale > 0 {
		return scale
	}
	return 1
}

// agentOptions is what the person said about letting agents in.
type agentOptions struct {
	addr string
	off  bool
}
