package toolbridge

import (
	"context"
	"slices"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

// Catalog lists and calls the tools of the plugins for a plugin that reaches them through the host.
type Catalog struct {
	plugins plugins
}

// NewCatalog is the tools the plugins of the host offer. The answers of the plugins are kept in the
// folder dir returns, and in memory only while it returns an empty path.
func NewCatalog(host Host, dir func() string) *Catalog {
	return &Catalog{plugins: plugins{host: host, listings: newListings(dir)}}
}

// ListTools lists the tools the caller is allowed: those the other plugins offer, for which the
// caller has the effect permission and all required permissions. The tools of one plugin are
// listed together, in the order of plugin ids.
func (c *Catalog) ListTools(ctx context.Context, callerID string) []*v1.PluginTool {
	allowed := c.listAllowedTools(ctx, callerID)
	specs := make([]*v1.PluginTool, 0, len(allowed))
	for _, one := range allowed {
		specs = append(specs, one.spec)
	}
	return specs
}

// CallTool runs the call on the plugin that lists the tool, with the context and in the vault the
// call carries. The tool is found in the kept listings, so only the plugins not listed yet and the
// one that runs the tool are started. A tool the caller is not allowed is answered as unknown.
func (c *Catalog) CallTool(ctx context.Context, callerID string, call *v1.CallToolRequest) (*v1.CallToolResponse, error) {
	allowed := c.listAllowedTools(ctx, callerID)
	index := slices.IndexFunc(allowed, func(one tool) bool { return one.spec.GetName() == call.GetName() })
	if index < 0 {
		return &v1.CallToolResponse{Error: v1.ToolErrorCode_TOOL_ERROR_CODE_UNKNOWN_TOOL}, nil
	}
	return c.plugins.callTool(ctx, allowed[index], call)
}

func (c *Catalog) listAllowedTools(ctx context.Context, callerID string) []tool {
	permissions := c.plugins.getPermissions(callerID)
	all := c.plugins.listTools(ctx, callerID)
	allowed := make([]tool, 0, len(all))
	for _, one := range all {
		if hasPermission(permissions, one.spec.GetEffect()) && hasAll(permissions, one.spec.GetRequires()) {
			allowed = append(allowed, one)
		}
	}
	return allowed
}

func hasAll(permissions, required []string) bool {
	for _, one := range required {
		if !slices.Contains(permissions, one) {
			return false
		}
	}
	return true
}
