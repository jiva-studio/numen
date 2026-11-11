// Package runs starts the Antigravity program for an agent request and streams what it prints as the
// steps of the run.
package runs

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/events"
)

const (
	// gomutants:disable-next-line INTEGER_INCREMENT,INTEGER_DECREMENT reason="Channel buffer capacity is a throughput bound."
	outputBuffer = 64
	// gomutants:disable-next-line INTEGER_INCREMENT,INTEGER_DECREMENT,BITWISE_SHIFT,ARITHMETIC_BASE reason="Scanner maximum token size is a capacity bound."
	maxLine = 4 << 20
)

var errNoOutput = errors.New("the program ended without output")

// Command is the program and environment of a run.
type Command struct {
	Path    string
	Args    []string
	Dir     string
	Env     []string
	Token   string
	Release func() error
}

// Process is a started program of a run.
type Process interface {
	Output() io.ReadCloser
	Stderr() string
	Wait() error
}

// Commands builds the command that runs a request.
type Commands interface {
	Build(
		ctx context.Context, req *agentrun.Request, reg *agentrun.Registry, executablePath string,
	) (Command, error)
}

// Launcher starts a command and ends its process group when its context is canceled.
type Launcher interface {
	Start(ctx context.Context, cmd Command) (Process, error)
}

// ModelResolver replaces the model of a request with the listed model that the request names.
type ModelResolver interface {
	ResolveModel(ctx context.Context, req *agentrun.Request)
}

// Runner runs the requests of one agent service.
type Runner struct {
	registry *agentrun.Registry
	commands Commands
	launcher Launcher
	resolver ModelResolver
}

// New returns a runner that records its runs and their sessions in registry.
func New(registry *agentrun.Registry, commands Commands, launcher Launcher, resolver ModelResolver) *Runner {
	return &Runner{registry: registry, commands: commands, launcher: launcher, resolver: resolver}
}

type spec struct {
	Conversation string
	MaxSteps     int
	Root         string
	Token        string
	Build        func(ctx context.Context) (Command, error)
	NewHandler   func(sink events.Sink) func(event map[string]any)
}

// Ask answers a request: it reports that the model started, runs the program, and forwards the steps
// to send. A run that cannot start ends with a stopped step that gives the reason.
func (r *Runner) Ask(
	ctx context.Context, req *agentrun.Request, executablePath string, send func(*agentrun.Response) error,
) error {
	if err := send(&agentrun.Response{Step: &agentrun.ModelStartedStep{}}); err != nil {
		return err
	}

	r.resolver.ResolveModel(ctx, req)
	work, err := r.Start(ctx, req, executablePath)
	if err != nil {
		return send(&agentrun.Response{
			Step: &agentrun.StoppedStep{Stopped: err.Error()},
		})
	}
	return agentrun.ForwardResponses(ctx, work, send)
}

// Start runs the program for req and returns the work whose output carries the steps.
func (r *Runner) Start(ctx context.Context, req *agentrun.Request, executablePath string) (*agentrun.Work, error) {
	return r.start(ctx, r.newSpec(req, executablePath))
}

func (r *Runner) newSpec(req *agentrun.Request, executablePath string) spec {
	s := spec{
		Conversation: req.Conversation,
		MaxSteps:     int(req.MaxSteps),
		NewHandler:   events.NewHandler,
		Root:         "",
		Token:        "",
		Build: func(ctx context.Context) (Command, error) {
			return r.commands.Build(ctx, req, r.registry, executablePath)
		},
	}
	if agentrun.HasTools(req) {
		s.Root = req.Root
		s.Token = req.ToolsToken
	}
	return s
}

