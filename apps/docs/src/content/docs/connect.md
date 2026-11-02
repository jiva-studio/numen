---
title: MCP server
description: The window serves your vault to any agent that supports MCP — where it listens, what it needs, and how to turn it off.
---

The [panel's agent](/agent/) is not the only one that can work in your vault. While the window is open it serves the same tools over MCP, so an assistant you already use — in your terminal, in your editor — can search your notes, read them, write them and connect them.

The port is open whenever the panel has an agent to use, which is the ordinary installation. If you turn the panel's agent off, the port closes too, unless you ask for the port on its own:

```json
{ "agent": { "use": "", "serve_tools": true } }
```

An installation with neither setting opens no port and creates no token.

## Where it listens

```
http://127.0.0.1:7717/mcp
```

That address is the loopback interface: nothing outside your computer can reach it. The port is fixed on purpose, because it goes into a configuration file you write once.

Every request has to send the token as a bearer credential. Without it the vault returns _this vault is not open to you_.

## What to point your agent at

Beside [the settings file](/install/#where-numen-keeps-its-own-files), the window writes **`agents.json`** while it is running:

```json
{
  "url": "http://127.0.0.1:7717/mcp",
  "token": "…"
}
```

That is the whole configuration: the address and the token, ready to be copied into your agent's configuration. The transport is streamable HTTP, and the header is the ordinary one:

```
Authorization: Bearer <token>
```

The token is kept in `agents.token` and **survives a restart**. A new token at each launch would break the line in your agent's configuration every time you closed the window.

`agents.json` is removed when the window closes: while it is there, the vault is being served.

## What an agent can do through it

The same four families the panel's agent has — [notes, links, documents and vaults](/agent/#what-it-can-do). It works on the vault the window is showing, and on no other.

There is no tool that erases a vault from disk. Removing a folder needs your confirmation in the window.

## Turning it off

|                                          |                                                                                  |
| ---------------------------------------- | -------------------------------------------------------------------------------- |
| `numen -no-mcp`                          | starts the window with the server off. Nothing is served and no file is written. |
| `numen -mcp-addr <address>`              | changes the address.                                                             |
| `agent.use: ""` and `serve_tools: false` | no port, permanently, without a flag on every launch.                            |

Any address other than a loopback address makes your vault reachable from the network. Then the token is the only protection against anyone who can reach that port.

Two things to know before you share the token: it grants everything the tools grant, and the file that stores it can be read by any program running as you.
