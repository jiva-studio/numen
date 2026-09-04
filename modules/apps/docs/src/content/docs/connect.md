---
title: MCP server
description: The window serves your vault to any agent that speaks MCP — where it listens, what it needs, and how to shut the door.
---

The [panel's agent](/agent/) is not the only one that can work in your vault. While the window is
open it serves the same tools over MCP, so an assistant you already use — in your terminal, in
your editor — can search your notes, read them, write them and connect them.

The port is open whenever the panel has an agent to answer with, which is the ordinary
installation. Turn the panel's agent off and the port goes with it — unless you ask for the port
on its own:

```json
{ "agent": { "use": "", "serve_tools": true } }
```

An installation asking for neither opens no port and mints no token at all.

## Where it listens

```
http://127.0.0.1:7717/mcp
```

That address is the loopback interface: nothing outside your computer can reach it. The port is
fixed on purpose, because it goes into a configuration file you write once.

Every request has to carry the token as a bearer credential; without it the vault answers *this
vault is not open to you*.

## What to point your agent at

Beside [the settings file](/install/#where-numen-keeps-its-own-files), the window writes
**`agents.json`** while it is running:

```json
{
  "url": "http://127.0.0.1:7717/mcp",
  "token": "…"
}
```

That is the whole configuration: the address and the token, ready to be copied into whatever
your agent reads. The transport is streamable HTTP, and the header is the ordinary one:

```
Authorization: Bearer <token>
```

The token is kept in `agents.token` and **survives a restart**. Minting a new one each launch
would mean the line in your agent's configuration stopped working every time you closed the
window, which is not a configuration file at all.

`agents.json` is removed when the window closes: while it is there, the vault is being served.

## What an agent can do through it

The same four families the panel's agent has — [notes, links, documents and
vaults](/agent/#what-it-can-do). It works whichever vault the window is showing, and no other.

There is no tool that erases a vault from disk. Taking a folder away is asked for in front of you.

## Shutting the door

| | |
| --- | --- |
| `numen -no-mcp` | starts the window with no door at all. Nothing is served and nothing is written down. |
| `numen -mcp-addr <address>` | moves it. |
| `agent.use: ""` and `serve_tools: false` | no port, for good, without a flag on every launch. |

Anything other than a loopback address opens your vault to the network, and then the token is
the only thing between your notes and whoever can reach that port.

Two things worth knowing before you hand the token around: it grants everything the tools grant,
and the file it lives in is readable by anything running as you.
