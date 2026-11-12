package connection

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

// adapter serves the roles of a Provider. A canceled call answers with the status code Canceled, and any
// other error with a served failure; the text of an error that is no failure kind never leaves the provider.
type adapter struct {
	numenv1connect.UnimplementedProviderServiceHandler
	provider Provider
}

func (a *adapter) Complete(
	ctx context.Context, request *connect.Request[v1.ProviderServiceCompleteRequest],
) (*connect.Response[v1.ProviderServiceCompleteResponse], error) {
	if a.provider.Completer == nil {
		return a.UnimplementedProviderServiceHandler.Complete(ctx, request)
	}
	messages := make([]Message, len(request.Msg.GetMessages()))
	for i, one := range request.Msg.GetMessages() {
		messages[i] = Message{Role: one.GetRole(), Text: one.GetText()}
	}
	var temperature *float64
	if request.Msg.Temperature != nil {
		value := request.Msg.GetTemperature()
		temperature = &value
	}
	result, err := a.provider.Completer.Complete(ctx, readCall(request.Msg.GetCall()), ProviderCompleteRequest{
		Messages: messages, Temperature: temperature, MaxTokens: int(request.Msg.GetMaxTokens()),
	})
	if err != nil {
		return failed(ctx, err, func(failure *v1.ProviderFailure) *v1.ProviderServiceCompleteResponse {
			return &v1.ProviderServiceCompleteResponse{Failure: failure}
		})
	}
	return connect.NewResponse(&v1.ProviderServiceCompleteResponse{Text: result.Text}), nil
}

func (a *adapter) Embed(
	ctx context.Context, request *connect.Request[v1.ProviderServiceEmbedRequest],
) (*connect.Response[v1.ProviderServiceEmbedResponse], error) {
	if a.provider.Embedder == nil {
		return a.UnimplementedProviderServiceHandler.Embed(ctx, request)
	}
	vectors, err := a.provider.Embedder.Embed(ctx, readCall(request.Msg.GetCall()), request.Msg.GetTexts())
	if err != nil {
		return failed(ctx, err, func(failure *v1.ProviderFailure) *v1.ProviderServiceEmbedResponse {
			return &v1.ProviderServiceEmbedResponse{Failure: failure}
		})
	}
	wire := make([]*v1.FloatVector, len(vectors))
	for i, one := range vectors {
		wire[i] = &v1.FloatVector{Values: one}
	}
	return connect.NewResponse(&v1.ProviderServiceEmbedResponse{Vectors: wire}), nil
}

func (a *adapter) ListModels(
	ctx context.Context, request *connect.Request[v1.ProviderServiceListModelsRequest],
) (*connect.Response[v1.ProviderServiceListModelsResponse], error) {
	if a.provider.Models == nil {
		return a.UnimplementedProviderServiceHandler.ListModels(ctx, request)
	}
	models, err := a.provider.Models.ListModels(ctx, readCall(request.Msg.GetCall()), request.Msg.GetOperation())
	if err != nil {
		return failed(ctx, err, func(failure *v1.ProviderFailure) *v1.ProviderServiceListModelsResponse {
			return &v1.ProviderServiceListModelsResponse{Failure: failure}
		})
	}
	wire := make([]*v1.ProviderModel, len(models))
	for i, one := range models {
		wire[i] = &v1.ProviderModel{Id: one.ID, Title: one.Title, Operations: one.Operations}
	}
	return connect.NewResponse(&v1.ProviderServiceListModelsResponse{Models: wire}), nil
}

func (a *adapter) ListReasoningLevels(
	ctx context.Context, request *connect.Request[v1.ProviderServiceListReasoningLevelsRequest],
) (*connect.Response[v1.ProviderServiceListReasoningLevelsResponse], error) {
	if a.provider.Models == nil {
		return a.UnimplementedProviderServiceHandler.ListReasoningLevels(ctx, request)
	}
	levels, err := a.provider.Models.ListReasoningLevels(ctx, readCall(request.Msg.GetCall()))
	if err != nil {
		return failed(ctx, err, func(failure *v1.ProviderFailure) *v1.ProviderServiceListReasoningLevelsResponse {
			return &v1.ProviderServiceListReasoningLevelsResponse{Failure: failure}
		})
	}
	wire := make([]*v1.ProviderReasoningLevel, len(levels))
	for i, one := range levels {
		wire[i] = &v1.ProviderReasoningLevel{Value: one.Value, Title: one.Title}
	}
	return connect.NewResponse(&v1.ProviderServiceListReasoningLevelsResponse{Levels: wire}), nil
}

