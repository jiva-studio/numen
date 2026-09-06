# A link note's text is its prose and what was fetched

- **Status:** Accepted
- **Date:** 2026-09-06
- **Applies to:** `modules/libs/core`
- **Related:** [A source is text in one table](0010-a-source-is-text-in-one-table.md), [A chunk is identified by its text](0012-a-chunk-is-identified-by-its-text.md), [A book's text is a cache or an artifact](0015-a-books-text-is-a-cache-or-an-artifact.md), [A passage is a range of bytes](0016-a-passage-is-a-range-of-bytes.md), [What comes back from an address is named by the address](0039-what-comes-back-from-an-address-is-named-by-it.md)

## Context

Every file with text is one row of one table, and one row has one text. A book's is its own or a model's, and never both.

A link note has two: the prose its person wrote, and what is at the address they wrote it about. Both are worth searching, and a question about either is a question about that note.

## Decision

### The two are one text, prose first

A link note is cut over its prose, a blank line, and what was fetched. An offset is into the two together, and the join is a part, so no chunk runs out of one into the other.

Whatever reads a passage back composes the same text the same way. One reader type answers for every kind of source, and this is that type reading this kind.

The prose comes first because it is what the person opened the note to write.

### The source names the producer, as every other source does

`producer` is what fetched the words and `hash` is the address they were fetched from, written by the same statement that writes the chunks — one fact, one write. A note that points nowhere clears all three.

### A chunk is identified by its text, and that is what makes this cheap

Editing the prose moves every offset below it and costs a hash of each chunk. No chunk of what was fetched loses its row or its vector, which is the property a chunk's identity was chosen for.

## Consequences

- One search answers over what a person wrote and what they were writing about,
and a hit says which of the two it is by where it falls.
- A link note's offsets are into a text that is not the file, and the reader is
the only thing that may turn one back into words.
- Nothing else in the vault has two texts, and this is the one exception the
one-text rule now carries.

## Alternatives considered

**A second `sources` row for the same path.** Rejected for the reason a book's reading is not one: the row could only be conjured out of band, a hit would carry a path the person cannot open, and the two rows would be joined by a column pointing the wrong way for reading a passage back.

**What was fetched is not searched at all.** Rejected: it is the whole reason the address was pasted.

**What was fetched is written into the note's body.** Rejected for the reason a recording's transcript is not a note: the words are what a machine made, the body is what a person typed, and files being the source of truth rests on that difference.
