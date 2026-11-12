package model

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

var errPluginToken = errors.New("ModelService is not served to a plugin")

func refuseToken(header http.Header) error {
	// A window sends no token; a plugin process sends the one the host issued it, and these calls name the
	// plugin they act for in the request.
	if header.Get("Authorization") != "" {
		return connect.NewError(connect.CodePermissionDenied, errPluginToken)
	}
	return nil
}

// Complete turns messages into the text of an answer through the rows of the plugin the request names.
func (h *Handler) Complete(
	ctx context.Context, request *connect.Request[v1.ModelServiceCompleteRequest],
) (*connect.Response[v1.ModelServiceCompleteResponse], error) {
	if err := refuseToken(request.Header()); err != nil {
		return nil, err
	}
	messages := make([]modelcall.Message, len(request.Msg.GetMessages()))
	for i, one := range request.Msg.GetMessages() {
		messages[i] = modelcall.Message{Role: one.GetRole(), Text: one.GetText()}
	}
	result, err := h.calls.Complete(ctx, request.Msg.GetPluginId(), modelcall.CompleteRequest{
		Messages: messages, Temperature: request.Msg.Temperature, MaxTokens: int(request.Msg.GetMaxTokens()),
	}, readPriority(request.Msg.GetPriority()))
	if err != nil {
		code, detail, failures, isServed := writeCallError(ctx, err)
		if !isServed {
			return nil, newStatusError(ctx, err)
		}
		return connect.NewResponse(&v1.ModelServiceCompleteResponse{Error: code, Detail: detail, Failures: failures}), nil
	}
	return connect.NewResponse(&v1.ModelServiceCompleteResponse{Text: result.Text}), nil
}

// Chat streams the next turn of a conversation through the rows of the plugin the request names. A turn the
// core could not serve ends the stream with an event that holds the error code.
func (h *Handler) Chat(
	ctx context.Context, request *connect.Request[v1.ModelServiceChatRequest], stream *connect.ServerStream[v1.ModelServiceChatResponse],
) error {
	if err := refuseToken(request.Header()); err != nil {
		return err
	}
	messages := make([]modelcall.ChatMessage, len(request.Msg.GetMessages()))
	for i, one := range request.Msg.GetMessages() {
		calls := make([]modelcall.ToolCall, len(one.GetToolCalls()))
		for j, call := range one.GetToolCalls() {
			calls[j] = modelcall.ToolCall{ID: call.GetId(), Name: call.GetName(), Arguments: call.GetArgumentsJson()}
		}
		messages[i] = modelcall.ChatMessage{Role: one.GetRole(), Text: one.GetText(), ToolCalls: calls, ToolCallID: one.GetToolCallId()}
	}
	tools := make([]modelcall.ToolDefinition, len(request.Msg.GetTools()))
	for i, one := range request.Msg.GetTools() {
		tools[i] = modelcall.ToolDefinition{Name: one.GetName(), Description: one.GetDescription(), Parameters: one.GetParametersJson()}
	}
	err := h.calls.Chat(ctx, request.Msg.GetPluginId(), modelcall.ChatRequest{
		Messages: messages, Tools: tools, Temperature: request.Msg.Temperature, MaxTokens: int(request.Msg.GetMaxTokens()),
	}, readPriority(request.Msg.GetPriority()), func(event modelcall.ChatEvent) error {
		return stream.Send(writeChatEvent(event))
	})
	if err == nil {
		return nil
	}
	code, detail, failures, isServed := writeCallError(ctx, err)
	if !isServed {
		return newStatusError(ctx, err)
	}
	return stream.Send(&v1.ModelServiceChatResponse{Event: &v1.ModelServiceChatResponse_Failure{
		Failure: &v1.ModelChatError{Error: code, Detail: detail, Failures: failures},
	}})
}

