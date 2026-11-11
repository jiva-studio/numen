package agentrun

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

// Client calls an agent service.
//
//numen:api
type Client struct {
	// Header is sent with every call.
	Header http.Header

	rpc numenv1connect.AgentServiceClient
}

// NewClient returns a client of the agent service at baseURL.
func NewClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) *Client {
	return &Client{Header: http.Header{}, rpc: numenv1connect.NewAgentServiceClient(httpClient, baseURL, opts...)}
}

func newRequest[T any](client *Client, msg *T) *connect.Request[T] {
	req := connect.NewRequest(msg)
	for key, values := range client.Header {
		for _, value := range values {
			req.Header().Add(key, value)
		}
	}
	return req
}

// Stream is the responses of one Ask call.
type Stream struct {
	rpc     *connect.ServerStreamForClient[v1.AskAgentResponse]
	current *Response
}

// Receive waits for the next response. It returns false when the stream has ended or failed.
func (s *Stream) Receive() bool {
	if !s.rpc.Receive() {
		return false
	}
	s.current = responseFromProto(s.rpc.Msg())
	return true
}

// Msg returns the response that Receive read last.
func (s *Stream) Msg() *Response { return s.current }

// Err returns the error that ended the stream, nil when it ended normally.
func (s *Stream) Err() error { return s.rpc.Err() }

// Close ends the stream and the work that answers it.
func (s *Stream) Close() error { return s.rpc.Close() }

// Ask gives the task to the agent.
func (c *Client) Ask(ctx context.Context, req *Request) (*Stream, error) {
	stream, err := c.rpc.AskAgent(ctx, newRequest(c, requestToProto(req)))
	if err != nil {
		return nil, err
	}
	return &Stream{rpc: stream, current: nil}, nil
}

// GetState reports whether an agent can be reached.
func (c *Client) GetState(ctx context.Context) (*State, error) {
	resp, err := c.rpc.GetAgentState(ctx, newRequest(c, &v1.GetAgentStateRequest{}))
	if err != nil {
		return nil, err
	}
	return stateFromProto(resp.Msg), nil
}

// GetPrograms lists the programs the agent runs.
func (c *Client) GetPrograms(ctx context.Context, commands []ProgramCommand) ([]*Program, error) {
	resp, err := c.rpc.GetAgentPrograms(ctx, newRequest(c, programCommandsToProto(commands)))
	if err != nil {
		return nil, err
	}
	return programsFromProto(resp.Msg), nil
}

func (c *Client) RegisterTool(ctx context.Context, tool *ToolDefinition) error {
	_, err := c.rpc.RegisterTool(ctx, newRequest(c, &v1.RegisterToolRequest{Tool: toolDefinitionToProto(tool)}))
	return err
}

func (c *Client) ExtendSystemPrompt(ctx context.Context, extension string) (string, error) {
	resp, err := c.rpc.ExtendSystemPrompt(ctx, newRequest(c, &v1.ExtendSystemPromptRequest{Extension: extension}))
	if err != nil {
		return "", err
	}
	return resp.Msg.GetFullPrompt(), nil
}

// GetTools lists the tools registered with the agent.
func (c *Client) GetTools(ctx context.Context) (*ToolSet, error) {
	resp, err := c.rpc.GetAgentTools(ctx, newRequest(c, &v1.GetAgentToolsRequest{}))
	if err != nil {
		return nil, err
	}
	return toolSetFromProto(resp.Msg), nil
}

// FinishConversation ends the conversation.
func (c *Client) FinishConversation(ctx context.Context, conversation string) error {
	_, err := c.rpc.FinishConversation(ctx, newRequest(c, &v1.FinishConversationRequest{Conversation: conversation}))
	return err
}
