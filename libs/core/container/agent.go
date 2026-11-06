package container

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/agentbridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/agenttoolsbridge"
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

// programsWait is how long a read of the settings waits for the providers to list their programs.
// Past it the read answers with the programs listed last, so a provider that is slow to start does
// not hold the window back.
const programsWait = time.Second

// getAgentPrograms returns the programs the agent plugins list, requested with the command lines
// from the settings. A provider that does not respond in time lists none, and the read waits for
// the providers no longer than programsWait: after that it answers with the programs listed last.
func (c Config) getAgentPrograms(provider agentport.AgentProvider, target settings.Agent) []agentport.AgentProgram {
	if provider == nil {
		return nil
	}
	list := func() []agentport.AgentProgram {
		ctx, stop := context.WithTimeout(context.Background(), programsLimit)
		defer stop()
		programs, err := provider.GetPrograms(ctx, target.GetCommands())
		if err != nil {
			c.handleError(fmt.Errorf("list agent programs: %w", err))
		}
		return programs
	}
	if c.programs == nil {
		return list()
	}
	return c.programs.Get(programsWait, list)
}

var errNoCompletionProvider = errors.New("no plugin host runs agent programs")

func (c Config) completionAgentSettings() completion.AgentSettings {
	target := c.getSettingsOrDefaults().Agent
	return completion.AgentSettings{
		IsNamed:  target.IsNamed(),
		Use:      target.GetUse(),
		Commands: target.GetCommands(),
		GetProgram: func(name string) completion.ProgramSettings {
			section := target.GetProgram(name)
			return completion.ProgramSettings{
				Model:   section.Model,
				Command: section.Command,
			}
		},
	}
}

// newCompletionRouter answers the questions plugins ask of a model with the
// default under ai.default in the settings file, read at every question,
// through the agent programs of the host's plugins.
func (c Config) newCompletionRouter() (*completion.Router, error) {
	provider := c.AgentProvider()
	if provider == nil {
		return nil, errNoCompletionProvider
	}
	return completion.NewRouter(
		func() completion.Config {
			return completion.Config{Default: completion.Route(c.getSettingsOrDefaults().AI.Default)}
		},
		completion.NewPrograms(provider, c.completionAgentSettings),
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

// GetSettingsAgent runs a window's tasks on the programs the plugins run, under the settings read at
// every task. A task naming no program goes to the one agent.use names, else to the default of the
// plugins. Each run connects to the tools of the plugins on an endpoint of its own. It is nil where
// AgentProvider is nil.
func (c Config) GetSettingsAgent() agentport.Agent {
	provider := c.getToolProvider()
	if provider == nil {
		return nil
	}
	return completion.NewSettingsAgent(completion.NewPrograms(provider, c.completionAgentSettings))
}

// getToolProvider returns AgentProvider with an MCP endpoint of its own for each run, serving the
// tools the plugins with the tools capability offer on the full server whose required permissions
// the plugin that runs the task has. It returns nil when AgentProvider is nil.
func (c Config) getToolProvider() agentport.AgentProvider {
	if c.PluginHost == nil || c.SafeMode {
		return nil
	}
	return agentbridge.NewServingRouter(c.PluginHost, func(agent agentport.Agent, pluginID string) agentport.Agent {
		return agenttoolsbridge.NewAgent(agent, c.PluginHost, agenttoolsbridge.Full, pluginID)
	})
}
