//go:build !debug

package main

import (
	"context"
	"flag"
	"net/http"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/agents"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/window/editor"
	"github.com/jiva-studio/numen/modules/libs/core/container"
)

// *debugOptions holds test settings for debug builds.
type debugOptions struct{}

func registerDebugFlags(*flag.FlagSet) *debugOptions {
	return &debugOptions{}
}

func (d *debugOptions) apply(*container.Config) {}

func (d *debugOptions) ensureVault(container.Config, string) error { return nil }

func (d *debugOptions) isServingHTTP() bool {
	return false
}

func (d *debugOptions) serveHTTP(
	context.Context, container.Config, *editor.Installation, http.Handler, *agents.Endpoint,
) error {
	return nil
}
