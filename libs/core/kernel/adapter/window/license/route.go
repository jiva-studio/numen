package license

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// NewRoute returns the path LicenseService is served at and the handler that serves it, and the handler it answers through.
func NewRoute(handler numenv1connect.LicenseServiceHandler, counted connect.HandlerOption) (path string, served http.Handler, service any) {
	path, served = numenv1connect.NewLicenseServiceHandler(handler, counted)
	return path, served, handler
}

// NewService returns the handler over the use cases, or the handler that answers every call as
// unimplemented where the installation has no licenses.
func NewService(useCases *UseCases) numenv1connect.LicenseServiceHandler {
	if useCases == nil {
		return numenv1connect.UnimplementedLicenseServiceHandler{}
	}
	return NewHandler(*useCases)
}
