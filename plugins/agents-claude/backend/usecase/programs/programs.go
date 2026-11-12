// Package programs reports whether Claude Code can be reached and which programs the plugin runs.
package programs

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/ports"
)

// Name is the name tasks use for the program, and the settings section key.
const Name = "claude"

// title is the display name of the program.
const title = "Claude Code"

// Models lists the models of the program on a command line.
type Models interface {
	List(ctx context.Context, command string, extra, env []string) []*reasoning.Model
}

// Dependencies are what Programs needs.
type Dependencies struct {
	Settings    *settings.Settings
	Finder      ports.ProgramFinder
	Models      Models
	Environment ports.Environment
}

// Programs answers what the host asks about the programs of the plugin.
type Programs struct {
	deps Dependencies
}

// New creates Programs, and refuses dependencies that are missing.
func New(deps Dependencies) (*Programs, error) {
	if deps.Settings == nil || deps.Finder == nil || deps.Models == nil || deps.Environment == nil {
		return nil, errors.New("programs: every dependency is required")
	}
	return &Programs{deps: deps}, nil
}

// ResolveCommand returns the program and its leading arguments: the command the task names, else
// the program found on the machine, which is none when it is not installed.
func (p *Programs) ResolveCommand(command []string) (string, []string) {
	if len(command) > 0 {
		return command[0], command[1:]
	}
	return p.deps.Finder.Find(p.deps.Settings.Get().ExecutablePath), nil
}

// GetState reports the program as unreachable when it cannot be found.
func (p *Programs) GetState() *agentrun.State {
	var unreachable string
	if p.deps.Finder.Find(p.deps.Settings.Get().ExecutablePath) == "" {
		unreachable = "Claude Code CLI not found"
	}
	return &agentrun.State{Unreachable: unreachable}
}

// List lists Claude Code, whether it is installed under the command line the installation set or
// where it is found, and the models it accepts.
func (p *Programs) List(ctx context.Context, commands []agentrun.ProgramCommand) []*agentrun.Program {
	var configured []string
	for _, one := range commands {
		if one.Program == Name {
			configured = one.Command
		}
	}
	command, extra := p.ResolveCommand(configured)
	return []*agentrun.Program{{
		Name:        Name,
		Title:       title,
		IsInstalled: command != "",
		IsDefault:   true,
		LimitsTools: true,
		Models:      p.deps.Models.List(ctx, command, extra, p.deps.Environment.GetVariables()),
	}}
}
