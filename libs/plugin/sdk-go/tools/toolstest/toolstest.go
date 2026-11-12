// Package toolstest serves PluginToolService from a stand-in of the host, for the tests of a plugin that
// lists and calls the tools of other plugins through tools.Client.
package toolstest

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

// ServiceHandler answers the tool calls of a plugin with the headers of each request.
type ServiceHandler interface {
	ListTools(ctx context.Context, header http.Header) (tools.Listing, error)
	InvokeTool(ctx context.Context, header http.Header, call tools.Call) (*tools.CallResponse, error)
}

type serviceAdapter struct {
	numenv1connect.UnimplementedPluginToolServiceHandler
	h ServiceHandler
}

func (a *serviceAdapter) ListTools(
	ctx context.Context, req *connect.Request[v1.ListToolsRequest],
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
	ctx context.Context, req *connect.Request[v1.CallToolRequest],
) (*connect.Response[v1.CallToolResponse], error) {
	res, err := a.h.InvokeTool(ctx, req.Header(), tools.Call{
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

// NewServiceHandler returns the path and the handler of PluginToolService over h.
func NewServiceHandler(h ServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewPluginToolServiceHandler(&serviceAdapter{h: h}, opts...)
}

func writeEffect(effect tools.Effect) v1.ToolEffect {
	switch effect {
	case tools.ReadOnly:
		return v1.ToolEffect_TOOL_EFFECT_READ_ONLY
	case tools.ReadWrite:
		return v1.ToolEffect_TOOL_EFFECT_READ_WRITE
	}
	return v1.ToolEffect_TOOL_EFFECT_UNSPECIFIED
}

func writeErrorCode(code tools.ErrorCode) v1.ToolErrorCode {
	switch code {
	case tools.ErrorCodeUnknownTool:
		return v1.ToolErrorCode_TOOL_ERROR_CODE_UNKNOWN_TOOL
	case tools.ErrorCodeBadArguments:
		return v1.ToolErrorCode_TOOL_ERROR_CODE_BAD_ARGUMENTS
	}
	return v1.ToolErrorCode_TOOL_ERROR_CODE_UNSPECIFIED
}

func writeToolInfo(t *tools.ToolInfo) *v1.PluginTool {
	if t == nil {
		return nil
	}
	return &v1.PluginTool{
		Name:         t.Name,
		Title:        t.Title,
		Description:  t.Description,
		InputSchema:  t.InputSchema,
		OutputSchema: t.OutputSchema,
		Effect:       writeEffect(t.Effect),
		Instructions: t.Instructions,
		Requires:     t.Requires,
	}
}

func writeCallResponse(res *tools.CallResponse) *v1.CallToolResponse {
	if res == nil {
		return &v1.CallToolResponse{}
	}
	out := &v1.CallToolResponse{Error: writeErrorCode(res.Error)}
	if res.Result != nil {
		out.Result = &v1.ToolResult{
			Text:       res.Result.Text,
			Structured: res.Result.Structured,
			IsError:    res.Result.IsError,
		}
	}
	return out
}
