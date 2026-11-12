package antigravity

import (
	"context"
	"fmt"
	"os/exec"
	"slices"

	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/model"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/usecase/task"
)

// Supervisor ends the whole process group of a command when its context is canceled.
type Supervisor interface {
	Supervise(cmd *exec.Cmd)
}

// CLI is the Antigravity program on this machine.
type CLI struct {
	supervisor Supervisor
}

var _ task.Program = (*CLI)(nil)

// New returns the program, whose child processes the supervisor ends.
func New(supervisor Supervisor) *CLI {
	return &CLI{supervisor: supervisor}
}

// IsInstalled reports whether the program is runnable, at path when given, or where it is found.
func (c *CLI) IsInstalled(path string) bool {
	return isInstalled(path)
}

// Find returns the program at path when it is runnable, else where it is found, else the bare name.
func (c *CLI) Find(path string) string {
	return findProgram(path)
}

// MakeTaskCommand returns the command of a run that may call the tools served at the address of run
// without asking, in a home of its own whose MCP configuration names them. A run with no address only
// answers.
func (c *CLI) MakeTaskCommand(path string, run task.Run) (task.Command, error) {
	return makeTaskCommand(path, run)
}

// MakeAnswerCommand returns the command of a run that only answers.
func (c *CLI) MakeAnswerCommand(path string, run task.Run) (task.Command, error) {
	return makeAnswerCommand(path, run)
}

// List runs the "models" command of the program and returns the models it prints.
func (c *CLI) List(ctx context.Context, command []string) ([]model.Model, error) {
	if len(command) == 0 {
		command = []string{findProgram()}
	}
	name, args := command[0], command[1:]
	cmd := exec.CommandContext(ctx, name, slices.Concat(args, []string{"models"})...)
	c.supervisor.Supervise(cmd)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list the models of %s: %w", name, err)
	}
	return model.Parse(out), nil
}
