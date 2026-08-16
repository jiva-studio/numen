# ADR-0025: A client talks to the core through a schema

- **Status:** Accepted
- **Date:** 2026-08-16
- **Applies to:** the product as a whole
- **Related:** ADR-0014, ADR-0020, ADR-0023

## Context

ADR-0014 decided one binary: the core is a package compiled into whatever runs
it, with no daemon and nothing listening on a socket. That still holds. Nothing
in this repository opens a port.

It also said the protocol layer stays empty until something outside the binary
needs to talk to the core. That sentence was written when the only entry point
was a command line, where the question does not arise: a command calls a
function. A client is not a command. The desktop window is a webview, and
JavaScript cannot call a Go function — the boundary is there because the two
sides are written in different languages, not because they are in different
processes.

So the choice ADR-0014 made, and made correctly, was between one process and
two. This decision is about what a client and the core say to each other, which
is a question the moment there is a client at all.

There will be more than one. The mobile client is a webview as well, and a web
client — reaching a core that does run somewhere else — is a plausible third.
They differ in how far the message travels and in nothing else.

## Decision

### The contract belongs to the core, not to a client

One `.proto` file describes what can be asked and what comes back. Every client
is generated from it, and the core answers it. A client is a consumer of the
contract and never the author of one: two clients that each grew their own
vocabulary would make the core answer to two, and the second one would be a
translation of the first written by hand.

The Go types and the TypeScript types are both generated. Neither side
hand-writes a type the schema already describes, so a field renamed on one side
fails a build rather than emptying a box on screen.

### The transport is the client's business

Today's clients are inside the binary, and the generated handler is given to
the webview's own asset server: requests are answered in-process, no socket is
opened, and nothing outside the binary can reach the core. What ADR-0014
refused is still refused.

A client that runs elsewhere is the case ADR-0014 anticipated. When one
arrives, what changes is where the bytes go — the questions, the answers and
the generated code do not.

### A change is pushed, not polled

A client is told when the vault changes, over a stream the schema declares
(ADR-0023). Asking on a timer is the alternative, and it either misses the
budget or spends the machine.

### The schema is checked where it can fail

Lint on every change, a breaking-change check against the branch being merged
into, and the generated output committed and regenerated in CI, so a schema and
its output cannot disagree in a merge.

### What travels is what a client asks and is answered

The schema is not a second model of the vault. It carries questions and their
answers. What the person wrote and what the application worked out do not
arrive looking alike.

## Consequences

**Positive**

- A second client inherits the contract instead of growing its own.
- A change to the vault reaches a client without it asking.
- The two languages disagree at build time, in one place, by name.
- Moving a client out of the binary later is a transport change, not a redesign.

**Negative**

- A schema to design, version and debug: the cost ADR-0014 named as the reason
  to wait. It is paid now because a client makes it unavoidable, not because a
  second process made it worthwhile.
- `modules/libs/` was to stay empty until two applications genuinely shared
  code. It holds the schema with one consumer today and a second one planned,
  which is a bet rather than an observation.
- The toolchain grows two generators, and a build that skips them is a build
  against yesterday's contract.

## Alternatives considered

**The webview toolkit's own bindings.** Rejected: they expose Go methods
directly, so the contract is whatever the methods happen to be that day. There
is nothing to lint, nothing to check for breaking changes, and nothing a second
client could be generated from.

**Hand-written messages over the same handler.** Rejected: the two sides drift,
and the first anyone hears of it is an empty field in the interface. This is the
failure the schema exists to make impossible.

**A socket and a separate process, now.** Rejected by ADR-0014, and nothing
here changes that reasoning: independent lifecycles buy nothing yet, and the
boundary being crossed today is between two languages rather than two machines.
