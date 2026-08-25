# ADR-0018: The note file

- **Status:** Accepted
- **Date:** 2026-08-25
- **Applies to:** the vault format — every application that reads or writes one
- **Related:** ADR-0001, ADR-0004, ADR-0017, ADR-0019, ADR-0026

## Context

The note file is the one artifact the person touches. Everything the application knows about a note has to survive inside it, and a vault full of syntax one application can read is a vault nobody can leave.

## Decision

### A note is markdown any editor opens

A note is a UTF-8 markdown file. An addition is permitted only where it renders as plain text in an editor that never heard of this application. Markup that renders as garbage is refused; slightly unusual text a person can read is allowed.

The permitted additions are exactly two: **YAML frontmatter** at the top of the file, and **`[[wikilink]]`** in the body. Nothing else — no custom fences, no HTML comments carrying data, no sidecar files, no private extension. What a wikilink resolves to is in [links](../links.md).

### Which files are notes

The extensions treated as notes are a setting, and the default is `.md` alone. Markdown is written under several names, and the answer is a preference.

Two places are never notes, whatever the setting says: the application's own folder inside the vault, and any directory whose name begins with a dot. Both are skipped whole, without being descended into.

### The frontmatter is where the application's fields live

The body is prose the person wrote. A field the application owns about the note as a whole goes in the frontmatter. The frontmatter is shared with the person, and three rules follow.

Keys the application does not own are **preserved verbatim**, order included, and that covers a key it may own in a future version. Owned keys are a **closed, documented set**, each introduced by a decision that says what it means. A **collision is reported and never resolved**: where a person's own key carries the name of an owned one, the application neither overwrites it nor reinterprets it.

The owned keys, the extensions a note may carry, and the order a note's displayed name is resolved in are tabulated in [the note format](../note-format.md).

## Consequences

- The format cannot express what markdown does not render, and anything that does not fit becomes an artifact in the application's own folder.
- Parsing has to tolerate what other editors produce, which is irregular.
- `title` is an ordinary English word taken as an owned key, and a person's own key of that name collides.
- A reported collision stands until the person settles it, and the note carries a field the application will not read.
- Extensions being a setting means one file is a note in one vault and prose in the next.

## Alternatives considered

**Sidecar metadata files**, `note.md` beside `note.meta.json`. Rejected: the pair splits on any external move, rename or copy, which are the operations this product promises are safe, and it doubles what the person sees in every folder.

**A fenced block of the application's own for machine-owned fields.** Rejected: it renders as a wall of syntax in every editor that never heard of it, and it sits in the body, where the person's prose is.

**The application owning the whole frontmatter**, rewriting it on every save. Rejected: the frontmatter is where another application's fields and the person's own fields already are, and a rewrite drops them.
