//go:build !numen_debug

package app

import (
	"context"
	"flag"
	"net/http"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

// DebugOptions holds test settings for debug builds.
type DebugOptions struct{}

func RegisterDebugFlags(*flag.FlagSet, *Options) {}

func (d *DebugOptions) apply(func(), *container.Resources) error { return nil }

func (d *DebugOptions) ensureVault(container.Config, *container.Resources, string) error {
	return nil
}

func (d *DebugOptions) isServingHTTP() bool {
	return false
}

func (d *DebugOptions) serveHTTP(
	context.Context, *container.Resources, *editor.Installation, http.Handler,
) error {
	return nil
}

func (d *DebugOptions) getFrame(frame editor.Frame) editor.Frame { return frame }
