//go:build !numen_debug

package app

import (
	"context"
	"flag"
	"net/http"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

// DebugOptions holds test settings for debug builds.
type DebugOptions struct{}

func RegisterDebugFlags(*flag.FlagSet, *Options) {}

func (d *DebugOptions) apply(*container.Config) error { return nil }

func (d *DebugOptions) ensureVault(container.Config, string) error { return nil }

func (d *DebugOptions) isServingHTTP() bool {
	return false
}

func (d *DebugOptions) serveHTTP(
	context.Context, container.Config, *container.OpenedWindow, http.Handler,
) error {
	return nil
}
