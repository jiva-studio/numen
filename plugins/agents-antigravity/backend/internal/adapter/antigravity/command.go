package antigravity

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/model"
	"github.com/jiva-studio/numen/plugins/agents-antigravity/backend/usecase/runs"
)

func resolveCommand(command []string, customPath ...string) (string, []string) {
	if len(command) > 0 {
		return command[0], command[1:]
	}
	return findProgram(customPath...), nil
}

func makeAntigravityCommand(
	runCtx context.Context, req *agentrun.Request, reg *agentrun.Registry, customPath ...string,
) (runs.Command, error) {
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
	if modelName := req.Model; modelName != "" && modelName != model.DefaultName {
		args = append(args, "--model", modelName)
	}
	if sessID := reg.GetSession(req.Conversation); sessID != "" {
		args = append(args, "--conversation", sessID)
	}

	if req.ToolsURL == "" {
		return runs.Command{Path: binName, Args: args, Dir: "", Env: nil, Token: "", Release: nil}, nil
	}
	tempHome, clean, err := setupAntigravityHome(&endpoint{URL: req.ToolsURL, Token: req.ToolsToken})
	if err != nil {
		return runs.Command{}, fmt.Errorf("set up the Antigravity home: %w", err)
	}
	return runs.Command{
		Path: binName, Args: args, Dir: "", Env: append(hostcall.GetChildEnvironment(), "HOME="+tempHome), Token: "", Release: clean,
	}, nil
}

func makeAnswerCommand(
	_ context.Context, req *agentrun.Request, reg *agentrun.Registry, customPath ...string,
) (runs.Command, error) {
	tempHome, cleanHome, err := setupAntigravityHome(nil)
	if err != nil {
		return runs.Command{}, fmt.Errorf("set up the Antigravity home: %w", err)
	}
	dir, cleanDir, err := agentrun.CreateTempDir("numen-antigravity-run-*")
	if err != nil {
		return runs.Command{}, errors.Join(fmt.Errorf("create the folder to run in: %w", err), cleanHome())
	}
	binName, baseArgs := resolveCommand(req.Command, customPath...)
	args := append([]string{}, baseArgs...)
	args = append(args,
		"-p", agentrun.WritePrompt(req.Instructions, req.Prompt),
		"--output-format", "stream-json",
	)
	if modelName := req.Model; modelName != "" && modelName != model.DefaultName {
		args = append(args, "--model", modelName)
	}
	if sessID := reg.GetSession(req.Conversation); sessID != "" {
		args = append(args, "--conversation", sessID)
	}
	return runs.Command{
		Path:  binName,
		Args:  args,
		Dir:   dir,
		Env:   append(hostcall.GetChildEnvironment(), "HOME="+tempHome),
		Token: "",
		Release: func() error {
			return errors.Join(cleanHome(), cleanDir())
		},
	}, nil
}
