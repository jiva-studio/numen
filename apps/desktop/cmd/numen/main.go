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
		"address agents connect to; any address other than loopback opens the vault to the network")
	flag.BoolVar(&opts.NoMcp, "no-mcp", false, "do not let agents connect to this vault")
	flag.Float64Var(&opts.InterfaceScale, "interface-scale", 0,
		"scale of the interface (controls and spacing); 1 is the default; this launch only")
	flag.Float64Var(&opts.TextScale, "text-scale", 0,
		"scale of note and book text; 1 is the default; this launch only")
	flag.BoolVar(&opts.RebuildIndex, "rebuild-index", false,
		"read every file again and rewrite its index entries, whatever the index already holds")
	flag.BoolVar(&opts.SafeMode, "safe-mode", false, "disable plugins for emergency recovery")
	flag.BoolVar(&opts.ShowVersion, "version", false, "print the version and exit")
	app.RegisterDebugFlags(flag.CommandLine, opts)
	flag.Parse()

	if opts.ShowVersion {
		if _, err := fmt.Fprintln(os.Stdout, version.GetVersionLine("numen")); err != nil {
			fmt.Fprintln(os.Stderr, "numen:", err)
			os.Exit(1)
		}
		return
	}

	if err := app.Run(opts); err != nil {
		fmt.Fprintln(os.Stderr, "numen:", err)
		os.Exit(1)
	}
}
