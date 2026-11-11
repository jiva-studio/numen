// Package servetest holds the HTTP handlers tests mount in place of a window and of a plugin.
package servetest

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

type PluginLifecycle struct {
	numenv1connect.UnimplementedPluginLifecycleServiceHandler
}

// Ping reports manifest.ProtocolVersion.
func (PluginLifecycle) Ping(
	_ context.Context,
	_ *connect.Request[v1.PingRequest],
) (*connect.Response[v1.PingResponse], error) {
	return connect.NewResponse(&v1.PingResponse{Status: "ok", Protocol: manifest.ProtocolVersion}), nil
}

// HandlePluginLifecycle mounts a PluginLifecycle on mux.
func HandlePluginLifecycle(mux *http.ServeMux) {
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(PluginLifecycle{}))
}
