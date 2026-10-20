package container

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/completion"
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
	provider port.AgentProvider
	getAgent func() settings.Agent
}

// GetProgram returns the program named. An empty name is the program the
// settings name under agent.use, else the default of the plugins.
func (p *completionPrograms) GetProgram(ctx context.Context, name string) (port.AgentProgram, error) {
	held := p.getAgent()
	if name == "" && !held.IsNamed() {
		return port.AgentProgram{}, fmt.Errorf("%w: the settings name no agent", port.ErrNoRoute)
	}
	if name == "" {
		name = held.GetUse()
	}
	listing, stop := context.WithTimeout(ctx, programsLimit)
	defer stop()
	programs, listErr := p.provider.GetPrograms(listing, held.GetCommands())
	found, isFound := findProgram(programs, name)
	switch {
	case !isFound && listErr != nil:
		return port.AgentProgram{}, fmt.Errorf("%w: %w", port.ErrUnavailable, listErr)
	case !isFound:
		return port.AgentProgram{}, fmt.Errorf("%w: no plugin lists the program %q", port.ErrNoRoute, name)
	case !found.IsInstalled:
		return port.AgentProgram{}, fmt.Errorf("%w: the program %q is not installed", port.ErrNoRoute, found.Name)
	}
	return found, nil
}

// findProgram is the program named, or, for an empty name, the default of the
// plugins.
func findProgram(programs []port.AgentProgram, name string) (port.AgentProgram, bool) {
	if name == "" {
		return port.GetDefaultProgram(programs)
	}
	for _, one := range programs {
		if one.Name == name {
			return one, true
		}
	}
	return port.AgentProgram{}, false
}

// StartRun starts the task on the program it names, with the command line,
// the options and, where the task names none, the model the settings give it.
func (p *completionPrograms) StartRun(ctx context.Context, task port.Task) (port.Run, error) {
	section := p.getAgent().GetProgram(task.Agent)
	if task.Model == "" {
		task.Model = section.Model
	}
	task.Command = section.Command
	task.Options = section.Options
	return p.provider.StartRun(ctx, task)
}
