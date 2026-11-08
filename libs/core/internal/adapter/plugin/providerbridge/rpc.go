package providerbridge

import (
	"context"
	"sync"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/bridge"
	modelcallport "github.com/jiva-studio/numen/libs/core/usecase/modelcall/modelcallcontract"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

// Source opens the provider of a plugin over its ProviderService.
type Source struct {
	host pluginport.PluginActivator
}

// NewSource returns the source over the plugins the host activates.
func NewSource(host pluginport.PluginActivator) *Source { return &Source{host: host} }

// OpenProvider activates the plugin, which starts it if it is not running, and returns its provider with
// the function that releases the plugin. The function may be called more than once.
func (s *Source) OpenProvider(ctx context.Context, pluginID string) (modelcallport.Provider, func(), error) {
	client, lease, err := bridge.Open(ctx, s.host, pluginID, numenv1connect.NewProviderServiceClient)
	if err != nil {
		return nil, nil, err
	}
	var once sync.Once
	return &provider{client: client}, func() { once.Do(func() { _ = lease.Close() }) }, nil
}

type provider struct {
	client numenv1connect.ProviderServiceClient
}

func (p *provider) Complete(
	ctx context.Context, call modelcallport.ProviderCall, request modelcallport.CompleteRequest,
) (modelcallport.CompleteResult, error) {
	messages := make([]*v1.ProviderMessage, len(request.Messages))
	for i, one := range request.Messages {
		messages[i] = &v1.ProviderMessage{Role: one.Role, Text: one.Text}
	}
	response, err := p.client.Complete(ctx, connect.NewRequest(&v1.ProviderCompleteRequest{
		Call: writeCall(call), Messages: messages, Temperature: request.Temperature, MaxTokens: int32(request.MaxTokens),
	}))
	if err != nil {
		return modelcallport.CompleteResult{}, readCallError(ctx, err)
	}
	if response.Msg.GetFailure() != nil {
		return modelcallport.CompleteResult{}, readFailure(response.Msg.GetFailure())
	}
	return modelcallport.CompleteResult{Text: response.Msg.GetText()}, nil
}

func (p *provider) Embed(ctx context.Context, call modelcallport.ProviderCall, texts []string) ([][]float32, error) {
	response, err := p.client.Embed(ctx, connect.NewRequest(&v1.ProviderEmbedRequest{Call: writeCall(call), Texts: texts}))
	if err != nil {
		return nil, readCallError(ctx, err)
	}
	if response.Msg.GetFailure() != nil {
		return nil, readFailure(response.Msg.GetFailure())
	}
	vectors := make([][]float32, len(response.Msg.GetVectors()))
	for i, one := range response.Msg.GetVectors() {
		vectors[i] = one.GetValues()
	}
	return vectors, nil
}

func (p *provider) ListModels(ctx context.Context, call modelcallport.ProviderCall, operation string) ([]modelcallport.Model, error) {
	response, err := p.client.ListModels(ctx, connect.NewRequest(&v1.ProviderListModelsRequest{Call: writeCall(call), Operation: operation}))
	if err != nil {
		return nil, readCallError(ctx, err)
	}
	if response.Msg.GetFailure() != nil {
		return nil, readFailure(response.Msg.GetFailure())
	}
	models := make([]modelcallport.Model, len(response.Msg.GetModels()))
	for i, one := range response.Msg.GetModels() {
		models[i] = modelcallport.Model{ID: one.GetId(), Title: one.GetTitle(), Operations: one.GetOperations()}
	}
	return models, nil
}

func (p *provider) ListReasoningLevels(ctx context.Context, call modelcallport.ProviderCall) ([]modelcallport.ReasoningLevel, error) {
	response, err := p.client.ListReasoningLevels(ctx, connect.NewRequest(&v1.ProviderListReasoningLevelsRequest{Call: writeCall(call)}))
	if err != nil {
		return nil, readCallError(ctx, err)
	}
	if response.Msg.GetFailure() != nil {
		return nil, readFailure(response.Msg.GetFailure())
	}
	levels := make([]modelcallport.ReasoningLevel, len(response.Msg.GetLevels()))
	for i, one := range response.Msg.GetLevels() {
		levels[i] = modelcallport.ReasoningLevel{Value: one.GetValue(), Title: one.GetTitle()}
	}
	return levels, nil
}

func (p *provider) GetModelProfile(ctx context.Context, call modelcallport.ProviderCall) (modelcallport.ModelProfile, error) {
	response, err := p.client.GetModelProfile(ctx, connect.NewRequest(&v1.ProviderGetModelProfileRequest{Call: writeCall(call)}))
	if err != nil {
		return modelcallport.ModelProfile{}, readCallError(ctx, err)
	}
	if response.Msg.GetFailure() != nil {
		return modelcallport.ModelProfile{}, readFailure(response.Msg.GetFailure())
	}
	return modelcallport.ModelProfile{
		IsKnown:        response.Msg.GetIsKnown(),
		Dimensions:     int(response.Msg.GetDimensions()),
		MaxTokens:      int(response.Msg.GetMaxTokens()),
		Pooling:        response.Msg.GetPooling(),
		QueryPrefix:    response.Msg.GetQueryPrefix(),
		DocumentPrefix: response.Msg.GetDocumentPrefix(),
		Source:         response.Msg.GetSource(),
	}, nil
}

func (p *provider) ProbeEmbeddingWidth(ctx context.Context, call modelcallport.ProviderCall) (int, error) {
	response, err := p.client.ProbeEmbeddingWidth(ctx, connect.NewRequest(&v1.ProviderProbeEmbeddingWidthRequest{Call: writeCall(call)}))
	if err != nil {
		return 0, readCallError(ctx, err)
	}
	if response.Msg.GetFailure() != nil {
		return 0, readFailure(response.Msg.GetFailure())
	}
	return int(response.Msg.GetDimensions()), nil
}

func writeCall(call modelcallport.ProviderCall) *v1.ProviderCall {
	secrets := make([]*v1.ProviderSecret, 0, len(call.Secrets))
	for key, value := range call.Secrets {
		secrets = append(secrets, &v1.ProviderSecret{Key: key, Value: value})
	}
	return &v1.ProviderCall{Type: call.Type, Fields: call.Fields, Secrets: secrets, Model: call.Model, Reasoning: call.Reasoning}
}

// readCallError returns the end of the context as it is, a role the plugin does not serve as a stop of
// unknown reason, and any other failure of the call as it is.
func readCallError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if connect.CodeOf(err) == connect.CodeUnimplemented {
		return &modelcallport.StopError{Reason: modelcallport.FailureReasonUnknown}
	}
	return err
}

