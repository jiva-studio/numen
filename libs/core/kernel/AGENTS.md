# The core — architecture and invariants

The conventions every language in this repository shares are in `AGENTS.md` at
the root. This file defines specific rules for `libs/core/kernel`. Where a piece of code goes and how it is shaped is the skill [`nm-place-core`](../../../.agents/skills/nm-place-core/SKILL.md).

The architecture follows [`.agents/rules/architecture.md`](../../../.agents/rules/architecture.md)
("Ports and adapters"). The following table summarizes architectural invariants
and the corresponding tests enforcing them.

## Layer boundaries and dependency direction

The checks `go/layers` and `go/direction` of `tools/checks/structure/go` enforce these boundaries (`make checks ONLY=go/layers,go/direction`).
Each check is a folder of `tools/checks/structure/go`.

| The rule                                                                                                                                                                                                                                                                                                                             | The check                                   |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------- |
| A package is on the layer its path states, and imports only from lower layers                                                                                                                                                                                                                                                        | `go/layers`                                 |
| No package that the core is compiled from imports an adapter                                                                                                                                                                                                                                                                         | `go/direction`                              |
| A use case package imports no other use case package (packages are isolated; inside one package the use cases may call each other and no check covers it); a `usecase/` package named `*test` is test support, imported only by tests                                                                                                | `go/usecase-edges`                          |
| No interface of a use case package or of `port/` is implemented by a use case of another package, and no `Execute` method value of a use case is passed as a function to a field or a function of another use case package: what use cases share is a rule of the domain or a port an adapter implements | `go/use-case-ports` |
| An adapter does not name the port it implements                                                                                                                                                                                                                                                                                      | `go/adapters-claim-no-ports`                |
| The composition root returns no adapter type                                                                                                                                                                                                                                                                                         | `go/ports`                                  |
| A package is imported under the name its layer gives it                                                                                                                                                                                                                                                                              | `go/layers`                                 |
| No package declares an alias of a type another package of the core declares                                                                                                                                                                                                                                                          | `go/core-reexport`                          |
| An adapter that references a use case is driving, one with a type that implements a port is driven, and none is both                                                                                                                                                                                                                 | `go/adapter-role`                           |
| A driving adapter references no type, function, variable, constant or method declared under `port/` and no port interface declared in a use case package, aliases resolved                                                                                                                                                           | `go/adapter-role`                           |
| The composition root is the one package `container`; no package lies below it                                                                                                                                                                                                                                                        | `go/composition-root`                       |
| The composition root imports no package that reaches the machine, randomness or cryptography, calls no `fmt` print function, and holds no function with a cyclomatic complexity above 3: it connects, and a rule belongs to a use case or an adapter                                                                                 | `go/wiring-only`                            |
| A package under `domain/` of the core imports the standard library, `libs/shared/note/domain`, `libs/plugin/catalog/manifest`, `libs/plugin/catalog/when` and other domain packages, no other layer of the core, no third-party code; every other layer and the composition root may import it                                                                          | `go/layers`, `go/domain-layer`              |
| No plugin, no plugin SDK and no plugin catalog package imports a package of the core, a domain package included                                                                                                                                                                                                                      | `go/plugin-independence`, `go/domain-layer` |
| The depguard rule `domain` denies `port` as well as adapters, use cases and the composition root                                                                                                                                                                                                                                     | `.golangci.yml`                             |
| No struct holds more than five use cases                                                                                                                                                                                                                                                                                             | `go/struct-use-cases`                       |
| No package outside `container`, `apps/*/cmd` and `apps/*/app` imports a use case package and a driven adapter package (one with a type that implements a port: an interface with methods under `port/`, or a named interface of a use case package that a use case holds as a field or takes as a parameter of an exported function) | `go/wiring-package`                         |

## What the core may not do

| The rule                                                                 | The check           |
| ------------------------------------------------------------------------ | ------------------- |
| The core does not access the machine: no `os`, no filesystem, no network | `go/machine-access` |
| Nothing reads the machine's clock — an instant is passed as an argument  | `go/machine-clock`  |
| Nothing logs                                                             | `go/loggers`        |

