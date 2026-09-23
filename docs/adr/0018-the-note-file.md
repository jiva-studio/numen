# The note file

- **Status:** Accepted
- **Date:** 2026-08-25
- **Applies to:** the vault format — every application that reads or writes one
- **Related:** [Files on disk are the source of truth](0001-files-are-the-source-of-truth.md), [A hexagonal core in Go](0004-a-hexagonal-core-in-go.md), [The application writes to the vault](0017-the-application-writes-to-the-vault.md), [A note is identified by a ULID in its frontmatter](0019-a-note-is-identified-by-a-ulid.md), [The stencil, the deck and the card](0026-the-stencil-and-the-deck.md)

## Context

The note file is the one artifact the person touches. Everything the application knows about a note has to survive inside it, and a vault full of syntax one application can read is a vault nobody can leave.

## Decision

### A note is markdown any editor opens

A note is a UTF-8 markdown file. An addition is permitted only where it renders as plain text in an editor that never heard of this application. Markup that renders as garbage is refused; slightly unusual text a person can read is allowed.

Every note may carry two: **YAML frontmatter** at the top of the file, and **`[[wikilink]]`** in the body. What a wikilink resolves to is in [links](../links.md).

A note of `type: stencil` or `type: deck` carries two more, and no other kind of note may: **`{{Field}}`** in a stencil's face, where a card's value is laid out, and **`^` and a card's mark** at the end of a card's heading in a deck. Both render as plain text in an editor that never heard of this application, which is the test above.

Nothing else — no custom fences, no HTML comments carrying data, no sidecar files, no private extension. A kind of note that wants an addition of its own asks for it in a record, and the count above is what is kept current.

### Which files are notes

A note is an ordinary file named `.md`, read whole up to a bound the core holds every reader to. A device, a socket or a FIFO hands over no bytes whatever it is named, and a file over the bound is a file the vault does not hold as a note; both are answered the same way. What the bound is, and what a person sees when a file passes it, are in [editing](../editing.md).

A stencil, a deck and a preset are notes, so one extension answers for all of them.

Two places are never notes: the application's own folder inside the vault, and any directory whose name begins with a dot. Both are skipped whole, without being descended into.

### The frontmatter is where the application's fields live

The body is prose the person wrote. A field the application owns about the note as a whole goes in the frontmatter. The frontmatter is shared with the person, and three rules follow.

Keys the application does not own are **preserved verbatim**, order included, and that covers a key it may own in a future version. Owned keys are a **closed, documented set**, each introduced by a decision that says what it means. A **collision is reported and never resolved**: where a person's own key carries the name of an owned one, the application neither overwrites it nor reinterprets it.

The owned keys, the extensions a note may carry, and the order a note's displayed name is resolved in are tabulated in [the note format](../note-format.md).

## Consequences

- The format cannot express what markdown does not render, and anything that does not fit becomes an artifact in the application's own folder.
- Parsing has to tolerate what other editors produce, which is irregular.
- `title` is an ordinary English word taken as an owned key, and a person's own key of that name collides.
- A reported collision stands until the person settles it, and the note carries a field the application will not read.
- Markdown written under another name — `.markdown`, `.mdown` — is prose to this application, and the person renames it to bring it in.
- A generated `.md` larger than the bound is outside the vault as far as this application is concerned, and splitting it is what brings it in.

## Alternatives considered

**Sidecar metadata files**, `note.md` beside `note.meta.json`. Rejected: the pair splits on any external move, rename or copy, which are the operations this product promises are safe, and it doubles what the person sees in every folder.

**A fenced block of the application's own for machine-owned fields.** Rejected: it renders as a wall of syntax in every editor that never heard of it, and it sits in the body, where the person's prose is.

**The application owning the whole frontmatter**, rewriting it on every save. Rejected: the frontmatter is where another application's fields and the person's own fields already are, and a rewrite drops them.
