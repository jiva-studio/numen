//go:build nomcp

package main

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/flashcardsui"
	"github.com/jiva-studio/numen/modules/libs/core/container"
)

// Built with `nomcp`: a card cannot be asked about here, and none of the code
// that would let one is in the binary. The flag stays so that a script written
// for the ordinary build does not fail on an unknown one.

const unnamed = "this build reaches no agent"

func serveAgents(
	_ context.Context,
	_ container.Config,
	_ *container.Index,
	api *flashcardsui.API,
	_ bool,
	_ io.Writer,
) func() error {
	api.Unreachable.Store(unnamed)
	return func() error { return nil }
}
