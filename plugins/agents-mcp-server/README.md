# MCP Server

Exposes Numen's vault and window tools over the Model Context Protocol (MCP) to external editors, IDEs, and coding agents.

<!-- more -->

## How It Works

- **External MCP endpoint**: Serves SSE, Streamable HTTP, and stdio MCP transports aggregating every tool declared by active Numen tool plugins.
- **Live vault routing**: Forwards incoming tool calls through `numen.v1.PluginToolService` on the host, executing against the currently active vault.
- **Supported clients**: Works with Claude Desktop, Cursor, Windsurf, Claude Code, Antigravity, and any MCP-compatible client.
