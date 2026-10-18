package main

import (
	"context"
	"errors"
	"os"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/completion"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

var errNoCompletionAgent = errors.New("no installed agent can give a single answer without tools")

// Complete answers one question from the installed Claude Code CLI, with the model the host names,
// and with no tools, no vault and no conversation. The CLI signs in with the person's own account.
func (s *agentServer) Complete(
	ctx context.Context,
	req *connect.Request[numenv1.CompleteRequest],
) (*connect.Response[numenv1.CompleteResponse], error) {
	command, model := s.command, req.Msg.GetModel()
	return completion.Answer(ctx, req.Msg.GetQuestion(), func(ctx context.Context, request completion.Request) (string, error) {
		name, args := getCompletionCommand(command)
		if name == "" {
			return "", errNoCompletionAgent
		}
		args = append(args, completion.ClaudeArguments(request.System, model)...)
		return completion.RunCommand(ctx, name, args, os.Environ(), request.Prompt)
	})
}

// getCompletionCommand is the command that gives a single answer: the one named, or the Claude Code
// CLI where it is installed.
func getCompletionCommand(command []string) (string, []string) {
	if len(command) > 0 {
		return command[0], command[1:]
	}
	if IsClaudeInstalled() {
		return FindClaude(), nil
	}
	return "", nil
}
