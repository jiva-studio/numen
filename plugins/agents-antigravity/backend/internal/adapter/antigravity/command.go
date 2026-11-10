package antigravity

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/models"
)

func resolveCommand(command []string, customPath ...string) (string, []string) {
	if len(command) > 0 {
		return command[0], command[1:]
	}
	return findProgram(customPath...), nil
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
	if model := req.Model; model != "" && model != models.DefaultName {
		args = append(args, "--model", model)
	}
	if sessID := reg.GetSession(req.Conversation); sessID != "" {
		args = append(args, "--conversation", sessID)
	}

	cmd := exec.CommandContext(runCtx, binName, args...)
	if req.ToolsURL != "" {
		tempHome, clean, err := setupAntigravityHome(&endpoint{URL: req.ToolsURL, Token: req.ToolsToken})
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
	if model := req.Model; model != "" && model != models.DefaultName {
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
