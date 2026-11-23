package identity

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// NewRoute is where IdentityService is mounted, what the schema generated to answer it, and the
// handler it answers through.
func NewRoute(handler numenv1connect.IdentityServiceHandler, counted connect.HandlerOption) (path string, served http.Handler, service any) {
	path, served = numenv1connect.NewIdentityServiceHandler(handler, counted)
	return path, served, handler
}

// NewService returns the handler over the use cases, or the handler that answers every call as
// unimplemented where the installation has no key store.
func NewService(useCases *UseCases) numenv1connect.IdentityServiceHandler {
	if useCases == nil {
		return numenv1connect.UnimplementedIdentityServiceHandler{}
	}
	return NewHandler(*useCases)
}
