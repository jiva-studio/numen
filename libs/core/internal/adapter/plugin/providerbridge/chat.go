package providerbridge

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	modelcallport "github.com/jiva-studio/numen/libs/core/usecase/modelcall/modelcallcontract"
)

var errStreamEnded = errors.New("the stream of the provider ended before the turn did")

// Chat streams one turn from the provider and sends each event as it arrives. A failure the provider
// reports ends the turn as *TryNextError or *StopError, the error of send ends it as it is, and a stream
// that ends with no finish is an error of no failure kind.
func (p *provider) Chat(
	ctx context.Context, call modelcallport.ProviderCall, request modelcallport.ChatRequest, send func(modelcallport.ChatEvent) error,
) error {
	stream, err := p.client.Chat(ctx, connect.NewRequest(writeChatRequest(call, request)))
	if err != nil {
		return readCallError(ctx, err)
	}
	defer func() { _ = stream.Close() }()
	for stream.Receive() {
		switch event := stream.Msg().GetEvent().(type) {
		case *v1.ProviderServiceChatResponse_TextDelta:
			err = send(modelcallport.ChatEvent{TextDelta: event.TextDelta})
		case *v1.ProviderServiceChatResponse_ToolCall:
			err = send(modelcallport.ChatEvent{ToolCall: &modelcallport.ToolCall{
				ID: event.ToolCall.GetId(), Name: event.ToolCall.GetName(), Arguments: event.ToolCall.GetArgumentsJson(),
			}})
		case *v1.ProviderServiceChatResponse_Finish:
			reason := readFinishReason(event.Finish.GetReason())
			if err = send(modelcallport.ChatEvent{Finish: &reason}); err == nil {
				return nil
			}
		case *v1.ProviderServiceChatResponse_Failure:
			return readFailure(event.Failure)
		}
		if err != nil {
			return err
		}
	}
	if err := stream.Err(); err != nil {
		return readCallError(ctx, err)
	}
	return errStreamEnded
}

func writeChatRequest(call modelcallport.ProviderCall, request modelcallport.ChatRequest) *v1.ProviderServiceChatRequest {
	messages := make([]*v1.ProviderChatMessage, len(request.Messages))
	for i, one := range request.Messages {
		calls := make([]*v1.ProviderToolCall, len(one.ToolCalls))
		for j, tool := range one.ToolCalls {
			calls[j] = &v1.ProviderToolCall{Id: tool.ID, Name: tool.Name, ArgumentsJson: tool.Arguments}
		}
		messages[i] = &v1.ProviderChatMessage{Role: one.Role, Text: one.Text, ToolCalls: calls, ToolCallId: one.ToolCallID}
	}
	tools := make([]*v1.ProviderToolDefinition, len(request.Tools))
	for i, one := range request.Tools {
		tools[i] = &v1.ProviderToolDefinition{Name: one.Name, Description: one.Description, ParametersJson: one.Parameters}
	}
	return &v1.ProviderServiceChatRequest{
		Call: writeCall(call), Messages: messages, Tools: tools, Temperature: request.Temperature, MaxTokens: int32(request.MaxTokens),
	}
}

func readFinishReason(reason v1.ProviderChatFinishReason) modelcallport.ChatFinishReason {
	switch reason {
	case v1.ProviderChatFinishReason_PROVIDER_CHAT_FINISH_REASON_TOOL_CALLS:
		return modelcallport.ChatFinishToolCalls
	case v1.ProviderChatFinishReason_PROVIDER_CHAT_FINISH_REASON_LENGTH:
		return modelcallport.ChatFinishLength
	case v1.ProviderChatFinishReason_PROVIDER_CHAT_FINISH_REASON_STOP, v1.ProviderChatFinishReason_PROVIDER_CHAT_FINISH_REASON_UNSPECIFIED:
		return modelcallport.ChatFinishStop
	}
	return modelcallport.ChatFinishStop
}
