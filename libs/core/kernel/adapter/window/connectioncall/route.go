package connectioncall

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

// NewRoute is the path ConnectionCallService is served at and the handler that serves it. Each call goes
// through counted.
func NewRoute(handler numenv1connect.ConnectionCallServiceHandler, counted connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewConnectionCallServiceHandler(handler, counted)
}

// NewService returns the handler over the calls and the rows, or the handler that answers every call as
// unimplemented where the installation has no calls.
func NewService(calls Calls, rows Rows) numenv1connect.ConnectionCallServiceHandler {
	if calls == nil || rows == nil {
		return numenv1connect.UnimplementedConnectionCallServiceHandler{}
	}
	return NewHandler(calls, rows)
}
