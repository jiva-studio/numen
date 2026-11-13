package toolbridge

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
)

// Host is the plugin host used by the tools: it lists the known plugins, tells the enabled ones and
// starts each on demand.
type Host interface {
	bridge.Activator
	GetPluginInfos() []plugin.Info
	IsEnabled(pluginID string) bool
}

// tool is one tool a plugin lists, with the plugin that runs it.
type tool struct {
	pluginID string
	spec     *v1.PluginTool
}

// plugins calls the PluginToolService of every plugin that declares the tools
// capability. A version of a plugin is asked for its tools once; its answer is kept in listings.
type plugins struct {
	host     Host
	listings *listings
}

func (p plugins) listTools(ctx context.Context, except string) []tool {
	var found []tool
	names := map[string]bool{}
	for _, info := range p.host.GetPluginInfos() {
		if info.ID == except || !slices.Contains(info.Capabilities, bridge.CapabilityTools) || !p.host.IsEnabled(info.ID) {
			continue
		}
		listed, err := p.getPluginTools(ctx, info)
		if err != nil {
			// A plugin that cannot be started offers no tools; the listing goes on with the tools of
			// the others.
			continue
		}
		for _, spec := range listed.GetTools() {
			if names[spec.GetName()] || !isAvailable(spec) || !hasPermission(info.Permissions, spec.GetEffect()) {
				continue
			}
			names[spec.GetName()] = true
			found = append(found, tool{pluginID: info.ID, spec: spec})
		}
	}
	return found
}

func (p plugins) getPluginTools(ctx context.Context, info plugin.Info) (*v1.ListToolsResponse, error) {
	if kept, isKept := p.listings.get(info.ID, info.Version); isKept {
		return kept, nil
	}
	listed, err := p.listPluginTools(ctx, info.ID)
	if err != nil {
		return nil, err
	}
	p.listings.put(info.ID, info.Version, listed)
	return listed, nil
}

func (p plugins) listPluginTools(ctx context.Context, pluginID string) (listing *v1.ListToolsResponse, err error) {
	client, lease, err := bridge.OpenForQuestion(ctx, p.host, pluginID, "list tools", numenv1connect.NewPluginToolServiceClient)
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

func (p plugins) callTool(ctx context.Context, one tool, call *v1.CallToolRequest) (response *v1.CallToolResponse, err error) {
	client, lease, err := bridge.Open(ctx, p.host, one.pluginID, "call tool "+one.spec.GetName(), numenv1connect.NewPluginToolServiceClient)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, lease.Close()) }()

	res, err := client.CallTool(ctx, connect.NewRequest(&v1.CallToolRequest{
		Name: one.spec.GetName(), Arguments: call.GetArguments(), Context: call.GetContext(), VaultId: call.GetVaultId(),
	}))
	if err != nil {
		return nil, fmt.Errorf("call %s on %s: %w", one.spec.GetName(), one.pluginID, err)
	}
	return res.Msg, nil
}

func isAvailable(spec *v1.PluginTool) bool {
	return spec.GetName() != "" && spec.GetEffect() != v1.ToolEffect_TOOL_EFFECT_UNSPECIFIED
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
