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
	outputBuffer = 64
	maxLine      = 4 << 20
)

var errNoOutput = errors.New("the program ended without output")

// disabledFeatures are the tools of Codex's own that a run naming no tool is started without.
var disabledFeatures = []string{
	"shell_tool", "unified_exec", "view_image", "multi_agent", "apps", "browser_use",
	"computer_use", "plugins", "image_generation", "hooks",
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

	go runReader(runCtx, stop, cmd, stdout, &stderr, w, reg, s, cleanup)
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

func runReader(
	runCtx context.Context, stop func(), cmd *exec.Cmd, stdout io.ReadCloser,
	stderr *strings.Builder, w *agentrun.Work, reg *agentrun.Registry, s spec, cleanup func() error,
) {
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
	if scanErr != nil {
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

// newSpec builds the Codex CLI invocation for the request.
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
		mcpHeader := fmt.Sprintf(`mcp_servers.numen.http_headers={"Authorization"="Bearer %s"}`, req.ToolsToken)
		args = append(args, "-c", mcpURL, "-c", mcpHeader)
	}
	args = appendModelArguments(args, req)
	args = append(args, req.Prompt)
	return exec.CommandContext(runCtx, binName, args...)
}

// makeAnswerCommand is the run of the Codex CLI that gives one answer: no MCP server, none of the
// tools Codex has of its own that can be turned off, nothing the user's configuration adds, a
// sandbox that writes nothing and an empty folder to run in. The folder goes when the run does.
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
	for _, feature := range disabledFeatures {
		args = append(args, "--disable", feature)
	}
	args = appendModelArguments(args, req)
	args = append(args, "--", agentrun.WritePrompt(req.Instructions, req.Prompt))
	cmd := exec.CommandContext(runCtx, binName, args...)
	cmd.Dir = dir
	return cmd, cleanDir, nil
}

// appendModelArguments adds the model the request names and the reasoning level it requests.
func appendModelArguments(args []string, req *agentrun.Request) []string {
	if model := req.Model; model != "" && model != "default" {
		args = append(args, "--model", model)
	}
	if level := req.Reasoning; reasoning.IsKnown(level) {
		args = append(args, "-c", fmt.Sprintf("model_reasoning_effort=%q", level))
	}
	return args
}