func writeChatEvent(event modelcall.ChatEvent) *v1.ModelServiceChatResponse {
	switch {
	case event.ToolCall != nil:
		return &v1.ModelServiceChatResponse{Event: &v1.ModelServiceChatResponse_ToolCall{ToolCall: &v1.HostModelToolCall{
			Id: event.ToolCall.ID, Name: event.ToolCall.Name, ArgumentsJson: event.ToolCall.Arguments,
		}}}
	case event.Finish != nil:
		return &v1.ModelServiceChatResponse{Event: &v1.ModelServiceChatResponse_Finish{
			Finish: &v1.HostModelChatFinish{Reason: writeChatFinishReason(*event.Finish)},
		}}
	}
	return &v1.ModelServiceChatResponse{Event: &v1.ModelServiceChatResponse_TextDelta{TextDelta: event.TextDelta}}
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

// Embed turns texts into vectors through the rows of the plugin the request names.
func (h *Handler) Embed(
	ctx context.Context, request *connect.Request[v1.ModelServiceEmbedRequest],
) (*connect.Response[v1.ModelServiceEmbedResponse], error) {
	if err := refuseToken(request.Header()); err != nil {
		return nil, err
	}
	vectors, err := h.calls.Embed(ctx, request.Msg.GetPluginId(), request.Msg.GetTexts(), readPriority(request.Msg.GetPriority()))
	if err != nil {
		code, detail, failures, isServed := writeCallError(ctx, err)
		if !isServed {
			return nil, newStatusError(ctx, err)
		}
		return connect.NewResponse(&v1.ModelServiceEmbedResponse{Error: code, Detail: detail, Failures: failures}), nil
	}
	wire := make([]*v1.FloatVector, len(vectors))
	for i, one := range vectors {
		wire[i] = &v1.FloatVector{Values: one}
	}
	return connect.NewResponse(&v1.ModelServiceEmbedResponse{Vectors: wire}), nil
}

// GetEmbeddingProfile returns the profile of the embedding model of the first row of the plugin that
// answers.
func (h *Handler) GetEmbeddingProfile(
	ctx context.Context, request *connect.Request[v1.ModelServiceGetEmbeddingProfileRequest],
) (*connect.Response[v1.ModelServiceGetEmbeddingProfileResponse], error) {
	if err := refuseToken(request.Header()); err != nil {
		return nil, err
	}
	profile, err := h.calls.GetEmbeddingProfile(ctx, request.Msg.GetPluginId())
	if err != nil {
		code, detail, failures, isServed := writeCallError(ctx, err)
		if !isServed {
			return nil, newStatusError(ctx, err)
		}
		return connect.NewResponse(&v1.ModelServiceGetEmbeddingProfileResponse{Error: code, Detail: detail, Failures: failures}), nil
	}
	return connect.NewResponse(&v1.ModelServiceGetEmbeddingProfileResponse{
		Model:          profile.Model,
		Source:         profile.Source,
		Pooling:        profile.Pooling,
		QueryPrefix:    profile.QueryPrefix,
		DocumentPrefix: profile.DocumentPrefix,
		Dimensions:     int32(profile.Dimensions),
		MaxTokens:      int32(profile.MaxTokens),
	}), nil
}

func readPriority(priority v1.Priority) modelcall.Priority {
	if priority == v1.Priority_PRIORITY_INTERACTIVE {
		return modelcall.PriorityInteractive
	}
	return modelcall.PriorityBackground
}

func writeCallError(ctx context.Context, err error) (v1.ModelErrorCode, string, []*v1.ModelFailure, bool) {
	if ctx.Err() != nil {
		return v1.ModelErrorCode_MODEL_ERROR_CODE_UNSPECIFIED, "", nil, false
	}
	code, isServed := getErrorCode(err)
	if !isServed {
		return code, "", nil, false
	}
	var allFailed *connection.AllFailedError
	var rejected *connection.RejectedError
	switch {
	case errors.As(err, &allFailed):
		failures := make([]*v1.ModelFailure, len(allFailed.Failures))
		for i, one := range allFailed.Failures {
			failures[i] = &v1.ModelFailure{ConnectionName: one.ConnectionName, Reason: one.Reason}
		}
		return code, allFailed.Error(), failures, true
	case errors.As(err, &rejected):
		return code, rejected.Error(), []*v1.ModelFailure{{ConnectionName: rejected.ConnectionName, Reason: rejected.Reason}}, true
	}
	return code, "", nil, true
}