func (r *Runner) start(ctx context.Context, s spec) (*agentrun.Work, error) {
	runCtx, stop := context.WithCancel(ctx)
	cmd, err := s.Build(runCtx)
	if err != nil {
		stop()
		return nil, err
	}
	cleanup := cmd.Release
	if cleanup == nil {
		cleanup = func() error { return nil }
	}
	configureCommand(&cmd, s)
	w := agentrun.NewWork(s.Conversation, outputBuffer, agentrun.StopByCancel(stop))
	if !r.registry.Add(w) {
		stop()
		return nil, errors.Join(errors.New("agent runner is terminating"), cleanup())
	}
	proc, err := r.launcher.Start(runCtx, cmd)
	if err != nil {
		w.End()
		r.registry.Remove(w)
		stop()
		return nil, errors.Join(fmt.Errorf("start %s: %w", cmd.Path, err), cleanup())
	}

	go runReader(runCtx, &runProcess{stop: stop, proc: proc, cleanup: cleanup}, w, r.registry, s)
	return w, nil
}

func configureCommand(cmd *Command, s spec) {
	if s.Root != "" {
		cmd.Dir = s.Root
	}
	if s.Token != "" {
		cmd.Token = s.Token
	}
}

// runProcess holds the started command of a run and what its reader needs to stop and release it.
type runProcess struct {
	stop    func()
	proc    Process
	cleanup func() error
}

func runReader(runCtx context.Context, p *runProcess, w *agentrun.Work, reg *agentrun.Registry, s spec) {
	stop, cleanup, proc := p.stop, p.cleanup, p.proc
	stdout := proc.Output()
	r := &reader{work: w, registry: reg, conversation: s.Conversation, limit: s.MaxSteps,
		completed: 0, sentCount: 0, isFinished: false}
	defer func() {
		stop()
		if err := cleanup(); err != nil {
			r.reportStop(fmt.Sprintf("release the resources of the run: %v", err))
		}
		w.End()
		reg.Remove(w)
		close(w.Output)
	}()
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(nil, maxLine)
	isEnded := scanOutput(scanner, s.NewHandler(r), r, w, stop)
	scanErr := scanner.Err()
	isCanceled := runCtx.Err() != nil
	// gomutants:disable-next-line BRANCH_IF reason="Stopping the context on scanner error cancels the child before defer calls stop()."
	if scanErr != nil {
		// gomutants:disable-next-line STATEMENT_REMOVE reason="Stopping the context on scanner error cancels the child before defer calls stop()."
		stop()
	}
	waitErr := errors.Join(stdout.Close(), proc.Wait())
	if isEnded || isCanceled {
		return
	}
	reportExit(r, scanErr, waitErr, proc.Stderr())
}

func scanOutput(
	scanner *bufio.Scanner, handle func(map[string]any), r *reader, w *agentrun.Work, stop func(),
) bool {
	for scanner.Scan() {
		line := scanner.Bytes()
		if event, ok := events.DecodeLine(line); ok {
			handle(event)
		} else if text := strings.TrimSpace(string(line)); text != "" {
			r.Send(&agentrun.Response{Step: &agentrun.TextStep{Text: text + "\n"}})
		}
		if r.isFinished {
			// gomutants:disable-next-line STATEMENT_REMOVE reason="Calling stop() when the CLI reports result completion cancels the child before runReader's defer calls stop()."
			stop()
			return true
		}
		if r.isOverLimit() {
			r.reportLimit()
			// gomutants:disable-next-line STATEMENT_REMOVE reason="Calling stop() when the step limit is reached cancels the child before runReader's defer calls stop()."
			stop()
			return true
		}
		select {
		case <-w.Stopped():
			return false
		default:
		}
	}
	return false
}

func reportExit(r *reader, scanErr, waitErr error, stderr string) {
	switch {
	case scanErr != nil:
		r.reportStop(scanErr.Error())
	case waitErr != nil:
		r.reportStop(agentrun.FailureMessage(waitErr, stderr))
	case r.sentCount == 0 && strings.TrimSpace(stderr) != "":
		r.reportStop(agentrun.FailureMessage(errNoOutput, stderr))
	}
}
