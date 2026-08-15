# ADR-0014: One binary, hexagonal core in Go

- **Status:** Accepted
- **Date:** 2026-08-15
- **Related:** ADR-0001, ADR-0002, ADR-0013, ADR-0015

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

### A Go module per application

The repository holds several applications, so each one is its own module with its
own `go.mod` beside its code. There is no module at the repository root: the root
is not a project, and a manifest there would claim otherwise.

`modules/libs/` exists for code two applications genuinely share, and stays empty
until that happens. Extracting a library because code might one day be shared
buys nothing and costs a versioned boundary through the middle of a codebase that
has one consumer.

### Layout inside an application

```
modules/apps/<app>/
  go.mod
  cmd/<binary>/main.go     entry point, and nothing else
  internal/
    core/
      domain/              entities
      port/                the interfaces the core needs from outside
      service/             use cases
      markdown/            the note format, parsed
    adapter/
      cli/                 driving: arguments in, text out
      filesystem/          driven: a vault on disk
      index/               driven: the cache the notes are queried from
      appstate/            driven: the list of vaults
    container/             composition root: which adapter satisfies which port
```

Three things in this are Go rather than architecture in general, and they are the
reason it does not look like the same diagram drawn in another language:

- **`internal/` is enforced by the compiler.** Nothing outside the application
  can import any of it, so the boundary is a fact rather than a convention. This
  is why the layers do not need to be separate modules to stay separate.
- **Interfaces belong to whoever needs them, not to whoever satisfies them.** The
  ports are declared in `core/port` because the core is what needs them; the
  adapters never mention the interface they implement, and Go checks the fit
  without either side saying so.
- **Ports are named after the need, adapters after the technology.** The core
  asks for a `VaultReader`; that the answer is a filesystem, and that the index
  is SQLite, is knowledge confined to `adapter/` and `container/`.

### SQL lives in files, not in string literals

Schema and queries are `.sql` files embedded into the binary, one statement per
file, loaded by name. SQL is a language of its own, and burying it in Go string
literals hides it from anything that reads, formats or checks SQL — including the
person reviewing a change to it.

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
- A module per application means a dependency shared by two of them is declared
  twice until it is worth extracting. That is the price of not drawing a library
  boundary before there is a second consumer.

## Alternatives considered

**A daemon with the clients as thin front ends.** Rejected for now: it demands a
protocol before it demands a feature. Reconsider when something outside the
binary needs the core — a mobile client is the obvious candidate, and it is out
of scope.

**One module for the whole repository.** Rejected: the repository holds several
applications rather than one, and a single manifest at the root would make the
root a project it is not.

**A shared library from the start.** Rejected: nothing is shared yet. A library
extracted on speculation is a versioned boundary through a codebase with one
consumer.

**`mattn/go-sqlite3`** (cgo). Not rejected on merit — it is the faster and more
mature driver. Deferred because cgo turns cross-compilation into a build matrix,
and that cost lands on every release while the speed difference is currently
unmeasured.
