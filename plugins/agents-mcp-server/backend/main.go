// Command agent-mcp-server serves the MCP server as a plugin process.
package main

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/agents-mcp-server/backend/service"
)

func main() {
	server.Run(service.Mount)
}
