package container

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/completion"

	"github.com/jiva-studio/numen/modules/libs/core/port/agentport"
	"github.com/jiva-studio/numen/modules/libs/core/port/completionport"
)

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
	return completion.NewRouter(
		func() completion.Config { return c.getSettingsOrDefaults().AI },
		c.PluginHost,
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
	c.PluginHost.SetCompleter(router)
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
