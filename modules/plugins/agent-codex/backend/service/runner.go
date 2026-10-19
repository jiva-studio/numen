package service

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// newSpec is the run of the Codex CLI the request asks for.
func newSpec(req *numenv1.AskAgentRequest) agentrun.Spec {
	return agentrun.Spec{
		Conversation: req.GetConversation(),
		MaxSteps:     int(req.GetMaxSteps()),
		Root:         req.GetRoot(),
		Token:        req.GetToolsToken(),
		Build: func(ctx context.Context) (*exec.Cmd, func(), error) {
			return makeCodexCommand(ctx, req), nil, nil
		},
		NewHandler: newEventHandler,
	}
}

func resolveCommand(command []string) (string, []string) {
	if len(command) > 0 {
		return command[0], command[1:]
	}
	return FindCodex(), nil
}

func makeCodexCommand(running context.Context, req *numenv1.AskAgentRequest) *exec.Cmd {
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
	return exec.CommandContext(running, binName, args...)
}
