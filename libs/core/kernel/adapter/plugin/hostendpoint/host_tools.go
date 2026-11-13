package hostendpoint

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

var errNoTools = errors.New("the host offers plugins no tools")

// PluginTools is the tools of the plugins, as a plugin reaches them through the host.
type PluginTools interface {
	// ListTools lists the tools the caller is offered.
	ListTools(ctx context.Context, callerID string) []*v1.PluginTool
	// CallTool runs a call of a tool the caller is offered, with the context and in the vault the
	// call carries. Any other tool is answered as unknown.
	CallTool(ctx context.Context, callerID string, call *v1.CallToolRequest) (*v1.CallToolResponse, error)
}

// toolEndpoint answers PluginToolService for a plugin that reaches the tools of the others.
type toolEndpoint struct {
	endpoint *Endpoint
}

var _ numenv1connect.PluginToolServiceHandler = (*toolEndpoint)(nil)

func (e *toolEndpoint) begin(header http.Header) (string, PluginTools, error) {
	pluginID, err := e.endpoint.Authorize(header.Get("Authorization"), manifest.PermissionVaultRead)
	if err != nil {
		return "", nil, err
	}
	tools := e.endpoint.callers.Tools()
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

// CallTool runs the tool with the context of the request, in the vault the request names. A request
// that names none runs in the vault named by the Numen-Vault header, else the vault the host shows. A
// vault the host does not know is not found.
func (e *toolEndpoint) CallTool(
	ctx context.Context, req *connect.Request[v1.CallToolRequest],
) (*connect.Response[v1.CallToolResponse], error) {
	pluginID, tools, err := e.begin(req.Header())
	if err != nil {
		return nil, err
	}
	header := req.Header()
	if named := req.Msg.GetVaultId(); named != "" {
		header = header.Clone()
		header.Set(vault.Header, named)
	}
	vaultID, err := e.endpoint.getVaultID(header)
	if err != nil {
		return nil, err
	}
	if connectionID := req.Msg.GetConnectionId(); connectionID != "" {
		ctx = plugin.WithCaller(ctx, plugin.Caller{PluginID: pluginID, ConnectionID: connectionID})
	}
	res, err := tools.CallTool(ctx, pluginID, &v1.CallToolRequest{
		Name: req.Msg.GetName(), Arguments: req.Msg.GetArguments(), Context: req.Msg.GetContext(), VaultId: string(vaultID),
	})
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, connect.NewError(connect.CodeCanceled, ctxErr)
		}
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewResponse(res), nil
}

func (e *Endpoint) getVaultID(header http.Header) (domain.VaultID, error) {
	vault, err := e.getVault(header)
	if err != nil || vault == nil {
		return "", err
	}
	v, showing := vault.Switch.GetVault()
	if !showing {
		return "", nil
	}
	return v.ID, nil
}
