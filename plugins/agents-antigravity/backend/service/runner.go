package service

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/internal/adapter/process"
)

const (
	// gomutants:disable-next-line INTEGER_INCREMENT,INTEGER_DECREMENT reason="Channel buffer capacity is a throughput bound."
	outputBuffer = 64
	// gomutants:disable-next-line INTEGER_INCREMENT,INTEGER_DECREMENT,BITWISE_SHIFT,ARITHMETIC_BASE reason="Scanner maximum token size is a capacity bound."
	maxLine = 4 << 20
)

var errNoOutput = errors.New("the program ended without output")

type spec struct {
	Conversation string
	MaxSteps     int
	Root         string
	Token        string
	Build        func(ctx context.Context) (*exec.Cmd, func() error, error)
	NewHandler   func(r *reader) func(event map[string]any)
}

type reader struct {
	work         *agentrun.Work
	registry     *agentrun.Registry
	conversation string
	limit        int
	completed    int
	sentCount    int
	isFinished   bool
}

func (r *reader) Send(resp *agentrun.Response) {
	select {
	case r.work.Output <- resp:
	case <-r.work.Stopped():
		return
	}
	r.sentCount++
	if resp.IsToolFinished() {
		r.completed++
	}
}

func (r *reader) CountStep() { r.completed++ }

func (r *reader) SetSession(session string) {
	if r.conversation != "" && session != "" {
		r.registry.SetSession(r.conversation, session)
	}
}

func (r *reader) isOverLimit() bool { return r.limit > 0 && r.completed >= r.limit }

func (r *reader) reportLimit() {
	r.reportStop(fmt.Sprintf("reached maximum limit of %d steps", r.limit))
}

func (r *reader) reportStop(reason string) {
	r.Send(&agentrun.Response{Step: &agentrun.StoppedStep{Stopped: reason}})
}

func start(ctx context.Context, reg *agentrun.Registry, s spec) (*agentrun.Work, error) {
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
	// gomutants:disable-next-line STATEMENT_REMOVE reason="process.Supervise sets OS process-group attributes for child cleanup."
	process.Supervise(cmd)

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
	isCancelled := runCtx.Err() != nil
	// gomutants:disable-next-line BRANCH_IF reason="Stopping the context on scanner error cancels the child before defer calls stop()."
	if scanErr != nil {
		// gomutants:disable-next-line STATEMENT_REMOVE reason="Stopping the context on scanner error cancels the child before defer calls stop()."
		stop()
	}
	waitErr := errors.Join(stdout.Close(), cmd.Wait())
	if isEnded || isCancelled {
		return
	}
	reportExit(r, scanErr, waitErr, stderr.String())
}

func scanOutput(
	scanner *bufio.Scanner, handle func(map[string]any), r *reader, w *agentrun.Work, stop func(),
) bool {
	for scanner.Scan() {
		line := scanner.Bytes()
		var event map[string]any
		if err := json.Unmarshal(line, &event); err == nil {
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

func newSpec(req *agentrun.Request, reg *agentrun.Registry, customPath ...string) spec {
	execPath := ""
	if len(customPath) > 0 {
		execPath = customPath[0]
	}
	s := spec{
		Conversation: req.Conversation,
		MaxSteps:     int(req.MaxSteps),
		NewHandler:   newEventHandler,
		Root:         "",
		Token:        "",
		Build:        nil,
	}
	if agentrun.HasTools(req) {
		s.Root = req.Root
		s.Token = req.ToolsToken
	}
	s.Build = func(ctx context.Context) (*exec.Cmd, func() error, error) {
		return makeAntigravityCommand(ctx, req, reg, execPath)
	}
	return s
}

func resolveCommand(command []string, customPath ...string) (string, []string) {
	if len(command) > 0 {
		return command[0], command[1:]
	}
	return FindAntigravity(customPath...), nil
}

func makeAntigravityCommand(
	runCtx context.Context, req *agentrun.Request, reg *agentrun.Registry, customPath ...string,
) (*exec.Cmd, func() error, error) {
	if !agentrun.HasTools(req) {
		return makeAnswerCommand(runCtx, req, reg, customPath...)
	}
	binName, baseArgs := resolveCommand(req.Command, customPath...)
	args := append([]string{}, baseArgs...)
	args = append(args,
		"-p", req.Prompt,
		"--output-format", "stream-json",
		"--dangerously-skip-permissions",
	)
	if model := req.Model; model != "" && model != defaultModelName {
		args = append(args, "--model", model)
	}
	if sessID := reg.GetSession(req.Conversation); sessID != "" {
		args = append(args, "--conversation", sessID)
	}

	cmd := exec.CommandContext(runCtx, binName, args...)
	if req.ToolsURL != "" {
		tempHome, clean, err := setupAntigravityHome(&Endpoint{URL: req.ToolsURL, Token: req.ToolsToken})
		if err != nil {
			return nil, nil, fmt.Errorf("set up the Antigravity home: %w", err)
		}
		cmd.Env = append(os.Environ(), "HOME="+tempHome)
		return cmd, clean, nil
	}
	return cmd, nil, nil
}

func makeAnswerCommand(
	runCtx context.Context, req *agentrun.Request, reg *agentrun.Registry, customPath ...string,
) (*exec.Cmd, func() error, error) {
	tempHome, cleanHome, err := setupAntigravityHome(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("set up the Antigravity home: %w", err)
	}
	dir, cleanDir, err := agentrun.CreateTempDir("numen-antigravity-run-*")
	if err != nil {
		return nil, nil, errors.Join(fmt.Errorf("create the folder to run in: %w", err), cleanHome())
	}
	binName, baseArgs := resolveCommand(req.Command, customPath...)
	args := append([]string{}, baseArgs...)
	args = append(args,
		"-p", agentrun.WritePrompt(req.Instructions, req.Prompt),
		"--output-format", "stream-json",
	)
	if model := req.Model; model != "" && model != defaultModelName {
		args = append(args, "--model", model)
	}
	if sessID := reg.GetSession(req.Conversation); sessID != "" {
		args = append(args, "--conversation", sessID)
	}
	cmd := exec.CommandContext(runCtx, binName, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "HOME="+tempHome)
	return cmd, func() error {
		return errors.Join(cleanHome(), cleanDir())
	}, nil
}