func (a *adapter) GetModelProfile(
	ctx context.Context, request *connect.Request[v1.ProviderServiceGetModelProfileRequest],
) (*connect.Response[v1.ProviderServiceGetModelProfileResponse], error) {
	if a.provider.Profiles == nil {
		return a.UnimplementedProviderServiceHandler.GetModelProfile(ctx, request)
	}
	profile, err := a.provider.Profiles.GetModelProfile(ctx, readCall(request.Msg.GetCall()))
	if err != nil {
		return failed(ctx, err, func(failure *v1.ProviderFailure) *v1.ProviderServiceGetModelProfileResponse {
			return &v1.ProviderServiceGetModelProfileResponse{Failure: failure}
		})
	}
	return connect.NewResponse(&v1.ProviderServiceGetModelProfileResponse{
		IsKnown:        profile.IsKnown,
		Dimensions:     int32(profile.Dimensions),
		MaxTokens:      int32(profile.MaxTokens),
		Pooling:        profile.Pooling,
		QueryPrefix:    profile.QueryPrefix,
		DocumentPrefix: profile.DocumentPrefix,
		Source:         profile.Source,
	}), nil
}

func (a *adapter) ProbeEmbeddingWidth(
	ctx context.Context, request *connect.Request[v1.ProviderServiceProbeEmbeddingWidthRequest],
) (*connect.Response[v1.ProviderServiceProbeEmbeddingWidthResponse], error) {
	if a.provider.Profiles == nil {
		return a.UnimplementedProviderServiceHandler.ProbeEmbeddingWidth(ctx, request)
	}
	width, err := a.provider.Profiles.ProbeEmbeddingWidth(ctx, readCall(request.Msg.GetCall()))
	if err != nil {
		return failed(ctx, err, func(failure *v1.ProviderFailure) *v1.ProviderServiceProbeEmbeddingWidthResponse {
			return &v1.ProviderServiceProbeEmbeddingWidthResponse{Failure: failure}
		})
	}
	return connect.NewResponse(&v1.ProviderServiceProbeEmbeddingWidthResponse{Dimensions: int32(width)}), nil
}

func failed[R any](ctx context.Context, err error, respond func(*v1.ProviderFailure) *R) (*connect.Response[R], error) {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, connect.NewError(connect.CodeCanceled, ctxErr)
	}
	if errors.Is(err, context.Canceled) {
		return nil, connect.NewError(connect.CodeCanceled, err)
	}
	return connect.NewResponse(respond(writeFailure(err))), nil
}

func readCall(wire *v1.ProviderCall) Call {
	call := Call{
		Type:      wire.GetType(),
		Fields:    wire.GetFields(),
		Secrets:   make(map[string]Secret, len(wire.GetSecrets())),
		Model:     wire.GetModel(),
		Reasoning: wire.GetReasoning(),
	}
	for _, one := range wire.GetSecrets() {
		call.Secrets[one.GetKey()] = Secret{Value: one.GetValue()}
	}
	return call
}

func writeFailure(err error) *v1.ProviderFailure {
	reason, _ := GetReason(err)
	retryAfter, isTryNext := IsTryNext(err)
	if !isTryNext {
		return &v1.ProviderFailure{Kind: v1.ProviderFailureKind_PROVIDER_FAILURE_KIND_STOP, Reason: writeReason(reason)}
	}
	return &v1.ProviderFailure{
		Kind:         v1.ProviderFailureKind_PROVIDER_FAILURE_KIND_TRY_NEXT,
		Reason:       writeReason(reason),
		RetryAfterMs: retryAfter.Round(time.Millisecond).Milliseconds(),
	}
}

func writeReason(reason Reason) v1.ProviderFailureReason {
	switch reason {
	case ReasonKeyRejected:
		return v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_KEY_REJECTED
	case ReasonPaymentRequired:
		return v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_PAYMENT_REQUIRED
	case ReasonRateLimited:
		return v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_RATE_LIMITED
	case ReasonUnreachable:
		return v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_UNREACHABLE
	case ReasonTimeout:
		return v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_TIMEOUT
	case ReasonBadRequest:
		return v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_BAD_REQUEST
	case ReasonUnknown:
		return v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_UNKNOWN
	}
	return v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_UNKNOWN
}
