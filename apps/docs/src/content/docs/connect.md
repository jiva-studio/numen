---
title: MCP server
description: The Model Context Protocol Server plugin serves your vault to any agent that supports MCP — where it listens, how to set its token, how to change the port and how to turn it off.
---

The [panel's agent](/agent/) is not the only one that can work in your vault. The plugin **Model Context Protocol Server** serves the vault's tools over MCP while the window is open, so an assistant you already use — in your terminal, in your editor — can search your notes, read them, write them and connect them. The application answers no questions from outside; it asks agents from inside and serves the vault to them through this plugin.

## Where it listens

```
http://127.0.0.1:3333/mcp
```

That address is the loopback interface: nothing outside your computer can reach it. While the plugin is enabled, Numen listens on the port of the plugin's **Port** setting, on its page in the settings (3333 by default, a number from 1024 to 65535), and starts the plugin when an agent connects. A changed port takes effect at once. The server follows the vault the window is showing.

## Setting the token

The server answers only a request that carries the plugin's token. Type a token of your choice, a long random string, into the plugin's **Token** setting on its page in the settings. The setting is empty until you do, and while it is empty the server answers nothing. A changed token takes effect at once, and an agent configured with the old one is refused.

The server also refuses a request sent by a web page that is not served from your own computer.

## What to point your agent at

Add the address and the token to your agent's MCP configuration, with your token in place of `<token>`. The transport is streamable HTTP:

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

## What an agent can do through it

The same four families the panel's agent has — [notes, links, documents and vaults](/agent/#what-it-can-do). It works on the vault the window is showing, and on no other.

There is no tool that erases a vault from disk. Removing a folder needs your confirmation in the window.

## Turning it off

Disable the plugin on its page in the settings. The port is closed while the plugin is disabled.

Any program running as you can reach the port while it is open. A program that has the token is granted everything the tools grant.
