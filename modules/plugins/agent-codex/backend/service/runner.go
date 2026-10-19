package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// disabledFeatures are the tools of Codex's own that a run naming no tool is started without.
var disabledFeatures = []string{
	"shell_tool", "unified_exec", "view_image", "multi_agent", "apps", "browser_use",
	"computer_use", "plugins", "image_generation", "hooks",
}

// newSpec is the run of the Codex CLI the request asks for.
func newSpec(req *numenv1.AskAgentRequest) agentrun.Spec {
	spec := agentrun.Spec{
		Conversation: req.GetConversation(),
		MaxSteps:     int(req.GetMaxSteps()),
		NewHandler:   newEventHandler,
	}
	if agentrun.HasTools(req) {
		spec.Root = req.GetRoot()
		spec.Token = req.GetToolsToken()
		spec.Build = func(ctx context.Context) (*exec.Cmd, func(), error) {
			return makeCodexCommand(ctx, req), nil, nil
		}
		return spec
	}
	spec.Build = func(ctx context.Context) (*exec.Cmd, func(), error) {
		return makeAnswerCommand(ctx, req)
	}
	return spec
}

func resolveCommand(command []string) (string, []string) {
	if len(command) > 0 {
		return command[0], command[1:]
	}
	return FindCodex(), nil
}

func makeCodexCommand(runCtx context.Context, req *numenv1.AskAgentRequest) *exec.Cmd {
	binName, baseArgs := resolveCommand(req.GetCommand())
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
	if model := req.GetModel(); model != "" && model != "default" {
		args = append(args, "--model", model)
	}
	args = append(args, req.GetPrompt())
	return exec.CommandContext(runCtx, binName, args...)
}

// makeAnswerCommand is the run of the Codex CLI that gives one answer: no MCP server, none of the
// tools Codex has of its own that can be turned off, nothing the user's configuration adds, a
// sandbox that writes nothing and an empty folder to run in. The folder goes when the run does.
func makeAnswerCommand(runCtx context.Context, req *numenv1.AskAgentRequest) (*exec.Cmd, func(), error) {
	dir, err := os.MkdirTemp("", "numen-codex-*")
	if err != nil {
		return nil, nil, fmt.Errorf("create the folder to run in: %w", err)
	}
	binName, baseArgs := resolveCommand(req.GetCommand())
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
	if model := req.GetModel(); model != "" && model != "default" {
		args = append(args, "--model", model)
	}
	args = append(args, "--", agentrun.WritePrompt(req.GetInstructions(), req.GetPrompt()))
	cmd := exec.CommandContext(runCtx, binName, args...)
	cmd.Dir = dir
	return cmd, func() { _ = os.RemoveAll(dir) }, nil
}
