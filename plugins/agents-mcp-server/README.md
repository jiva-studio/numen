# Model Context Protocol Server

Let external AI apps—such as Claude Desktop, Cursor, or Windsurf—read, search, and work with your open Numen vault.

<!-- more -->

## What You Can Do

- **Use your vault from other AI apps**: Ask an external AI assistant to search your notes, read passages from your books, or create new notes while you work in another app.
- **Same vault**: Changes and searches apply to the vault that is open in Numen.

## How to Connect

While the plugin is enabled, Numen listens on `127.0.0.1` at the plugin's **Port** setting (default `3333`), listens on the new port when the setting changes, and starts the MCP server when a client connects:

- **Streamable HTTP endpoint**: `http://127.0.0.1:3333/mcp`

A client authenticates with the header `Authorization: Bearer <token>`, where `<token>` is the value of the plugin's **Token** setting on its page in the settings. Numen generates the token, 64 hexadecimal characters, when it first loads the enabled plugin; copy it from the setting into the client configuration. You can replace it with a token of your own, and while the setting is empty the server returns 401 for every request. A changed token takes effect at once. A request sent from a web page whose origin is not `localhost` or a loopback address gets 403.

### Example JSON Configuration (`mcpServers`)

Add the following entry to your client's MCP configuration file (such as `claude_desktop_config.json`, `.cursor/mcp.json`, or `~/.codeium/windsurf/mcp_config.json`), with your token in place of `<token>`:

```json
{
  "mcpServers": {
    "numen": {
      "type": "http",
      "url": "http://127.0.0.1:3333/mcp",
      "headers": { "Authorization": "Bearer <token>" }
    }
  }
}
```
