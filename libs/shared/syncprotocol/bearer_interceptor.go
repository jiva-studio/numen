package syncprotocol

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
)

type bearerInterceptor struct{ token func() string }

// NewBearerInterceptor returns the client interceptor that sends the token of the session as a bearer in every
// request. It reads the token again for each request; an empty token sends no header, and removes one the request already carries.
func NewBearerInterceptor(token func() string) connect.Interceptor {
	return bearerInterceptor{token: token}
}

func (i bearerInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		i.setHeader(req.Header())
		return next(ctx, req)
	}
}

func (i bearerInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		i.setHeader(conn.RequestHeader())
		return conn
	}
}

func (bearerInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

func (i bearerInterceptor) setHeader(header http.Header) {
	header.Del("Authorization")
	if token := i.token(); token != "" {
		header.Set("Authorization", "Bearer "+token)
	}
}
