package search

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

// NewRoute is the path SearchService is served at and the handler that serves it.
func NewRoute(handler *Handler, options ...connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewSearchServiceHandler(handler, options...)
}
