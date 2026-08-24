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
docs/adr/                    architecture decision records
modules/apps/desktop/        desktop client
modules/apps/mobile/         mobile client
modules/apps/landing/        the page the product is read about on
modules/libs/protocol/       wire/vault protocol
modules/libs/ui/             shared interface components
modules/tools/git-hooks/     repo-level tooling
```

Mixed-language by design. Each module owns its toolchain; the root carries no
build system.

## Status

Early. The architecture is settled and recorded as ADRs; implementation has not
started.

- [Architecture decisions](docs/adr/)
- [Contributing](CONTRIBUTING.md) — layout, labels, commit format
