package connection

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// NewRoute is where ProviderConnectionService is mounted, what the schema generated to answer it, and the
// handler it answers through. Each call goes through counted.
func NewRoute(handler numenv1connect.ProviderConnectionServiceHandler, counted connect.HandlerOption) (path string, served http.Handler, service any) {
	path, served = numenv1connect.NewProviderConnectionServiceHandler(handler, counted)
	return path, served, handler
}

// NewService returns the handler over the use cases, or the handler that answers every call as
// unimplemented where the installation has no connections.
func NewService(useCases *UseCases) numenv1connect.ProviderConnectionServiceHandler {
	if useCases == nil {
		return numenv1connect.UnimplementedProviderConnectionServiceHandler{}
	}
	return NewHandler(*useCases)
}
