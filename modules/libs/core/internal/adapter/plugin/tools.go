package plugin

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// ErrUnknownTool is a call for a tool the plugin did not list, or a plugin that does not list it
// any longer.
var ErrUnknownTool = errors.New("no plugin lists that tool")

// ErrBadToolArguments is arguments that are not a JSON object.
var ErrBadToolArguments = errors.New("tool arguments are not a JSON object")

// ToolsPluginHost provides plugin resolution, activation and permissions for the tools.
type ToolsPluginHost interface {
	port.PluginResolver
	port.PluginActivator
	port.PluginAuthorizer
}

// ToolsBridge connects port.PluginTools to the plugins that declare the tools capability,
// over their PluginToolService.
type ToolsBridge struct {
	host ToolsPluginHost

	mu sync.Mutex
	// listed is the effect of each tool the last listing found, by plugin and name.
	listed map[toolKey]port.ToolEffect
}

type toolKey struct {
	pluginID string
	name     string
}

// OpenToolsBridge makes the bridge to the plugins the host runs. It starts none.
func OpenToolsBridge(host ToolsPluginHost) *ToolsBridge {
	return &ToolsBridge{host: host, listed: map[toolKey]port.ToolEffect{}}
}

// Close releases nothing: a call holds its reference on the plugin for the length of the call.
func (b *ToolsBridge) Close() error {
	return nil
}

// ListTools lists the tools of every enabled plugin that declares the capability and holds the
// permission their effect needs. A plugin that cannot be started or reached is left out.
func (b *ToolsBridge) ListTools(ctx context.Context) []port.PluginTool {
	var found []port.PluginTool
	listed := map[toolKey]port.ToolEffect{}
	for _, one := range b.host.ResolveCapabilities(CapabilityTools) {
		tools, err := b.ListPluginTools(ctx, one.Manifest.ID)
		if err != nil {
			continue
		}
		for _, tool := range tools {
			if b.host.CheckPluginPermission(tool.PluginID, getToolPermission(tool.Effect)) != nil {
				continue
			}
			listed[toolKey{tool.PluginID, tool.Name}] = tool.Effect
			found = append(found, tool)
		}
	}
	b.mu.Lock()
	b.listed = listed
	b.mu.Unlock()
	return found
}

// CallTool runs the tool the plugin listed, once the plugin is known to hold the permission its
// effect needs.
func (b *ToolsBridge) CallTool(
	ctx context.Context, pluginID, name string, arguments []byte, asked map[string]string,
) (port.PluginToolResult, error) {
	b.mu.Lock()
	effect, known := b.listed[toolKey{pluginID, name}]
	b.mu.Unlock()
	if !known {
		return port.PluginToolResult{}, ErrUnknownTool
	}
	if err := b.host.CheckPluginPermission(pluginID, getToolPermission(effect)); err != nil {
		return port.PluginToolResult{}, fmt.Errorf("%w: %w", port.ErrPermissionDenied, err)
	}
	client, lease, err := openPlugin(ctx, b.host, pluginID, numenv1connect.NewPluginToolServiceClient)
	if err != nil {
		return port.PluginToolResult{}, readActivationError(err)
	}
	defer func() { _ = lease.Close() }()

	res, err := client.CallTool(ctx, connect.NewRequest(&v1.CallToolRequest{Name: name, Arguments: arguments, Asked: asked}))
	if err != nil {
		return port.PluginToolResult{}, readToolError(ctx, err)
	}
	switch res.Msg.GetError() {
	case v1.ToolErrorCode_TOOL_ERROR_CODE_UNSPECIFIED:
	case v1.ToolErrorCode_TOOL_ERROR_CODE_UNKNOWN_TOOL:
		return port.PluginToolResult{}, ErrUnknownTool
	case v1.ToolErrorCode_TOOL_ERROR_CODE_BAD_ARGUMENTS:
		return port.PluginToolResult{}, ErrBadToolArguments
	}
	result := res.Msg.GetResult()
	return port.PluginToolResult{
		Text:       result.GetText(),
		Structured: result.GetStructured(),
		IsRefusal:  result.GetIsRefusal(),
	}, nil
}

