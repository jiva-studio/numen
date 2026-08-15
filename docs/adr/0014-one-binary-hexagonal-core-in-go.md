# ADR-0014: One binary, hexagonal core in Go

- **Status:** Accepted
- **Date:** 2026-08-15
- **Related:** ADR-0001, ADR-0002, ADR-0013

## Context

The first tool walks a vault and builds the index. Before it is written, the
shape of the code has to be settled: what the core is allowed to know, how it is
started, and what it is compiled into. These are cheap to decide now and
expensive to change once there is code in every corner of the answer.

## Decision

### One binary; the core is a package, not a process

The core is compiled into whatever runs it. There is no background daemon and
nothing talks to anything over a socket.

A separate process would buy independent lifecycles and a wire protocol usable by
other clients later. Neither is needed to scan a folder, and both cost a protocol
to design, version and debug before a single note has been read. The protocol
layer stays empty until something outside the binary genuinely needs to talk to
the core.

### Hexagonal: the domain knows nothing that has a lifetime

The domain holds notes, vaults, the index model and the rules that operate on
them. It may not know that files exist, that SQLite exists, or what time it is.
Everything with a lifetime — a filesystem, a database, a clock — is reached
through a port the domain declares and an adapter that implements it.

The point is not layering for its own sake. It is that ADR-0002 makes the index
replaceable and ADR-0001 makes the vault an unreliable, externally-edited thing;
both of those are survivable only if the code that decides *what* is true is
separated from the code that decides *where it came from*.

**Entry points are adapters too.** The CLI is the first one. A Wails-based
desktop shell will be the second. The core knows about neither, and no decision
about the interface is made here — the CLI exists because a scan has to be
runnable, not because the product is a command-line tool.

### One Go module for the repository

A single `go.mod`; `modules/libs/*` and `modules/apps/*` are directories of
packages, not modules of their own. Splitting into separate modules buys
independent versioning, which is worth having when there is a second consumer and
is pure overhead until then. Nothing is extracted into a shared library on
speculation.

### SQLite driver: pure Go for now, behind a port

`modernc.org/sqlite`, chosen for cross-compilation: a desktop application ships
to macOS, Windows and Linux, and a cgo-free driver builds all three from one
machine. It is slower than the C library and slightly behind upstream, and it
supports the vector extension when that becomes relevant (ADR-0002).

This is a decision made without measurements, and it is recorded as replaceable
rather than as right. The driver lives behind the index port, so replacing it is
one adapter. It should be replaced if a benchmark on a vault at target scale —
100k notes, full rebuild, incremental update, full-text query — shows the cgo
driver clearing a budget the pure-Go one misses.

### Schema version, not migrations

ADR-0002 says a model change drops the tables and replays from the vault, so
there are no migration scripts. What is needed instead is a version stored in the
database and one rule: if it does not match the version the binary expects, the
index is dropped and rebuilt.

The rebuild is minutes at target scale, so it is a visible event with progress,
not a silent stall on startup.

## Consequences

**Positive**

- The first tool can be written now, and the desktop shell later, without either
  constraining the other.
- The parts most likely to change — driver, storage, entry point — are each one
  adapter.
- No versioning, protocol or process-management work is done before something
  needs it.

**Negative**

- Ports and adapters are indirection, and on a codebase this small the
  indirection is visible before the payoff is. Accepted on the grounds that the
  two things it protects are the two ADR-0001 and ADR-0002 identify as certain to
  churn.
- The pure-Go driver is a bet on portability over speed, taken before there is
  anything to measure. It is recorded as such, and the measurement is defined.
- One module means every package shares one dependency set. That is fine at this
  size and will need revisiting if a mobile client ever shares code.

## Alternatives considered

**A daemon with the clients as thin front ends.** Rejected for now: it demands a
protocol before it demands a feature. Reconsider when something outside the
binary needs the core — a mobile client is the obvious candidate, and it is out
of scope.

**A module per directory.** Rejected until there is a second consumer.
Independent versioning of packages nobody imports twice is bookkeeping.

**`mattn/go-sqlite3`** (cgo). Not rejected on merit — it is the faster and more
mature driver. Deferred because cgo turns cross-compilation into a build matrix,
and that cost lands on every release while the speed difference is currently
unmeasured.
