package hostendpoint

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	modelcallport "github.com/jiva-studio/numen/libs/core/usecase/modelcall/modelcallcontract"
)

// modelEndpoint answers HostModelService for the plugins that call the host. The connections are the
// rows of the calling plugin, so the call needs no permission and names no connection.
type modelEndpoint struct {
	endpoint *Endpoint
}

var _ numenv1connect.HostModelServiceHandler = (*modelEndpoint)(nil)

func (e *modelEndpoint) begin(header string) (string, modelcallport.ModelCalls, error) {
	pluginID, err := e.endpoint.authenticate(header)
	if err != nil {
		return "", nil, err
	}
	calls := e.endpoint.callers.GetModelCalls()
	if calls == nil {
		return pluginID, noCalls{}, nil
	}
	return pluginID, calls, nil
}

// noCalls answers every call with no connection.
type noCalls struct{}

func (noCalls) Complete(context.Context, string, modelcallport.CompleteRequest, modelcallport.Priority) (modelcallport.CompleteResult, error) {
	return modelcallport.CompleteResult{}, modelcallport.ErrNoConnection
}

func (noCalls) Embed(context.Context, string, []string, modelcallport.Priority) ([][]float32, error) {
	return nil, modelcallport.ErrNoConnection
}

func (noCalls) GetEmbeddingProfile(context.Context, string) (modelcallport.EmbeddingProfile, error) {
	return modelcallport.EmbeddingProfile{}, modelcallport.ErrNoConnection
}

func (e *modelEndpoint) Complete(
	ctx context.Context, request *connect.Request[v1.HostModelServiceCompleteRequest],
) (*connect.Response[v1.HostModelServiceCompleteResponse], error) {
	pluginID, calls, err := e.begin(request.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}
	messages := make([]modelcallport.Message, len(request.Msg.GetMessages()))
	for i, one := range request.Msg.GetMessages() {
		messages[i] = modelcallport.Message{Role: one.GetRole(), Text: one.GetText()}
	}
	result, err := calls.Complete(ctx, pluginID, modelcallport.CompleteRequest{
		Messages: messages, Temperature: request.Msg.Temperature, MaxTokens: int(request.Msg.GetMaxTokens()),
	}, readPriority(request.Msg.GetPriority()))
	if err != nil {
		code, detail, failures, served := writeModelError(ctx, err)
		if !served {
			return nil, readModelStatus(ctx, err)
		}
		return connect.NewResponse(&v1.HostModelServiceCompleteResponse{Error: code, Detail: detail, Failures: failures}), nil
	}
	return connect.NewResponse(&v1.HostModelServiceCompleteResponse{Text: result.Text}), nil
}

func (e *modelEndpoint) Embed(
	ctx context.Context, request *connect.Request[v1.HostModelServiceEmbedRequest],
) (*connect.Response[v1.HostModelServiceEmbedResponse], error) {
	pluginID, calls, err := e.begin(request.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}
	vectors, err := calls.Embed(ctx, pluginID, request.Msg.GetTexts(), readPriority(request.Msg.GetPriority()))
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
	profile, err := calls.GetEmbeddingProfile(ctx, pluginID)
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
	var allFailed *modelcallport.AllFailedError
	var rejected *modelcallport.RejectedError
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
	var allFailed *modelcallport.AllFailedError
	var rejected *modelcallport.RejectedError
	switch {
	case errors.Is(err, modelcallport.ErrNoConnection):
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

// readPriority maps the wire priority to the port one. An unspecified priority is background.
func readPriority(priority v1.Priority) modelcallport.Priority {
	if priority == v1.Priority_PRIORITY_INTERACTIVE {
		return modelcallport.PriorityInteractive
	}
	return modelcallport.PriorityBackground
}
