package providerbridge

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
)

var errStreamEnded = errors.New("the stream of the provider ended before the turn did")

// Chat streams one turn from the provider and sends each event as it arrives. A failure the provider
// reports ends the turn as *TryNextError or *StopError, the error of send ends it as it is, and a stream
// that ends with no finish is an error of no failure kind.
func (p *provider) Chat(
	ctx context.Context, call modelcall.ProviderCall, request modelcall.ChatRequest, send func(modelcall.ChatEvent) error,
) (err error) {
	stream, err := p.client.Chat(ctx, connect.NewRequest(writeChatRequest(call, request)))
	if err != nil {
		return readCallError(ctx, err)
	}
	defer func() { err = errors.Join(err, stream.Close()) }()
	for stream.Receive() {
		switch event := stream.Msg().GetEvent().(type) {
		case *v1.ProviderServiceChatResponse_TextDelta:
			err = send(modelcall.ChatEvent{TextDelta: event.TextDelta})
		case *v1.ProviderServiceChatResponse_ToolCall:
			err = send(modelcall.ChatEvent{ToolCall: &modelcall.ToolCall{
				ID: event.ToolCall.GetId(), Name: event.ToolCall.GetName(), Arguments: event.ToolCall.GetArgumentsJson(),
			}})
		case *v1.ProviderServiceChatResponse_Finish:
			reason := readFinishReason(event.Finish.GetReason())
			if err = send(modelcall.ChatEvent{Finish: &reason}); err == nil {
				return nil
			}
		case *v1.ProviderServiceChatResponse_Failure:
			return readFailure(event.Failure)
		}
		if err != nil {
			return err
		}
	}
	if streamErr := stream.Err(); streamErr != nil {
		return readCallError(ctx, streamErr)
	}
	return errStreamEnded
}

func writeChatRequest(call modelcall.ProviderCall, request modelcall.ChatRequest) *v1.ProviderServiceChatRequest {
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

func readFinishReason(reason v1.ProviderChatFinishReason) modelcall.ChatFinishReason {
	switch reason {
	case v1.ProviderChatFinishReason_PROVIDER_CHAT_FINISH_REASON_TOOL_CALLS:
		return modelcall.ChatFinishToolCalls
	case v1.ProviderChatFinishReason_PROVIDER_CHAT_FINISH_REASON_LENGTH:
		return modelcall.ChatFinishLength
	case v1.ProviderChatFinishReason_PROVIDER_CHAT_FINISH_REASON_STOP, v1.ProviderChatFinishReason_PROVIDER_CHAT_FINISH_REASON_UNSPECIFIED:
		return modelcall.ChatFinishStop
	}
	return modelcall.ChatFinishStop
}
