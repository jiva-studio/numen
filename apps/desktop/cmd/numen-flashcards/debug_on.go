//go:build debug

package main

import (
	"context"
	"flag"
	"net/http"
	"os"

	"github.com/jiva-studio/numen/apps/desktop/internal/debugserve"
)

// debugOptions are the settings of a test run of a debug build.
type debugOptions struct {
	httpAddr string
}

func registerDebugFlags(fs *flag.FlagSet, opts *debugOptions) {
	fs.StringVar(&opts.httpAddr, "debug-http-addr", "", "serve the window over HTTP without opening an OS window")
}

func (d debugOptions) isServingHTTP() bool { return d.httpAddr != "" }

func (d debugOptions) serveHTTP(ctx context.Context, handler http.Handler) error {
	return debugserve.Serve(ctx, d.httpAddr, handler, os.Stdout)
}
