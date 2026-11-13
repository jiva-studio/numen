package connection

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

// NewRoute is the path ProviderConnectionService is served at and the handler that serves it. Each call goes through counted.
func NewRoute(handler numenv1connect.ProviderConnectionServiceHandler, counted connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewProviderConnectionServiceHandler(handler, counted)
}

// NewService returns the handler over the use cases, or the handler that answers every call as
// unimplemented where the installation has no connections.
func NewService(useCases *UseCases) numenv1connect.ProviderConnectionServiceHandler {
	if useCases == nil {
		return numenv1connect.UnimplementedProviderConnectionServiceHandler{}
	}
	return NewHandler(*useCases)
}
