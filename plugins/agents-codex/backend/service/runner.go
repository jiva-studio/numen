package service

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/jsonlrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/reasoning"
)

// disabledFeatures are the tools of Codex's own that a run naming no tool is started without.
var disabledFeatures = []string{
	"shell_tool", "unified_exec", "view_image", "multi_agent", "apps", "browser_use",
	"computer_use", "plugins", "image_generation", "hooks",
}

// newSpec builds the Codex CLI invocation for the request.
func newSpec(req *agentrun.AskAgentRequest, customPath ...string) jsonlrun.Spec {
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

func makeCodexCommand(runCtx context.Context, req *agentrun.AskAgentRequest, customPath ...string) *exec.Cmd {
	binName, baseArgs := resolveCommand(req.GetCommand(), customPath...)
	args := append([]string{}, baseArgs...)
	args = append(args,
		"exec",
		"--json",
		"--dangerously-bypass-approvals-and-sandbox",
	)
	if req.GetToolsUrl() != "" {
		mcpURL := fmt.Sprintf(`mcp_servers.numen.url=%q`, req.GetToolsUrl())
		mcpHeader := fmt.Sprintf(`mcp_servers.numen.http_headers={"Authorization"="Bearer %s"}`, req.GetToolsToken())
		args = append(args, "-c", mcpURL, "-c", mcpHeader)
	}
	args = appendModelArguments(args, req)
	args = append(args, req.GetPrompt())
	return exec.CommandContext(runCtx, binName, args...)
}

// makeAnswerCommand is the run of the Codex CLI that gives one answer: no MCP server, none of the
// tools Codex has of its own that can be turned off, nothing the user's configuration adds, a
// sandbox that writes nothing and an empty folder to run in. The folder goes when the run does.
func makeAnswerCommand(runCtx context.Context, req *agentrun.AskAgentRequest, customPath ...string) (*exec.Cmd, func() error, error) {
	dir, cleanDir, err := agentrun.CreateTempDir("numen-codex-*")
	if err != nil {
		return nil, nil, fmt.Errorf("create the folder to run in: %w", err)
	}
	binName, baseArgs := resolveCommand(req.GetCommand(), customPath...)
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
	args = append(args, "--", agentrun.WritePrompt(req.GetInstructions(), req.GetPrompt()))
	cmd := exec.CommandContext(runCtx, binName, args...)
	cmd.Dir = dir
	return cmd, cleanDir, nil
}

// appendModelArguments adds the model the request names and the reasoning level it requests.
func appendModelArguments(args []string, req *agentrun.AskAgentRequest) []string {
	if model := req.GetModel(); model != "" && model != "default" {
		args = append(args, "--model", model)
	}
	if level := req.GetReasoning(); reasoning.IsKnown(level) {
		args = append(args, "-c", fmt.Sprintf("model_reasoning_effort=%q", level))
	}
	return args
}
