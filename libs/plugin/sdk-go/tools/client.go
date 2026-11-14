package tools

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

// ErrorCode is the result status of a tool call.
type ErrorCode int32

const (
	ErrorCodeUnspecified  ErrorCode = 0
	ErrorCodeUnknownTool  ErrorCode = 1
	ErrorCodeBadArguments ErrorCode = 2
)

// Call describes an invocation of a tool. ConnectionID is the connection whose agent asks for the call; the
// list of running plugins names that connection as the holder of the tool call.
type Call struct {
	Name         string
	Arguments    []byte
	Context      map[string]string
	VaultID      string
	ConnectionID string
}

func (c *Call) GetName() string {
	if c == nil {
		return ""
	}
	return c.Name
}

func (c *Call) GetArguments() []byte {
	if c == nil {
		return nil
	}
	return c.Arguments
}

func (c *Call) GetContext() map[string]string {
	if c == nil {
		return nil
	}
	return c.Context
}

// CallResult is the output returned by a tool execution.
type CallResult struct {
	Text       string
	Structured []byte
	IsError    bool
}

func (r *CallResult) GetText() string {
	if r == nil {
		return ""
	}
	return r.Text
}

func (r *CallResult) GetStructured() []byte {
	if r == nil {
		return nil
	}
	return r.Structured
}

func (r *CallResult) GetIsError() bool {
	if r == nil {
		return false
	}
	return r.IsError
}

type CallResponse struct {
	Result *CallResult
	Error  ErrorCode
}

func (r *CallResponse) GetResult() *CallResult {
	if r == nil {
		return nil
	}
	return r.Result
}

func (r *CallResponse) GetError() ErrorCode {
	if r == nil {
		return ErrorCodeUnspecified
	}
	return r.Error
}

// ToolInfo describes a declared tool.
type ToolInfo struct {
	Name         string
	Title        string
	Description  string
	InputSchema  []byte
	OutputSchema []byte
	Effect       Effect
	Instructions []string
	Requires     []string
}

func (t *ToolInfo) GetName() string {
	if t == nil {
		return ""
	}
	return t.Name
}

func (t *ToolInfo) GetTitle() string {
	if t == nil {
		return ""
	}
	return t.Title
}

func (t *ToolInfo) GetDescription() string {
	if t == nil {
		return ""
	}
	return t.Description
}

func (t *ToolInfo) GetInputSchema() []byte {
	if t == nil {
		return nil
	}
	return t.InputSchema
}

func (t *ToolInfo) GetOutputSchema() []byte {
	if t == nil {
		return nil
	}
	return t.OutputSchema
}

func (t *ToolInfo) GetEffect() Effect {
	if t == nil {
		return 0
	}
	return t.Effect
}

func (t *ToolInfo) GetInstructions() []string {
	if t == nil {
		return nil
	}
	return t.Instructions
}

func (t *ToolInfo) GetRequires() []string {
	if t == nil {
		return nil
	}
	return t.Requires
}

// Listing is the list of tools a PluginToolService returns.
type Listing struct {
	Tools []*ToolInfo
}

func readListing(res *v1.ListToolsResponse) Listing {
	out := make([]*ToolInfo, len(res.GetTools()))
	for i, t := range res.GetTools() {
		out[i] = readToolInfo(t)
	}
	return Listing{Tools: out}
}

// Client calls a PluginToolService endpoint.
type Client interface {
	ListTools(ctx context.Context) (Listing, error)
	InvokeTool(ctx context.Context, call Call) (*CallResponse, error)
}

type rpcClient struct {
	rpc numenv1connect.PluginToolServiceClient
}

// NewClient creates a Client reaching the given endpoint.
func NewClient(endpoint hostcall.Endpoint) Client {
	return &rpcClient{
		rpc: numenv1connect.NewPluginToolServiceClient(hostcall.HTTPClient(), endpoint.BaseURL, endpoint.Options()...),
	}
}

func (c *rpcClient) ListTools(ctx context.Context) (Listing, error) {
	res, err := c.rpc.ListTools(ctx, connect.NewRequest(&v1.ListToolsRequest{}))
	if err != nil {
		return Listing{}, err
	}
	return readListing(res.Msg), nil
}

func (c *rpcClient) InvokeTool(ctx context.Context, call Call) (*CallResponse, error) {
	res, err := c.rpc.CallTool(ctx, connect.NewRequest(&v1.CallToolRequest{
		Name:         call.Name,
		Arguments:    call.Arguments,
		Context:      call.Context,
		VaultId:      call.VaultID,
		ConnectionId: call.ConnectionID,
	}))
	if err != nil {
		return nil, err
	}
	return readCallResponse(res.Msg), nil
}
