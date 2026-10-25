// Package hostcall is how a plugin calls the host: where the host endpoint is, and the token that
// says which plugin is calling.
package hostcall

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"

	"connectrpc.com/connect"
)

const (
	// HostPortEnv names the environment variable holding the host endpoint port.
	HostPortEnv = "NUMEN_HOST_PORT"
	// PluginTokenEnv names the environment variable holding the plugin launch token.
	PluginTokenEnv = "NUMEN_PLUGIN_TOKEN"
)

// ErrNoHost means the environment names no host endpoint.
var ErrNoHost = errors.New("host: " + HostPortEnv + " is not set")

// Endpoint is where the host is, and the token the plugin calls it with.
type Endpoint struct {
	BaseURL string
	Token   string
}

// FromEnv is the endpoint the environment of the process names.
func FromEnv() (Endpoint, error) {
	return FromVars(map[string]string{
		HostPortEnv:    os.Getenv(HostPortEnv),
		PluginTokenEnv: os.Getenv(PluginTokenEnv),
	})
}

// FromVars is the endpoint the NUMEN_HOST_PORT and NUMEN_PLUGIN_TOKEN entries of vars name. They are
// the variables a host gives a plugin it links into its own process.
func FromVars(vars map[string]string) (Endpoint, error) {
	port := strings.TrimSpace(vars[HostPortEnv])
	if port == "" {
		return Endpoint{}, ErrNoHost
	}
	return Endpoint{BaseURL: "http://127.0.0.1:" + port, Token: vars[PluginTokenEnv]}, nil
}

// Options are what a client of the host is made with: the token on every call, streaming ones
// included.
func (e Endpoint) Options() []connect.ClientOption {
	return []connect.ClientOption{connect.WithInterceptors(Bearer(e.Token))}
}

// HTTPClient is what a client of the host sends its calls with.
func HTTPClient() connect.HTTPClient { return http.DefaultClient }

type bearer struct {
	token string
}

// Bearer presents the token as the plugin's own on every call, unary or streaming. An empty token
// presents none.
func Bearer(token string) connect.Interceptor {
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
