package providerbridge

import (
	"context"
	"sync"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
)

// Source opens the provider of a plugin over its ProviderService.
type Source struct {
	host bridge.Activator
}

// NewSource returns the source over the plugins the host activates.
func NewSource(host bridge.Activator) *Source { return &Source{host: host} }

// OpenProvider activates the plugin, which starts it if it is not running, and returns its provider with
// the function that releases the plugin. The function may be called more than once.
func (s *Source) OpenProvider(ctx context.Context, pluginID string) (connectionport.Provider, func(), error) {
	return s.open(ctx, pluginID)
}

// OpenCatalog activates the plugin in the same way and returns its model catalog.
func (s *Source) OpenCatalog(ctx context.Context, pluginID string) (connectionport.ModelCatalog, func(), error) {
	return s.open(ctx, pluginID)
}

func (s *Source) open(ctx context.Context, pluginID string) (*provider, func(), error) {
	client, lease, err := bridge.Open(ctx, s.host, pluginID, numenv1connect.NewProviderServiceClient)
	if err != nil {
		return nil, nil, err
	}
	var once sync.Once
	return &provider{client: client}, func() { once.Do(func() { lease.Host.Release(lease.PluginID) }) }, nil
}

// provider carries the calls of the core to the ProviderService of a plugin. A role the plugin does not serve
// is a stop of unknown reason, and the end of the context is returned as it is.
type provider struct {
	client numenv1connect.ProviderServiceClient
}

func (p *provider) Complete(
	ctx context.Context, call modelcall.ProviderCall, request modelcall.CompleteRequest,
) (modelcall.CompleteResult, error) {
	messages := make([]*v1.ProviderMessage, len(request.Messages))
	for i, one := range request.Messages {
		messages[i] = &v1.ProviderMessage{Role: one.Role, Text: one.Text}
	}
	response, err := p.client.Complete(ctx, connect.NewRequest(&v1.ProviderServiceCompleteRequest{
		Call: writeCall(call), Messages: messages, Temperature: request.Temperature, MaxTokens: int32(request.MaxTokens),
	}))
	if err != nil {
		return modelcall.CompleteResult{}, readCallError(ctx, err)
	}
	if response.Msg.GetFailure() != nil {
		return modelcall.CompleteResult{}, readFailure(response.Msg.GetFailure())
	}
	return modelcall.CompleteResult{Text: response.Msg.GetText()}, nil
}

func (p *provider) Embed(ctx context.Context, call modelcall.ProviderCall, texts []string) ([][]float32, error) {
	response, err := p.client.Embed(ctx, connect.NewRequest(&v1.ProviderServiceEmbedRequest{Call: writeCall(call), Texts: texts}))
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

func (p *provider) ListModels(ctx context.Context, call modelcall.ProviderCall, operation string) ([]modelcall.Model, error) {
	response, err := p.client.ListModels(ctx, connect.NewRequest(&v1.ProviderServiceListModelsRequest{Call: writeCall(call), Operation: operation}))
	if err != nil {
		return nil, readCallError(ctx, err)
	}
	if response.Msg.GetFailure() != nil {
		return nil, readFailure(response.Msg.GetFailure())
	}
	models := make([]modelcall.Model, len(response.Msg.GetModels()))
	for i, one := range response.Msg.GetModels() {
		models[i] = modelcall.Model{ID: one.GetId(), Title: one.GetTitle(), Operations: one.GetOperations()}
	}
	return models, nil
}

func (p *provider) ListReasoningLevels(ctx context.Context, call modelcall.ProviderCall) ([]modelcall.ReasoningLevel, error) {
	response, err := p.client.ListReasoningLevels(ctx, connect.NewRequest(&v1.ProviderServiceListReasoningLevelsRequest{Call: writeCall(call)}))
	if err != nil {
		return nil, readCallError(ctx, err)
	}
	if response.Msg.GetFailure() != nil {
		return nil, readFailure(response.Msg.GetFailure())
	}
	levels := make([]modelcall.ReasoningLevel, len(response.Msg.GetLevels()))
	for i, one := range response.Msg.GetLevels() {
		levels[i] = modelcall.ReasoningLevel{Value: one.GetValue(), Title: one.GetTitle()}
	}
	return levels, nil
}

func (p *provider) GetModelProfile(ctx context.Context, call modelcall.ProviderCall) (modelcall.ModelProfile, error) {
	response, err := p.client.GetModelProfile(ctx, connect.NewRequest(&v1.ProviderServiceGetModelProfileRequest{Call: writeCall(call)}))
	if err != nil {
		return modelcall.ModelProfile{}, readCallError(ctx, err)
	}
	if response.Msg.GetFailure() != nil {
		return modelcall.ModelProfile{}, readFailure(response.Msg.GetFailure())
	}
	return modelcall.ModelProfile{
		IsKnown:        response.Msg.GetIsKnown(),
		Dimensions:     int(response.Msg.GetDimensions()),
		MaxTokens:      int(response.Msg.GetMaxTokens()),
		Pooling:        response.Msg.GetPooling(),
		QueryPrefix:    response.Msg.GetQueryPrefix(),
		DocumentPrefix: response.Msg.GetDocumentPrefix(),
		Source:         response.Msg.GetSource(),
	}, nil
}

func (p *provider) ProbeEmbeddingWidth(ctx context.Context, call modelcall.ProviderCall) (int, error) {
	response, err := p.client.ProbeEmbeddingWidth(ctx, connect.NewRequest(&v1.ProviderServiceProbeEmbeddingWidthRequest{Call: writeCall(call)}))
	if err != nil {
		return 0, readCallError(ctx, err)
	}
	if response.Msg.GetFailure() != nil {
		return 0, readFailure(response.Msg.GetFailure())
	}
	return int(response.Msg.GetDimensions()), nil
}

func writeCall(call modelcall.ProviderCall) *v1.ProviderCall {
	secrets := make([]*v1.ProviderSecret, 0, len(call.Secrets))
	for key, value := range call.Secrets {
		secrets = append(secrets, &v1.ProviderSecret{Key: key, Value: value})
	}
	return &v1.ProviderCall{Type: call.Type, Fields: call.Fields, Secrets: secrets, Model: call.Model, Reasoning: call.Reasoning}
}

func readCallError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if connect.CodeOf(err) == connect.CodeUnimplemented {
		return &connection.StopError{Reason: connection.FailureReasonUnknown}
	}
	return err
}

func readFailure(failure *v1.ProviderFailure) error {
	reason := readReason(failure.GetReason())
	if failure.GetKind() == v1.ProviderFailureKind_PROVIDER_FAILURE_KIND_TRY_NEXT {
		return &connection.TryNextError{Reason: reason, RetryAfter: time.Duration(max(failure.GetRetryAfterMs(), 0)) * time.Millisecond}
	}
	return &connection.StopError{Reason: reason}
}

func readReason(reason v1.ProviderFailureReason) connection.FailureReason {
	switch reason {
	case v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_KEY_REJECTED:
		return connection.FailureReasonKeyRejected
	case v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_PAYMENT_REQUIRED:
		return connection.FailureReasonPaymentRequired
	case v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_RATE_LIMITED:
		return connection.FailureReasonRateLimited
	case v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_UNREACHABLE:
		return connection.FailureReasonUnreachable
	case v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_TIMEOUT:
		return connection.FailureReasonTimeout
	case v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_BAD_REQUEST:
		return connection.FailureReasonBadRequest
	case v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_UNKNOWN, v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_UNSPECIFIED:
		return connection.FailureReasonUnknown
	}
	return connection.FailureReasonUnknown
}
