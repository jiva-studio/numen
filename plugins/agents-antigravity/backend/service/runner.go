package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/jsonlrun"
)

// newSpec builds the Antigravity CLI invocation for the request.
func newSpec(req *agentrun.AskAgentRequest, reg *agentrun.Registry, customPath ...string) jsonlrun.Spec {
	execPath := ""
	if len(customPath) > 0 {
		execPath = customPath[0]
	}
	s := jsonlrun.Spec{
		Conversation: req.GetConversation(),
		MaxSteps:     int(req.GetMaxSteps()),
		NewHandler:   newEventHandler,
		Root:         "",
		Token:        "",
		Build:        nil,
	}
	if agentrun.HasTools(req) {
		s.Root = req.GetRoot()
		s.Token = req.GetToolsToken()
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
	runCtx context.Context, req *agentrun.AskAgentRequest, reg *agentrun.Registry, customPath ...string,
) (*exec.Cmd, func() error, error) {
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
	if model := req.GetModel(); model != "" && model != defaultModelName {
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
) (*exec.Cmd, func() error, error) {
	tempHome, cleanHome, err := setupAntigravityHome(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("set up the Antigravity home: %w", err)
	}
	dir, cleanDir, err := agentrun.CreateTempDir("numen-antigravity-run-*")
	if err != nil {
		return nil, nil, errors.Join(fmt.Errorf("create the folder to run in: %w", err), cleanHome())
	}
	binName, baseArgs := resolveCommand(req.GetCommand(), customPath...)
	args := append([]string{}, baseArgs...)
	args = append(args,
		"-p", agentrun.WritePrompt(req.GetInstructions(), req.GetPrompt()),
		"--output-format", "stream-json",
	)
	if model := req.GetModel(); model != "" && model != defaultModelName {
		args = append(args, "--model", model)
	}
	if sessID := reg.GetSession(req.GetConversation()); sessID != "" {
		args = append(args, "--conversation", sessID)
	}
	cmd := exec.CommandContext(runCtx, binName, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "HOME="+tempHome)
	return cmd, func() error { return errors.Join(cleanHome(), cleanDir()) }, nil
}
