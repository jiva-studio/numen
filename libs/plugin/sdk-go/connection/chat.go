package connection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

// ErrStreamEnded means the stream of a turn ended with neither a finish nor an error.
var ErrStreamEnded = errors.New("connection: the stream ended before the turn did")

// ToolCall is a call of a tool the model asked for. Arguments is one JSON object.
//
//numen:api
type ToolCall struct {
	// ID names the call; the tool message that answers it carries the same id.
	ID string
	// Name is the name of the tool.
	Name string
	// Arguments is the arguments as one JSON object.
	Arguments string
}

// ToolDefinition is a tool offered to the model.
//
//numen:api
type ToolDefinition struct {
	// Name is the name the model calls the tool by.
	Name string
	// Description tells the model what the tool does.
	Description string
	// Parameters is the JSON schema of the arguments.
	Parameters json.RawMessage
}

// ChatMessage is one message of a conversation. Role is system, user, assistant or tool. An assistant
// message carries the calls it asked for, and a tool message carries the id of the call it answers.
//
//numen:api
type ChatMessage struct {
	// Role is system, user, assistant or tool.
	Role string
	// Text is the content of the message.
	Text string
	// ToolCalls are the calls an assistant message asked for.
	ToolCalls []ToolCall
	// ToolCallID is the call a tool message answers.
	ToolCallID string
}

// FinishReason is why a turn ended. The zero value is FinishStop.
type FinishReason int

// The reasons a turn ends.
const (
	// FinishStop is an answer the model finished.
	FinishStop FinishReason = iota
	// FinishToolCalls is a turn that asks for tool calls.
	FinishToolCalls
	// FinishLength is an answer cut off at the length limit.
	FinishLength
)

// Finish ends a turn.
type Finish struct{ Reason FinishReason }

// ChatEvent is one event of a turn: exactly one field is set. A text delta is never empty.
//
//numen:api
type ChatEvent struct {
	// TextDelta is the next piece of the answer.
	TextDelta string
	// ToolCall is a complete call of a tool.
	ToolCall *ToolCall
	// Finish ends the turn and is the last event.
	Finish *Finish
}

// ChatRequest is one turn a consumer asks a model for. The history travels with it.
type ChatRequest struct {
	Messages    []ChatMessage
	Tools       []ToolDefinition
	Temperature *float64
	MaxTokens   int
	Priority    Priority
}

// ChatStream is the events of one turn. A reader calls Next until it returns false and then reads Err.
type ChatStream struct {
	stream     *connect.ServerStreamForClient[v1.HostModelServiceChatResponse]
	contextErr func() error
	err        error
	isDone     bool
}

// Chat asks the rows of the plugin, in order, for the next turn of a conversation. The core goes to the
// next row only when a row fails before its first event. The caller closes the stream.
func (c *Client) Chat(ctx context.Context, request ChatRequest) (*ChatStream, error) {
	messages := make([]*v1.HostModelChatMessage, len(request.Messages))
	for i, one := range request.Messages {
		calls := make([]*v1.HostModelToolCall, len(one.ToolCalls))
		for j, call := range one.ToolCalls {
			calls[j] = &v1.HostModelToolCall{Id: call.ID, Name: call.Name, ArgumentsJson: call.Arguments}
		}
		messages[i] = &v1.HostModelChatMessage{Role: one.Role, Text: one.Text, ToolCalls: calls, ToolCallId: one.ToolCallID}
	}
	tools := make([]*v1.HostModelToolDefinition, len(request.Tools))
	for i, one := range request.Tools {
		tools[i] = &v1.HostModelToolDefinition{Name: one.Name, Description: one.Description, ParametersJson: string(one.Parameters)}
	}
	stream, err := c.host.Chat(ctx, connect.NewRequest(&v1.HostModelServiceChatRequest{
		Messages: messages, Tools: tools, Temperature: request.Temperature, MaxTokens: int32(request.MaxTokens),
		Priority: writePriority(request.Priority),
	}))
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, fmt.Errorf("connection: chat: %w", err)
	}
	return &ChatStream{stream: stream, contextErr: ctx.Err}, nil
}

// Next returns the next event. It returns false at the end of the turn and on an error; Err tells which.
func (s *ChatStream) Next() (ChatEvent, bool) {
	if s.isDone {
		return ChatEvent{}, false
	}
	if !s.stream.Receive() {
		s.isDone = true
		s.err = s.readEnd()
		return ChatEvent{}, false
	}
	switch event := s.stream.Msg().GetEvent().(type) {
	case *v1.HostModelServiceChatResponse_TextDelta:
		return ChatEvent{TextDelta: event.TextDelta}, true
	case *v1.HostModelServiceChatResponse_ToolCall:
		return ChatEvent{ToolCall: &ToolCall{
			ID: event.ToolCall.GetId(), Name: event.ToolCall.GetName(), Arguments: event.ToolCall.GetArgumentsJson(),
		}}, true
	case *v1.HostModelServiceChatResponse_Finish:
		s.isDone = true
		return ChatEvent{Finish: &Finish{Reason: readHostFinishReason(event.Finish.GetReason())}}, true
	case *v1.HostModelServiceChatResponse_Failure:
		s.isDone = true
		s.err = readError(event.Failure.GetError(), event.Failure.GetDetail(), event.Failure.GetFailures())
		return ChatEvent{}, false
	}
	return s.Next()
}

// Err returns why the stream ended: nil after a finish, ErrNoConnection, *AllFailedError or
// *RejectedError for a turn the core could not serve, the error of the context for a canceled call, or
// ErrStreamEnded.
func (s *ChatStream) Err() error { return s.err }

// Close ends the stream and cancels the call if the turn has not ended.
func (s *ChatStream) Close() error { return s.stream.Close() }

func (s *ChatStream) readEnd() error {
	if err := s.stream.Err(); err != nil {
		if ctxErr := s.contextErr(); ctxErr != nil {
			return ctxErr
		}
		return fmt.Errorf("connection: chat: %w", err)
	}
	return ErrStreamEnded
}

func readHostFinishReason(reason v1.HostModelChatFinishReason) FinishReason {
	switch reason {
	case v1.HostModelChatFinishReason_HOST_MODEL_CHAT_FINISH_REASON_TOOL_CALLS:
		return FinishToolCalls
	case v1.HostModelChatFinishReason_HOST_MODEL_CHAT_FINISH_REASON_LENGTH:
		return FinishLength
	case v1.HostModelChatFinishReason_HOST_MODEL_CHAT_FINISH_REASON_STOP, v1.HostModelChatFinishReason_HOST_MODEL_CHAT_FINISH_REASON_UNSPECIFIED:
		return FinishStop
	}
	return FinishStop
}
