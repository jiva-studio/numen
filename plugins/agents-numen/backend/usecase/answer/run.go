package answer

import (
	"context"
	"errors"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/toolcall"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/ports"
)

// defaultTurns is how many requests to the model one task may make where the task names no limit.
const defaultTurns = 25

// ReasonDone is why a run stops when the model has answered.
const ReasonDone = "done"

// ReasonTurns is why a run stops when the model is still calling tools after its last turn.
const ReasonTurns = "the agent reached the limit of steps without finishing"

// ReasonCut is why a run stops when the model's answer was cut off at its length limit.
const ReasonCut = "the answer was cut off at the model's length limit"

// chatError is a run stopped by a turn that failed. It stops the run and is reported as why it stopped.
type chatError struct{ err error }

func (f *chatError) Error() string {
	if errors.Is(f.err, conversation.ErrNoConnection) {
		return conversation.ReasonNoConnection
	}
	return f.err.Error()
}

func (f *chatError) Unwrap() error { return f.err }

// run is one task worked by the model: the way to ask it, the tools, and where its steps go.
type run struct {
	chat        ports.Chat
	temperature *float64
	maxTokens   int
	tools       ports.Toolbox
	words       map[string]toolcall.Declaration
	turns       int
	send        func(*agentrun.Response) error
}

func (r *run) answer(ctx context.Context, messages []conversation.Message) ([]conversation.Message, string, error) {
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
	return messages, ReasonTurns, nil
}

func (r *run) turn(ctx context.Context, messages []conversation.Message) ([]conversation.Message, string, bool, error) {
	if err := r.send(&agentrun.Response{Step: &agentrun.ModelStartedStep{}}); err != nil {
		return messages, "", false, err
	}
	var unsent error
	emit := func(text string) error {
		unsent = r.send(&agentrun.Response{Step: &agentrun.TextStep{Text: text}})
		return unsent
	}
	reply, err := r.chat.Stream(ctx, conversation.Turn{
		Messages: messages, Tools: r.tools.Tools(), Temperature: r.temperature, MaxTokens: r.maxTokens,
	}, emit)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return messages, "", false, ctxErr
	}
	if unsent != nil {
		return messages, "", false, unsent
	}
	if err != nil {
		return messages, "", false, &chatError{err: err}
	}
	message := reply.Message
	if reply.IsCut {
		message.ToolCalls = nil
		return append(messages, message), ReasonCut, true, nil
	}
	cleanToolCalls(message.ToolCalls)
	messages = append(messages, message)
	if len(message.ToolCalls) == 0 {
		return messages, ReasonDone, true, nil
	}
	messages, err = r.runToolCalls(ctx, messages, message.ToolCalls)
	return messages, "", false, err
}

func cleanToolCalls(calls []conversation.ToolCall) {
	for i := range calls {
		if strings.TrimSpace(calls[i].Arguments) == "" {
			calls[i].Arguments = "{}"
		}
	}
}

func (r *run) runToolCalls(ctx context.Context, messages []conversation.Message, calls []conversation.ToolCall) ([]conversation.Message, error) {
	for _, call := range calls {
		result, err := r.callTool(ctx, call)
		if err != nil {
			return messages, err
		}
		messages = append(messages, conversation.Message{Role: "tool", ToolCallID: call.ID, Text: result, ToolCalls: nil})
	}
	return messages, nil
}

func (r *run) callTool(ctx context.Context, call conversation.ToolCall) (string, error) {
	described := toolcall.Describe(r.words, call.Name, call.ID, call.Arguments)
	if err := r.send(&agentrun.Response{Step: &agentrun.ToolCallStep{ToolCall: described}}); err != nil {
		return "", err
	}
	result := r.tools.Call(ctx, call.Name, call.Arguments)
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := r.send(&agentrun.Response{Step: &agentrun.ToolFinishedStep{}}); err != nil {
		return "", err
	}
	return result, nil
}
