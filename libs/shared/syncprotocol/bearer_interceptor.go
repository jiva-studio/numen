package syncprotocol

import (
	"context"

	"connectrpc.com/connect"
)

type bearerInterceptor struct{ token func() string }

// NewBearerInterceptor returns the client interceptor that sends the token of the session as a bearer in every
// request. It reads the token again for each request; an empty token sends no header.
func NewBearerInterceptor(token func() string) connect.Interceptor {
	return bearerInterceptor{token: token}
}

func (i bearerInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		i.setHeader(req.Header().Set)
		return next(ctx, req)
	}
}

func (i bearerInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		i.setHeader(conn.RequestHeader().Set)
		return conn
	}
}

func (bearerInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

func (i bearerInterceptor) setHeader(set func(key, value string)) {
	if token := i.token(); token != "" {
		set("Authorization", "Bearer "+token)
	}
}
