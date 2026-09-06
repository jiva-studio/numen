# An agent reaches the vault through tools

- **Status:** Accepted
- **Date:** 2026-08-25
- **Applies to:** `modules/libs/core` — `adapter/mcp`
- **Related:** [A hexagonal core in Go](0004-a-hexagonal-core-in-go.md), [A client is generated from the protocol](0005-a-client-is-generated-from-the-protocol.md), [The vault is watched](0009-the-vault-is-watched.md), [The application writes to the vault](0017-the-application-writes-to-the-vault.md), [One process, one lifetime](0020-one-process-one-lifetime.md), [The agent this application starts is a port](0022-the-agent-this-application-starts-is-a-port.md)

## Context

A language model working on someone's behalf reads a vault, changes it, and draws nothing. It arrives speaking a protocol of its own and expects to reach a server. The person watches what it does in the window they already have open.

## Decision

### An agent is a word of its own, and so is a tool

An **agent** is a program that acts on a vault on a person's behalf, through tools. A **tool** is one operation it can call.

Neither is a client. A client is generated from the protocol and draws what it is told. An agent is written by someone else and acts.

### The tool endpoint is a driving adapter inside the window

`adapter/mcp` is compiled into the window binary and listens on a port. It knows nothing of the webview toolkit: it is handed what the window has open, and it names the schema for the reason a refusal carries. It consumes the ports the core declares, as the command line does.

```mermaid
graph TD
    OA["an agent a person configured"]
    PA["the panel's agent<br/>started by this window"]

    subgraph win["the window binary"]
        direction TB
        G["token compared in constant time<br/>Origin checked"]
        M["adapter/mcp<br/>the tools"]
        U["core/usecase"]
        P["core/port"]
    end

    F["a file in the vault"]
    W["the watcher"]
    I["the index"]
    C["the window's client"]

    OA --> G
    PA --> G
    G --> M
    M --> U
    U --> P
    P --> F
    F --> W
    W --> I
    I --> C
    I -.a tool that writes returns here.-> M
```

The agent this window starts for the panel is a port of its own, and reaches the vault through the same endpoint as any other.

### The tool surface is authored

The tools are written by hand. The schema is shaped for a window — one focus, one neighbourhood, a stream of what changed. A tool surface is shaped for a model: few tools, named for what they do, described in prose the model reads before choosing. Every surface consumes the same use cases.

### Three surfaces, one core

Each family of tools registers its reading half and its writing half separately, and a surface is a choice of halves. `adapter/mcp` builds three:

- **The whole surface** is every tool the vault has, and the editor window serves it. That window is where a person asks for the vault to be changed, and the endpoint it puts up is the one an agent a person configured themselves reaches, with the vault's writers on it. The set is exact, so a tool added to the server is a tool this window is knowingly given. → `TestTheWindowAPersonWritesInServesEveryToolTheVaultHas`
- **The reading surface** is every tool that reads and no writer at all. An agent answering from it changes nothing.
- **The reviewing surface** is what the window a person runs their cards in serves: everything that reads, and the cards of a deck. Nothing on it makes a deck or a stencil, and nothing on it writes a note, a link or a document.

The panel's child is narrowed by its allowance: it is told, tool by tool, which of the surface it may call, and under the mode it runs in a tool outside that list is refused.

### A call carrying names takes many; a call carrying a document's text takes one

Paths and links are written in a moment, so looking up twenty notes, moving twenty or joining twenty is one call. A note carries what the note says: several in one call means nothing reaches the vault until the last word of the last one, and a person watching a graph sees it move once, minutes late. Every note is worth a call of its own.

### A note has one address

A tool names a note by its path relative to the vault root — the same address the schema and the command line use. A note that can be named two ways is a note an agent has to keep two names for.

### One vault at a time, and the endpoint follows the window

Every tool works the vault the window is showing, and no other. `vault_list`, `vault_add`, `vault_rename`, `vault_forget` and `vault_open` are the list and what a person does to it, and a vault whose folder is gone is marked and stays on the list until somebody forgets it.

The window opening another vault stops the endpoint and serves it again on the one the swap ended on. One swap holds that at a time, so a second waits for the first.

### The port is opened where a person asks for it

An agent named for the panel puts the tools on a port, since that is how the agent this window starts reaches them. A person who runs an agent of their own asks for the port itself, by a setting. An installation asking for neither opens no port and mints no token: a port and a secret on somebody's machine belong to an installation that was asked for them.

### What is served, and to whom

The loopback interface, on a fixed port, by default. Any other address is allowed, and choosing one is said plainly at startup.

A token, always, compared in constant time. On loopback it is the second line of defence behind an unreachable port; anywhere else it is the only one. The `Origin` header is checked: a page open in a browser can reach a port on this machine, and is the one caller that arrives without being invited.

### The token is kept between launches, and the address is written beside it

An agent is configured once, by a person, in a file of its own. So the token is generated once and kept with the application's own state, and beside it goes the address the endpoint is answering on. Both files are written beside themselves and renamed over the top, at mode `0600`, and the address file is removed on the way out. A file left behind points the next agent at a dead port with a live-looking token.

### The tools ship in the binary a person runs

The binaries are split by build profile: a window links against the system's browser through cgo, and the command line is pure Go and cross-compiles. The tools go in a window, because they exist to be watched — `numen` serves the whole surface, and `numen-flashcards` the reviewing one. `numen-cli` is the developer's and automation's tool, and nothing of the tools is linked into it.

### A change reaches the window the way any other edit does

A tool writes a file. The watcher sees it, a refresh brings the index level, and every listening client is told. There is no path reserved for an agent, so there is one path to get right and it is the one already tested.

**A tool that writes returns only once the index is level again.** An agent that creates a note and searches for it in the next breath finds it.

The tools themselves, family by family, and the limits on what one call may carry, are in [`../agents.md`](../agents.md).

## Consequences

- The repository opens a port where an installation asks for one, and what stands in front of it is a token and an origin check.
- Three surfaces to keep true as the core changes. Nothing generates them against the schema; only tests hold them there.
- An agent with write tools can damage a vault as thoroughly as a person can. The damage is visible and reversible, which is weaker than prevention and is what is available.
- A tool that writes waits for a refresh, so a write costs what indexing that note costs.
- The tools are absent from the binary that cross-compiles.

## Alternatives considered

**A separate process serving the tools.** Rejected: the change an agent makes has to appear in the open window, so a second process would still have to reach the first, it would be a second writer against one index, and it is a second thing for a person to start.

**Generating the tools from the schema.** Rejected: it produces a tool surface shaped by what a window needs to draw, and it makes both surfaces worse for the reader each was written for.

**Serving the generated handler on the port and teaching an agent the schema.** Rejected: an agent that must first be taught a schema spends its context learning one.
