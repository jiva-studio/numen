package completion

import (
	"context"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/port/agentport"
	"github.com/jiva-studio/numen/libs/core/port/completionport"
)

const programsLimit = 8 * time.Second

// ProgramSettings holds the configured model, command line, and options for an agent program.
type ProgramSettings struct {
	Model   string
	Command []string
	Options map[string]string
}

// AgentSettings holds the active agent configuration read from the settings file.
type AgentSettings struct {
	IsNamed    bool
	Use        string
	Commands   map[string][]string
	GetProgram func(name string) ProgramSettings
}

// Programs finds the agent programs the plugins list and starts the tasks of single answers on them.
type Programs struct {
	provider agentport.AgentProvider
	getAgent func() AgentSettings
}

// NewPrograms creates a Programs adapter over an AgentProvider and a settings callback.
func NewPrograms(provider agentport.AgentProvider, getAgent func() AgentSettings) *Programs {
	return &Programs{provider: provider, getAgent: getAgent}
}

// GetProgram returns the program named. An empty name is the program the settings name under
// agent.use, else the default of the plugins.
func (p *Programs) GetProgram(ctx context.Context, name string) (agentport.AgentProgram, error) {
	target := p.getAgent()
	if name == "" && !target.IsNamed {
		return agentport.AgentProgram{}, fmt.Errorf("%w: the settings name no agent", completionport.ErrNoRoute)
	}
	if name == "" {
		name = target.Use
	}
	listing, stop := context.WithTimeout(ctx, programsLimit)
	defer stop()
	programs, listErr := p.provider.GetPrograms(listing, target.Commands)
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

// StartRun starts the task on the program it names, with the command line, the options and, where
// the task names none, the model the settings give it.
func (p *Programs) StartRun(ctx context.Context, task agentport.Task) (agentport.Run, error) {
	section := p.getAgent().GetProgram(task.Agent)
	if task.Model == "" {
		task.Model = section.Model
	}
	task.Command = section.Command
	task.Options = section.Options
	return p.provider.StartRun(ctx, task)
}

// SettingsAgent starts each task on the program the task or the settings name.
type SettingsAgent struct {
	programs *Programs
}

// NewSettingsAgent creates a SettingsAgent backed by Programs.
func NewSettingsAgent(programs *Programs) *SettingsAgent {
	return &SettingsAgent{programs: programs}
}

// StartRun starts the task on the program it names, else on the one the settings choose.
func (a *SettingsAgent) StartRun(ctx context.Context, task agentport.Task) (agentport.Run, error) {
	if task.Agent == "" {
		program, err := a.programs.GetProgram(ctx, "")
		if err != nil {
			return nil, err
		}
		task.Agent = program.Name
	}
	return a.programs.StartRun(ctx, task)
}

// EndConversation notifies every plugin the conversation was started in that it is over.
func (a *SettingsAgent) EndConversation(ctx context.Context, conversation string) error {
	return a.programs.provider.EndConversation(ctx, conversation)
}
