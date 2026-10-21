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

// errNoProgram is what a task is told where the settings name no program and the plugins list none.
var errNoProgram = errors.New("no plugin lists an agent program")

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
	limitedOnly  bool

	mu       sync.RWMutex
	isClosed bool
	// conversations are the ones tasks were started in, and not ended.
	conversations map[string]struct{}
}

// NewRouter builds the router for the settings, against the tools served at url.
func NewRouter(opts Options, url, secret string, vocabulary map[string]mcp.Tool) *Router {
	names := make([]string, 0, len(vocabulary))
	for name := range vocabulary {
		names = append(names, name)
	}
	slices.Sort(names)

	return &Router{
		provider:     getProvider(opts),
		settings:     opts.Config.Agent,
		tools:        names,
		declarations: getToolDeclarations(names, vocabulary),
		endpoint:     port.ToolEndpoint{URL: url, Token: secret},
		root:         opts.Root,
		preview:      opts.Preview,
		out:          opts.Out,
		limitedOnly:  opts.LimitedToolsOnly,
	}
}

// getProvider is the provider the options give, else the one the plugins make. Nil where there is
// none.
func getProvider(opts Options) port.AgentProvider {
	if opts.Provider != nil {
		return opts.Provider
	}
	return opts.Config.AgentProvider()
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

// Close refuses the work that comes after it, and ends the conversations tasks were started in.
func (r *Router) Close() error {
	r.mu.Lock()
	r.isClosed = true
	ended := make([]string, 0, len(r.conversations))
	for conversation := range r.conversations {
		ended = append(ended, conversation)
	}
	r.conversations = nil
	r.mu.Unlock()

	if r.provider == nil {
		return nil
	}
	ctx, stop := context.WithTimeout(context.Background(), bound)
	defer stop()

	var failed []error
	for _, conversation := range ended {
		failed = append(failed, r.provider.EndConversation(ctx, conversation))
	}
	return errors.Join(failed...)
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
	if r.limitedOnly {
		if err := r.checkLimited(ctx, use); err != nil {
			return nil, err
		}
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
	task.Options = section.Options
	task.Root = r.root
	task.Endpoint = r.endpoint
	task.ToolDeclarations = r.declarations

	r.mu.Lock()
	if r.isClosed {
		r.mu.Unlock()
		return nil, errors.New("this router is closed")
	}
	if task.Conversation != "" {
		if r.conversations == nil {
			r.conversations = make(map[string]struct{})
		}
		r.conversations[task.Conversation] = struct{}{}
	}
	r.mu.Unlock()
	fmt.Fprintf(r.out, "agents: dispatching task to %s (model: %s)\n", use, task.Model)
	run, err := r.provider.StartRun(ctx, task)
	if err != nil {
		return nil, err
	}
	return newDraftingRun(ctx, run, r.preview), nil
}

// checkLimited refuses a program that runs tools of its own.
func (r *Router) checkLimited(ctx context.Context, use string) error {
	programs, listErr := r.provider.GetPrograms(ctx, r.settings.GetCommands())
	for _, one := range programs {
		if one.Name == use && one.LimitsTools {
			return nil
		}
	}
	return errors.Join(fmt.Errorf("the program %q runs tools of its own, which this window does not allow", use), listErr)
}

// getProgram is the program a task goes to: the one it names, else the one the settings name, else
// the default of the plugins.
func (r *Router) getProgram(ctx context.Context, asked string) (string, error) {
	if asked != "" {
		return asked, nil
	}
	if !r.settings.IsNamed() {
		return "", errors.New("no agent is named in the settings")
	}
	if use := r.settings.GetUse(); use != "" {
		return use, nil
	}

	programs, listErr := r.provider.GetPrograms(ctx, r.settings.GetCommands())
	chosen, found := port.GetDefaultProgram(programs)
	if !found {
		return "", errors.Join(errNoProgram, listErr)
	}
	return chosen.Name, nil
}

// EndConversation ends the conversation on the program it was asked of.
func (r *Router) EndConversation(ctx context.Context, conversation string) error {
	r.mu.Lock()
	delete(r.conversations, conversation)
	r.mu.Unlock()
	if r.provider == nil {
		return nil
	}
	return r.provider.EndConversation(ctx, conversation)
}
