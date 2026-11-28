package identity

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// NewRoute returns the path IdentityService is served at and the handler that serves it.
func NewRoute(handler numenv1connect.IdentityServiceHandler, counted connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewIdentityServiceHandler(handler, counted)
}

// NewService returns the handler over the use cases, or the handler that answers every call as
// unimplemented where the installation has no key store.
func NewService(useCases *UseCases) numenv1connect.IdentityServiceHandler {
	if useCases == nil {
		return numenv1connect.UnimplementedIdentityServiceHandler{}
	}
	return NewHandler(*useCases)
}
