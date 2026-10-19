package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// newSpec is the run of the Antigravity CLI the request asks for.
func newSpec(req *numenv1.AskAgentRequest, reg *agentrun.Registry) agentrun.Spec {
	spec := agentrun.Spec{
		Conversation: req.GetConversation(),
		MaxSteps:     int(req.GetMaxSteps()),
		NewHandler:   newEventHandler,
	}
	if agentrun.HasTools(req) {
		spec.Root = req.GetRoot()
		spec.Token = req.GetToolsToken()
	}
	spec.Build = func(ctx context.Context) (*exec.Cmd, func(), error) {
		return makeAntigravityCommand(ctx, req, reg)
	}
	return spec
}

func resolveCommand(command []string) (string, []string) {
	if len(command) > 0 {
		return command[0], command[1:]
	}
	return FindAntigravity(), nil
}

func makeAntigravityCommand(
	runCtx context.Context, req *numenv1.AskAgentRequest, reg *agentrun.Registry,
) (*exec.Cmd, func(), error) {
	if !agentrun.HasTools(req) {
		return makeAnswerCommand(runCtx, req, reg)
	}
	binName, baseArgs := resolveCommand(req.GetCommand())
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
	runCtx context.Context, req *numenv1.AskAgentRequest, reg *agentrun.Registry,
) (*exec.Cmd, func(), error) {
	tempHome, cleanHome, err := setupAntigravityHome(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("set up the Antigravity home: %w", err)
	}
	dir, err := os.MkdirTemp("", "numen-antigravity-run-*")
	if err != nil {
		cleanHome()
		return nil, nil, fmt.Errorf("create the folder to run in: %w", err)
	}
	binName, baseArgs := resolveCommand(req.GetCommand())
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
		_ = os.RemoveAll(dir)
	}, nil
}
