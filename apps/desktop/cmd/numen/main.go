// Command numen is the window: one vault, one note, and the plex around it.
//
// It is a separate binary from the command-line one because a webview links
// against the system browser, and a binary that links one cannot be
// cross-compiled for another machine from this one. Both are entry points onto
// the same core.
//
// It also carries the tools an agent uses to work on the vault.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jiva-studio/numen/apps/desktop/app"
	"github.com/jiva-studio/numen/apps/desktop/internal/plugincli"
	"github.com/jiva-studio/numen/apps/desktop/internal/version"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "plugin" {
		os.Exit(plugincli.Run(os.Args[2:], os.Stdout, os.Stderr))
	}

	opts := app.NewOptions()
	flag.StringVar(&opts.IndexPath, "index", "", "path to the index database")
	flag.StringVar(&opts.RegistryPath, "registry", "", "path to the vault list")
	flag.StringVar(&opts.Vault, "vault", "",
		"the vault to open: a name, a path or an identity; the one opened last by default")
	flag.StringVar(&opts.McpAddr, "mcp-addr", app.DefaultAgentAddr,
		"where agents reach this vault; anything but a loopback address opens it to the network")
	flag.BoolVar(&opts.NoMcp, "no-mcp", false, "do not let agents reach this vault")
	flag.Float64Var(&opts.InterfaceScale, "interface-scale", 0,
		"how large the interface is drawn, 1 being as designed; this launch alone")
	flag.Float64Var(&opts.TextScale, "text-scale", 0,
		"how large the text a person reads is set, 1 being as designed; this launch alone")
	flag.BoolVar(&opts.RebuildIndex, "rebuild-index", false,
		"read every file and put it in the index again, whatever the index remembers")
	flag.BoolVar(&opts.SafeMode, "safe-mode", false, "disable plugins for emergency recovery")
	flag.BoolVar(&opts.ShowVersion, "version", false, "say what this build is and stop")
	app.RegisterDebugFlags(flag.CommandLine, opts)
	flag.Parse()

	if opts.ShowVersion {
		fmt.Println(version.GetVersionLine("numen"))
		return
	}

	if err := app.Run(opts); err != nil {
		fmt.Fprintln(os.Stderr, "numen:", err)
		os.Exit(1)
	}
}
