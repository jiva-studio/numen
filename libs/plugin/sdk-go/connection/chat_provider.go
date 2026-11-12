package connection

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

// ProviderChatRequest is the turn a provider carries out. The history is in Messages.
type ProviderChatRequest struct {
	Messages    []ChatMessage
	Tools       []ToolDefinition
	Temperature *float64
	MaxTokens   int
}

// Chatter carries out the operation chat. It sends the events of one turn in order: text deltas, the
// complete tool calls, then one Finish. It returns the error of the first call of send that fails, and
// NewTryNextError or NewStopError for a turn that failed.
type Chatter interface {
	Chat(ctx context.Context, call Call, request ProviderChatRequest, send func(ChatEvent) error) error
}

func (a *adapter) Chat(
	ctx context.Context, request *connect.Request[v1.ProviderServiceChatRequest], stream *connect.ServerStream[v1.ProviderServiceChatResponse],
) error {
	if a.provider.Chatter == nil {
		return connect.NewError(connect.CodeUnimplemented, errors.New("numen.v1.ProviderService.Chat is not implemented"))
	}
	messages := make([]ChatMessage, len(request.Msg.GetMessages()))
	for i, one := range request.Msg.GetMessages() {
		calls := make([]ToolCall, len(one.GetToolCalls()))
		for j, call := range one.GetToolCalls() {
			calls[j] = readProviderToolCall(call)
		}
		messages[i] = ChatMessage{Role: one.GetRole(), Text: one.GetText(), ToolCalls: calls, ToolCallID: one.GetToolCallId()}
	}
	tools := make([]ToolDefinition, len(request.Msg.GetTools()))
	for i, one := range request.Msg.GetTools() {
		tools[i] = ToolDefinition{Name: one.GetName(), Description: one.GetDescription(), Parameters: []byte(one.GetParametersJson())}
	}
	var temperature *float64
	if request.Msg.Temperature != nil {
		value := request.Msg.GetTemperature()
		temperature = &value
	}
	var sendErr error
	send := func(event ChatEvent) error {
		if sendErr = stream.Send(writeChatEvent(event)); sendErr != nil {
			return sendErr
		}
		return nil
	}
	err := a.provider.Chatter.Chat(ctx, readCall(request.Msg.GetCall()), ProviderChatRequest{
		Messages: messages, Tools: tools, Temperature: temperature, MaxTokens: int(request.Msg.GetMaxTokens()),
	}, send)
	switch {
	case err == nil:
		return nil
	case sendErr != nil:
		return sendErr
	case ctx.Err() != nil || errors.Is(err, context.Canceled):
		return connect.NewError(connect.CodeCanceled, err)
	}
	return stream.Send(&v1.ProviderServiceChatResponse{Event: &v1.ProviderServiceChatResponse_Failure{Failure: writeFailure(err)}})
}

func writeChatEvent(event ChatEvent) *v1.ProviderServiceChatResponse {
	switch {
	case event.ToolCall != nil:
		return &v1.ProviderServiceChatResponse{Event: &v1.ProviderServiceChatResponse_ToolCall{ToolCall: writeProviderToolCall(*event.ToolCall)}}
	case event.Finish != nil:
		return &v1.ProviderServiceChatResponse{Event: &v1.ProviderServiceChatResponse_Finish{
			Finish: &v1.ProviderChatFinish{Reason: writeFinishReason(event.Finish.Reason)},
		}}
	}
	return &v1.ProviderServiceChatResponse{Event: &v1.ProviderServiceChatResponse_TextDelta{TextDelta: event.TextDelta}}
}

func writeProviderToolCall(call ToolCall) *v1.ProviderToolCall {
	return &v1.ProviderToolCall{Id: call.ID, Name: call.Name, ArgumentsJson: call.Arguments}
}

func readProviderToolCall(call *v1.ProviderToolCall) ToolCall {
	return ToolCall{ID: call.GetId(), Name: call.GetName(), Arguments: call.GetArgumentsJson()}
}

func writeFinishReason(reason FinishReason) v1.ProviderChatFinishReason {
	switch reason {
	case FinishToolCalls:
		return v1.ProviderChatFinishReason_PROVIDER_CHAT_FINISH_REASON_TOOL_CALLS
	case FinishLength:
		return v1.ProviderChatFinishReason_PROVIDER_CHAT_FINISH_REASON_LENGTH
	case FinishStop:
		return v1.ProviderChatFinishReason_PROVIDER_CHAT_FINISH_REASON_STOP
	}
	return v1.ProviderChatFinishReason_PROVIDER_CHAT_FINISH_REASON_STOP
}
