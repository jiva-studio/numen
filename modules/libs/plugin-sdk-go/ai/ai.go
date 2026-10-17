// Package ai asks the host for a model completion.
//
// A plugin names a purpose and states what it needs; the host decides which
// model answers.
package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

const (
	// HostPortEnv names the environment variable holding the host endpoint port.
	HostPortEnv = "NUMEN_HOST_PORT"
	// PluginTokenEnv names the environment variable holding the plugin launch token.
	PluginTokenEnv = "NUMEN_PLUGIN_TOKEN"
)

var (
	// ErrNoHost means the environment names no host endpoint.
	ErrNoHost = errors.New("ai: " + HostPortEnv + " is not set")
	// ErrNoAnswer means the host replied without an answer.
	ErrNoAnswer = errors.New("ai: host sent no answer")
)

// Client calls the host on behalf of one plugin.
type Client struct {
	host numenv1connect.HostServiceClient
}

// NewClient creates a client for the host at baseURL, authenticating with token.
func NewClient(httpClient connect.HTTPClient, baseURL, token string) *Client {
	return &Client{host: numenv1connect.NewHostServiceClient(
		httpClient, baseURL, connect.WithInterceptors(newBearerInterceptor(token)),
	)}
}

// NewClientFromEnv creates a client for the host named by NUMEN_HOST_PORT,
// authenticating with NUMEN_PLUGIN_TOKEN.
func NewClientFromEnv() (*Client, error) {
	port := strings.TrimSpace(os.Getenv(HostPortEnv))
	if port == "" {
		return nil, ErrNoHost
	}
	return NewClient(http.DefaultClient, "http://127.0.0.1:"+port, os.Getenv(PluginTokenEnv)), nil
}

// Complete asks the host to answer the question and returns the text.
func (c *Client) Complete(ctx context.Context, question *numenv1.CompletionQuestion) (string, error) {
	res, err := c.host.Complete(ctx, connect.NewRequest(&numenv1.HostServiceCompleteRequest{Question: question}))
	if err != nil {
		return "", fmt.Errorf("ai: complete: %w", err)
	}
	answer := res.Msg.GetAnswer()
	if answer == nil {
		return "", ErrNoAnswer
	}
	return answer.GetText(), nil
}

// Complete asks the host named by the environment to answer the question.
func Complete(ctx context.Context, question *numenv1.CompletionQuestion) (string, error) {
	c, err := NewClientFromEnv()
	if err != nil {
		return "", err
	}
	return c.Complete(ctx, question)
}

type bearerInterceptor struct {
	token string
}

func newBearerInterceptor(token string) connect.Interceptor {
	return &bearerInterceptor{token: token}
}

func (b *bearerInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if b.token != "" {
			req.Header().Set("Authorization", "Bearer "+b.token)
		}
		return next(ctx, req)
	}
}

func (b *bearerInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		if b.token != "" {
			conn.RequestHeader().Set("Authorization", "Bearer "+b.token)
		}
		return conn
	}
}

func (b *bearerInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}
