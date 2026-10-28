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
)

const (
	outputBuffer = 64
	maxLine      = 4 << 20
)

var errNoOutput = errors.New("the program ended without output")

type spec struct {
	Conversation string
	MaxSteps     int
	Root         string
	Token        string
	Build        func(ctx context.Context) (*exec.Cmd, func(), error)
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

func (r *reader) Send(resp *agentrun.AskAgentResponse) {
	select {
	case r.work.Output <- resp:
	case <-r.work.Stopped():
		return
	}
	r.sentCount++
	if resp.GetToolFinished() != nil {
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
	r.Send(&agentrun.AskAgentResponse{Step: &agentrun.AskAgentResponse_Stopped{Stopped: reason}})
}

func start(ctx context.Context, reg *agentrun.Registry, s spec) (*agentrun.Work, error) {
	runCtx, stop := context.WithCancel(ctx)
	cmd, cleanup, err := s.Build(runCtx)
	if err != nil {
		stop()
		return nil, err
	}
	if cleanup == nil {
		cleanup = func() {}
	}
	configureCommand(cmd, s)
	var stderr strings.Builder
	if cmd.Stderr == nil {
		cmd.Stderr = &stderr
	}
	supervise(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cleanup()
		stop()
		return nil, err
	}
	w := agentrun.NewWork(s.Conversation, outputBuffer, stop)
	if !reg.Add(w) {
		_ = stdout.Close()
		cleanup()
		stop()
		return nil, errors.New("agent runner is terminating")
	}
	if err := cmd.Start(); err != nil {
		w.End()
		reg.Remove(w)
		_ = stdout.Close()
		cleanup()
		stop()
		return nil, fmt.Errorf("start %s: %w", cmd.Path, err)
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
	stderr *strings.Builder, w *agentrun.Work, reg *agentrun.Registry, s spec, cleanup func(),
) {
	defer func() {
		stop()
		cleanup()
		w.End()
		reg.Remove(w)
		close(w.Output)
	}()
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(nil, maxLine)
	r := &reader{work: w, registry: reg, conversation: s.Conversation, limit: s.MaxSteps}
	isEnded := scanOutput(scanner, s.NewHandler(r), r, w, stop)
	scanErr := scanner.Err()
	isCancelled := runCtx.Err() != nil
	if scanErr != nil {
		stop()
	}
	_ = stdout.Close()
	waitErr := cmd.Wait()
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
			r.Send(&agentrun.AskAgentResponse{Step: &agentrun.AskAgentResponse_Text{Text: text + "\n"}})
		}
		if r.isFinished {
			stop()
			return true
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

// newSpec is the run of the Antigravity CLI the request asks for.
func newSpec(req *agentrun.AskAgentRequest, reg *agentrun.Registry, customPath ...string) spec {
	execPath := ""
	if len(customPath) > 0 {
		execPath = customPath[0]
	}
	s := spec{
		Conversation: req.GetConversation(),
		MaxSteps:     int(req.GetMaxSteps()),
		NewHandler:   newEventHandler,
	}
	if agentrun.HasTools(req) {
		s.Root = req.GetRoot()
		s.Token = req.GetToolsToken()
	}
	s.Build = func(ctx context.Context) (*exec.Cmd, func(), error) {
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
	runCtx context.Context, req *agentrun.AskAgentRequest, reg *agentrun.Registry, customPath ...string,
) (*exec.Cmd, func(), error) {
	if !agentrun.HasTools(req) {
		return makeAnswerCommand(runCtx, req, reg, customPath...)
	}
	binName, baseArgs := resolveCommand(req.GetCommand(), customPath...)
	args := append([]string{}, baseArgs...)
	args = append(args,
		"-p", req.GetPrompt(),
		"--output-format", "stream-json",
		"--dangerously-skip-permissions",
	)
	if model := req.GetModel(); model != "" && model != "default" {
		args = append(args, "--model", model)
	}
	if sessID := reg.GetSession(req.GetConversation()); sessID != "" {
		args = append(args, "--conversation", sessID)
	}

	cmd := exec.CommandContext(runCtx, binName, args...)
	if req.GetToolsUrl() != "" {
		tempHome, clean, err := setupAntigravityHome(&Endpoint{URL: req.GetToolsUrl(), Token: req.GetToolsToken()})
		if err != nil {
			return nil, nil, fmt.Errorf("set up the Antigravity home: %w", err)
		}
		cmd.Env = append(os.Environ(), "HOME="+tempHome)
		return cmd, clean, nil
	}
	return cmd, nil, nil
}

// makeAnswerCommand is the run of the Antigravity CLI that gives one answer: no MCP server, no
// permission granted beforehand, and an empty folder to run in. The home and the folder go when the
// run does.
func makeAnswerCommand(
	runCtx context.Context, req *agentrun.AskAgentRequest, reg *agentrun.Registry, customPath ...string,
) (*exec.Cmd, func(), error) {
	tempHome, cleanHome, err := setupAntigravityHome(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("set up the Antigravity home: %w", err)
	}
	dir, cleanDir, err := agentrun.CreateTempDir("numen-antigravity-run-*")
	if err != nil {
		cleanHome()
		return nil, nil, fmt.Errorf("create the folder to run in: %w", err)
	}
	binName, baseArgs := resolveCommand(req.GetCommand(), customPath...)
	args := append([]string{}, baseArgs...)
	args = append(args,
		"-p", agentrun.WritePrompt(req.GetInstructions(), req.GetPrompt()),
		"--output-format", "stream-json",
	)
	if model := req.GetModel(); model != "" && model != "default" {
		args = append(args, "--model", model)
	}
	if sessID := reg.GetSession(req.GetConversation()); sessID != "" {
		args = append(args, "--conversation", sessID)
	}
	cmd := exec.CommandContext(runCtx, binName, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "HOME="+tempHome)
	return cmd, func() {
		cleanHome()
		cleanDir()
	}, nil
}
