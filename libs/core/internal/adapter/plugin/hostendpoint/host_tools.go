package hostendpoint

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/notes/domain"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

var errNoTools = errors.New("the host offers plugins no tools")

// PluginTools is the tools of the plugins, as a plugin serves them to programs outside Numen.
type PluginTools interface {
	// ListTools lists the tools the caller is offered.
	ListTools(ctx context.Context, callerID string) []*v1.PluginTool
	// CallTool runs a tool the caller is offered, in the vault given. Any other is answered as unknown.
	CallTool(ctx context.Context, callerID, name string, arguments []byte, vault domain.VaultID) (*v1.CallToolResponse, error)
}

// toolEndpoint answers PluginToolService for a plugin that serves the tools of the others.
type toolEndpoint struct {
	endpoint *Endpoint
}

var _ numenv1connect.PluginToolServiceHandler = (*toolEndpoint)(nil)

// begin authorizes the call and returns the plugin that made it and the tools it is offered.
func (e *toolEndpoint) begin(header http.Header) (string, PluginTools, error) {
	pluginID, err := e.endpoint.authorize(header.Get("Authorization"), pluginport.PermissionVaultRead)
	if err != nil {
		return "", nil, err
	}
	tools := e.endpoint.callers.GetPluginTools()
	if tools == nil {
		return "", nil, connect.NewError(connect.CodeUnavailable, errNoTools)
	}
	return pluginID, tools, nil
}

// ListTools lists the tools of the other plugins the caller is offered.
func (e *toolEndpoint) ListTools(
	ctx context.Context, req *connect.Request[v1.ListToolsRequest],
) (*connect.Response[v1.ListToolsResponse], error) {
	pluginID, tools, err := e.begin(req.Header())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.ListToolsResponse{Tools: tools.ListTools(ctx, pluginID)}), nil
}

// CallTool runs the tool in the vault the Numen-Vault header names, else the one the host shows. The
// context the caller sends is dropped: no question was asked in Numen.
func (e *toolEndpoint) CallTool(
	ctx context.Context, req *connect.Request[v1.CallToolRequest],
) (*connect.Response[v1.CallToolResponse], error) {
	pluginID, tools, err := e.begin(req.Header())
	if err != nil {
		return nil, err
	}
	vault, err := e.endpoint.getVaultID(req.Header())
	if err != nil {
		return nil, err
	}
	res, err := tools.CallTool(ctx, pluginID, req.Msg.GetName(), req.Msg.GetArguments(), vault)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, connect.NewError(connect.CodeCanceled, ctxErr)
		}
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewResponse(res), nil
}

// getVaultID is the identity of the vault a call works, empty where the host has none to offer.
func (e *Endpoint) getVaultID(header http.Header) (domain.VaultID, error) {
	vault, err := e.getVault(header)
	if err != nil || vault == nil {
		return "", err
	}
	v, showing := vault.GetVault()
	if !showing {
		return "", nil
	}
	return v.ID, nil
}
