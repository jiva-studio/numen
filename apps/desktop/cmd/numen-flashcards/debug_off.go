//go:build !debug

package main

import (
	"context"
	"flag"
	"net/http"
)

// debugOptions are the settings of a test run, which a release build has none of.
type debugOptions struct{}

func registerDebugFlags(*flag.FlagSet, *debugOptions) {}

func (debugOptions) isServingHTTP() bool { return false }

func (debugOptions) serveHTTP(context.Context, http.Handler) error { return nil }
