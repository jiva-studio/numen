# Model Context Protocol Server

Let external AI apps—such as Claude Desktop, Cursor, or Windsurf—read, search, and work with your open Numen vault.

<!-- more -->

## What You Can Do

- **Use your vault from other AI apps**: Ask an external AI assistant to search your notes, read passages from your books, or create new notes while you work in another app.
- **Always in sync**: Changes and searches happen directly in the vault you currently have open in Numen.

## How to Connect

When enabled, the MCP server listens locally on `127.0.0.1` using the configured **Port** (default `3333`):

- **Streamable HTTP endpoint**: `http://127.0.0.1:3333/mcp`
- **SSE endpoint**: `http://127.0.0.1:3333/sse`

### Example JSON Configuration (`mcpServers`)

Add the following entry to your client's MCP configuration file (such as `claude_desktop_config.json`, `.cursor/mcp.json`, or `~/.codeium/windsurf/mcp_config.json`):

```json
{
  "mcpServers": {
    "numen": {
      "url": "http://127.0.0.1:3333/mcp"
    }
  }
}
```

For clients that require the legacy SSE transport, use `"http://127.0.0.1:3333/sse"` as the URL.
