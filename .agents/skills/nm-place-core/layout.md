# The target layout

The tree below is the target of `libs/core/kernel`, `libs/note` and the Go side of `apps/desktop`. A new package goes where the tree says; a package that sits elsewhere in main moves here when it is touched for another reason.

```text
libs/note/                      the domain; imports nothing of this repository and does no I/O
├── domain/                      entities, value objects, identifiers, refusals, rules over values
├── markdown/                    reading and editing the text of a note's file
├── edit/                        edit operations over a note's text
├── textfile/                    the format of a text version's file
└── ulid/                        the identifiers of notes; time and entropy come in as arguments

libs/core/kernel/                       the hexagon
├── domain/<area>/               small packages of the core's own concepts (settings, theme, connections, the index and its chunking and vector shapes, search sources, plugin information) and their sentinel errors; never imported by a plugin or the SDK
├── port/<name>port/             the interfaces use cases require, interfaces several use cases share, at most five methods each; an interface with one consumer is in that use case's package
├── usecase/<noun>/              a bounded context that holds several use cases, one scenario per file, each isolated from the others, with its request and result types and the interfaces only it consumes
├── adapter/
│   ├── cli/                     driving: the command line; parses arguments, calls use cases, prints
│   ├── window/                  driving: cut by wire service, one adapter type per RPC service; no object holds them all
│   ├── index/                   driven: the SQLite index; stores and reads, never cuts or ranks
│   └── settings/                driven: the settings file, its lock and its on-disk layout
├── internal/
│   ├── adapter/
│   │   ├── filesystem/          driven: vault reader, writer, watcher, identity files
│   │   ├── clock/               driven: the machine clock and a settable clock for tests
│   │   ├── trash/               driven: the operating system's trash
│   │   ├── plugin/              the plugin adapter: host* packages drive the core, *bridge packages are driven by it
│   │   ├── theme/               driving: the stylesheet catalogue a window asks for
│   │   └── (one folder per other technology: provision, onnxruntime, appstate, machine)
│   ├── adapterlib/              code several adapters share and that is not a port (retry, watchhub, wire)
│   └── testsupport/             helpers for tests
└── container/                   the composition root: one package, wiring only

apps/desktop/                    thin shells
├── cmd/<binary>/                main: flags, then app.Run
├── app/                         the window application: options, toolkit, menus, quit, wiring of the toolkit to the container
└── internal/                    pieces only this application uses (keychain, platform paths, shutdown, version)
```

## What each folder may import

- `libs/note/*` imports the standard library and its own siblings only, along the edges the `go/note-library` check lists.
- `domain/<area>` imports the standard library, `libs/note/domain` and itself.
- `port/*` imports `domain/<area>`, `libs/note/domain` and the standard library.
- `usecase/*` imports `port`, `domain/<area>`, `libs/note` and no other use case.
- A driven adapter imports `port`, `libs/note` and its technology; it imports no other adapter.
- A driving adapter imports `usecase`, `libs/note` and the generated protocol; it imports nothing under `port/`.
- `container` imports everything and returns no adapter type.
- `apps/desktop` imports `container`, the toolkit and the types `container` hands out.

## Where a type is declared

- A fact about notes, vaults, links, cards or text, or a value both a use case and a port exchange: `libs/note/domain`.
- The vocabulary of a port that a domain type does not hold (a stored vector, a plugin manifest entry): the port package that owns it.
- The request and result of one scenario: its use case package.
- A shape that exists only to cross a wire: the generated protocol, mapped in the driving adapter.