## Ports

A port is declared where it is consumed, and every method on one is called from
outside `port/`.

| The rule                                                                                                                                                                                                   | The check             |
| ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------- |
| Every port is referenced somewhere else                                                                                                                                                                    | `go/ports-referenced` |
| Every type a port declares is named somewhere else                                                                                                                                                         | `go/ports-referenced` |
| Every port method is called through the port from outside `port/`                                                                                                                                          | `go/ports-referenced` |
| No function or method of a port, and no function of a use case package that returns a port, calls a method of an interface value                                                                           | `go/port-calls`       |
| An interface has at most the interfacebloat `max` of `.golangci-structure.yml` methods, embedded ones included, in the kernel, the notes library, the SDK, the applications and the plugins, not in a tool | `go/interface-size`   |

## Names in the core

| The rule                                                                                                                                                                                                               | The check                                               |
| ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------- |
| An identity is named with the core's own term, in fields and in signatures                                                                                                                                             | `go/identities`, `go/identities`                        |
| An instant is a `time.Time`                                                                                                                                                                                            | `go/instants`                                           |
| A driving adapter calls no port outside its wiring files, and is covered by the depguard rule                                                                                                                          | `go/driving-calls`                                      |
| The machine clock (`time.Now`, `Since`, `Until`) is read only inside a clock port implementation; timers and randomness are not checked; a timer that changes behavior comes from the clock port (`container.installationClock`) | `go/machine-clock`, `go/machine-clock`, `go/clock-port` |
| The core probes no value for another interface                                                                                                                                                                         | `go/port-shape`                                         |
| A port names no type from `internal/`                                                                                                                                                                                  | `go/port-shape`                                         |

The target shape, which the packages of the refactor reach: the core's own concepts are small packages `domain/<area>` that import the standard library, `libs/shared/note/domain`, `libs/plugin/catalog/manifest`, `libs/plugin/catalog/when` and themselves, and that no plugin and not the SDK imports; a port has at most five methods and three where practical, lives in the package of its one consumer and in `port/<name>port` when several production consumers share it, including driven adapters that delegate or return a factory result; the window adapter is one adapter type per RPC service, with no object that holds them all; no struct holds more than five use cases.

A `domain/<area>` package takes the value types, constants and sentinel errors of its area.

No type is named by a gerund or a participle.

A concept is named with the standard term of its field.

## Constructors

A use case is created with everything it needs. The constructor takes the
collaborators, stores them in unexported fields and returns the concrete type. It
does not validate them and returns no error. A functional option passed to the
constructor and a `With` method that returns a new value are allowed. A method that
changes the object after construction is not.

| The rule                                                                                 | The check                         |
| ---------------------------------------------------------------------------------------- | --------------------------------- |
| A use case does not store a collaborator that its constructor takes in an exported field | `go/use-case-fields`              |
| A constructor of a use case or an adapter returns the value and no error                 | `go/constructors-return-no-error` |

No use case exposes a collaborator.

A dependency that may be absent is an optional field or a separate type, and the
composition root decides it explicitly. A use case does not test a required
collaborator for nil. An adapter does not test a required collaborator for nil either: where the installation may lack the whole service, the composition root passes the generated Unimplemented handler of that service instead.

A policy is not a collaborator. How many pages a batch holds, where progress is
reported, what to do after a write — those are set by an option of the constructor
and have a default.

## Migrations

A numbered migration of the index that a build has run is not changed, renamed or removed: the schema goes forward by another number.

| The rule                                                                           | The check                      |
| ---------------------------------------------------------------------------------- | ------------------------------ |
| A migration of the base is in the folder with the same name and the same contents | `core/migration-forward-only` |

## Where a rule lives

A rule about one value is a method on that value. A rule over several is a
function in the package that defines the subject. `domain/` contains what is true of
a note, and the format rules are in `markdown`.

## Writing a new rule

A rule is two things, and a rule missing either is not merged:

1. the walk over the tree, with a guard that fails when it reads nothing;
2. a negative control — a fixture written to be rejected, asserting exactly what
   the rule catches and what it lets through.
