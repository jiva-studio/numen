package service

import (
	"github.com/jiva-studio/numen/plugins/agents-mcp-server/backend/hostserver"
	"github.com/jiva-studio/numen/plugins/agents-mcp-server/backend/internal/adapter/diagnostics"
	"github.com/jiva-studio/numen/plugins/agents-mcp-server/backend/internal/adapter/mcphttp"
	"github.com/jiva-studio/numen/plugins/agents-mcp-server/backend/usecase/exposetools"
)

// endpoint is the MCP server with its HTTP transport.
type endpoint struct {
	*mcphttp.Server
	host *hostserver.Server
}

func newEndpoint(token string, client exposetools.Host) *endpoint {
	host := hostserver.New(client, diagnostics.ReportSyncError)
	return &endpoint{Server: mcphttp.New(host.MCP, token), host: host}
}
