//go:build numen_debug

package main

import (
	"context"
	"flag"
	"net/http"
	"os"

	"github.com/jiva-studio/numen/apps/desktop/internal/debugserve"
	"github.com/jiva-studio/numen/libs/core/container"
)

// debugOptions are the settings of a test run of a debug build.
type debugOptions struct {
	httpAddr  string
	vaultPath string
}

func registerDebugFlags(fs *flag.FlagSet, opts *debugOptions) {
	fs.StringVar(&opts.httpAddr, "debug-http-addr", "", "serve the window over HTTP without opening an OS window")
	fs.StringVar(&opts.vaultPath, "debug-vault", "", "put this folder in the list of vaults before the window opens")
}

func (d debugOptions) registerVault(cfg container.Config) error {
	return debugserve.RegisterVault(cfg, d.vaultPath)
}

func (d debugOptions) isServingHTTP() bool { return d.httpAddr != "" }

func (d debugOptions) serveHTTP(ctx context.Context, cfg container.Config, handler http.Handler) error {
	return debugserve.Serve(ctx, cfg.Clock(), d.httpAddr, handler, os.Stdout)
}
