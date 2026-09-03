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

## Layout

```
docs/                        specifications and architecture decision records
modules/apps/desktop/        desktop client, and the flashcards window beside it
modules/apps/mobile/         mobile client
modules/apps/landing/        the page the product is read about on
modules/apps/docs/           the manual, for the person using the application
modules/libs/core/           the core every client is built on
modules/libs/protocol/       wire/vault protocol
modules/libs/ui/             shared interface components
modules/tools/git-hooks/     repo-level tooling
```

Mixed-language by design. Each module owns its toolchain; the root carries no
build system.

## Status

Early, and running. The desktop client writes notes, links them, draws the plex,
searches by name, word and meaning, reads scanned books with OCR, transcribes
recordings, and runs the cards a vault holds in a window of its own. Nothing has
been released yet.

- [The manual](https://docs.numen.md) — how the application is used
- [Specifications](docs/) — how each part behaves
- [Architecture decisions](docs/adr/)
- [Contributing](CONTRIBUTING.md) — layout, labels, commit format
