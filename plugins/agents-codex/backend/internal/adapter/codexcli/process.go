// Package process starts and ends the agent programs the plugin runs, each in a process group of
// its own.
package process

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/jiva-studio/numen/plugins/agents-codex/backend/usecase/run"
)

// waitDelay is how long the pipes of a killed child are waited for.
const waitDelay = 2 * time.Second

// ExitGrace is how long a program whose input has ended is given to exit before it is killed.
const ExitGrace = time.Second

// Supervise puts the command in a process group of its own, and ends the whole group when the
// command's context is canceled.
func Supervise(cmd *exec.Cmd) {
	detach(cmd)
	cmd.Cancel = func() error { return kill(cmd) }
	cmd.WaitDelay = waitDelay
}

// WaitForExit waits for the started command to exit, and calls stop, which cancels the command's
// context, if it is still running after ExitGrace.
//
// A program that exits on the end of its input is reaped by Wait before its group is signaled, so it
// finishes the way it chose to and is not reported as killed.
func WaitForExit(cmd *exec.Cmd, stop func()) error {
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	timer := time.NewTimer(ExitGrace)
	defer timer.Stop()
	select {
	case err := <-exited:
		return err
	case <-timer.C:
		stop()
		return <-exited
	}
}

// Group starts commands in a process group of their own.
type Group struct{}

// Supervise puts cmd in a process group of its own, and ends the whole group when the command's
// context is canceled.
func (Group) Supervise(cmd *exec.Cmd) {
	Supervise(cmd)
}

// Launcher starts a command in its own process group.
type Launcher struct{}

// Start starts spec and returns the running process.
func (Launcher) Start(ctx context.Context, spec run.Command) (run.Process, error) {
	cmd := exec.CommandContext(ctx, spec.Path, spec.Args...)
	cmd.Dir = spec.Dir
	env := spec.Env
	if len(env) == 0 {
		env = os.Environ()
	}
	if spec.Token != "" {
		env = append(env, "NUMEN_TOKEN="+spec.Token)
	}
	cmd.Env = env
	// gomutants:disable-next-line STATEMENT_REMOVE reason="Supervise sets OS process-group attributes for child cleanup."
	Supervise(cmd)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	// gomutants:disable-next-line BRANCH_IF reason="cmd.StdoutPipe on a fresh command before Start fails only on OS descriptor exhaustion."
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &startedProcess{cmd: cmd, stdout: stdout, stderr: &stderr}, nil
}

type startedProcess struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
	stderr *bytes.Buffer
}

func (p *startedProcess) Output() io.ReadCloser { return p.stdout }
func (p *startedProcess) Stderr() string        { return p.stderr.String() }
func (p *startedProcess) Wait() error           { return p.cmd.Wait() }
