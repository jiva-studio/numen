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
	// ErrNoRoute means no agent answers the plugin: the agent the settings name is
	// not installed, or no plugin answers completions.
	ErrNoRoute = errors.New("ai: no agent answers")
	// ErrUnavailable means the agent the plugin is routed to cannot be reached.
	ErrUnavailable = errors.New("ai: routed agent unavailable")
	// ErrNoConsent means the person has not let the plugin ask a model, or has
	// refused it. The window asks them in the settings.
	ErrNoConsent = errors.New("ai: the person has not let this plugin ask a model")
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
	return NewClientFromVars(map[string]string{
		HostPortEnv:    os.Getenv(HostPortEnv),
		PluginTokenEnv: os.Getenv(PluginTokenEnv),
	})
}

// NewClientFromVars creates a client for the host named by the NUMEN_HOST_PORT entry of vars,
// authenticating with its NUMEN_PLUGIN_TOKEN entry. They are the variables a host gives a plugin
// it links into its own process.
func NewClientFromVars(vars map[string]string) (*Client, error) {
	port := strings.TrimSpace(vars[HostPortEnv])
	if port == "" {
		return nil, ErrNoHost
	}
	return NewClient(http.DefaultClient, "http://127.0.0.1:"+port, vars[PluginTokenEnv]), nil
}

// Complete asks the host to answer the question and returns the text.
func (c *Client) Complete(ctx context.Context, question *numenv1.CompletionQuestion) (string, error) {
	res, err := c.host.Complete(ctx, connect.NewRequest(&numenv1.HostServiceCompleteRequest{Question: question}))
	if err != nil {
		return "", fmt.Errorf("ai: complete: %w", err)
	}
	if err := readRefusal(res.Msg.GetError()); err != nil {
		return "", err
	}
	answer := res.Msg.GetAnswer()
	if answer == nil {
		return "", ErrNoAnswer
	}
	return answer.GetText(), nil
}

// readRefusal maps the code of a refused call to its error. Unspecified is no refusal.
func readRefusal(code numenv1.CompletionErrorCode) error {
	switch code {
	case numenv1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNSPECIFIED:
		return nil
	case numenv1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_ROUTE:
		return ErrNoRoute
	case numenv1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE:
		return ErrUnavailable
	case numenv1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_CONSENT:
		return ErrNoConsent
	}
	return fmt.Errorf("ai: unknown refusal code %d", code)
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
