package toolsbridge

import (
	"context"
	"slices"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// Catalog lists and calls the tools of the plugins for a plugin that serves them to programs
// outside Numen. No question was asked in Numen, so every call is made with no context.
type Catalog struct {
	plugins plugins
	server  Server
}

// NewCatalog is the tools the plugins of the host offer on the server.
func NewCatalog(host Host, server Server) *Catalog {
	return &Catalog{plugins: plugins{host: host}, server: server}
}

// ListTools lists the tools the caller is allowed: those the other plugins offer on the server,
// whose effect the caller holds the permission of as well.
func (c *Catalog) ListTools(ctx context.Context, callerID string) []*v1.PluginTool {
	allowed := c.listAllowed(ctx, callerID)
	specs := make([]*v1.PluginTool, 0, len(allowed))
	for _, one := range allowed {
		specs = append(specs, one.spec)
	}
	return specs
}

// CallTool runs the tool of the name on the plugin that lists it, in the vault given. A tool the
// caller is not allowed is answered as unknown.
func (c *Catalog) CallTool(
	ctx context.Context, callerID, name string, arguments []byte, vault domain.VaultID,
) (*v1.CallToolResponse, error) {
	allowed := c.listAllowed(ctx, callerID)
	index := slices.IndexFunc(allowed, func(one tool) bool { return one.spec.GetName() == name })
	if index < 0 {
		return &v1.CallToolResponse{Error: v1.ToolErrorCode_TOOL_ERROR_CODE_UNKNOWN_TOOL}, nil
	}
	return c.plugins.callTool(ctx, allowed[index], arguments, callScope{vault: vault})
}

// listAllowed is the tools of the other plugins whose effect the caller holds the permission of.
func (c *Catalog) listAllowed(ctx context.Context, callerID string) []tool {
	var permissions []string
	for _, info := range c.plugins.host.GetPluginInfos() {
		if info.ID == callerID {
			permissions = info.Permissions
		}
	}
	all := c.plugins.listTools(ctx, c.server, callerID)
	allowed := make([]tool, 0, len(all))
	for _, one := range all {
		if hasPermission(permissions, one.spec.GetEffect()) {
			allowed = append(allowed, one)
		}
	}
	return allowed
}
