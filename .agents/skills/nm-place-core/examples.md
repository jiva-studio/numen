# Worked examples

Each example shows the target shape. The cited lines in main follow that shape for the part named; the parts of the same files that differ from the target are not cited.

## A small port, its adapter and the composition root: the clock

The need: a scenario stamps a note with the current time.

- The port is one method, declared by the need and named after it: `Clock` with `Now() time.Time` at `libs/core/kernel/port/machineport/clock.go:11-13`.
- The driven adapter is named after the technology, `clock.System`, and holds the only read of the machine clock: `libs/core/kernel/internal/adapter/clock/clock.go:10-19`. It does not name `machineport.Clock`.
- The use case receives the port and calls `Now()`; the domain receives the instant it is given as an argument.
- `container` writes the one line that binds `clock.System{}` to the port.

## A driven adapter that stores and does not decide: the similarity floor

The need: a search remembers the floor a plugin measured for a vault and a recipe.

- The port has three methods, read, write and delete, each called from a use case: `FloorStore` at `libs/core/kernel/port/searchpolicyport/policy.go:9-17`.
- The adapter is SQL and nothing else: `ReadFloor` runs a query and maps `sql.ErrNoRows` to "none kept" at `libs/core/kernel/adapter/index/searchpolicy/repository.go:36-48`. It does not choose which floor wins.
- The decision is the use case's: the stored floor, then the hint of the embedding model, then the default, in `FloorPolicy.Get` at `libs/core/kernel/usecase/search/floor.go:75-82`.

Where the next change goes: a new source of floors is a new case in `FloorPolicy.Get` and, if it needs data, a new small port. The adapter is untouched.

## A rule in the domain, data from a port, the order in a use case: connections

The need: a connection between two notes must have a valid shape and must not make a note its own ancestor.

- The rule is the domain's because it follows from the connection and the parents of a note: `Connection.Validate` at `libs/note/domain/connection.go:104-134` and `Connection.CheckAcyclic` at `libs/note/domain/connection.go:159-190`. `CheckAcyclic` takes a function that answers the parents of a note, so the domain reads nothing.
- The use case supplies the data from a port and calls the rule: `connectionEditor.check` at `libs/core/kernel/usecase/note/connection.go:91-96` calls `Validate`, then `CheckAcyclic` with a function over `getParents`, which reads links through `noteport.LinkQueries` (`libs/core/kernel/usecase/note/connection.go:68-89`).
- The use case then writes the notes and updates the index; none of that is in the domain, and none of the rule is in the use case.

Where the next change goes: a new rule about what a valid connection is goes beside `Validate`; a new source of parents changes `getParents` only.

## A driving adapter that only routes: the vault command of the CLI

The need: `numen-cli vault add <path>` creates a vault entry.

- The adapter dispatches on the sub-command, parses flags and arguments, and refuses a malformed command line: `libs/core/kernel/adapter/cli/vault.go:26-60`.
- After parsing it calls one use case with plain arguments and prints the result.
- In the target the adapter receives the created use case from `container` (`vault.Add` with its `Execute(root, name)`) and names no port; it neither builds the use case from ports nor reads the clock. The use case is created once in `container`.

The same shape holds for a window service method: decode the request, call one use case, map the result and its error code to the wire type.

## A pure use case over a port: create a folder

The need: the window creates an empty folder in a vault.

- The type is a verb phrase and holds the one port it calls, in an unexported field: `CreateFolder` at `libs/core/kernel/usecase/file/create_folder.go:11-15`.
- The method is `Execute(ctx, vault, path)`, opens the writer through the port, makes the folder and returns the error: `libs/core/kernel/usecase/file/create_folder.go:21-27`.
- In the target the constructor is `NewCreateFolder(writers) *CreateFolder`, takes the port without checking it, and the port it takes is the one-method need "make a folder in a vault", not the whole writer of the vault.

## A new capability end to end: "export a note as PDF text"

1. The rule about what is exported (which parts of a note, in which order) is a function over `domain` values in `libs/note`, or in `markdown` when it reads the text of the file.
2. The scenario is `usecase/note/export_note.go`: `ExportNote` with `Execute(ctx, v, path) (Export, error)`. It reads the note through the existing note reader port, calls the rule and writes the result through a writer port declared for this use case.
3. The writer port is declared next to the use case (one consumer) with one method, `WriteExport`. The adapter that writes the file implements it in the filesystem adapter.
4. `container` creates `ExportNote` with the two ports and adds it to the group of note use cases.
5. The driving adapter gets a method that decodes the request, calls `Execute` and maps `Export`; the schema change goes through `libs/core/protocol`.
6. The application changes nothing.
