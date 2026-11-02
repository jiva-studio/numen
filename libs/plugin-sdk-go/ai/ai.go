// Package ai requests a model completion from the host.
//
// A plugin names a purpose and states what it needs; the host decides which
// model answers.
package ai

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

const (
	// HostPortEnv names the environment variable holding the host endpoint port.
	HostPortEnv = hostcall.HostPortEnv
	// PluginTokenEnv names the environment variable holding the plugin launch token.
	PluginTokenEnv = hostcall.PluginTokenEnv
)

var (
	// ErrNoHost means the environment names no host endpoint.
	ErrNoHost = hostcall.ErrNoHost
	// ErrNoRoute means no agent answers the plugin: the program the settings name
	// is not installed or no plugin lists it, or no program is installed.
	ErrNoRoute = errors.New("ai: no agent answers")
	// ErrUnavailable means the agent the plugin is routed to cannot be reached or
	// stopped without an answer.
	ErrUnavailable = errors.New("ai: routed agent did not answer")
	// ErrNoConsent means user consent is required to invoke an AI model.
	ErrNoConsent = errors.New("ai: user consent required to invoke model")
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
		httpClient, baseURL, connect.WithInterceptors(hostcall.Bearer(token)),
	)}
}

// NewClientFromEnv creates a client for the host named by NUMEN_HOST_PORT,
// authenticating with NUMEN_PLUGIN_TOKEN.
func NewClientFromEnv() (*Client, error) {
	endpoint, err := hostcall.FromEnv()
	if err != nil {
		return nil, err
	}
	return newClientAt(endpoint), nil
}

// NewClientFromVars creates a client for the host named by the NUMEN_HOST_PORT entry of vars,
// authenticating with its NUMEN_PLUGIN_TOKEN entry. They are the variables a host gives a plugin
// it links into its own process.
func NewClientFromVars(vars map[string]string) (*Client, error) {
	endpoint, err := hostcall.FromVars(vars)
	if err != nil {
		return nil, err
	}
	return newClientAt(endpoint), nil
}

func newClientAt(endpoint hostcall.Endpoint) *Client {
	return NewClient(hostcall.HTTPClient(), endpoint.BaseURL, endpoint.Token)
}

// Message is one message in a completion question.
type Message struct {
	Role string
	Text string
}

// GetRole returns the role of the message.
func (m Message) GetRole() string { return m.Role }

// GetText returns the text of the message.
func (m Message) GetText() string { return m.Text }

// Question is a completion question sent to the host's routed AI model.
type Question struct {
	Purpose  string
	Messages []Message
}

// GetPurpose returns the purpose of the question.
func (q *Question) GetPurpose() string {
	if q == nil {
		return ""
	}
	return q.Purpose
}

// GetMessages returns the messages of the question.
func (q *Question) GetMessages() []Message {
	if q == nil {
		return nil
	}
	return q.Messages
}

func toProtoQuestion(q *Question) *v1.CompletionQuestion {
	if q == nil {
		return nil
	}
	msgs := make([]*v1.CompletionMessage, len(q.Messages))
	for i, m := range q.Messages {
		msgs[i] = &v1.CompletionMessage{Role: m.Role, Text: m.Text}
	}
	return &v1.CompletionQuestion{
		Purpose:  q.Purpose,
		Messages: msgs,
	}
}

// Complete sends the question to the host and returns the answer text.
func (c *Client) Complete(ctx context.Context, question *Question) (string, error) {
	res, err := c.host.Complete(ctx, connect.NewRequest(&v1.HostServiceCompleteRequest{Question: toProtoQuestion(question)}))
	if err != nil {
		return "", fmt.Errorf("ai: complete: %w", err)
	}
	if err := readRefusal(res.Msg.GetError(), res.Msg.GetDetail()); err != nil {
		return "", err
	}
	answer := res.Msg.GetAnswer()
	if answer == nil {
		return "", ErrNoAnswer
	}
	return answer.GetText(), nil
}

func readRefusal(code v1.CompletionErrorCode, detail string) error {
	switch code {
	case v1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNSPECIFIED:
		return nil
	case v1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_ROUTE:
		return addDetail(ErrNoRoute, detail)
	case v1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE:
		return addDetail(ErrUnavailable, detail)
	case v1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_CONSENT:
		return addDetail(ErrNoConsent, detail)
	}
	return fmt.Errorf("ai: unknown error code %d", code)
}

// Complete sends the question to the host named by the environment.
func Complete(ctx context.Context, question *Question) (string, error) {
	c, err := NewClientFromEnv()
	if err != nil {
		return "", err
	}
	return c.Complete(ctx, question)
}

// addDetail is the rejection followed by the detail the host gave, or the rejection alone where it gave none.
func addDetail(targetErr error, detail string) error {
	if detail == "" {
		return targetErr
	}
	return fmt.Errorf("%w: %s", targetErr, detail)
}
