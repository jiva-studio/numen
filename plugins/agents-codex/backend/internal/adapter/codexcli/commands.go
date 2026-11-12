package codexcli

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/usecase/run"
)

// Commands builds the command lines of a Codex run.
type Commands struct{}

func getDisabledFeatures() []string {
	return []string{
		"shell_tool", "unified_exec", "view_image", "multi_agent", "apps", "browser_use",
		"computer_use", "plugins", "image_generation", "hooks",
	}
}

func resolveCommand(command []string, customPath string) (string, []string) {
	if len(command) > 0 {
		return command[0], command[1:]
	}
	return FindCodex(customPath), nil
}

// MakeToolsCommand returns the command of a run that may call the tools the request names. The
// program is given the tools server and runs without approvals or a sandbox.
func (Commands) MakeToolsCommand(_ context.Context, req *agentrun.Request, customPath string) run.Command {
	binName, baseArgs := resolveCommand(req.Command, customPath)
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
	return run.Command{Path: binName, Args: args, Dir: "", Env: nil, Token: "", Release: nil}
}

// MakeAnswerCommand returns the command of a run that only answers, in a new empty folder.
func (Commands) MakeAnswerCommand(_ context.Context, req *agentrun.Request, customPath string) (run.Command, error) {
	dir, cleanDir, err := agentrun.CreateTempDir("numen-codex-*")
	if err != nil {
		return run.Command{}, fmt.Errorf("create the folder to run in: %w", err)
	}
	binName, baseArgs := resolveCommand(req.Command, customPath)
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
	return run.Command{Path: binName, Args: args, Dir: dir, Env: nil, Token: "", Release: cleanDir}, nil
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
