# ADR-0031: The agent this application starts, and what it may reach

- **Status:** Accepted
- **Date:** 2026-08-17
- **Applies to:** `modules/apps/desktop`
- **Related:** ADR-0013, ADR-0024, ADR-0026, ADR-0027

## Context

ADR-0026 decided that an agent reaches the vault through tools, and described one
direction: the application serves a port, and a person pastes one line into the
configuration of an agent they run themselves. Whose agent it is, and what else
it can do, was not this application's business.

The panel changed that. A person types a question into the window, and the
application starts an agent to answer it. It chooses the program, the model, the
tools, the working directory and the environment. Every one of those is a
decision, and none of them was written down.

What the panel makes different is not the tools — those are ADR-0026's, and are
the same tools either way. It is that nobody is watching. A person who runs their
own agent in a terminal sees each thing it reaches for and answers a prompt. A
person reading a narrow panel beside their notes sees a line of text, and is in
no position to be asked whether a command may run.

The agent this application starts today is Claude Code, and it is one of several
it could start: an agent is a port, and starting a program is an adapter behind
it. What follows is about the agent numen starts, whichever it is, and where a
detail belongs to Claude Code alone it is said so.

Claude Code reads configuration from this machine: hooks, skills, standing
instructions, plugins, custom agents. A hook is a shell command that program runs
itself — not a tool a model calls — so nothing about which tools an agent may use
has any bearing on it.

## Decision

### The agent is started by this application, and everything it may reach is named here

Not inherited, not discovered. The command line, the environment and the working
directory are constructed, and what is absent is absent because this decision
says so.

### It reaches this vault through this vault's tools, and the web

The tools it brings are `WebSearch` and `WebFetch`. Every other built-in — a
shell, a file writer, a file reader, a subagent — is disabled by naming the whole
set the agent may bring.

A shell and a file writer reach past every use case: into the vault under a name
the index does not know, and into the rest of the machine. The web tools reach
neither, and looking something up is part of writing a note about it.

Approving the tools an agent may use is a separate thing from which tools exist.
Both are said: the set is named, and this vault's tools are approved ahead of the
run so that nothing is asked about them.

**The web reaches outward as well as in.** An agent holding a vault's contents
and a fetcher can send them somewhere. This is accepted: the same agent is
trusted with the notes themselves, and a person who does not want that turns the
panel off.

### It reads nothing this machine holds for it

Hooks, skills, standing instructions, plugins and custom agents are all refused
by default. A question typed into a panel is not asking for a shell command to
run, and a hook is a shell command.

A person may ask for their own back, as a setting. Then what they configured for
themselves is read, and nothing else.

**The sources of configuration are named, not turned off.** Refusing every
customisation wholesale refuses this vault's own tools with them: they arrive on a
command line and are read as a customisation like any other. An agent that loses
them answers from what the model already knows and mentions the vault was missing
after the answer — which is worse than saying nothing, and is what naming the
sources avoids.

**What a vault carries is refused either way.** A vault is a folder that arrives
from elsewhere — synced, cloned, shared, restored — and a configuration file
inside one is a folder naming commands for this machine to run. The working
directory is the vault, so this is not hypothetical.

### Which agent answers is a setting, and each is named

An agent is a port: a task goes in, steps come out, and everything above that
interface sees one agent and never which. What starts a program, writes its
command line and reads what it prints is an adapter behind that port, and reading
one program's stream is that adapter's business alone.

So the setting names the agent, and the file carries a section for each that
could: which model it answers on, how many steps it may take, and whatever else
that one is reached by. Today one adapter is written — Claude Code — and a second
is a second adapter, not a change to anything above it. A local model served over
an OpenAI-compatible interface, an agent of another maker, a program somebody
writes for themselves: each is a section and an adapter.

A section is kept whether it is the one in use or not, so trying another for an
afternoon does not cost the settings of the one before.

Empty names no agent, and the panel says there is none.

### The environment is inherited, less what describes somebody else's session

Variables that describe a Claude Code session belong to whoever is running one.
The window is not running one. They are dropped: a session's identity, its
socket, its messaging token, and how hard it was told to think.

**Everything else passes**, and that is the honest shape of it. What says how to
reach a model passes, which is the point — and so does what says *which server*
sees the notes, which configuration root is read, which proxy the traffic goes
through, and which program the name `claude` resolves to. A denylist of nine names
is not a built environment; it is an inherited one with nine holes plugged.

The shape that matches the claim is an allowlist: a named set, and nothing else.
It is not written, and until it is, this section says what is true rather than
what was intended.

### What it is told about the person

That it is answering inside the application the notes are kept in, beside the
note in front of them; that a note is called by its title and never by a path;
and that the answer is read in a narrow panel.

## Consequences

**Positive**

- What the agent may reach is one decision in one file, and a test asserts each
  part of it.
- A vault from anywhere is safe to add: nothing inside it runs.
- The panel's agent cannot write outside the vault, and cannot write inside it
  except through a use case that keeps the index level with the file.

**Negative**

- **The person loses their own agent's habits inside the panel.** Their skills,
  their standing instructions, their hooks: none of it applies. The setting gives
  it back, and turning it on runs their shell commands on every question.

- **A file the agent leaves in the vault is read by the person's own terminal.**
  The panel's agent does not read the vault's instructions in either mode — the
  sources it is given do not include the vault. But a note it writes stays in the
  folder, and the person's own agent, started in that folder, reads what is there.
  Where a note may be written is bounded to places a note may live and this
  application's own folders, so another tool's folder is refused; the vault's root
  is not, and a file named for what another tool reads can still land in it.

- **The bearer token is on the command line.** Any local process that can read
  `/proc` can read it, and it grants what the tools grant. The token file is
  0600 and this undoes that. It is a known hole with a known fix, and the fix is
  not in yet.

- **An agent whose tools did not arrive answers anyway.** The stream says the
  server was unreachable, and the run continues; the reason is reported after the
  answer. The posture intended is fail-closed and what is implemented is
  answer-then-explain.

- **A list of refused variables, and a list of named tools, are both lists.** The
  program they filter is somebody else's and changes without asking. Each release
  is a thing to check, and nothing checks it automatically.

- **Every agent added is these decisions made again.** What tools it brings, what
  it reads from this machine, what environment it is given: each adapter answers
  for itself, and nothing above the port can answer for it. A second adapter that
  forgets one of them is a hole with no compiler to catch it.

## Alternatives considered

**Serve the tools and let the person run their own agent, as ADR-0026 alone
allows.** Rejected as the only answer: a panel beside the notes is worth having,
and it cannot exist without the application starting something. ADR-0026 stands
for the other direction, which is unchanged.

**Let the agent bring everything, and rely on the permission prompt.** Rejected:
the prompt is a terminal's, and this has no terminal. Auto-approving in a panel
is approving nothing.

**Refuse the web as well.** Rejected: an agent writing about a subject wants to
look it up, and the same agent is already trusted with the notes.

**Sandbox the process — a container, a namespace, a seccomp filter.** Not
rejected, deferred. It is the answer that does not depend on a list being current
and does not depend on somebody else's flags. Nothing here is a substitute for
it; what is here is what one afternoon buys, and the sandbox is the thing to
build next.
