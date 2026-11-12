// Package hostchat asks the model API of the host for the turns of a conversation. The host picks the
// connection that answers.
package hostchat

import (
	"context"
	"errors"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/ports"
)

// Stream is the events of one turn.
type Stream interface {
	Next() (connection.ChatEvent, bool)
	Err() error
	Close() error
}

// Client is the model API of the host, as the agent uses it.
type Client interface {
	Chat(ctx context.Context, request connection.ChatRequest) (*connection.ChatStream, error)
}

// Open starts a turn.
type Open func(ctx context.Context, request connection.ChatRequest) (Stream, error)

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
	return fromClient(client), nil
}

// FromVariables returns a Chat that connects to the host the variables name.
func FromVariables(vars map[string]string) (*Chat, error) {
	client, err := connection.NewClientFromVars(vars)
	if err != nil {
		return nil, err
	}
	return fromClient(client), nil
}

func fromClient(client Client) *Chat {
	return New(func(ctx context.Context, request connection.ChatRequest) (Stream, error) {
		stream, err := client.Chat(ctx, request)
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
	stream, err := c.open(ctx, connection.ChatRequest{
		Messages: writeMessages(turn.Messages), Tools: writeTools(turn.Tools),
		Temperature: turn.Temperature, MaxTokens: turn.MaxTokens, Priority: connection.PriorityInteractive,
	})
	if err != nil {
		return conversation.Reply{}, readError(err)
	}
	defer func() { err = errors.Join(err, stream.Close()) }()

	reply.Message = conversation.Message{Role: "assistant", Text: "", ToolCalls: nil, ToolCallID: ""}
	var text strings.Builder
	for {
		event, ok := stream.Next()
		if !ok {
			break
		}
		switch {
		case event.ToolCall != nil:
			reply.Message.ToolCalls = append(reply.Message.ToolCalls, readCall(*event.ToolCall))
		case event.Finish != nil:
			reply.IsCut = event.Finish.Reason == connection.FinishLength
		default:
			text.WriteString(event.TextDelta)
			if err := emit(event.TextDelta); err != nil {
				return reply, err
			}
		}
	}
	reply.Message.Text = text.String()
	if streamErr := stream.Err(); streamErr != nil {
		return reply, readError(streamErr)
	}
	return reply, nil
}

func writeMessages(messages []conversation.Message) []connection.ChatMessage {
	out := make([]connection.ChatMessage, len(messages))
	for i, one := range messages {
		calls := make([]connection.ToolCall, len(one.ToolCalls))
		for j, call := range one.ToolCalls {
			calls[j] = connection.ToolCall{ID: call.ID, Name: call.Name, Arguments: call.Arguments}
		}
		out[i] = connection.ChatMessage{Role: one.Role, Text: one.Text, ToolCalls: calls, ToolCallID: one.ToolCallID}
	}
	return out
}

func writeTools(tools []conversation.Tool) []connection.ToolDefinition {
	out := make([]connection.ToolDefinition, len(tools))
	for i, one := range tools {
		out[i] = connection.ToolDefinition{Name: one.Name, Description: one.Description, Parameters: one.Parameters}
	}
	return out
}

func readCall(call connection.ToolCall) conversation.ToolCall {
	return conversation.ToolCall{ID: call.ID, Name: call.Name, Arguments: call.Arguments}
}

// readError names each connection that failed and the reason, or says that the agent has none.
func readError(err error) error {
	var allFailed *connection.AllFailedError
	var rejected *connection.RejectedError
	switch {
	case errors.Is(err, connection.ErrNoConnection):
		return errors.Join(conversation.ErrNoConnection, err)
	case errors.As(err, &allFailed):
		parts := make([]string, len(allFailed.Failures))
		for i, one := range allFailed.Failures {
			parts[i] = one.ConnectionName + ": " + one.Reason
		}
		return errors.New(strings.Join(parts, "; "))
	case errors.As(err, &rejected):
		return errors.New(rejected.ConnectionName + ": " + rejected.Reason)
	}
	return err
}
