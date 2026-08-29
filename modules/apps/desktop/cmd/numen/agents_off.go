//go:build nomcp

package main

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/webui"
	"github.com/jiva-studio/numen/modules/libs/core/container"
)

// Built with `nomcp`: no agent reaches this vault, and none of the code that
// would let one is in the binary. The flags stay so that a script written for
// the ordinary build does not fail on an unknown one.

const defaultAgentAddr = ""

func serveAgents(context.Context, container.Config, *webui.Opened, agentOptions, io.Writer) (func() error, error) {
	return func() error { return nil }, nil
}
