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
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/internal/adapter/process"
)

const (
	// gomutants:disable-next-line INTEGER_INCREMENT,INTEGER_DECREMENT reason="Channel buffer capacity is a throughput bound."
	outputBuffer = 64
	// gomutants:disable-next-line INTEGER_INCREMENT,INTEGER_DECREMENT,BITWISE_SHIFT,ARITHMETIC_BASE reason="Scanner maximum token size is a capacity bound."
	maxLine = 4 << 20
)

var errNoOutput = errors.New("the program ended without output")

func getDisabledFeatures() []string {
	return []string{
		"shell_tool", "unified_exec", "view_image", "multi_agent", "apps", "browser_use",
		"computer_use", "plugins", "image_generation", "hooks",
	}
}

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
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(nil, maxLine)
	isLimited := scanOutput(scanner, s.NewHandler(r), r, w, stop)
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

func newSpec(req *agentrun.Request, customPath ...string) spec {
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
		s.Build = func(ctx context.Context) (*exec.Cmd, func() error, error) {
			return makeCodexCommand(ctx, req, execPath), nil, nil
		}
		return s
	}
	s.Build = func(ctx context.Context) (*exec.Cmd, func() error, error) {
		return makeAnswerCommand(ctx, req, execPath)
	}
	return s
}

func resolveCommand(command []string, customPath ...string) (string, []string) {
	if len(command) > 0 {
		return command[0], command[1:]
	}
	return FindCodex(customPath...), nil
}

func makeCodexCommand(runCtx context.Context, req *agentrun.Request, customPath ...string) *exec.Cmd {
	binName, baseArgs := resolveCommand(req.Command, customPath...)
	args := append([]string{}, baseArgs...)
	args = append(args,
		"exec",
		"--json",
		"--dangerously-bypass-approvals-and-sandbox",
	)
	if req.ToolsURL != "" {
		mcpURL := fmt.Sprintf(`mcp_servers.numen.url=%q`, req.ToolsURL)
		args = append(args, "-c", mcpURL, "-c", `mcp_servers.numen.bearer_token_env_var="NUMEN_TOKEN"`)
	}
	args = appendModelArguments(args, req)
	args = append(args, req.Prompt)
	return exec.CommandContext(runCtx, binName, args...)
}

func makeAnswerCommand(runCtx context.Context, req *agentrun.Request, customPath ...string) (*exec.Cmd, func() error, error) {
	dir, cleanDir, err := agentrun.CreateTempDir("numen-codex-*")
	if err != nil {
		return nil, nil, fmt.Errorf("create the folder to run in: %w", err)
	}
	binName, baseArgs := resolveCommand(req.Command, customPath...)
	args := append([]string{}, baseArgs...)
	args = append(args,
		"exec",
		"--json",
		"--ephemeral",
		"--skip-git-repo-check",
		"--ignore-user-config",
		"--sandbox", "read-only",
		"-c", `web_search="disabled"`,
	)
	for _, feature := range getDisabledFeatures() {
		args = append(args, "--disable", feature)
	}
	args = appendModelArguments(args, req)
	args = append(args, "--", agentrun.WritePrompt(req.Instructions, req.Prompt))
	cmd := exec.CommandContext(runCtx, binName, args...)
	cmd.Dir = dir
	return cmd, cleanDir, nil
}

func appendModelArguments(args []string, req *agentrun.Request) []string {
	if model := req.Model; model != "" && model != "default" {
		args = append(args, "--model", model)
	}
	if level := req.Reasoning; reasoning.IsKnown(level) {
		args = append(args, "-c", fmt.Sprintf("model_reasoning_effort=%q", level))
	}
	return args
}
