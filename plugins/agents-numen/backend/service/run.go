package service

import (
	"context"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/toolcall"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/connection"
)

// defaultTurns is how many requests to the model one task may make where the task names no limit.
const defaultTurns = 25

// reasonDone is why a run stops when the model has answered.
const reasonDone = "done"

// reasonTurns is why a run stops when the model is still calling tools after its last turn.
const reasonTurns = "the agent reached the limit of steps without finishing"

// reasonCut is why a run stops when the model's answer was cut off at its length limit.
const reasonCut = "the answer was cut off at the model's length limit"

// run is one task worked by the model: the way to ask it, the tools, and where its steps go.
type run struct {
	ask         askModel
	temperature *float64
	maxTokens   int
	tools       *toolbox
	words       map[string]toolcall.Declaration
	turns       int
	send        func(*agentrun.Response) error
}

func (r *run) answer(ctx context.Context, messages []connection.ChatMessage) ([]connection.ChatMessage, string, error) {
	turns := r.turns
	if turns <= 0 {
		turns = defaultTurns
	}
	for range turns {
		var (
			reason string
			isDone bool
			err    error
		)
		messages, reason, isDone, err = r.turn(ctx, messages)
		if err != nil {
			return messages, "", err
		}
		if isDone {
			return messages, reason, nil
		}
	}
	return messages, reasonTurns, nil
}

func (r *run) turn(ctx context.Context, messages []connection.ChatMessage) ([]connection.ChatMessage, string, bool, error) {
	if err := r.send(&agentrun.Response{Step: &agentrun.ModelStartedStep{}}); err != nil {
		return messages, "", false, err
	}
	reply, finish, err := r.read(ctx, messages)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return messages, "", false, ctxErr
	}
	if err != nil {
		return messages, "", false, err
	}
	if finish == connection.FinishLength {
		reply.ToolCalls = nil
		return append(messages, reply), reasonCut, true, nil
	}
	messages = append(messages, reply)
	if len(reply.ToolCalls) == 0 {
		return messages, reasonDone, true, nil
	}
	messages, err = r.runToolCalls(ctx, messages, reply.ToolCalls)
	return messages, "", false, err
}

// read asks for one turn and reports its text as it arrives. A turn that fails keeps the text already
// reported, calls no tool and is not asked again.
func (r *run) read(ctx context.Context, messages []connection.ChatMessage) (connection.ChatMessage, connection.FinishReason, error) {
	reply := connection.ChatMessage{Role: "assistant"}
	stream, err := r.ask(ctx, connection.ChatRequest{
		Messages: messages, Tools: r.tools.tools, Temperature: r.temperature, MaxTokens: r.maxTokens,
		Priority: connection.PriorityInteractive,
	})
	if err != nil {
		return reply, connection.FinishStop, &failure{err: err}
	}
	defer func() { _ = stream.Close() }()
	var text strings.Builder
	finish := connection.FinishStop
	for {
		event, ok := stream.Next()
		if !ok {
			break
		}
		switch {
		case event.ToolCall != nil:
			reply.ToolCalls = append(reply.ToolCalls, *event.ToolCall)
		case event.Finish != nil:
			finish = event.Finish.Reason
		default:
			text.WriteString(event.TextDelta)
			if err := r.send(&agentrun.Response{Step: &agentrun.TextStep{Text: event.TextDelta}}); err != nil {
				return reply, finish, err
			}
		}
	}
	reply.Text = text.String()
	if err := stream.Err(); err != nil {
		return reply, finish, &failure{err: err}
	}
	for i := range reply.ToolCalls {
		if strings.TrimSpace(reply.ToolCalls[i].Arguments) == "" {
			reply.ToolCalls[i].Arguments = "{}"
		}
	}
	return reply, finish, nil
}

func (r *run) runToolCalls(ctx context.Context, messages []connection.ChatMessage, calls []connection.ToolCall) ([]connection.ChatMessage, error) {
	for _, call := range calls {
		result, err := r.callTool(ctx, call)
		if err != nil {
			return messages, err
		}
		messages = append(messages, connection.ChatMessage{Role: "tool", ToolCallID: call.ID, Text: result})
	}
	return messages, nil
}

func (r *run) callTool(ctx context.Context, call connection.ToolCall) (string, error) {
	name, args := call.Name, call.Arguments
	described := toolcall.Describe(r.words, name, call.ID, args)
	if err := r.send(&agentrun.Response{Step: &agentrun.ToolCallStep{ToolCall: described}}); err != nil {
		return "", err
	}
	result := r.tools.call(ctx, name, args)
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := r.send(&agentrun.Response{Step: &agentrun.ToolFinishedStep{}}); err != nil {
		return "", err
	}
	return result, nil
}
