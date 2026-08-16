# ADR-0026: An agent reaches the vault through tools

- **Status:** Accepted
- **Date:** 2026-08-16
- **Applies to:** `modules/apps/desktop`
- **Related:** ADR-0014, ADR-0018, ADR-0023, ADR-0024, ADR-0025

## Context

ADR-0014 decided one binary and refused a daemon: nothing listens on a socket,
and the protocol layer stays empty until something outside the binary needs the
core. ADR-0025 restated that a year of code later — the clients are inside the
binary, the generated handler is answered in-process, and nothing outside can
reach the core.

What arrives now is not another client. It is a language model working on
someone's behalf: it reads the vault, changes it, and draws nothing. The person
watches the change appear in the window they already have open, which is the
whole point of letting it in.

Such a program already speaks a protocol — the Model Context Protocol — and
every program that speaks it expects to *reach* a server rather than be compiled
into one. So the condition ADR-0014 named has arrived, and it arrived from the
direction ADR-0014 did not anticipate: not a second client of ours, but a
stranger with its own protocol.

## Decision

### An agent is a word of its own

An **agent** is a program that acts on a vault on a person's behalf, through
tools. A **tool** is one operation it can call.

Neither is a client (ADR-0025). A client is generated from the schema and draws
what it is told; an agent is written by someone else, arrives speaking its own
protocol, and acts. Calling both by one word would hide exactly the difference
that this decision turns on.

### The server is inside the application, and the port is all that leaves it

The MCP server is a driving adapter compiled into the desktop application,
listening on a port. One binary is installed, one thing is started, and the
agent has somewhere to connect.

A separate process was the obvious shape and is the wrong one today. The change
an agent makes has to appear in the open window, so a second process would still
have to reach the first; it would be a second writer against one index, which
ADR-0018 forbids; and it would be a second thing for a person to remember to
start. In one process there is one writer, one lifetime, and nothing to
coordinate.

**This is reversible by construction.** The adapter knows nothing of the window,
of the webview toolkit, or of the schema — it consumes the ports the core
declares, as the command line does. A headless binary later is an entry point
and a transport, not a redesign. That is the same sentence ADR-0025 wrote about
its own clients, and it is true here for the same reason.

### The tools are their own vocabulary, and are not generated from the schema

ADR-0025 requires that a client consume the contract rather than author one. An
agent is not bound by that, and must not be: the two surfaces answer different
questions for different readers.

The schema is shaped for a window — one focus, one neighbourhood, a stream of
what changed. A tool surface is shaped for a model: few tools, named for what
they do, described in prose the model reads before choosing, batched because a
model pays for every round trip.

Generating either from the other would make both worse, and nothing is written
twice by keeping them apart: both consume the same use cases. What differs is
only the shape of the question. **Two surfaces, one core.**

### A note has one address, and the vault is located once

A tool names a note by its path relative to the vault root — the same address
the schema, the command line and the index use (ADR-0024). Not two addresses,
not an absolute path repeated in every answer; a note that can be named two ways
is a note an agent has to keep two names for.

Where the vault is on disk is said once, when the agent connects, and can be
asked for again. An agent that can open files joins the two itself; an agent
that cannot reads through a tool. Which of the two it is is the agent's
business, and never an assumption in the answer.

### What is served, and to whom

**The loopback interface, by default.** Any other address is allowed — a person
may want to reach their own notes from their own network — and choosing one is
said plainly at startup rather than discovered later.

**A token, always.** On loopback it is the second line of defence behind an
unreachable port; anywhere else it is the only one. The `Origin` header is
checked, because a page open in a browser can reach a port on the same machine
and is the one attacker who arrives without being invited.

**The token is kept between launches, and where to reach the vault is written
down.** An agent is configured once, by a person, in a file of its own; a token
minted at every start would break that line every time the application is
restarted, which is not a configuration at all. So the token is generated once
and kept with the application's own state, and beside it goes the address the
server is answering on, written the way everything else here is written —
beside itself and renamed over the top, readable by nobody else, and removed on
the way out. A file left behind points the next agent at a dead port with a
live-looking token.

### The binary that carries this is the one a person runs

There are two binaries, split by build profile rather than by function: the
window links against the system's browser through cgo and cannot be
cross-compiled, and the command line is pure Go and can. The tools go in the
first, because they exist to be watched.

So the binary a person installs and starts is `numen` — the window, with the
tools inside it — and the developer's and automation's tool is `numen-cli`. The
name a person types is the thing they meant to run.

### A change reaches the window the way any other edit does

A tool writes a file. The watcher sees it, a refresh brings the index level, and
every listening client is told (ADR-0023). There is no path reserved for the
agent, so there is one path to get right and it is the one already tested — the
same one that carries an edit made in another editor.

One thing follows that would otherwise surprise: **a tool that writes returns
only once the index is level again.** An agent that creates a note and searches
for it in the next breath finds it, rather than learning that the vault is
eventually consistent and working around it forever after.

### Nothing here changes who writes

One process, one writing connection, short transactions: ADR-0018 stands
unamended. That is a consequence of putting the server inside the application
rather than beside it, and it is the largest single reason to do so.

## Consequences

**Positive**

- A person installs one thing, starts one thing, and pastes one line into their
  agent's configuration.
- What the agent does is visible as it happens, through machinery that already
  exists and is tested.
- The index keeps one writer.
- Moving the server out of the binary later is an entry point and a transport.

**Negative**

- The repository now opens a port, which ADR-0014 refused and ADR-0025 restated
  as still refused. The refusal held while nothing outside needed the core; that
  is no longer the case, and what replaces it is not "a port is fine" but the
  paragraph above about which port, for whom, and behind what.
- A second surface to keep true as the core changes. It is not generated, so
  nothing checks it against the schema; only tests can.
- An agent with write tools can damage a vault as thoroughly as a person can.
  The answer is not to withhold the tools but to make the damage visible and
  reversible (ADR-0027), which is a weaker guarantee than prevention and is the
  one available.

## Alternatives considered

**A separate binary now.** Rejected for today, kept as the destination. The
binaries in this repository are split by build profile — the window links
against the system's browser through cgo and cannot be cross-compiled; the
command line is pure Go and can. An MCP server shares the command line's
profile, so a third binary buys no portability, while costing a second writer
and a second thing to start.

**stdio instead of a port.** This is how most MCP servers are launched: the
agent spawns the process and speaks over its pipes. It is the right transport
for the headless binary when it exists, and the wrong one now, because a
spawned process is a second process again. The adapter is written so that
serving over stdio is a file.

**Generating the tools from `vault.proto`.** Rejected above, and recorded here
because it is the tidy-looking mistake someone will propose later: one contract,
two consumers, no duplication. What it produces is a tool surface shaped by what
a window needs to draw.

**Teaching the agent the schema instead.** Serve the existing Connect handler on
the port and let the agent speak it. Rejected: an agent that must first be
taught a schema spends its context learning one, and MCP exists precisely so
that it does not have to.
