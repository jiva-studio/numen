package app

import (
	"context"
	"io"
	"path/filepath"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/agents"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/window/editor"
	"github.com/jiva-studio/numen/modules/libs/core/container"
)

// DefaultAgentAddr is the default MCP listening address.
const DefaultAgentAddr = "127.0.0.1:23579"

// unnamed is what the panel is told where the settings name no agent.
const unnamed = "no agent is named in the settings"

func serveAgents(ctx context.Context, cfg container.Config, opened *editor.Installation, opts agentOptions, out io.Writer) (func() error, error) {
	if opts.isOff {
		return func() error { return nil }, nil
	}

	if !cfg.Agent.IsServingTools() {
		opened.State.Unreachable.Store(unnamed)
		return func() error { return nil }, nil
	}

	root, err := filepath.Abs(opened.GetCurrentVault().Path)
	if err != nil {
		return nil, err
	}

	addr := opts.addr
	if addr == "" {
		addr = DefaultAgentAddr
	}

	served, err := agents.Serve(ctx, agents.Options{
		Config:       cfg,
		Addr:         addr,
		IsAnnouncing: true,
		Root:         root,
		Out:          out,
	})
	if err != nil && opts.addr == "" {
		served, err = agents.Serve(ctx, agents.Options{
			Config:       cfg,
			Addr:         "127.0.0.1:0",
			IsAnnouncing: false,
			Root:         root,
			Out:          out,
		})
	}
	if err != nil {
		return nil, err
	}

	return served.Close, nil
}
