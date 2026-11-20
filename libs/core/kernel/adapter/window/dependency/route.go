// Package dependency serves model file requests from plugin frontends.
package dependency

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugindependency"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

type handler struct {
	models *plugindependency.Fetch
}

func NewRoute(models *plugindependency.Fetch, counted connect.HandlerOption) (string, http.Handler, any) {
	service := &handler{models: models}
	path, served := numenv1connect.NewHostDependencyServiceHandler(service, counted)
	return path, served, service
}

func (h *handler) FetchModelFile(ctx context.Context, req *connect.Request[v1.FetchModelFileRequest]) (*connect.Response[v1.FetchModelFileResponse], error) {
	path, err := h.models.Execute(ctx, req.Msg.GetUrl(), req.Msg.GetSha256())
	return connect.NewResponse(wire.DescribeModelFile(path, err)), nil
}
