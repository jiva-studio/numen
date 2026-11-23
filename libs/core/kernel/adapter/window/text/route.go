package text

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// Service answers TextService.
type Service interface {
	ReadLiveText(context.Context, *connect.Request[v1.ReadLiveTextRequest]) (*connect.Response[v1.ReadLiveTextResponse], error)
	CreateTextEdit(context.Context, *connect.Request[v1.CreateTextEditRequest]) (*connect.Response[v1.CreateTextEditResponse], error)
}

// NewUnavailable is the service of an installation that keeps no text: it answers every call as unimplemented.
func NewUnavailable() Service {
	return numenv1connect.UnimplementedTextServiceHandler{}
}

// NewRoute is the path TextService is served at and the handler that serves it.
func NewRoute(handler Service, options ...connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewTextServiceHandler(handler, options...)
}
