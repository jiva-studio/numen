// Package programs reports whether OpenAI Codex is installed and lists it with its models.
package programs

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/agentmodel"
)

const programName = "codex"

// Installation reports if the program is installed, at path when given.
type Installation interface {
	IsInstalled(path string) bool
}

// Models lists the models of the program that command starts.
type Models interface {
	Get(ctx context.Context, command []string) []agentmodel.Model
}

// Programs describes Codex.
type Programs struct {
	installation Installation
	models       Models
}

func New(installation Installation, models Models) *Programs {
	return &Programs{installation: installation, models: models}
}

// GetState reports if the program is reachable, with the reason when it is not.
func (p *Programs) GetState(executablePath string) *agentrun.State {
	if !p.installation.IsInstalled(executablePath) {
		return &agentrun.State{Unreachable: "OpenAI Codex was not found on this machine"}
	}
	return &agentrun.State{Unreachable: ""}
}

// GetPrograms lists OpenAI Codex, whether it is installed, and the models it lists.
func (p *Programs) GetPrograms(ctx context.Context, executablePath string, commands []agentrun.ProgramCommand) []*agentrun.Program {
	command := findCommand(commands, programName)
	if len(command) == 0 && executablePath != "" {
		command = []string{executablePath}
	}

	codex := &agentrun.Program{
		Name: programName, Title: "OpenAI Codex", IsInstalled: p.installation.IsInstalled(executablePath) || len(command) > 0,
		IsDefault: false, Models: nil, LimitsTools: false,
	}
	if codex.IsInstalled {
		for _, one := range p.models.Get(ctx, command) {
			codex.Models = append(codex.Models, &reasoning.Model{
				Name: one.Name, Title: one.Title, IsDefault: one.IsDefault,
				ReasoningLevels: one.Reasoning, DefaultReasoning: one.DefaultReasoning,
			})
		}
	}
	return []*agentrun.Program{codex}
}

func findCommand(commands []agentrun.ProgramCommand, name string) []string {
	for _, one := range commands {
		if one.Program == name {
			return one.Command
		}
	}
	return nil
}
