//go:build !nomcp

package agents

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/claudecode"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/runner"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/mcp"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Router sends each task to the agent it names, or to the configured one. Every agent is built
// once, when the router is, and the router is not changed afterwards.
type Router struct {
	agents map[string]port.Agent
	// maxSteps is how many steps each agent may take on a task, from the settings.
	maxSteps map[string]int
	// tools are the vault tools every agent may call.
	tools    []string
	use      string
	out      io.Writer
	mu       sync.RWMutex
	isClosed bool
}

// NewRouter builds every agent the settings can name, against the tools served at url.
func NewRouter(opts Options, url, secret string, vocabulary map[string]mcp.Tool) *Router {
	settings := opts.Config.Agent
	report := func(err error) { fmt.Fprintln(opts.Out, "agent:", err) }

	use := settings.Use
	if use == "" {
		use = agent.UseClaude
	}
	names := make([]string, 0, len(vocabulary))
	for name := range vocabulary {
		names = append(names, name)
	}
	slices.Sort(names)

	closedAntigravity := &atomic.Bool{}
	closedCodex := &atomic.Bool{}

	claudeAgent := resolveClaudeAgent(opts, url, secret, vocabulary, report)

	return &Router{
		agents: map[string]port.Agent{
			agent.UseClaude: claudeAgent,
			agent.UseAntigravity: externalAgent{
				name:    agent.UseAntigravity,
				cfg:     opts.Config,
				command: settings.Antigravity.Command,
				root:    opts.Root,
				closed:  closedAntigravity,
			},
			agent.UseCodex: externalAgent{
				name:    agent.UseCodex,
				cfg:     opts.Config,
				command: settings.Codex.Command,
				root:    opts.Root,
				closed:  closedCodex,
			},
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

type externalAgent struct {
	name    string
	cfg     container.Config
	command []string
	root    string
	closed  *atomic.Bool
}

func (e externalAgent) Close() error {
	e.closed.Store(true)
	return nil
}

func (e externalAgent) StartRun(ctx context.Context, task port.Task) (port.Run, error) {
	if e.closed.Load() {
		return nil, errors.New("agent is closed")
	}
	task.Agent = e.name
	if port := e.cfg.AgentPort(); port != nil {
		return port.StartRun(ctx, task)
	}
	if len(e.command) > 0 {
		return runFallbackCommand(ctx, e.command, e.root, task)
	}
	return nil, fmt.Errorf("no agent plugin available for %q", e.name)
}

func (e externalAgent) EndConversation(ctx context.Context, conversation string) error {
	if port := e.cfg.AgentPort(); port != nil {
		return port.EndConversation(ctx, conversation)
	}
	return nil
}

type fallbackRun struct {
	steps  chan port.Step
	cmd    *exec.Cmd
	cancel context.CancelFunc
}

func (f *fallbackRun) GetSteps() <-chan port.Step {
	return f.steps
}

func (f *fallbackRun) Stop() error {
	f.cancel()
	if f.cmd != nil && f.cmd.Process != nil {
		_ = f.cmd.Process.Kill()
	}
	return nil
}

func runFallbackCommand(ctx context.Context, command []string, root string, task port.Task) (port.Run, error) {
	runCtx, cancel := context.WithCancel(ctx)
	args := append([]string{}, command[1:]...)
	args = append(args, task.Question)
	cmd := exec.CommandContext(runCtx, command[0], args...)
	cmd.Dir = root
	runner.Supervise(cmd)
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}
	run := &fallbackRun{
		steps:  make(chan port.Step, 4),
		cmd:    cmd,
		cancel: cancel,
	}
	go func() {
		defer close(run.steps)
		_ = cmd.Wait()
	}()
	return run, nil
}

func resolveClaudeAgent(opts Options, url, secret string, vocabulary map[string]mcp.Tool, report func(error)) port.Agent {
	if opts.ClaudeAgent != nil {
		return opts.ClaudeAgent
	}
	if p := opts.Config.AgentPortFor("md.numen.agent-claude"); p != nil {
		return p
	}
	if p := opts.Config.AgentPortFor("agent-claude"); p != nil {
		return p
	}
	if p := opts.Config.AgentPort(); p != nil {
		return p
	}
	return claudecode.New(claudecode.Options{
		Settings:         opts.Config.Agent.Claude,
		Root:             opts.Root,
		Tools:            claudecode.Endpoint{URL: url, Token: secret},
		ToolDeclarations: getToolDeclarations(vocabulary),
		Preview:          opts.Preview,
		ErrorHandler:     report,
	})
}

// getToolDeclarations is how the agent speaks about the tools the vault serves.
func getToolDeclarations(vocabulary map[string]mcp.Tool) map[string]claudecode.ToolDeclaration {
	declarations := make(map[string]claudecode.ToolDeclaration, len(vocabulary))
	for name, tool := range vocabulary {
		declarations[claudecode.ToolName(name)] = claudecode.ToolDeclaration{
			Title: tool.Title,
			Kind:  tool.Kind,
			Arguments: claudecode.Arguments{
				About:   tool.About,
				Element: tool.Inside,
				Match:   tool.Match,
				Text:    tool.Text,
			},
		}
	}
	return declarations
}

// Close stops every agent and waits for them.
func (r *Router) Close() error {
	r.mu.Lock()
	r.isClosed = true
	r.mu.Unlock()

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
	r.mu.RLock()
	closed := r.isClosed
	r.mu.RUnlock()
	if closed {
		return nil, errors.New("this router is closed")
	}

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
