// Package hostchat asks the host for the turns of a conversation through the operation llm.chat. The host
// picks the connection that answers.
package hostchat

import (
	"context"
	"errors"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/ports"
)

// Stream is the frames of one turn, as llm.ChatStream reads them.
type Stream interface {
	Next() bool
	Frame() *llm.ChatFrame
	Finish() *llm.ChatFinish
	Err() error
	Close() error
}

// Open starts a turn.
type Open func(ctx context.Context, request *llm.ChatRequest, priority connection.Priority) (Stream, error)

// Chat asks the host for turns through open.
type Chat struct {
	open Open
}

// New returns a Chat that starts each turn with open.
func New(open Open) *Chat {
	return &Chat{open: open}
}

// FromEnvironment returns a Chat that connects to the host the process environment names.
func FromEnvironment() (*Chat, error) {
	client, err := connection.NewClientFromEnv()
	if err != nil {
		return nil, err
	}
	return fromCaller(client), nil
}

// FromVariables returns a Chat that connects to the host the variables name.
func FromVariables(vars map[string]string) (*Chat, error) {
	client, err := connection.NewClientFromVars(vars)
	if err != nil {
		return nil, err
	}
	return fromCaller(client), nil
}

func fromCaller(caller connection.Caller) *Chat {
	return New(func(ctx context.Context, request *llm.ChatRequest, priority connection.Priority) (Stream, error) {
		stream, err := llm.Chat(ctx, caller, request, priority)
		if err != nil {
			return nil, err
		}
		return stream, nil
	})
}

var _ ports.Chat = (*Chat)(nil)

// Stream asks the host for the next turn of the conversation and passes the text to emit as it arrives.
// A person waits for the answer, so the turn is interactive.
func (c *Chat) Stream(ctx context.Context, turn conversation.Turn, emit func(string) error) (reply conversation.Reply, err error) {
	stream, err := c.open(ctx, &llm.ChatRequest{
		Messages: writeMessages(turn.Messages), Tools: writeTools(turn.Tools),
		Temperature: turn.Temperature, MaxTokens: int32(turn.MaxTokens),
	}, connection.PriorityInteractive)
	if err != nil {
		return conversation.Reply{}, readError(err)
	}
	defer func() { err = errors.Join(err, stream.Close()) }()

	reply.Message = conversation.Message{Role: "assistant", Text: "", ToolCalls: nil, ToolCallID: ""}
	var text strings.Builder
	for stream.Next() {
		frame := stream.Frame()
		if call := frame.GetToolCall(); call != nil {
			reply.Message.ToolCalls = append(reply.Message.ToolCalls, readCall(call))
			continue
		}
		delta := frame.GetTextDelta()
		text.WriteString(delta)
		if err := emit(delta); err != nil {
			return reply, err
		}
	}
	reply.Message.Text = text.String()
	if streamErr := stream.Err(); streamErr != nil {
		return reply, readError(streamErr)
	}
	reply.IsCut = stream.Finish().GetReason() == llm.FinishReason_FINISH_REASON_LENGTH
	return reply, nil
}

func writeMessages(messages []conversation.Message) []*llm.ChatMessage {
	out := make([]*llm.ChatMessage, len(messages))
	for i, one := range messages {
		calls := make([]*llm.ToolCall, len(one.ToolCalls))
		for j, call := range one.ToolCalls {
			calls[j] = &llm.ToolCall{Id: call.ID, Name: call.Name, ArgumentsJson: call.Arguments}
		}
		out[i] = &llm.ChatMessage{Role: one.Role, Text: one.Text, ToolCalls: calls, ToolCallId: one.ToolCallID}
	}
	return out
}

func writeTools(tools []conversation.Tool) []*llm.ToolDefinition {
	out := make([]*llm.ToolDefinition, len(tools))
	for i, one := range tools {
		out[i] = &llm.ToolDefinition{Name: one.Name, Description: one.Description, ParametersJson: string(one.Parameters)}
	}
	return out
}

func readCall(call *llm.ToolCall) conversation.ToolCall {
	return conversation.ToolCall{ID: call.GetId(), Name: call.GetName(), Arguments: call.GetArgumentsJson()}
}

func readError(err error) error {
	var allFailed *connection.AllFailedError
	var stopped *connection.StoppedError
	switch {
	case errors.Is(err, connection.ErrNoConnection):
		return errors.Join(conversation.ErrNoConnection, err)
	case errors.As(err, &allFailed):
		parts := make([]string, len(allFailed.Failures))
		for i, one := range allFailed.Failures {
			parts[i] = one.ConnectionName + ": " + one.Reason
		}
		return errors.New(strings.Join(parts, "; "))
	case errors.As(err, &stopped):
		return errors.New(stopped.ConnectionName + ": " + stopped.Reason)
	}
	return err
}
