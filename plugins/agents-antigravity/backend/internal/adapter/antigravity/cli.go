package antigravity

import (
	"context"
	"fmt"
	"os/exec"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/model"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/runs"
)

// Supervisor ends the whole process group of a command when its context is canceled.
type Supervisor interface {
	Supervise(cmd *exec.Cmd)
}

// CLI is the Antigravity program on this machine.
type CLI struct {
	supervisor Supervisor
}

// New returns the program, whose child processes the supervisor ends.
func New(supervisor Supervisor) *CLI {
	return &CLI{supervisor: supervisor}
}

// IsInstalled reports whether the program is runnable, at path when given, or where it is found.
func (c *CLI) IsInstalled(path string) bool {
	return isInstalled(path)
}

// List runs the "models" command of the program and returns the models it prints.
func (c *CLI) List(ctx context.Context, command []string) ([]model.Model, error) {
	name, args := resolveCommand(command)
	cmd := exec.CommandContext(ctx, name, slices.Concat(args, []string{"models"})...)
	// gomutants:disable-next-line STATEMENT_REMOVE reason="Supervise sets OS process-group attributes for child cleanup."
	c.supervisor.Supervise(cmd)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list the models of %s: %w", name, err)
	}
	return model.Parse(out), nil
}

// Build returns the command that runs req.
func (c *CLI) Build(
	ctx context.Context, req *agentrun.Request, reg *agentrun.Registry, executablePath string,
) (runs.Command, error) {
	return makeAntigravityCommand(ctx, req, reg, executablePath)
}
