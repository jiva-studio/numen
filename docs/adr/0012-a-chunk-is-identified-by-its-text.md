# ADR-0012: A chunk is identified by its text

- **Status:** Accepted
- **Date:** 2026-08-25
- **Applies to:** `modules/apps/desktop`
- **Related:** ADR-0008, ADR-0010, ADR-0011, ADR-0013, ADR-0014, ADR-0016

## Context

A source is cut again whenever its file changes, and a note changes at every keystroke. What hangs on a chunk — its full-text row, its vector — is worth more than the chunk, so what makes two cuts of one text the same chunk has to be settled before either is written.

## Decision

### A chunk carries the SHA-256 of its text

Cutting a source again is a comparison against that column.

A window whose hash is on a row of this source **keeps that row**, its vector and its full-text row. A window whose hash is on no row is a new chunk. A row whose hash is in no window is a chunk that is gone.

**A row is claimed once**, so text occurring twice in one source is two rows and stays two. **A large window and a window inside one are two populations**, so the same text cut at both sizes is a row at each.

The text is hashed and indexed. No table holds a copy of it (ADR-0010).

### `start` and `length` are where a chunk is

They are what a passage is read back through, and they move. On a row that kept its identity they are updated to where its text now stands, along with the large window it now sits inside.

## Consequences

- Which of two identical passages keeps which row is whichever order the rows come back in, so a cut can move both and embed neither.
- A note edited at the top keeps every row below it and pays a hash of each.
- An offset into text that has changed reads out the wrong place, and the next scan is what mends it.
- A source whose text is unchanged is cut into the rows it already had, and the comparison is what says so.

## Alternatives considered

**Keep `start` and `length` as a chunk's key.** Rejected: an offset moves when anything above it changes, so a keystroke at the top of a note asks the model for every window below it, and the note being typed into is the one that costs most.

**Delete a source's chunks and write the cut again.** Rejected: every window loses its row and its full-text row, and a chunk that is word for word what it was comes back as a new one.

**Hash both cuts into one population.** Rejected: a large window whose text is also a small window would collapse to one row, and the window a result shows would have no row of its own.
