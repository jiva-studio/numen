package connection

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

// ErrNoConnection means the settings of the plugin hold no row that can serve the operation.
var ErrNoConnection = errors.New("connection: no connection serves the operation")

// Failure is one row that did not answer: the name of its connection and the reason.
type Failure struct{ ConnectionName, Reason string }

// AllFailedError means every row of the plugin was set aside or failed.
type AllFailedError struct{ Failures []Failure }

// Error names each connection and its reason, in the order of the rows.
func (e *AllFailedError) Error() string {
	parts := make([]string, len(e.Failures))
	for i, one := range e.Failures {
		parts[i] = one.ConnectionName + ": " + one.Reason
	}
	return "connection: every connection failed: " + strings.Join(parts, "; ")
}

// RejectedError means a connection found the request itself wrong, so no other connection was tried.
type RejectedError struct{ ConnectionName, Reason string }

// Error names the connection and the reason.
func (e *RejectedError) Error() string {
	return "connection: " + e.ConnectionName + " rejected the request: " + e.Reason
}

// Client asks the core to carry out operations for one plugin. The core takes the connections from the
// rows of the settings of that plugin; a request names no connection and a result holds no key.
type Client struct {
	host numenv1connect.HostModelServiceClient
}

// NewClient creates a client for the host at baseURL, authenticating with token.
func NewClient(httpClient connect.HTTPClient, baseURL, token string) *Client {
	return &Client{host: numenv1connect.NewHostModelServiceClient(
		httpClient, baseURL, connect.WithInterceptors(hostcall.Bearer(token)),
	)}
}

// NewClientFromEnv creates a client for the host named by NUMEN_HOST_PORT, authenticating with
// NUMEN_PLUGIN_TOKEN.
func NewClientFromEnv() (*Client, error) {
	endpoint, err := hostcall.FromEnv()
	if err != nil {
		return nil, err
	}
	return newClientAt(endpoint), nil
}

// NewClientFromVars creates a client for the host named by the NUMEN_HOST_PORT entry of vars,
// authenticating with its NUMEN_PLUGIN_TOKEN entry. They are the variables a host gives a plugin it links
// into its own process.
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

// Priority says whose action a call serves. A call that sets none is background, so it waits behind the
// calls of a person's actions when its connection runs as many calls as it may.
//
//numen:api
type Priority int

const (
	// PriorityBackground is work that serves no person's action, such as indexing.
	PriorityBackground Priority = iota
	// PriorityInteractive is a person's action, such as a search or a check of a line.
	PriorityInteractive
)

// CompleteRequest is what a consumer asks a model to complete.
type CompleteRequest struct {
	Messages    []Message
	Temperature *float64
	MaxTokens   int
	Priority    Priority
}

// EmbedRequest is the texts a consumer asks a model to embed.
type EmbedRequest struct {
	Texts    []string
	Priority Priority
}

// EmbedResult holds one vector for each text, in the order of the texts.
type EmbedResult struct{ Vectors [][]float32 }

// EmbeddingProfile is what the first row that answers says about its embedding model. Model is the
// model of that row and MaxTokens is zero when the limit is not known.
type EmbeddingProfile struct {
	Model          string
	Source         string
	Pooling        string
	QueryPrefix    string
	DocumentPrefix string
	Dimensions     int
	MaxTokens      int
}

// Complete asks the rows of the plugin, in order, to complete the messages. It returns ErrNoConnection,
// *AllFailedError or *RejectedError where no row answers.
func (c *Client) Complete(ctx context.Context, request CompleteRequest) (CompleteResult, error) {
	messages := make([]*v1.HostModelMessage, len(request.Messages))
	for i, one := range request.Messages {
		messages[i] = &v1.HostModelMessage{Role: one.Role, Text: one.Text}
	}
	wire := &v1.HostModelServiceCompleteRequest{Messages: messages, Temperature: request.Temperature, MaxTokens: int32(request.MaxTokens), Priority: writePriority(request.Priority)}
	response, err := c.host.Complete(ctx, connect.NewRequest(wire))
	if err != nil {
		return CompleteResult{}, fmt.Errorf("connection: complete: %w", err)
	}
	if err := readError(response.Msg.GetError(), response.Msg.GetDetail(), response.Msg.GetFailures()); err != nil {
		return CompleteResult{}, err
	}
	return CompleteResult{Text: response.Msg.GetText()}, nil
}

// Embed asks the rows of the plugin, in order, to embed the texts. It returns the errors Complete does.
func (c *Client) Embed(ctx context.Context, request EmbedRequest) (EmbedResult, error) {
	response, err := c.host.Embed(ctx, connect.NewRequest(&v1.HostModelServiceEmbedRequest{Texts: request.Texts, Priority: writePriority(request.Priority)}))
	if err != nil {
		return EmbedResult{}, fmt.Errorf("connection: embed: %w", err)
	}
	if err := readError(response.Msg.GetError(), response.Msg.GetDetail(), response.Msg.GetFailures()); err != nil {
		return EmbedResult{}, err
	}
	vectors := make([][]float32, len(response.Msg.GetVectors()))
	for i, one := range response.Msg.GetVectors() {
		vectors[i] = one.GetValues()
	}
	return EmbedResult{Vectors: vectors}, nil
}

// GetEmbeddingProfile asks the rows of the plugin, in order, for the profile of their embedding model.
// It returns the errors Complete does.
func (c *Client) GetEmbeddingProfile(ctx context.Context) (EmbeddingProfile, error) {
	response, err := c.host.GetEmbeddingProfile(ctx, connect.NewRequest(&v1.HostModelServiceGetEmbeddingProfileRequest{}))
	if err != nil {
		return EmbeddingProfile{}, fmt.Errorf("connection: get embedding profile: %w", err)
	}
	if err := readError(response.Msg.GetError(), response.Msg.GetDetail(), response.Msg.GetFailures()); err != nil {
		return EmbeddingProfile{}, err
	}
	return EmbeddingProfile{
		Model:          response.Msg.GetModel(),
		Source:         response.Msg.GetSource(),
		Pooling:        response.Msg.GetPooling(),
		QueryPrefix:    response.Msg.GetQueryPrefix(),
		DocumentPrefix: response.Msg.GetDocumentPrefix(),
		Dimensions:     int(response.Msg.GetDimensions()),
		MaxTokens:      int(response.Msg.GetMaxTokens()),
	}, nil
}

func readError(code v1.HostModelErrorCode, detail string, failures []*v1.HostModelFailure) error {
	switch code {
	case v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_UNSPECIFIED:
		return nil
	case v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_NO_CONNECTION:
		return ErrNoConnection
	case v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_ALL_FAILED:
		all := &AllFailedError{Failures: make([]Failure, len(failures))}
		for i, one := range failures {
			all.Failures[i] = Failure{ConnectionName: one.GetConnectionName(), Reason: one.GetReason()}
		}
		return all
	case v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_REJECTED:
		rejected := &RejectedError{}
		if len(failures) > 0 {
			rejected.ConnectionName, rejected.Reason = failures[0].GetConnectionName(), failures[0].GetReason()
		}
		return rejected
	}
	return fmt.Errorf("connection: unknown error code %d: %s", code, detail)
}

func writePriority(priority Priority) v1.Priority {
	if priority == PriorityInteractive {
		return v1.Priority_PRIORITY_INTERACTIVE
	}
	return v1.Priority_PRIORITY_BACKGROUND
}
