//go:build !debug

package main

import (
	"context"
	"flag"
	"net/http"

	"github.com/jiva-studio/numen/libs/core/container"
)

// debugOptions are the settings of a test run, which a release build has none of.
type debugOptions struct{}

func registerDebugFlags(*flag.FlagSet, *debugOptions) {}

func (debugOptions) registerVault(container.Config) error { return nil }

func (debugOptions) isServingHTTP() bool { return false }

func (debugOptions) serveHTTP(context.Context, http.Handler) error { return nil }