func readFailure(failure *v1.ProviderFailure) error {
	reason := readReason(failure.GetReason())
	if failure.GetKind() == v1.ProviderFailureKind_PROVIDER_FAILURE_KIND_TRY_NEXT {
		return &modelcallport.TryNextError{Reason: reason, RetryAfter: time.Duration(max(failure.GetRetryAfterMs(), 0)) * time.Millisecond}
	}
	return &modelcallport.StopError{Reason: reason}
}

func readReason(reason v1.ProviderFailureReason) modelcallport.FailureReason {
	switch reason {
	case v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_KEY_REJECTED:
		return modelcallport.FailureReasonKeyRejected
	case v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_PAYMENT_REQUIRED:
		return modelcallport.FailureReasonPaymentRequired
	case v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_RATE_LIMITED:
		return modelcallport.FailureReasonRateLimited
	case v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_UNREACHABLE:
		return modelcallport.FailureReasonUnreachable
	case v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_TIMEOUT:
		return modelcallport.FailureReasonTimeout
	case v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_BAD_REQUEST:
		return modelcallport.FailureReasonBadRequest
	case v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_UNKNOWN, v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_UNSPECIFIED:
		return modelcallport.FailureReasonUnknown
	}
	return modelcallport.FailureReasonUnknown
}
