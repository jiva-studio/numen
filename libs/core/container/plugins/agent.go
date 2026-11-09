package plugins

import (
	"context"
	"fmt"
	"time"

	settingsadapter "github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/agentbridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/agenttoolsbridge"
	"github.com/jiva-studio/numen/libs/core/port/agentport"
	"github.com/jiva-studio/numen/libs/core/usecase/agent"
)

// ProgramsLimit is how long the providers have to list the programs they run.
const ProgramsLimit = 8 * time.Second

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

// GetAgentPrograms lists the agent programs from provider, using the cached program list when set.
func (c Config) GetAgentPrograms(provider agentport.AgentProvider, target settingsadapter.Agent) []agentport.AgentProgram {
	if provider == nil {
		return nil
	}
	list := func() []agentport.AgentProgram {
		ctx, stop := context.WithTimeout(context.Background(), ProgramsLimit)
		defer stop()
		programs, err := provider.GetPrograms(ctx, target.GetCommands())
		if err != nil {
			c.handleError(fmt.Errorf("list agent programs: %w", err))
		}
		return programs
	}
	if c.Programs == nil {
		return list()
	}
	return c.Programs.Get(programsWait, list, c.RefreshSettingEvents)
}

func (c Config) getAgentSettings() agent.AgentSettings {
	target := c.GetSettingsOrDefaults().Agent
	return agent.AgentSettings{
		IsNamed:  target.IsNamed(),
		Use:      target.GetUse(),
		Commands: target.GetCommands(),
		GetProgram: func(name string) agent.ProgramSettings {
			section := target.GetProgram(name)
			return agent.ProgramSettings{
				Model:   section.Model,
				Command: section.Command,
			}
		},
	}
}

// GetSettingsAgent runs a window's tasks on the programs the plugins run, under the settings read at
// every task. A task naming no program goes to the one agent.use names, else to the default of the
// plugins. Each run connects to the tools of the plugins on an endpoint of its own. It is nil where
// AgentProvider is nil.
func (c Config) GetSettingsAgent() agentport.Agent {
	provider := c.GetToolProvider()
	if provider == nil {
		return nil
	}
	return agent.NewSettingsAgent(agent.NewPrograms(provider, c.getAgentSettings))
}

// GetToolProvider returns an agent provider that attaches plugin tools to each run.
func (c Config) GetToolProvider() agentport.AgentProvider {
	if c.PluginHost == nil || c.SafeMode {
		return nil
	}
	return agentbridge.NewServingRouter(c.PluginHost, func(agent agentport.Agent, pluginID string) agentport.Agent {
		return agenttoolsbridge.NewAgent(agent, c.PluginHost, agenttoolsbridge.Full, pluginID)
	})
}
