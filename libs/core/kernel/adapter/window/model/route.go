package model

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/modellist"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

// NewRoute is the path ModelService is served at and the handler that serves it. Each call goes through counted.
func NewRoute(handler numenv1connect.ModelServiceHandler, counted connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewModelServiceHandler(handler, counted)
}

// NewService returns the handler over the lists and the calls, or the handler that answers every call
// as unimplemented where the installation has no lists.
func NewService(lists *modellist.Lists, calls Calls) numenv1connect.ModelServiceHandler {
	if lists == nil {
		return numenv1connect.UnimplementedModelServiceHandler{}
	}
	return NewHandler(lists, calls)
}
