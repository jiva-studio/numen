# ADR-0004: A hexagonal core in Go

- **Status:** Accepted
- **Date:** 2026-08-25
- **Applies to:** `modules/libs/core`, `modules/apps/desktop`, `modules/apps/mobile`
- **Related:** ADR-0001, ADR-0002, ADR-0005, ADR-0006, ADR-0020, ADR-0021, ADR-0023, ADR-0025

## Context

The core decides what is true about a vault. A window, a phone, a command line and an agent each need to reach it. The shape of the code is settled once, because every later file is written inside the answer.

## Decision

### The architecture has names, and they are these

The application is **hexagonal architecture**: the domain declares ports, and adapters implement them. It follows the dependency rule of **clean architecture**: imports point inward only, and nothing under the core's own packages imports an adapter. It is organised by the tactical patterns of **domain-driven design**: aggregates, repositories, queries, and use cases named after the scenario, in the ubiquitous language of ADR-0026.

What a client says to the core is ADR-0005.

```mermaid
graph TD
    subgraph apps["modules/apps/*"]
        DK["desktop<br/>cmd + a window"]
        MB["mobile<br/>the core, bound"]
    end

    subgraph core["modules/libs/core"]
        direction TB
        subgraph driving["driving adapters"]
            W["webui<br/>serves a client"]
            C["cli"]
            M["mcp<br/>agent tools"]
        end
        P["port<br/>interfaces the core needs"]
        U["usecase<br/>one file per scenario"]
        D["domain<br/>notes, vaults, links"]
        CN["container<br/>composition root"]
        subgraph driven["driven adapters"]
            F["filesystem"]
            I["index — SQLite"]
            A["appstate"]
        end
    end

    DK --> CN
    MB --> CN
    DK --> W
    MB --> W
    W --> U
    C --> U
    M --> U
    U --> D
    U --> P
    P -.declared by the core.-> U
    F --> P
    I --> P
    A --> P
    CN -.binds adapter to port.-> P
```

### One binary; the core is a package, not a process

The core is compiled into whatever runs it. There is no background daemon.

An application serves the generated handler in-process and the tool endpoint on a loopback port; both are adapters inside the same binary.

### The domain knows nothing that has a lifetime

The domain holds notes, vaults, the index model and the rules over them. It may not know that files exist, that SQLite exists, or what time it is. A filesystem, a database, a clock and an agent are each a port the core declares and an adapter that implements it.

Ports are declared in `port`, because the core is what needs them. Adapters never name the interface they satisfy. Ports are named after the need, adapters after the technology: the core asks for a `VaultReader`, and that the answer is a filesystem is knowledge confined to `adapter/` and `container/`.

Entry points are adapters. The command line, the window and the tool endpoint are three of them, and the core knows about none.

### A module for the core, a module for each application

The core is a Go module, and each application is another beside its own code. There is no module at the repository root. `modules/libs/` holds what an application imports or is generated from: the core, the protocol and the interface library.

An application names a configuration, mounts the driving adapters it serves, and runs. Its manifest declares what it starts and nothing else: the desktop's names a window toolkit and the core.

### An implementation only one application can start lives in that application

An adapter behind a port is the core's when every application can run it, and the application's when one can. The agent behind `port.Agent` starts a program on the machine, so it sits in the desktop application; a phone binds that port to nothing and the panel says there is no agent.

### Layout

```
modules/libs/core/
  go.mod
  domain/                  one file per type: vault.go, note.go, link.go
  port/                    one file per port: vault_reader.go, note_queries.go
  usecase/<aggregate>/     one file per scenario: add.go, scan.go, rename.go
  markdown/                the note format, parsed
  container/               composition root: adapter to port
  adapter/
    cli/                   driving: arguments in, text out
    webui/                 driving: the handler a client asks
    mcp/                   driving: tools an agent calls
    filesystem/            driven: a vault on disk
    index/                 driven: the cache, a folder per aggregate
      <aggregate>/         repository.go, queries.go, sql/*.sql
      migration/           numbered schema changes
    settings/              driven: what a person configured
    agent/                 driven: which agent answers
  internal/
    adapter/               driven: what nothing outside composes
    ulid/                  identifiers
    testsupport/           fixtures and generated vaults, for tests only

modules/apps/<app>/
  go.mod
  cmd/<binary>/            entry point and the process's own concerns
  internal/adapter/        what this application alone can start
```

The unit of organisation is the thing, not the kind of thing. An aggregate is a folder holding its repository, its queries and its SQL together; a use case is a file named after the scenario.

A repository is a collection of aggregates: put one in, take one out, remove one. Anything answering across many of them, in a shape that is not an aggregate, is a query and lives beside the repository.

### `internal/` marks what nothing outside composes

An application reaches the core's own language, the driving adapters it serves, and `container`. A driven adapter it does not name sits under `internal/`, where the compiler holds it, so binding an adapter to a port stays one package's work.

### SQL lives in files

Schema and queries are `.sql` files embedded into the binary, one statement per file, loaded by name.

### The SQLite driver is pure Go, behind the index port

`modernc.org/sqlite`, so a desktop application builds for macOS, Windows and Linux from one machine, and a phone carries the same code. It carries the vector extension. It lives behind the index port, so replacing it is one adapter.

## Consequences

- The parts most likely to change — driver, storage, entry point — are each one adapter.
- Ports and adapters are indirection, and it is visible before the payoff is.
- An application declares what it starts and nothing else, and a dependency two of them share is declared once.
- A driven adapter an application composes is public, and the list of them is the core's surface to keep small.

## Alternatives considered

**A daemon with the clients as thin front ends.** Rejected: independent lifecycles buy nothing here, and the boundary being crossed is between two languages rather than two machines.

**One module for the whole repository.** Rejected: the repository holds several applications, and a manifest at the root would make the root a project it is not.

**The core inside the desktop application, imported from nowhere else.** Rejected: `internal/` is not importable across modules, so a second application reaches the core only by living inside the first.

**`mattn/go-sqlite3`.** Deferred: cgo turns cross-compilation into a build matrix, and that cost lands on every release.
