// Package transport is how the clients of the SDK reach the host over Connect: the options of a
// client, the HTTP client it sends with and the token it presents. No type of it is part of the
// public API of the SDK.
package transport

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

// BuildOptions returns what a client of the host is made with: the token on every call, streaming ones
// included.
func BuildOptions(endpoint hostcall.Endpoint) []connect.ClientOption {
	return []connect.ClientOption{connect.WithInterceptors(NewBearer(endpoint.Token))}
}

// GetHTTPClient returns what a client of the host sends its calls with.
func GetHTTPClient() *http.Client { return http.DefaultClient }

type bearer struct {
	token string
}

// NewBearer presents the token as the plugin's own on every call, unary or streaming. An empty token
// presents none.
func NewBearer(token string) connect.Interceptor {
	return &bearer{token: token}
}

func (b *bearer) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if b.token != "" {
			req.Header().Set("Authorization", "Bearer "+b.token)
		}
		return next(ctx, req)
	}
}

func (b *bearer) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		if b.token != "" {
			conn.RequestHeader().Set("Authorization", "Bearer "+b.token)
		}
		return conn
	}
}

func (b *bearer) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}