// ListPluginTools lists the tools of one plugin, whatever permissions it holds, holding its
// reference for the length of the call. A plugin that cannot be started or reached is an error.
func (b *ToolsBridge) ListPluginTools(ctx context.Context, pluginID string) ([]port.PluginTool, error) {
	client, lease, err := openPlugin(ctx, b.host, pluginID, numenv1connect.NewPluginToolServiceClient)
	if err != nil {
		return nil, readActivationError(err)
	}
	defer func() { _ = lease.Close() }()

	res, err := client.ListTools(ctx, connect.NewRequest(&v1.ListToolsRequest{}))
	if err != nil {
		return nil, readToolError(ctx, err)
	}
	tools := make([]port.PluginTool, 0, len(res.Msg.GetTools()))
	for _, wire := range res.Msg.GetTools() {
		effect, isKnown := readToolEffect(wire.GetEffect())
		if !isKnown || wire.GetName() == "" {
			continue
		}
		tools = append(tools, port.PluginTool{
			PluginID:     pluginID,
			Name:         wire.GetName(),
			Title:        wire.GetTitle(),
			Description:  wire.GetDescription(),
			InputSchema:  wire.GetInputSchema(),
			OutputSchema: wire.GetOutputSchema(),
			Effect:       effect,
			Step:         readToolStep(wire.GetStep()),
			Servers:      readToolServers(wire.GetServers()),
		})
	}
	return tools, nil
}

// readToolError sorts the failure of a call to a tools plugin: the end of the caller's context is
// the caller's own, a plugin that cannot be reached is port.ErrUnavailable, and anything else it
// said is returned as it is.
func readToolError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	var failure *connect.Error
	if !errors.As(err, &failure) {
		return fmt.Errorf("%w: %w", port.ErrUnavailable, err)
	}
	switch failure.Code() {
	case connect.CodeUnavailable, connect.CodeUnimplemented:
		return fmt.Errorf("%w: %w", port.ErrUnavailable, err)
	default:
		return fmt.Errorf("tool rpc: %w", err)
	}
}

// getToolPermission is the permission a plugin must hold for a tool of the effect.
func getToolPermission(effect port.ToolEffect) string {
	if effect == port.ToolWrites {
		return port.PermissionVaultWrite
	}
	return port.PermissionVaultRead
}

func readToolEffect(wire v1.ToolEffect) (port.ToolEffect, bool) {
	switch wire {
	case v1.ToolEffect_TOOL_EFFECT_READS:
		return port.ToolReads, true
	case v1.ToolEffect_TOOL_EFFECT_WRITES:
		return port.ToolWrites, true
	case v1.ToolEffect_TOOL_EFFECT_UNSPECIFIED:
	}
	return 0, false
}

func readToolServers(wire []v1.ToolServer) []port.ToolServer {
	servers := make([]port.ToolServer, 0, len(wire))
	for _, one := range wire {
		switch one {
		case v1.ToolServer_TOOL_SERVER_FULL:
			servers = append(servers, port.ToolServerFull)
		case v1.ToolServer_TOOL_SERVER_READING:
			servers = append(servers, port.ToolServerReading)
		case v1.ToolServer_TOOL_SERVER_REVIEW:
			servers = append(servers, port.ToolServerReview)
		case v1.ToolServer_TOOL_SERVER_UNSPECIFIED:
		}
	}
	return servers
}

func readToolStep(wire v1.ToolStep) port.ToolStep {
	switch wire {
	case v1.ToolStep_TOOL_STEP_SEARCH:
		return port.ToolStepSearch
	case v1.ToolStep_TOOL_STEP_REMOVE:
		return port.ToolStepRemove
	case v1.ToolStep_TOOL_STEP_MOVE:
		return port.ToolStepMove
	case v1.ToolStep_TOOL_STEP_UNSPECIFIED:
	}
	return port.ToolStepByEffect
}
