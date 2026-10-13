//go:build !nomcp

package agents

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/acp"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/claudecode"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/mcp"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Router sends each task to the agent it names, or to the configured one. Every agent is built
// once, when the router is, and the router is not changed afterwards.
type Router struct {
	agents map[string]port.Agent
	// maxSteps is how many steps each agent may take on a task, from the settings.
	maxSteps map[string]int
	// tools are the vault tools every agent may call.
	tools []string
	use   string
	out   io.Writer
}

// NewRouter builds every agent the settings can name, against the tools served at url.
func NewRouter(opts Options, url, secret string, vocabulary map[string]mcp.Tool) *Router {
	settings := opts.Config.Agent
	report := func(err error) { fmt.Fprintln(opts.Out, "agent:", err) }
	tools := acp.Endpoint{URL: url, Token: secret}

	use := settings.Use
	if use == "" {
		use = agent.UseClaude
	}
	names := make([]string, 0, len(vocabulary))
	for name := range vocabulary {
		names = append(names, name)
	}
	slices.Sort(names)

	return &Router{
		agents: map[string]port.Agent{
			agent.UseClaude: claudecode.New(claudecode.Options{
				Settings:     settings.Claude,
				Root:         opts.Root,
				Tools:        claudecode.Endpoint{URL: url, Token: secret},
				Vocabulary:   vocabulary,
				Preview:      opts.Preview,
				ErrorHandler: report,
			}),
			agent.UseAntigravity: acp.New(acp.Options{
				Program:      agent.UseAntigravity,
				Settings:     settings.Antigravity,
				Root:         opts.Root,
				Tools:        tools,
				ErrorHandler: report,
			}),
			agent.UseCodex: acp.New(acp.Options{
				Program:      agent.UseCodex,
				Settings:     settings.Codex,
				Root:         opts.Root,
				Tools:        tools,
				ErrorHandler: report,
			}),
		},
		maxSteps: map[string]int{
			agent.UseClaude:      settings.Claude.MaxSteps,
			agent.UseAntigravity: settings.Antigravity.MaxSteps,
			agent.UseCodex:       settings.Codex.MaxSteps,
		},
		tools: names,
		use:   use,
		out:   opts.Out,
	}
}

// Close stops every agent and waits for them.
func (r *Router) Close() error {
	var errs []error
	for _, one := range r.agents {
		if closer, ok := one.(io.Closer); ok {
			if err := closer.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

// StartRun starts the task on the agent it names, or on the configured one, under the tools and
// step limit the settings give.
func (r *Router) StartRun(ctx context.Context, task port.Task) (port.Run, error) {
	use := r.use
	if task.Agent != "" {
		use = task.Agent
	}
	answering, ok := r.agents[use]
	if !ok {
		return nil, fmt.Errorf("no agent is named %q", use)
	}
	if task.Tools == nil {
		task.Tools = r.tools
	}
	if task.MaxSteps == 0 {
		task.MaxSteps = r.maxSteps[use]
	}
	fmt.Fprintf(r.out, "agents: dispatching task to %s (model: %s)\n", use, task.Model)
	return answering.StartRun(ctx, task)
}

// EndConversation ends the conversation on every agent.
func (r *Router) EndConversation(ctx context.Context, conversation string) error {
	var errs []error
	for _, one := range r.agents {
		if err := one.EndConversation(ctx, conversation); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
