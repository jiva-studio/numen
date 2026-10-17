package main

import (
	"context"
	"errors"
	"os"
	"strings"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/completion"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

var (
	errNoKey     = errors.New("no Anthropic key is configured")
	errNoCommand = errors.New("the Claude Code CLI is not installed")
)

// Complete answers one question from the model with no tools, no vault and no conversation.
// The key is the one the host configured.
func (s *agentServer) Complete(
	ctx context.Context,
	req *connect.Request[numenv1.CompleteRequest],
) (*connect.Response[numenv1.CompleteResponse], error) {
	s.mu.RLock()
	model := s.completionModel
	s.mu.RUnlock()
	key, base := s.catalogue.credentials()

	return completion.Answer(ctx, req.Msg.GetQuestion(), func(ctx context.Context, request completion.Request) (string, error) {
		if key == "" {
			return "", errNoKey
		}
		name, args := findClaude(), []string(nil)
		if name == "" {
			return "", errNoCommand
		}
		args = append(args, completion.ClaudeArguments(request.System, model)...)
		return completion.RunCommand(ctx, name, args, buildCompletionEnvironment(os.Environ(), key, base), request.Prompt)
	})
}

// buildCompletionEnvironment is the environment of the command with the configured key and base
// URL in place of any the process holds.
func buildCompletionEnvironment(held []string, key, base string) []string {
	out := make([]string, 0, len(held)+2)
	for _, entry := range buildEnvironment(held) {
		name, _, _ := strings.Cut(entry, "=")
		if name == anthropicKeyVar || name == anthropicBaseVar {
			continue
		}
		out = append(out, entry)
	}
	out = append(out, anthropicKeyVar+"="+key)
	if base != "" {
		out = append(out, anthropicBaseVar+"="+base)
	}
	return out
}
