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

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/webui"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
)

func main() {
	var cfg container.Config
	var agents agentOptions
	flag.StringVar(&cfg.IndexPath, "index", "", "path to the index database")
	flag.StringVar(&cfg.RegistryPath, "registry", "", "path to the vault list")
	flag.StringVar(&agents.addr, "mcp-addr", defaultAgentAddr,
		"where agents reach this vault; anything but a loopback address opens it to the network")
	flag.BoolVar(&agents.off, "no-mcp", false, "do not let agents reach this vault")
	flag.Parse()

	if err := run(cfg, agents); err != nil {
		fmt.Fprintln(os.Stderr, "numen:", err)
		os.Exit(1)
	}
}

func run(cfg container.Config, agents agentOptions) error {
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
	})

	return app.Run()
}

// agentOptions is what the person said about letting agents in.
type agentOptions struct {
	addr string
	off  bool
}
