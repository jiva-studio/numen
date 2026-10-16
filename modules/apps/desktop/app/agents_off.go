//go:build nomcp

package app

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/window/editor"
	"github.com/jiva-studio/numen/modules/libs/core/container"
)

// DefaultAgentAddr is empty in nomcp build.
const DefaultAgentAddr = ""

func serveAgents(context.Context, container.Config, *editor.Installation, agentOptions, io.Writer) (func() error, error) {
	return func() error { return nil }, nil
}
