package hostendpoint

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
)

// modelEndpoint answers HostModelService for the plugins that call the host. The connections are the
// rows of the calling plugin, so the call needs no permission and names no connection.
type modelEndpoint struct {
	endpoint *Endpoint
}

var _ numenv1connect.HostModelServiceHandler = (*modelEndpoint)(nil)

func (e *modelEndpoint) begin(header string) (string, ModelCalls, error) {
	pluginID, err := e.endpoint.authenticate(header)
	if err != nil {
		return "", nil, err
	}
	return pluginID, e.endpoint.callers.Lent.GetModelCalls(), nil
}

// A host with no calls on models answers every call with connection.ErrNoConnection.
func (e *modelEndpoint) Complete(
	ctx context.Context, request *connect.Request[v1.HostModelServiceCompleteRequest],
) (*connect.Response[v1.HostModelServiceCompleteResponse], error) {
	pluginID, calls, err := e.begin(request.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}
	messages := make([]modelcall.Message, len(request.Msg.GetMessages()))
	for i, one := range request.Msg.GetMessages() {
		messages[i] = modelcall.Message{Role: one.GetRole(), Text: one.GetText()}
	}
	var result modelcall.CompleteResult
	err = connection.ErrNoConnection
	if calls != nil {
		result, err = calls.Complete(ctx, pluginID, modelcall.CompleteRequest{
			Messages: messages, Temperature: request.Msg.Temperature, MaxTokens: int(request.Msg.GetMaxTokens()),
		}, readPriority(request.Msg.GetPriority()))
	}
	if err != nil {
		code, detail, failures, served := writeModelError(ctx, err)
		if !served {
			return nil, readModelStatus(ctx, err)
		}
		return connect.NewResponse(&v1.HostModelServiceCompleteResponse{Error: code, Detail: detail, Failures: failures}), nil
	}
	return connect.NewResponse(&v1.HostModelServiceCompleteResponse{Text: result.Text}), nil
}

// Chat streams the events of one turn. A turn the core could not serve ends the stream with an event that
// holds the error code; any other failure is a status code.
func (e *modelEndpoint) Chat(
	ctx context.Context, request *connect.Request[v1.HostModelServiceChatRequest], stream *connect.ServerStream[v1.HostModelServiceChatResponse],
) error {
	pluginID, calls, err := e.begin(request.Header().Get("Authorization"))
	if err != nil {
		return err
	}
	err = connection.ErrNoConnection
	if calls != nil {
		err = calls.Chat(ctx, pluginID, readChatRequest(request.Msg), readPriority(request.Msg.GetPriority()), func(event modelcall.ChatEvent) error {
			return stream.Send(writeChatEvent(event))
		})
	}
	if err == nil {
		return nil
	}
	code, detail, failures, served := writeModelError(ctx, err)
	if !served {
		return readModelStatus(ctx, err)
	}
	return stream.Send(&v1.HostModelServiceChatResponse{Event: &v1.HostModelServiceChatResponse_Failure{
		Failure: &v1.HostModelChatError{Error: code, Detail: detail, Failures: failures},
	}})
}

func readChatRequest(request *v1.HostModelServiceChatRequest) modelcall.ChatRequest {
	messages := make([]modelcall.ChatMessage, len(request.GetMessages()))
	for i, one := range request.GetMessages() {
		calls := make([]modelcall.ToolCall, len(one.GetToolCalls()))
		for j, call := range one.GetToolCalls() {
			calls[j] = modelcall.ToolCall{ID: call.GetId(), Name: call.GetName(), Arguments: call.GetArgumentsJson()}
		}
		messages[i] = modelcall.ChatMessage{Role: one.GetRole(), Text: one.GetText(), ToolCalls: calls, ToolCallID: one.GetToolCallId()}
	}
	tools := make([]modelcall.ToolDefinition, len(request.GetTools()))
	for i, one := range request.GetTools() {
		tools[i] = modelcall.ToolDefinition{Name: one.GetName(), Description: one.GetDescription(), Parameters: one.GetParametersJson()}
	}
	return modelcall.ChatRequest{Messages: messages, Tools: tools, Temperature: request.Temperature, MaxTokens: int(request.GetMaxTokens())}
}

func writeChatEvent(event modelcall.ChatEvent) *v1.HostModelServiceChatResponse {
	switch {
	case event.ToolCall != nil:
		return &v1.HostModelServiceChatResponse{Event: &v1.HostModelServiceChatResponse_ToolCall{ToolCall: &v1.HostModelToolCall{
			Id: event.ToolCall.ID, Name: event.ToolCall.Name, ArgumentsJson: event.ToolCall.Arguments,
		}}}
	case event.Finish != nil:
		return &v1.HostModelServiceChatResponse{Event: &v1.HostModelServiceChatResponse_Finish{
			Finish: &v1.HostModelChatFinish{Reason: writeChatFinishReason(*event.Finish)},
		}}
	}
	return &v1.HostModelServiceChatResponse{Event: &v1.HostModelServiceChatResponse_TextDelta{TextDelta: event.TextDelta}}
}

