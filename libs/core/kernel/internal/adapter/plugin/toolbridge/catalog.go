package toolbridge

import (
	"context"
	"log/slog"
	"slices"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// Catalog lists and calls the tools of the plugins for a plugin that reaches them through the host.
type Catalog struct {
	plugins plugins
	clock   machineport.Clock
	logger  *slog.Logger
}

// NewCatalog is the tools the plugins of the host offer. The answers of the plugins are kept in memory
// and in files. Each call is written to the logger at the info level, timed by clock.
func NewCatalog(host Host, files ListingFiles, clock machineport.Clock, logger *slog.Logger) *Catalog {
	return &Catalog{plugins: plugins{host: host, listings: newListings(files)}, clock: clock, logger: logger}
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
	started := c.clock.Now()
	record := callRecord{callerID: callerID, call: call}
	allowed := c.listAllowedTools(ctx, callerID)
	index := slices.IndexFunc(allowed, func(one tool) bool { return one.spec.GetName() == call.GetName() })
	if index < 0 {
		record.response = &v1.CallToolResponse{Error: v1.ToolErrorCode_TOOL_ERROR_CODE_UNKNOWN_TOOL}
	} else {
		record.pluginID = allowed[index].pluginID
		record.response, record.err = c.plugins.callTool(ctx, allowed[index], call)
	}
	record.duration = c.clock.Now().Sub(started)
	logCall(ctx, c.logger, record)
	return record.response, record.err
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
