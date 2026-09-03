# Numen

A knowledge tool that merges three things that have never been in one place:

- **notes in markdown** with links between them, local and in an open format;
- **a typed navigational hierarchy** with multiple parents — a DAG, not a tree;
- **spaced repetition** with cards attached to those notes;

plus a **source layer** — PDF, audio, books — with text extraction, search, and
the hit highlighted in the original document, down to the phrase on the page or
the moment in the recording.

## Your notes stay yours

Everything you write is a plain file in an ordinary folder you choose. Not a
database, not an export — the files *are* the notes. Edit them in vim, keep them
in git, search them with ripgrep, run your own scripts over them, with the
application closed. Nothing is lost if it never opens again.

## What it does

**The vault.** A folder you point at. Markdown files are notes from the first
byte — nothing is imported or converted. The application walks it in the
background, watches it while it runs, and picks up what changed underneath it
while it was closed. Several vaults, one window.

**Links with meaning.** A link carries a role, a type and a label, written in
the note's own frontmatter. Where it points is worked out when you ask, so
adding a file mends a link that was dangling. The **plex** draws the
neighbourhood around a note — what it points at and what points back.

**Search, three ways at once.** Words, meaning, and names, merged into one
ranking. Vectors live inside SQLite beside the index; nothing leaves the
machine to answer a query.

**Books and recordings.** A PDF is read by a local OCR model, a recording is
transcribed by a local speech model, and both become searchable text. A hit
lands on the phrase on the page or the second in the audio. A second model
proofreads what the first one wrote, and the original is never overwritten.

**Cards.** A stencil declares fields and faces; a deck is a file of cards
written in ordinary markdown headings. Scheduling is FSRS, and a preset — itself
a note in the vault — says how much a day holds and what closes it. Review runs
in a window of its own, `numen-flashcards`, which you can use without ever
opening the editor.

**An agent, if you want one.** An LLM reaches the vault through an authored set
of tools over MCP — reading notes, searching, writing cards — and never through
a shell. What it does arrives in the window the way any other edit does.

## Architecture

A **hexagonal core in Go** ([ADR-0004](docs/adr/0004-a-hexagonal-core-in-go.md)),
compiled into whatever runs it. There is no daemon.

```
domain/     notes, vaults, links, cards — no clock, no disk, no database
port/       the interfaces the core needs, named after the need
usecase/    one file per scenario
adapter/    driving: cli, webui, mcp · driven: filesystem, index, settings
container/  the composition root, where an adapter meets a port
```

Clients are **generated from one `.proto`**
([ADR-0005](docs/adr/0005-a-client-is-generated-from-the-protocol.md)): Go
handlers and a TypeScript client both come out of the schema, so a renamed field
fails a build rather than a request.

The index is **one SQLite file for every vault**, outside them all — a cache
that can be deleted, with FTS5 for words and a vector extension for meaning.
The driver is pure Go, so one machine cross-compiles for macOS, Windows and
Linux.

The windows are **Vue on Wails**, over a shared component library that knows
nothing about vaults. Local models run in-process: ONNX Runtime for speech and
OCR, and pdfium under WebAssembly for pages.

Two ideas hold the rest together. **Files on disk are the truth** and the index
is a cache ([ADR-0001](docs/adr/0001-files-are-the-source-of-truth.md)); and
what a model made and cannot cheaply remake is an **artifact**, kept in the
vault's own folder ([ADR-0015](docs/adr/0015-a-books-text-is-a-cache-or-an-artifact.md)).

## Layout

```
docs/adr/                    architecture decision records
docs/*.md                    the specifications: the note format, cards, search
modules/apps/desktop/        the editor, the review window, the command line
modules/apps/mobile/         mobile client
modules/apps/landing/        the page the product is read about on
modules/apps/docs/           the manual, for the person using the application
modules/libs/core/           the core every client is built on
modules/libs/protocol/       the schema every client is generated from
modules/libs/ui/             shared interface components
modules/tools/git-hooks/     repo-level tooling
```

Mixed-language by design. Each module owns its toolchain; the root carries no
build system.

- [The manual](https://docs.numen.md) — how the application is used
- [Specifications](docs/) — how each part behaves
- [Architecture decisions](docs/adr/) — what was settled, and what it cost
- [Contributing](CONTRIBUTING.md) — layout, labels, commit format
