package connection

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

type adapter struct {
	numenv1connect.UnimplementedProviderServiceHandler
	provider Provider
}

func (a *adapter) Complete(
	ctx context.Context, request *connect.Request[v1.ProviderCompleteRequest],
) (*connect.Response[v1.ProviderCompleteResponse], error) {
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
		return failed(ctx, err, func(failure *v1.ProviderFailure) *v1.ProviderCompleteResponse {
			return &v1.ProviderCompleteResponse{Failure: failure}
		})
	}
	return connect.NewResponse(&v1.ProviderCompleteResponse{Text: result.Text}), nil
}

func (a *adapter) Embed(
	ctx context.Context, request *connect.Request[v1.ProviderEmbedRequest],
) (*connect.Response[v1.ProviderEmbedResponse], error) {
	if a.provider.Embedder == nil {
		return a.UnimplementedProviderServiceHandler.Embed(ctx, request)
	}
	vectors, err := a.provider.Embedder.Embed(ctx, readCall(request.Msg.GetCall()), request.Msg.GetTexts())
	if err != nil {
		return failed(ctx, err, func(failure *v1.ProviderFailure) *v1.ProviderEmbedResponse {
			return &v1.ProviderEmbedResponse{Failure: failure}
		})
	}
	wire := make([]*v1.FloatVector, len(vectors))
	for i, one := range vectors {
		wire[i] = &v1.FloatVector{Values: one}
	}
	return connect.NewResponse(&v1.ProviderEmbedResponse{Vectors: wire}), nil
}

func (a *adapter) ListModels(
	ctx context.Context, request *connect.Request[v1.ProviderListModelsRequest],
) (*connect.Response[v1.ProviderListModelsResponse], error) {
	if a.provider.Models == nil {
		return a.UnimplementedProviderServiceHandler.ListModels(ctx, request)
	}
	models, err := a.provider.Models.ListModels(ctx, readCall(request.Msg.GetCall()), request.Msg.GetOperation())
	if err != nil {
		return failed(ctx, err, func(failure *v1.ProviderFailure) *v1.ProviderListModelsResponse {
			return &v1.ProviderListModelsResponse{Failure: failure}
		})
	}
	wire := make([]*v1.ProviderModel, len(models))
	for i, one := range models {
		wire[i] = &v1.ProviderModel{Id: one.ID, Title: one.Title, Operations: one.Operations}
	}
	return connect.NewResponse(&v1.ProviderListModelsResponse{Models: wire}), nil
}

func (a *adapter) ListReasoningLevels(
	ctx context.Context, request *connect.Request[v1.ProviderListReasoningLevelsRequest],
) (*connect.Response[v1.ProviderListReasoningLevelsResponse], error) {
	if a.provider.Models == nil {
		return a.UnimplementedProviderServiceHandler.ListReasoningLevels(ctx, request)
	}
	levels, err := a.provider.Models.ListReasoningLevels(ctx, readCall(request.Msg.GetCall()))
	if err != nil {
		return failed(ctx, err, func(failure *v1.ProviderFailure) *v1.ProviderListReasoningLevelsResponse {
			return &v1.ProviderListReasoningLevelsResponse{Failure: failure}
		})
	}
	wire := make([]*v1.ProviderReasoningLevel, len(levels))
	for i, one := range levels {
		wire[i] = &v1.ProviderReasoningLevel{Value: one.Value, Title: one.Title}
	}
	return connect.NewResponse(&v1.ProviderListReasoningLevelsResponse{Levels: wire}), nil
}

func (a *adapter) GetModelProfile(
	ctx context.Context, request *connect.Request[v1.ProviderGetModelProfileRequest],
) (*connect.Response[v1.ProviderGetModelProfileResponse], error) {
	if a.provider.Profiles == nil {
		return a.UnimplementedProviderServiceHandler.GetModelProfile(ctx, request)
	}
	profile, err := a.provider.Profiles.GetModelProfile(ctx, readCall(request.Msg.GetCall()))
	if err != nil {
		return failed(ctx, err, func(failure *v1.ProviderFailure) *v1.ProviderGetModelProfileResponse {
			return &v1.ProviderGetModelProfileResponse{Failure: failure}
		})
	}
	return connect.NewResponse(&v1.ProviderGetModelProfileResponse{
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
	ctx context.Context, request *connect.Request[v1.ProviderProbeEmbeddingWidthRequest],
) (*connect.Response[v1.ProviderProbeEmbeddingWidthResponse], error) {
	if a.provider.Profiles == nil {
		return a.UnimplementedProviderServiceHandler.ProbeEmbeddingWidth(ctx, request)
	}
	width, err := a.provider.Profiles.ProbeEmbeddingWidth(ctx, readCall(request.Msg.GetCall()))
	if err != nil {
		return failed(ctx, err, func(failure *v1.ProviderFailure) *v1.ProviderProbeEmbeddingWidthResponse {
			return &v1.ProviderProbeEmbeddingWidthResponse{Failure: failure}
		})
	}
	return connect.NewResponse(&v1.ProviderProbeEmbeddingWidthResponse{Dimensions: int32(width)}), nil
}

// failed answers a call that returned an error: a cancelled call by its status code, any other by a
// served failure. The text of an error that is no failure kind never leaves the provider.
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
