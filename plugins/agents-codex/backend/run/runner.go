// Package run starts the Codex program for an agent request, reads its output as it runs, and ends the
// run when the program exits, the request ends or the step limit is reached.
package run

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/codexevent"
)

const (
	// gomutants:disable-next-line INTEGER_INCREMENT,INTEGER_DECREMENT reason="Channel buffer capacity is a throughput bound."
	outputBuffer = 64
	// gomutants:disable-next-line INTEGER_INCREMENT,INTEGER_DECREMENT,BITWISE_SHIFT,ARITHMETIC_BASE reason="Scanner maximum token size is a capacity bound."
	maxLine = 4 << 20
)

var errNoOutput = errors.New("the program ended without output")

// Commands builds the command of a run. customPath is the program the installation set, or empty.
type Commands interface {
	// MakeToolsCommand builds the command of a run that may call the tools of the request.
	MakeToolsCommand(ctx context.Context, req *agentrun.Request, customPath string) *exec.Cmd
	// MakeAnswerCommand builds the command of a run that only answers, and the function that
	// releases what the command needs.
	MakeAnswerCommand(ctx context.Context, req *agentrun.Request, customPath string) (*exec.Cmd, func() error, error)
}

// Supervisor prepares a command so that ending its context ends everything it started.
type Supervisor interface {
	Supervise(cmd *exec.Cmd)
}

// Runner starts runs.
type Runner struct {
	commands   Commands
	supervisor Supervisor
}

// NewRunner creates a runner that builds its commands with commands and supervises them with
// supervisor.
func NewRunner(commands Commands, supervisor Supervisor) *Runner {
	return &Runner{commands: commands, supervisor: supervisor}
}

type spec struct {
	Conversation string
	MaxSteps     int
	Root         string
	Token        string
	Build        func(ctx context.Context) (*exec.Cmd, func() error, error)
}

// Start starts the program for req and returns the work whose output carries the steps of the run.
func (r *Runner) Start(ctx context.Context, reg *agentrun.Registry, req *agentrun.Request, customPath string) (*agentrun.Work, error) {
	return r.start(ctx, reg, r.newSpec(req, customPath))
}

func (r *Runner) start(ctx context.Context, reg *agentrun.Registry, s spec) (*agentrun.Work, error) {
	runCtx, stop := context.WithCancel(ctx)
	cmd, cleanup, err := s.Build(runCtx)
	if err != nil {
		stop()
		return nil, err
	}
	if cleanup == nil {
		cleanup = func() error { return nil }
	}
	configureCommand(cmd, s)
	var stderr strings.Builder
	if cmd.Stderr == nil {
		cmd.Stderr = &stderr
	}
	// gomutants:disable-next-line STATEMENT_REMOVE reason="Supervise sets OS process-group attributes for child cleanup."
	r.supervisor.Supervise(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stop()
		return nil, errors.Join(err, cleanup())
	}
	w := agentrun.NewWork(s.Conversation, outputBuffer, agentrun.StopByCancel(stop))
	if !reg.Add(w) {
		stop()
		return nil, errors.Join(errors.New("agent runner is terminating"), stdout.Close(), cleanup())
	}
	if err := cmd.Start(); err != nil {
		w.End()
		reg.Remove(w)
		stop()
		return nil, errors.Join(fmt.Errorf("start %s: %w", cmd.Path, err), stdout.Close(), cleanup())
	}

	go runReader(runCtx, &runProcess{stop: stop, cmd: cmd, stdout: stdout, stderr: &stderr, cleanup: cleanup}, w, reg, s)
	return w, nil
}

func configureCommand(cmd *exec.Cmd, s spec) {
	if s.Root != "" {
		cmd.Dir = s.Root
	}
	if len(cmd.Env) == 0 {
		cmd.Env = os.Environ()
	}
	if s.Token != "" {
		cmd.Env = append(cmd.Env, "NUMEN_TOKEN="+s.Token)
	}
}

func (r *Runner) newSpec(req *agentrun.Request, customPath string) spec {
	s := spec{
		Conversation: req.Conversation,
		MaxSteps:     int(req.MaxSteps),
		Root:         "",
		Token:        "",
		Build:        nil,
	}
	if agentrun.HasTools(req) {
		s.Root = req.Root
		s.Token = req.ToolsToken
		s.Build = func(ctx context.Context) (*exec.Cmd, func() error, error) {
			return r.commands.MakeToolsCommand(ctx, req, customPath), nil, nil
		}
		return s
	}
	s.Build = func(ctx context.Context) (*exec.Cmd, func() error, error) {
		return r.commands.MakeAnswerCommand(ctx, req, customPath)
	}
	return s
}

// runProcess holds the started command of a run and what its reader needs to stop and release it.
type runProcess struct {
	stop    func()
	cmd     *exec.Cmd
	stdout  io.ReadCloser
	stderr  *strings.Builder
	cleanup func() error
}

func runReader(runCtx context.Context, p *runProcess, w *agentrun.Work, reg *agentrun.Registry, s spec) {
	stop, cleanup, stdout, cmd, stderr := p.stop, p.cleanup, p.stdout, p.cmd, p.stderr
	r := &reader{
		work: w, registry: reg, conversation: s.Conversation, limit: s.MaxSteps,
		completed: 0, sentCount: 0,
	}
	defer func() {
		stop()
		if err := cleanup(); err != nil {
			r.reportStop(fmt.Sprintf("release the resources of the run: %v", err))
		}
		w.End()
		reg.Remove(w)
		close(w.Output)
	}()
	scanner := newOutputScanner(stdout)
	isLimited := scanOutput(scanner, func(event map[string]any) { codexevent.Handle(r, event) }, r, w, stop)
	scanErr := scanner.Err()
	isCancelled := runCtx.Err() != nil
	// gomutants:disable-next-line BRANCH_IF reason="Stopping the context on scanner error cancels the child before defer calls stop()."
	if scanErr != nil {
		// gomutants:disable-next-line STATEMENT_REMOVE reason="Stopping the context on scanner error cancels the child before defer calls stop()."
		stop()
	}
	waitErr := errors.Join(stdout.Close(), cmd.Wait())
	if isLimited || isCancelled {
		return
	}
	reportExit(r, scanErr, waitErr, stderr.String())
}

// Ask runs req and sends the steps of the run. A run that cannot start ends with a stopped step.
func (r *Runner) Ask(ctx context.Context, reg *agentrun.Registry, req *agentrun.Request, customPath string, send func(*agentrun.Response) error) error {
	if err := send(&agentrun.Response{Step: &agentrun.ModelStartedStep{}}); err != nil {
		return err
	}
	work, err := r.Start(ctx, reg, req, customPath)
	if err != nil {
		return send(&agentrun.Response{
			Step: &agentrun.StoppedStep{Stopped: err.Error()},
		})
	}
	return agentrun.ForwardResponses(ctx, work, send)
}
