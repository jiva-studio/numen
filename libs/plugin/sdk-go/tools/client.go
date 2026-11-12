package tools

import (
	"context"
	"net/http"
	"slices"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

// ErrorCode is the result status of a tool call.
type ErrorCode int32

const (
	ErrorCodeUnspecified  ErrorCode = 0
	ErrorCodeUnknownTool  ErrorCode = 1
	ErrorCodeBadArguments ErrorCode = 2
)

// Call describes an invocation of a tool.
type Call struct {
	Name      string
	Arguments []byte
	Context   map[string]string
	VaultID   string
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
	Step         Step
	Servers      []Server
	Instructions []string
	Requires     []string
	Hints        Hints
}

// Hints are the MCP annotations of a tool.
//
//numen:api
type Hints struct {
	// ReadOnly is a tool that changes nothing.
	ReadOnly bool
	// Destructive is a tool that removes something.
	Destructive bool
	// Idempotent is a tool whose repeated call with the same arguments has no further effect.
	Idempotent bool
	// OpenWorld is a tool that reaches outside the machine.
	OpenWorld bool
}

func createHints(spec Spec) Hints {
	return Hints{
		ReadOnly:    spec.Effect == ReadOnly,
		Destructive: spec.Effect == ReadWrite && spec.Step == Remove,
		Idempotent:  spec.Effect == ReadOnly,
		OpenWorld:   slices.Contains(spec.Requires, host.PermissionLinkOpen),
	}
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

func (t *ToolInfo) GetStep() Step {
	if t == nil {
		return ByEffect
	}
	return t.Step
}

func (t *ToolInfo) GetServers() []Server {
	if t == nil {
		return nil
	}
	return t.Servers
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

func (t *ToolInfo) GetHints() Hints {
	if t == nil {
		return Hints{}
	}
	return t.Hints
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
		Name:      call.Name,
		Arguments: call.Arguments,
		Context:   call.Context,
		VaultId:   call.VaultID,
	}))
	if err != nil {
		return nil, err
	}
	return readCallResponse(res.Msg), nil
}

// ServiceHandler handles tool listing and invocation with request headers.
type ServiceHandler interface {
	ListTools(ctx context.Context, header http.Header) (Listing, error)
	InvokeTool(ctx context.Context, header http.Header, call Call) (*CallResponse, error)
}

type serviceAdapter struct {
	numenv1connect.UnimplementedPluginToolServiceHandler
	h ServiceHandler
}

func (a *serviceAdapter) ListTools(
	ctx context.Context,
	req *connect.Request[v1.ListToolsRequest],
) (*connect.Response[v1.ListToolsResponse], error) {
	listing, err := a.h.ListTools(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	wire := make([]*v1.PluginTool, len(listing.Tools))
	for i, t := range listing.Tools {
		wire[i] = writeToolInfo(t)
	}
	return connect.NewResponse(&v1.ListToolsResponse{Tools: wire}), nil
}

func (a *serviceAdapter) CallTool(
	ctx context.Context,
	req *connect.Request[v1.CallToolRequest],
) (*connect.Response[v1.CallToolResponse], error) {
	res, err := a.h.InvokeTool(ctx, req.Header(), Call{
		Name:      req.Msg.GetName(),
		Arguments: req.Msg.GetArguments(),
		Context:   req.Msg.GetContext(),
		VaultID:   req.Msg.GetVaultId(),
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(writeCallResponse(res)), nil
}

func NewServiceHandler(h ServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewPluginToolServiceHandler(&serviceAdapter{h: h}, opts...)
}
