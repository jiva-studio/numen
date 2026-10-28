package container

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/agentbridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/toolserver"
	"github.com/jiva-studio/numen/libs/core/port/agentport"
	"github.com/jiva-studio/numen/libs/core/port/completionport"
	"github.com/jiva-studio/numen/libs/core/usecase/completion"
)

// programsLimit is how long the providers have to list the programs they run.
const programsLimit = 8 * time.Second

// AgentProvider runs tasks on the programs the plugins with the agent provider capability run. It
// is nil where there is no plugin host, or the host is in safe mode.
func (c Config) AgentProvider() agentport.AgentProvider {
	if c.PluginHost == nil || c.SafeMode {
		return nil
	}
	return agentbridge.NewRouter(c.PluginHost)
}

// getAgentPrograms are the programs the agent plugins list, asked for with the command lines the
// settings give. A provider that does not answer in time lists none.
func (c Config) getAgentPrograms(provider agentport.AgentProvider, target settings.Agent) []agentport.AgentProgram {
	if provider == nil {
		return nil
	}
	ctx, stop := context.WithTimeout(context.Background(), programsLimit)
	defer stop()
	programs, err := provider.GetPrograms(ctx, target.GetCommands())
	if err != nil {
		c.handleError(fmt.Errorf("list agent programs: %w", err))
	}
	return programs
}

var errNoCompletionProvider = errors.New("no plugin host runs agent programs")

// newCompletionRouter answers the questions plugins ask of a model with the
// agent each plugin's setting names, else the default under ai.default in the
// settings file, both read at every question, through the agent programs of the
// host's plugins.
func (c Config) newCompletionRouter() (*completion.Router, error) {
	provider := c.AgentProvider()
	if provider == nil {
		return nil, errNoCompletionProvider
	}
	choices, _ := c.PluginHost.(completionport.CompletionChoices)
	return completion.NewRouter(
		func() completion.Config {
			return completion.Config{Default: completion.Route(c.getSettingsOrDefaults().AI.Default)}
		},
		choices,
		&completionPrograms{provider: provider, getAgent: func() settings.Agent { return c.getSettingsOrDefaults().Agent }},
	)
}

// setCompleter has the plugin host answer the questions its plugins ask of a
// model. Safe mode serves none.
func (c Config) setCompleter() {
	if c.PluginHost == nil || c.SafeMode {
		return
	}
	router, err := c.newCompletionRouter()
	if err != nil {
		c.handleError(err)
		return
	}
	if host, ok := c.PluginHost.(completionport.CompleterHost); ok {
		host.SetCompleter(router)
	}
}

// completionPrograms finds the agent programs the plugins list and starts the
// tasks of single answers on them, under the settings read at every call.
type completionPrograms struct {
	provider agentport.AgentProvider
	getAgent func() settings.Agent
}

// GetProgram returns the program named. An empty name is the program the
// settings name under agent.use, else the default of the plugins.
func (p *completionPrograms) GetProgram(ctx context.Context, name string) (agentport.AgentProgram, error) {
	target := p.getAgent()
	if name == "" && !target.IsNamed() {
		return agentport.AgentProgram{}, fmt.Errorf("%w: the settings name no agent", completionport.ErrNoRoute)
	}
	if name == "" {
		name = target.GetUse()
	}
	listing, stop := context.WithTimeout(ctx, programsLimit)
	defer stop()
	programs, listErr := p.provider.GetPrograms(listing, target.GetCommands())
	found, isFound := findProgram(programs, name)
	switch {
	case !isFound && listErr != nil:
		return agentport.AgentProgram{}, fmt.Errorf("%w: %w", completionport.ErrUnavailable, listErr)
	case !isFound:
		return agentport.AgentProgram{}, fmt.Errorf("%w: no plugin lists the program %q", completionport.ErrNoRoute, name)
	case !found.IsInstalled:
		return agentport.AgentProgram{}, fmt.Errorf("%w: the program %q is not installed", completionport.ErrNoRoute, found.Name)
	}
	return found, nil
}

// findProgram is the program named, or, for an empty name, the default of the
// plugins.
func findProgram(programs []agentport.AgentProgram, name string) (agentport.AgentProgram, bool) {
	if name == "" {
		return agentport.GetDefaultProgram(programs)
	}
	for _, one := range programs {
		if one.Name == name {
			return one, true
		}
	}
	return agentport.AgentProgram{}, false
}

// StartRun starts the task on the program it names, with the command line,
// the options and, where the task names none, the model the settings give it.
func (p *completionPrograms) StartRun(ctx context.Context, task agentport.Task) (agentport.Run, error) {
	section := p.getAgent().GetProgram(task.Agent)
	if task.Model == "" {
		task.Model = section.Model
	}
	task.Command = section.Command
	task.Options = section.Options
	return p.provider.StartRun(ctx, task)
}

// GetSettingsAgent runs a window's tasks on the programs the plugins run, under the settings read at
// every task. A task naming no program goes to the one agent.use names, else to the default of the
// plugins. Each run reaches the tools of the plugins on an endpoint of its own. It is nil where
// AgentProvider is.
func (c Config) GetSettingsAgent() agentport.Agent {
	provider := c.AgentProvider()
	if provider == nil {
		return nil
	}
	return c.serveTools(&settingsAgent{programs: &completionPrograms{
		provider: provider,
		getAgent: func() settings.Agent { return c.getSettingsOrDefaults().Agent },
	}})
}

// settingsAgent starts each task on the program the task or the settings name.
type settingsAgent struct {
	programs *completionPrograms
}

// StartRun starts the task on the program it names, else on the one the settings choose, with the
// command line, the options and the model the settings give that program.
func (a *settingsAgent) StartRun(ctx context.Context, task agentport.Task) (agentport.Run, error) {
	if task.Agent == "" {
		program, err := a.programs.GetProgram(ctx, "")
		if err != nil {
			return nil, err
		}
		task.Agent = program.Name
	}
	return a.programs.StartRun(ctx, task)
}

// EndConversation says a conversation is over to every plugin it was asked in.
func (a *settingsAgent) EndConversation(ctx context.Context, conversation string) error {
	return a.programs.provider.EndConversation(ctx, conversation)
}

// serveTools gives each run of the agent an MCP endpoint of its own, serving the tools the plugins
// with the tools capability offer on the full server. A nil agent stays nil.
func (c Config) serveTools(agent agentport.Agent) agentport.Agent {
	if agent == nil {
		return nil
	}
	return toolserver.NewAgent(agent, c.PluginHost, toolserver.Full)
}
