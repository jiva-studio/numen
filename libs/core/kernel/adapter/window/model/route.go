package model

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

// NewRoute is the path ModelService is served at and the handler that serves it.
func NewRoute(handler numenv1connect.ModelServiceHandler, options ...connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewModelServiceHandler(handler, options...)
}
