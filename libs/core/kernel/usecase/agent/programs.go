package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

const programsLimit = 8 * time.Second

// ProgramSettings holds the configured model, command line, and options for an agent program.
type ProgramSettings struct {
	Model   string
	Command []string
}

// AgentSection is the agent part of the settings file.
type AgentSection interface {
	IsNamed() bool
	GetUse() string
	GetCommands() map[string][]string
	GetProgramSettings(name string) (model string, command []string)
}

// NewAgentSettings reads the active agent configuration from the agent part of the settings.
func NewAgentSettings(section AgentSection) AgentSettings {
	return AgentSettings{
		IsNamed:  section.IsNamed(),
		Use:      section.GetUse(),
		Commands: section.GetCommands(),
		GetProgram: func(name string) ProgramSettings {
			model, command := section.GetProgramSettings(name)
			return ProgramSettings{Model: model, Command: command}
		},
	}
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
func (p *Programs) GetProgram(ctx context.Context, name string) (agent.Program, error) {
	target := p.getAgent()
	if name == "" && !target.IsNamed {
		return agent.Program{}, fmt.Errorf("%w: the settings name no agent", domain.ErrUnavailable)
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
		return agent.Program{}, fmt.Errorf("%w: %w", domain.ErrUnavailable, listErr)
	case !isFound:
		return agent.Program{}, fmt.Errorf("%w: no plugin lists the program %q", domain.ErrUnavailable, name)
	case !found.IsInstalled:
		return agent.Program{}, fmt.Errorf("%w: the program %q is not installed", domain.ErrUnavailable, found.Name)
	}
	return found, nil
}

func findProgram(programs []agent.Program, name string) (agent.Program, bool) {
	if name == "" {
		return agent.GetDefaultProgram(programs)
	}
	for _, one := range programs {
		if one.Name == name {
			return one, true
		}
	}
	return agent.Program{}, false
}

// StartRun starts the task on the program it names, with the command line, the options and, where
// the task names none, the model the settings give it.
func (p *Programs) StartRun(ctx context.Context, task agent.Task) (agentport.Run, error) {
	section := p.getAgent().GetProgram(task.Agent)
	if task.Model == "" {
		task.Model = section.Model
	}
	task.Command = section.Command
	return p.provider.StartRun(ctx, task)
}

// SettingsAgent starts each task on the program the task or the settings name.
type SettingsAgent struct {
	programs *Programs
}

func NewSettingsAgent(programs *Programs) *SettingsAgent {
	return &SettingsAgent{programs: programs}
}

// StartRun starts the task on the program it names, else on the one the settings choose.
func (a *SettingsAgent) StartRun(ctx context.Context, task agent.Task) (agentport.Run, error) {
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
