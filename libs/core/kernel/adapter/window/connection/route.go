package connection

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

// NewRoute is the path ProviderConnectionService is served at and the handler that serves it.
func NewRoute(handler numenv1connect.ProviderConnectionServiceHandler, options ...connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewProviderConnectionServiceHandler(handler, options...)
}
