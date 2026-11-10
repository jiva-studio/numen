package catalog

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

// NewRoute is the path CatalogService is served at and the handler that serves it.
func NewRoute(handler *Handler, options ...connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewCatalogServiceHandler(handler, options...)
}
