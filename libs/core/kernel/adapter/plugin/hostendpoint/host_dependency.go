package hostendpoint

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

type dependencyEndpoint struct {
	endpoint *Endpoint
}

func (e *dependencyEndpoint) FetchModelFile(
	ctx context.Context, req *connect.Request[v1.FetchModelFileRequest],
) (*connect.Response[v1.FetchModelFileResponse], error) {
	if _, err := e.endpoint.Authenticate(req.Header().Get("Authorization")); err != nil {
		return nil, err
	}
	fetch := e.endpoint.callers.Dependencies
	if fetch == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("the host has no dependency service"))
	}
	file, err := fetch.Execute(ctx, req.Msg.GetUrl(), req.Msg.GetSha256())
	response := &v1.FetchModelFileResponse{}
	if err != nil {
		response.Error = v1.ErrorCode_ERROR_CODE_INTERNAL
		response.Detail = err.Error()
	} else {
		response.Path = file
	}
	return connect.NewResponse(response), nil
}
