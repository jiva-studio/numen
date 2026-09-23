# A source is text in one table

- **Status:** Accepted
- **Date:** 2026-08-25
- **Applies to:** `modules/libs/core`
- **Related:** [Files on disk are the source of truth](0001-files-are-the-source-of-truth.md), [One database for all vaults, outside them](0002-one-database-for-all-vaults.md), [What the index stores](0006-what-the-index-stores.md), [Text is cut twice](0011-text-is-cut-twice.md), [A chunk is identified by its text](0012-a-chunk-is-identified-by-its-text.md), [The vector index stays inside SQLite](0013-the-vector-index-stays-inside-sqlite.md), [One search, three rankings, merged by rank](0014-one-search-three-rankings.md), [A book's text is a cache or an artifact](0015-a-books-text-is-a-cache-or-an-artifact.md), [A passage is a range of bytes](0016-a-passage-is-a-range-of-bytes.md), [The note file](0018-the-note-file.md)

## Context

A vault holds notes and books together, and the index answers over both at once. A note is a file somebody types into; a book is a file somebody bought. Both are text, both are searched by the same question, and how they are filed decides whether one answer can hold them both.

## Decision

### One table holds everything with text, and `kind` says which

A note, a book, a recording, and a url are kinds of source. Every indexable file or address is a `sources` row, and `kind` says which sort it is (`note`, `book`, `recording`, `url`).

What only a note has — its title, its identifier, its frontmatter and the basename a link reaches it by — sits beside the source in a table of its own, under the same row number. Chunks and vectors hang on the source.

The columns that sources, notes and chunks occupy are defined in [What the index stores](0006-what-the-index-stores.md).

### Nothing stores the text

The index keeps where a passage is. Showing one re-reads the file (or the cached text artifact in the service folder for URL sources).

The full-text index over chunks is contentless and keeps no copy of what it indexed. It does keep term positions, so the sequence of words in a chunk is recovered from it without opening the file, lowercased and without punctuation.

### Extraction never refuses

An extractor that finds text returns it. **No structure is a normal outcome**: a book with no navigation document and no headings is one division. A part that will not parse is dropped and the rest is returned.

Structure is taken at the best level the file offers, and each level is judged by what it yields. A navigation document with two usable entries over a whole book is not structure.

### The recipe names a procedure

`sources.recipe` says what produced the text and is null while nothing has. It is the name of a procedure and carries no value that varies from one source to the next.

It names the sizes the text is cut at as well as the reader that read it, so a source cut at the command line and a source cut in the window are one source cut once. The recipe a vector was made under is a separate name, kept with the vector.

## Consequences

- Every result costs a read of the file it came from, on the search path.
- A file no reader reaches is a row with no chunks, found by its name alone.
- What one kind has and the other does not is a table and a join.
- Changing what a reader produces changes the recipe, and every source that reader served owes its text again.

## Alternatives considered

**Store the extracted text in a column.** Rejected: it is a second copy of what is on disk, and slicing a passage out of a stored value costs in proportion to that value, so the largest books are the ones it serves worst.

**A table per kind of source.** Rejected: every ranking would begin by asking which table to read, and one mixed answer would be a union written out at each of them.

**Leave the cut sizes out of the recipe.** Rejected: a source cut under one set of sizes and read under another has no column that says so, and the file is cut again by whatever opens it next.
