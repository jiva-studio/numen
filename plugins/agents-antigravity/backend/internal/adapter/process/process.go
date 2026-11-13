// Package process starts and ends the agent programs the plugin runs, each in a process group of
// its own.
package process

import (
	"context"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/usecase/task"
)

// waitDelay is how long the pipes of a killed child are waited for.
const waitDelay = 2 * time.Second

// Supervise puts the command in a process group of its own, and ends the whole group when the
// command's context is canceled.
func Supervise(cmd *exec.Cmd) {
	detach(cmd)
	cmd.Cancel = func() error { return kill(cmd) }
	cmd.WaitDelay = waitDelay
}

// Supervisor supervises commands with Supervise.
type Supervisor struct{}

// Supervise puts the command in a process group of its own, and ends the whole group when the
// command's context is canceled.
func (Supervisor) Supervise(cmd *exec.Cmd) {
	Supervise(cmd)
}

// Launcher starts a command in its own process group.
type Launcher struct{}

// Start starts spec and returns the running process.
func (Launcher) Start(ctx context.Context, spec task.Command) (task.Process, error) {
	cmd := exec.CommandContext(ctx, spec.Path, spec.Args...)
	cmd.Dir = spec.Dir
	env := spec.Env
	if len(env) == 0 {
		env = hostcall.GetChildEnvironment()
	}
	cmd.Env = env
	// gomutants:disable-next-line STATEMENT_REMOVE reason="Supervise sets OS process-group attributes for child cleanup."
	Supervise(cmd)
	if spec.Stdin != "" {
		cmd.Stdin = strings.NewReader(spec.Stdin)
	}
	stderr := new(tailBuffer)
	cmd.Stderr = stderr
	stdout, err := cmd.StdoutPipe()
	// gomutants:disable-next-line BRANCH_IF reason="cmd.StdoutPipe on a fresh command before Start fails only on OS descriptor exhaustion."
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &startedProcess{cmd: cmd, stdout: stdout, stderr: stderr}, nil
}

type startedProcess struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
	stderr *tailBuffer
}

func (p *startedProcess) Output() io.ReadCloser { return p.stdout }
func (p *startedProcess) Stderr() string        { return p.stderr.String() }
func (p *startedProcess) Wait() error           { return p.cmd.Wait() }
