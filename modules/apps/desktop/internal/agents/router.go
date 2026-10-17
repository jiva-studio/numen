//go:build !nomcp

package agents

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/mcp"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// errNoProvider is what a task is told where no plugin runs agent programs.
var errNoProvider = errors.New("no plugin provides agents")

// Router sends each task to the program it names, or to the configured one, with the tools, the
// folder, the step limit and the command line the settings give. The router is not changed after
// it is built.
type Router struct {
	provider port.AgentProvider
	settings settings.Agent
	// tools are the vault tools every agent may call.
	tools []string
	// declarations are how those tools are spoken about.
	declarations []port.ToolDeclaration
	endpoint     port.ToolEndpoint
	root         string
	preview      EditPreview
	out          io.Writer

	mu       sync.RWMutex
	isClosed bool
}

// NewRouter builds the router for the settings, against the tools served at url.
func NewRouter(opts Options, url, secret string, vocabulary map[string]mcp.Tool) *Router {
	names := make([]string, 0, len(vocabulary))
	for name := range vocabulary {
		names = append(names, name)
	}
	slices.Sort(names)

	return &Router{
		provider:     opts.Config.AgentProvider(),
		settings:     opts.Config.Agent,
		tools:        names,
		declarations: getToolDeclarations(names, vocabulary),
		endpoint:     port.ToolEndpoint{URL: url, Token: secret},
		root:         opts.Root,
		preview:      opts.Preview,
		out:          opts.Out,
	}
}

// getToolDeclarations is how the agent speaks about the tools the vault serves.
func getToolDeclarations(names []string, vocabulary map[string]mcp.Tool) []port.ToolDeclaration {
	declarations := make([]port.ToolDeclaration, 0, len(names))
	for _, name := range names {
		tool := vocabulary[name]
		declarations = append(declarations, port.ToolDeclaration{
			Name:    name,
			Title:   tool.Title,
			Kind:    tool.Kind,
			About:   tool.About,
			Element: tool.Inside,
			Match:   tool.Match,
			Text:    tool.Text,
		})
	}
	return declarations
}

// Close refuses the work that comes after it.
func (r *Router) Close() error {
	r.mu.Lock()
	r.isClosed = true
	r.mu.Unlock()
	return nil
}

// StartRun starts the task on the program it names, or on the configured one, under the tools,
// the folder and the program's section of the settings.
func (r *Router) StartRun(ctx context.Context, task port.Task) (port.Run, error) {
	r.mu.RLock()
	closed := r.isClosed
	r.mu.RUnlock()
	if closed {
		return nil, errors.New("this router is closed")
	}
	if r.provider == nil {
		return nil, errNoProvider
	}

	use, err := r.getProgram(ctx, task.Agent)
	if err != nil {
		return nil, err
	}
	section := r.settings.GetProgram(use)

	task.Agent = use
	if task.Tools == nil {
		task.Tools = r.tools
	}
	if task.MaxSteps == 0 {
		task.MaxSteps = section.MaxSteps
	}
	if task.Model == "" {
		task.Model = section.Model
	}
	task.Command = section.Command
	task.ShouldReadHooksAndSkills = section.ShouldReadHooksAndSkills
	task.Root = r.root
	task.Endpoint = r.endpoint
	task.ToolDeclarations = r.declarations

	fmt.Fprintf(r.out, "agents: dispatching task to %s (model: %s)\n", use, task.Model)
	run, err := r.provider.StartRun(ctx, task)
	if err != nil {
		return nil, err
	}
	return newDraftingRun(ctx, run, r.preview), nil
}

// getProgram is the program a task goes to: the one it names, else the one the settings name, else
// the default of the plugins.
func (r *Router) getProgram(ctx context.Context, asked string) (string, error) {
	if asked != "" {
		return asked, nil
	}
	return r.settings.GetAnswering(func() ([]port.AgentProgram, error) {
		return r.provider.GetPrograms(ctx, r.settings.GetCommands())
	})
}

// EndConversation ends the conversation on the program it was asked of.
func (r *Router) EndConversation(ctx context.Context, conversation string) error {
	if r.provider == nil {
		return nil
	}
	return r.provider.EndConversation(ctx, conversation)
}
