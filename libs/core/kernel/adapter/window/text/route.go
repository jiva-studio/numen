package text

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// NewRoute is where TextService is mounted, what the schema generated to answer it, and the
// handler it answers through.
func NewRoute(handler *Handler, options ...connect.HandlerOption) (path string, served http.Handler, service any) {
	path, served = numenv1connect.NewTextServiceHandler(handler, options...)
	return path, served, handler
}
