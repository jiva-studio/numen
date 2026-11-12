package answer

import (
	"context"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/ports"
)

// chatError is a turn the model could not be asked for. It stops the run.
type chatError struct{ err error }

func (f *chatError) Error() string { return f.err.Error() }

func (f *chatError) Unwrap() error { return f.err }

// run is one task worked by the model: the way to ask it, the tools, and where its frames go.
type run struct {
	chat        ports.Chat
	temperature *float64
	maxTokens   int
	tools       ports.Tools
	offered     []conversation.Tool
	listed      map[string]conversation.Tool
	scope       conversation.Scope
	turns       int
	send        func(*acp.TaskFrame) error
}

// answer asks the model turn by turn until it answers without a tool call, its answer is cut off or the
// limit of turns is reached, and returns the transcript and the stop reason.
func (r *run) answer(ctx context.Context, messages []conversation.Message) ([]conversation.Message, acp.StopReason, error) {
	for index := range r.turns {
		var (
			reason acp.StopReason
			isDone bool
			err    error
		)
		messages, reason, isDone, err = r.turn(ctx, messages, "message-"+strconv.Itoa(index+1))
		if err != nil {
			return messages, acp.StopReason_STOP_REASON_UNSPECIFIED, err
		}
		if isDone {
			return messages, reason, nil
		}
	}
	return messages, acp.StopReason_STOP_REASON_MAX_TURN_REQUESTS, nil
}

func (r *run) turn(ctx context.Context, messages []conversation.Message, messageID string) ([]conversation.Message, acp.StopReason, bool, error) {
	var unsent error
	emit := func(text string) error {
		if text == "" {
			return nil
		}
		unsent = r.send(&acp.TaskFrame{Update: &acp.TaskFrame_AgentMessageChunk{
			AgentMessageChunk: &acp.MessageChunk{MessageId: messageID, Text: text},
		}})
		return unsent
	}
	reply, err := r.chat.Stream(ctx, conversation.Turn{
		Messages: messages, Tools: r.offered, Temperature: r.temperature, MaxTokens: r.maxTokens,
	}, emit)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return messages, acp.StopReason_STOP_REASON_UNSPECIFIED, false, ctxErr
	}
	if unsent != nil {
		return messages, acp.StopReason_STOP_REASON_UNSPECIFIED, false, unsent
	}
	if err != nil {
		return messages, acp.StopReason_STOP_REASON_UNSPECIFIED, false, &chatError{err: err}
	}
	message := reply.Message
	if reply.IsCut {
		message.ToolCalls = nil
		return append(messages, message), acp.StopReason_STOP_REASON_MAX_TOKENS, true, nil
	}
	cleanToolCalls(message.ToolCalls)
	messages = append(messages, message)
	if len(message.ToolCalls) == 0 {
		return messages, acp.StopReason_STOP_REASON_END_TURN, true, nil
	}
	messages, err = r.runToolCalls(ctx, messages, message.ToolCalls)
	return messages, acp.StopReason_STOP_REASON_UNSPECIFIED, false, err
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
	if err := r.send(&acp.TaskFrame{Update: &acp.TaskFrame_ToolCall{ToolCall: describeCall(r.listed, call)}}); err != nil {
		return "", err
	}
	result := conversation.ToolResult{Text: "error: there is no tool " + strconv.Quote(call.Name), IsError: true}
	if _, isListed := r.listed[call.Name]; isListed {
		result = r.tools.CallTool(ctx, call.Name, call.Arguments, r.scope)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	status := acp.ToolCallStatus_TOOL_CALL_STATUS_COMPLETED
	if result.IsError {
		status = acp.ToolCallStatus_TOOL_CALL_STATUS_FAILED
	}
	if err := r.send(&acp.TaskFrame{Update: &acp.TaskFrame_ToolCallUpdate{
		ToolCallUpdate: &acp.ToolCallUpdate{ToolCallId: call.ID, Status: &status},
	}}); err != nil {
		return "", err
	}
	return result.Text, nil
}
