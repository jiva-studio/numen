# A link is a note that carries an address

- **Status:** Accepted
- **Date:** 2026-09-06
- **Applies to:** `modules/libs/core`, `modules/apps/desktop`
- **Related:** [The note file](0018-the-note-file.md), [The stencil and the deck](0026-the-stencil-and-the-deck.md), [The preset](0029-the-preset.md), [A recording is a source of its own](0030-a-recording-is-a-source-of-its-own.md)

## Context

A person watches a lecture, reads an article, and wants what is in it where the rest of their thinking is. What they have is an address.

A vault holds files. An address is not one, and what is at it is somebody else's and may go away.

## Decision

### A link is the fifth kind of note

`type: link` in the frontmatter, and `url` beside it: the address, read into the one form every spelling of it reaches. `NoteType` takes a fifth value and `notes.type` takes a fifth word.

The note is a note. Its body is prose the person wrote, its links are links, its headings are headings, and every rule of the note file holds over it unchanged.

### `url` is read where every other kind of note's keys are read

A deck's cards, a stencil's fields and a preset's settings are not fields of `domain.Note`: each is read out of the frontmatter by the package that owns that kind, which answers with the value and what was wrong with it. An address is read the same way, and the domain's note carries no more than it did.

A link note with nowhere to point is missing its whole subject. That is a problem against the note, said and not guessed at, and the note is read as every other note is.

### The source kind is untouched

A `.md` file is a note from its extension alone, which is what every other kind of source is decided by. What a link note is, is `type` in its frontmatter, and that already crosses the wire.

## Consequences

- Everything that opens, renames, moves, links and searches a note works on a
link note the day it is made.
- A person who writes `type: link` and `url` by hand in another editor has made
one, and nothing has to be registered or imported.
- The closed set of owned frontmatter keys grows by one, and the note format
page is what keeps it current.

## Alternatives considered

**A source kind of its own, in a file of its own** — `.url`, the shortcut format an operating system already opens. Rejected: the person's prose about a video would then live in a second file beside it, and what they wrote and what they wrote it about are one thing to them.

**A field on `domain.Note`.** Rejected: four kinds of note already keep their own keys out of it, and a fifth that did not would be the beginning of one type that is every kind of note at once.
