package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"slices"
	"strings"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agentrun"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// disallowed are the tools the program is started without when it answers a question. It reads one
// question and writes one answer, and touches neither this machine nor the web.
var disallowed = []string{
	"Write", "Edit", "Bash", "Read", "WebFetch", "WebSearch",
	"Glob", "Grep", "NotebookEdit", "Task", "TodoWrite",
}

// buildAnswerArguments returns what the program is started with to answer a question: one question,
// no servers, no tools, and the instructions appended to what the program says of itself.
func buildAnswerArguments(model, instructions string) []string {
	args := []string{"-p", "--strict-mcp-config"}
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, "--disallowed-tools")
	args = append(args, disallowed...)
	return append(args, "--append-system-prompt", instructions)
}

// AnswerAgent puts one question to the program and answers with what it wrote.
//
// The question goes on the input: --disallowed-tools takes as many names as follow it, and a
// question is longer than a command line holds. The program reads whatever standing instructions
// and settings live where it was started, so it is started in an empty folder of the run's own.
func (s *agentServer) AnswerAgent(
	ctx context.Context,
	req *connect.Request[numenv1.AnswerAgentRequest],
) (*connect.Response[numenv1.AnswerAgentResponse], error) {
	msg := req.Msg
	if msg.GetInstructions() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("no instructions for the program"))
	}
	name, rest := getCommand(msg.GetCommand())
	if name == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("claude is not installed"))
	}

	empty, err := os.MkdirTemp("", "numen-answer-")
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer os.RemoveAll(empty)

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	work := agentrun.NewWork("", 0, cancel)
	if !s.registry.Add(work) {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("this agent is closing"))
	}
	defer s.registry.Remove(work)
	defer work.End()

	cmd := exec.CommandContext(runCtx, name, slices.Concat(rest, buildAnswerArguments(msg.GetModel(), msg.GetInstructions()))...)
	cmd.Dir = empty
	cmd.Env = buildEnvironment(os.Environ())
	cmd.Stdin = strings.NewReader(msg.GetPrompt())
	var said, stderr bytes.Buffer
	cmd.Stdout = &said
	cmd.Stderr = &stderr
	agentrun.Supervise(cmd)

	if err := cmd.Run(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New(getFailureMessage(err, stderr.String())))
	}
	return connect.NewResponse(&numenv1.AnswerAgentResponse{Text: strings.TrimSpace(said.String())}), nil
}
