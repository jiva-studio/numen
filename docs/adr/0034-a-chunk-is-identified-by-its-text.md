# ADR-0034: A chunk is identified by the text it holds

- **Status:** Accepted
- **Date:** 2026-08-17
- **Applies to:** `modules/apps/desktop`
- **Partly supersedes:** ADR-0006 — `start` and `length` as a chunk's key
- **Related:** ADR-0006, ADR-0007, ADR-0016, ADR-0024, ADR-0029

## Context

ADR-0006 stores a chunk as a place in a file — an offset and a length in the text of
its source — and calls that pair the chunk's key. A note is now edited in a tab
(ADR-0032), so a source is cut again while somebody is typing into it.

An offset moves when anything above it changes. A key made of offsets therefore names
a different chunk after a keystroke at the top of a note, and every vector below that
keystroke is asked of a model again. What one chunk costs is in docs/performance.md.

## Decision

### The hash of a chunk's text is what identifies it

A chunk carries the SHA-256 of the text it holds, and cutting a source again is a
comparison against that column.

A window whose hash is on a row of this source **keeps that row**, and its vector and
its full-text row with it. A window whose hash is on no row is a new chunk. A row
whose hash is in no window is a chunk that is gone.

**A row is claimed once**, so text that occurs twice in one source is two rows and
stays two. **A large window and a window inside one are two populations**, so the same
text cut at both sizes is a row at each: a vector belongs to the small size, and the
large one is the chunk with no parent (ADR-0006).

The text is hashed and indexed and is not stored, which is ADR-0006's rule unchanged.

### `start` and `length` are where a chunk is, and they move

They are what a passage is read back through, and everything the application does
with a chunk still goes through them. On a row that kept its identity they are
updated to where its text now is, along with the large window it now sits inside.

A large window covers the whole of a note, so its hash moves whenever the note is
edited at all, and every window inside it goes with it.

### A row written before the column existed carries the empty string

No text hashes to that, so such a row is replaced the first time its source is cut
again.

### Two hashes, and each says what it is over

A source's hash is over the bytes of a file and says which file it is (ADR-0006). A
chunk's hash is over the text of one window and says which chunk it is. Each is a
column of the table it belongs to, and nothing compares one with the other. This is
entered in ADR-0024.

## Consequences

**Positive**

- An edit above a chunk costs an update of two numbers, and the vector belonging to
  that text stays where it is.
- A line added to a note's frontmatter asks the model for nothing. What a save asks
  for is a count, and a test asserts it (docs/performance.md).
- What needs embedding is still a chunk with no vector, so there is nothing to
  reconcile after a crash (ADR-0006).

**Negative**

- **A note's offsets are derived from a size measured before its body was read.** A
  scan and a refresh each look at a file and then read it, and a note's windows begin
  at the file's size less the length of its body. A file written between the two puts
  every offset in that note out by the difference, and a passage read back through
  them reads out the wrong place. A body longer than the size that was measured
  begins at the start of the file. The next scan or refresh of that path corrects it,
  and until one happens nothing says the offsets are wrong.
- **The word `hash` now means two things**, and a sentence that says it alone has
  said nothing. The settlement is in ADR-0024.
- Which of two identical passages keeps which row is whichever order the rows come
  back in, so a cut can move both and embed neither.
- Cutting a source computes a hash per window, over text already in hand.

## Alternatives considered

**Keep offsets as the key, and embed again whatever moved.** Rejected: an edit at the
top of a note moves every offset below it, so one keystroke costs the whole note, and
the note being typed into is the one that costs most.

**A key made of the source and the window's ordinal.** Rejected: a paragraph inserted
anywhere shifts every ordinal after it, which is the failure the offsets already have
with a longer arithmetic.

**Store the text of a window and compare it.** Rejected: the index keeps no copy of
what it indexed (ADR-0006), and a hash is what a comparison needs.

**Hash the file and re-cut nothing when it matches.** Rejected as a substitute: it
answers whether a source changed, which the fingerprint and the recipe already answer
(ADR-0006), and says nothing about which of its windows survived the change.
