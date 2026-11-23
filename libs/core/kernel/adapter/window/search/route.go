package search

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// NewRoute is where SearchService is mounted, what the schema generated to answer it, and the
// handler it answers through.
func NewRoute(handler *Handler, options ...connect.HandlerOption) (path string, served http.Handler, service any) {
	path, served = numenv1connect.NewSearchServiceHandler(handler, options...)
	return path, served, handler
}
