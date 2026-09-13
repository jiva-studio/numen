# The agent this application starts is a port

- **Status:** Accepted
- **Date:** 2026-08-25
- **Applies to:** `modules/libs/core`, `modules/apps/desktop`
- **Related:** [A hexagonal core in Go](0004-a-hexagonal-core-in-go.md), [One process, one lifetime](0020-one-process-one-lifetime.md), [An agent reaches the vault through tools](0021-an-agent-reaches-the-vault-through-tools.md)

## Context

A person types a question into the panel beside their notes, and the application answers it by starting a program on this machine. The program is somebody else's, it is released without asking, and it arrives carrying a shell, a file writer and whatever the person has configured it with.

The folder it is started in is the vault, and a vault arrives from elsewhere: synced, cloned, shared, restored.

## Decision

### The agent is a port

A task goes in and steps come out. Everything above that interface sees one agent and never which. Starting a program, writing its command line and reading what it prints is an adapter behind the port, and one program's stream is that adapter's business alone.

Which agent answers is a setting, and each carries a section of its own; a section is kept whether it is the one in use or not. Empty names no agent, and the panel says there is none. Whether the tools go on a port is a setting of its own, so an installation that names none opens one only where it asks for one.

### What that agent may reach is constructed here

The command line, the environment and the working directory are built by this application. What is absent is absent because this decision says so.

### It brings a web search, and no other built-in

The set of built-in tools is named in full, so every other one is absent: no shell, no file reader, no file writer, no subagent. A shell and a file writer reach into the vault under a name the index does not know, and into the rest of the machine. A search reaches neither, and looking something up is part of writing a note about it.

A web *fetch* is absent, and the reason is not the same. A search cannot be told where to send anything: an injected note names no recipient, and what comes back is read by the model that was already reading the note. A fetch takes its address from the text, so a note somebody synced chooses where the run reaches — and the run is approved ahead of time, so nothing intervenes. That makes it a channel out of the vault rather than a way into the web, and the proofreader in the same package already runs with both refused.

This vault's tools are approved ahead of the run, so nothing is asked about them while it is on.

### Configuration sources are named

Hooks, skills, standing instructions, plugins and custom agents are refused by default: the sources of configuration the agent may read are named one by one, and by default the list is empty. A setting gives back the person's own, and only theirs.

Each source is named on its own. This vault's tools arrive on a command line and are read as a customisation like any other, so a blanket refusal takes them with it.

### What a vault itself carries is refused in either mode

A vault is a folder that arrives from elsewhere, and a configuration file inside one is a folder naming commands for this machine to run. The working directory is the vault.

The tools this agent is given, the settings keys, the environment variables that are dropped, what the agent is told about the person, and the security holes still open are in [`../agents.md`](../agents.md).

## Consequences

- Everything above the port sees one agent and never which.
- Every agent adapter added answers these questions again, and nothing above the port can answer for it.
- The named set of built-ins tracks a program released by somebody else, and each release of it is something to read.
- The person's own hooks and skills are one setting away, and what that setting admits is whatever this machine holds.
- The agent works in a folder that arrived from elsewhere, and what it writes lands where the person's own tools read.

## Alternatives considered

**Turning every customisation off with the single switch the program offers.** Rejected: this vault's own tools arrive as a customisation and go off with it, and an agent with no tools answers from the model alone.

**Letting the agent keep its shell and its file writer, bounded to the vault.** Rejected: a shell writes into the vault under a name the index does not know, and the bound is a promise made by the program it bounds.

**Calling a model directly and running the steps here.** Rejected: the step loop, the tool dispatch and the credentials become this application's, and the agent a person already runs is then one it cannot start.
