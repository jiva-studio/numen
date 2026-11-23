package catalog

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// NewRoute is the path CatalogService is served at and the handler that serves it, and the handler it answers through.
func NewRoute(handler *Handler, options ...connect.HandlerOption) (path string, served http.Handler, service any) {
	path, served = numenv1connect.NewCatalogServiceHandler(handler, options...)
	return path, served, handler
}
