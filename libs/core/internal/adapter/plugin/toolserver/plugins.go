package toolserver

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/bridge"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

// Host is the plugin host as the tools reach it: the plugins it knows, each started on demand.
type Host interface {
	pluginport.PluginActivator
	GetPluginInfos() []pluginport.PluginInfo
}

// Server is the set of tools a window allows an agent, as a tool declares the servers it is
// offered on.
type Server v1.ToolServer

// Full is the set an agent works the whole vault through.
const Full = Server(v1.ToolServer_TOOL_SERVER_FULL)

// callScope is what every call of a run is made with: the context of the question and the vault
// the run works.
type callScope struct {
	context map[string]string
	vault   domain.VaultID
}

// tool is one tool a plugin lists, with the plugin that runs it.
type tool struct {
	pluginID string
	spec     *v1.PluginTool
}

// plugins reaches the PluginToolService of every plugin that declares the tools capability.
type plugins struct {
	host Host
}

// listTools lists the tools offered on the server by the plugins that declare the tools capability,
// in the order of the plugin ids, leaving out the plugin of the id given. A tool whose effect needs a
// permission its plugin lacks is left out, and of two tools with one name the first is kept.
func (p plugins) listTools(ctx context.Context, server Server, except string) []tool {
	var found []tool
	names := map[string]bool{}
	for _, info := range p.host.GetPluginInfos() {
		if info.ID == except || !slices.Contains(info.Capabilities, bridge.CapabilityTools) {
			continue
		}
		listed, err := p.listPluginTools(ctx, info.ID)
		if err != nil {
			// A disabled plugin, or one that cannot be started, offers no tools; the run goes on with
			// the tools of the others.
			continue
		}
		for _, spec := range listed {
			if names[spec.GetName()] || !isOffered(spec, server) || !hasPermission(info.Permissions, spec.GetEffect()) {
				continue
			}
			names[spec.GetName()] = true
			found = append(found, tool{pluginID: info.ID, spec: spec})
		}
	}
	return found
}

// listPluginTools lists the tools of one plugin, holding its reference for the length of the call.
func (p plugins) listPluginTools(ctx context.Context, pluginID string) (tools []*v1.PluginTool, err error) {
	client, lease, err := bridge.Open(ctx, p.host, pluginID, numenv1connect.NewPluginToolServiceClient)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, lease.Close()) }()

	res, err := client.ListTools(ctx, connect.NewRequest(&v1.ListToolsRequest{}))
	if err != nil {
		return nil, fmt.Errorf("list the tools of %s: %w", pluginID, err)
	}
	return res.Msg.GetTools(), nil
}

// callTool runs the tool on the plugin that listed it, in the scope of the run.
func (p plugins) callTool(ctx context.Context, one tool, arguments []byte, scope callScope) (response *v1.CallToolResponse, err error) {
	client, lease, err := bridge.Open(ctx, p.host, one.pluginID, numenv1connect.NewPluginToolServiceClient)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, lease.Close()) }()

	res, err := client.CallTool(ctx, connect.NewRequest(&v1.CallToolRequest{
		Name: one.spec.GetName(), Arguments: arguments, Context: scope.context, VaultId: string(scope.vault),
	}))
	if err != nil {
		return nil, fmt.Errorf("call %s on %s: %w", one.spec.GetName(), one.pluginID, err)
	}
	return res.Msg, nil
}

// isOffered says whether the tool is offered on the server. A tool that names no server is
// offered on the full one alone, and a tool of no effect nowhere.
func isOffered(spec *v1.PluginTool, server Server) bool {
	if spec.GetName() == "" || spec.GetEffect() == v1.ToolEffect_TOOL_EFFECT_UNSPECIFIED {
		return false
	}
	if len(spec.GetServers()) == 0 {
		return server == Full
	}
	return slices.Contains(spec.GetServers(), v1.ToolServer(server))
}

// hasPermission says whether the plugin holds the permission a tool of the effect needs.
func hasPermission(permissions []string, effect v1.ToolEffect) bool {
	if effect == v1.ToolEffect_TOOL_EFFECT_READ_WRITE {
		return slices.Contains(permissions, pluginport.PermissionVaultWrite)
	}
	return slices.Contains(permissions, pluginport.PermissionVaultRead) ||
		slices.Contains(permissions, pluginport.PermissionVaultWrite)
}