func writeChatFinishReason(reason modelcall.ChatFinishReason) v1.HostModelChatFinishReason {
	switch reason {
	case modelcall.ChatFinishToolCalls:
		return v1.HostModelChatFinishReason_HOST_MODEL_CHAT_FINISH_REASON_TOOL_CALLS
	case modelcall.ChatFinishLength:
		return v1.HostModelChatFinishReason_HOST_MODEL_CHAT_FINISH_REASON_LENGTH
	case modelcall.ChatFinishStop:
		return v1.HostModelChatFinishReason_HOST_MODEL_CHAT_FINISH_REASON_STOP
	}
	return v1.HostModelChatFinishReason_HOST_MODEL_CHAT_FINISH_REASON_STOP
}

func (e *modelEndpoint) Embed(
	ctx context.Context, request *connect.Request[v1.HostModelServiceEmbedRequest],
) (*connect.Response[v1.HostModelServiceEmbedResponse], error) {
	pluginID, calls, err := e.begin(request.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}
	var vectors [][]float32
	err = connection.ErrNoConnection
	if calls != nil {
		vectors, err = calls.Embed(ctx, pluginID, request.Msg.GetTexts(), readPriority(request.Msg.GetPriority()))
	}
	if err != nil {
		code, detail, failures, served := writeModelError(ctx, err)
		if !served {
			return nil, readModelStatus(ctx, err)
		}
		return connect.NewResponse(&v1.HostModelServiceEmbedResponse{Error: code, Detail: detail, Failures: failures}), nil
	}
	wire := make([]*v1.FloatVector, len(vectors))
	for i, one := range vectors {
		wire[i] = &v1.FloatVector{Values: one}
	}
	return connect.NewResponse(&v1.HostModelServiceEmbedResponse{Vectors: wire}), nil
}

func (e *modelEndpoint) GetEmbeddingProfile(
	ctx context.Context, request *connect.Request[v1.HostModelServiceGetEmbeddingProfileRequest],
) (*connect.Response[v1.HostModelServiceGetEmbeddingProfileResponse], error) {
	pluginID, calls, err := e.begin(request.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}
	var profile modelcall.EmbeddingProfile
	err = connection.ErrNoConnection
	if calls != nil {
		profile, err = calls.GetEmbeddingProfile(ctx, pluginID)
	}
	if err != nil {
		code, detail, failures, served := writeModelError(ctx, err)
		if !served {
			return nil, readModelStatus(ctx, err)
		}
		return connect.NewResponse(&v1.HostModelServiceGetEmbeddingProfileResponse{Error: code, Detail: detail, Failures: failures}), nil
	}
	return connect.NewResponse(&v1.HostModelServiceGetEmbeddingProfileResponse{
		Model:          profile.Model,
		Source:         profile.Source,
		Pooling:        profile.Pooling,
		QueryPrefix:    profile.QueryPrefix,
		DocumentPrefix: profile.DocumentPrefix,
		Dimensions:     int32(profile.Dimensions),
		MaxTokens:      int32(profile.MaxTokens),
	}), nil
}

func writeModelError(ctx context.Context, err error) (v1.HostModelErrorCode, string, []*v1.HostModelFailure, bool) {
	if ctx.Err() != nil {
		return v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_UNSPECIFIED, "", nil, false
	}
	code, isServed := getModelErrorCode(err)
	if !isServed {
		return code, "", nil, false
	}
	var failures []*v1.HostModelFailure
	var allFailed *connection.AllFailedError
	var rejected *connection.RejectedError
	switch {
	case errors.As(err, &allFailed):
		for _, one := range allFailed.Failures {
			failures = append(failures, &v1.HostModelFailure{ConnectionName: one.ConnectionName, Reason: one.Reason})
		}
		return code, allFailed.Error(), failures, true
	case errors.As(err, &rejected):
		failures = append(failures, &v1.HostModelFailure{ConnectionName: rejected.ConnectionName, Reason: rejected.Reason})
		return code, rejected.Error(), failures, true
	}
	return code, "", nil, true
}

func getModelErrorCode(err error) (v1.HostModelErrorCode, bool) {
	var allFailed *connection.AllFailedError
	var rejected *connection.RejectedError
	switch {
	case errors.Is(err, connection.ErrNoConnection):
		return v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_NO_CONNECTION, true
	case errors.As(err, &allFailed):
		return v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_ALL_FAILED, true
	case errors.As(err, &rejected):
		return v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_REJECTED, true
	}
	return v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_UNSPECIFIED, false
}

func readModelStatus(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return connect.NewError(connect.CodeCanceled, ctxErr)
	}
	return connect.NewError(connect.CodeInternal, err)
}

func readPriority(priority v1.Priority) modelcall.Priority {
	if priority == v1.Priority_PRIORITY_INTERACTIVE {
		return modelcall.PriorityInteractive
	}
	return modelcall.PriorityBackground
}
