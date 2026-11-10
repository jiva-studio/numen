package agenttoolsbridge

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
)

// Host is the plugin host used by the tools: it lists the known plugins and
// starts each on demand.
type Host interface {
	bridge.Activator
	GetPluginInfos() []plugin.Info
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

// plugins calls the PluginToolService of every plugin that declares the tools
// capability.
type plugins struct {
	host Host
}

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
		for _, spec := range listed.GetTools() {
			if names[spec.GetName()] || !isAvailable(spec, server) || !hasPermission(info.Permissions, spec.GetEffect()) {
				continue
			}
			names[spec.GetName()] = true
			found = append(found, tool{pluginID: info.ID, spec: spec})
		}
	}
	return found
}

func (p plugins) listPluginTools(ctx context.Context, pluginID string) (listing *v1.ListToolsResponse, err error) {
	client, lease, err := bridge.Open(ctx, p.host, pluginID, numenv1connect.NewPluginToolServiceClient)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, lease.Close()) }()

	res, err := client.ListTools(ctx, connect.NewRequest(&v1.ListToolsRequest{}))
	if err != nil {
		return nil, fmt.Errorf("list the tools of %s: %w", pluginID, err)
	}
	return res.Msg, nil
}

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

func isAvailable(spec *v1.PluginTool, server Server) bool {
	if spec.GetName() == "" || spec.GetEffect() == v1.ToolEffect_TOOL_EFFECT_UNSPECIFIED {
		return false
	}
	if len(spec.GetServers()) == 0 {
		return server == Full
	}
	return slices.Contains(spec.GetServers(), v1.ToolServer(server))
}

func hasPermission(permissions []string, effect v1.ToolEffect) bool {
	if effect == v1.ToolEffect_TOOL_EFFECT_READ_WRITE {
		return slices.Contains(permissions, manifest.PermissionVaultWrite)
	}
	return slices.Contains(permissions, manifest.PermissionVaultRead) ||
		slices.Contains(permissions, manifest.PermissionVaultWrite)
}

func (p plugins) getPermissions(pluginID string) []string {
	for _, info := range p.host.GetPluginInfos() {
		if info.ID == pluginID {
			return info.Permissions
		}
	}
	return nil
}

func (p plugins) listToolsFor(ctx context.Context, server Server, callerID string) []tool {
	permissions := p.getPermissions(callerID)
	all := p.listTools(ctx, server, "")
	required := make([]tool, 0, len(all))
	for _, one := range all {
		if hasPermission(permissions, one.spec.GetEffect()) && hasAll(permissions, one.spec.GetRequires()) {
			required = append(required, one)
		}
	}
	return required
}
