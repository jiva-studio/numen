# Model Context Protocol Server

Let external AI apps—such as Claude Desktop, Cursor, or Windsurf—read, search, and work with your open Numen vault.

<!-- more -->

## What You Can Do

- **Use your vault from other AI apps**: Ask an external AI assistant to search your notes, read passages from your books, or create new notes while you work in another app.
- **Same vault**: Changes and searches apply to the vault that is open in Numen.

## How to Connect

While the plugin is enabled, Numen listens on `127.0.0.1:3333` and starts the MCP server when a client connects:

- **Streamable HTTP endpoint**: `http://127.0.0.1:3333/mcp`

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
