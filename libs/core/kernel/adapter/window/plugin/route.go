// Package plugin serves the route on which the windows reach the services plugins offer.
package plugin

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

// NewRoute is where PluginService is mounted, what the schema generated to answer it, and the
// handler it answers through. The window does not manage plugins, so every call is unimplemented.
func NewRoute(options connect.HandlerOption) (path string, served http.Handler, service any) {
	handler := numenv1connect.UnimplementedPluginServiceHandler{}
	path, served = numenv1connect.NewPluginServiceHandler(handler, options)
	return path, served, handler
}
