package testsupport

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

type PluginLifecycle struct {
	numenv1connect.UnimplementedPluginLifecycleServiceHandler
}

// Ping reports port.PluginProtocolVersion.
func (PluginLifecycle) Ping(
	_ context.Context,
	_ *connect.Request[v1.PingRequest],
) (*connect.Response[v1.PingResponse], error) {
	return connect.NewResponse(&v1.PingResponse{Status: "ok", Protocol: pluginport.PluginProtocolVersion}), nil
}

// HandlePluginLifecycle mounts a PluginLifecycle on mux.
func HandlePluginLifecycle(mux *http.ServeMux) {
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(PluginLifecycle{}))
}
