# ADR-0006: Sources: extracted text into the cache, unreproducible output into the vault

- **Status:** Accepted
- **Date:** 2026-08-17
- **Applies to:** `modules/apps/desktop`
- **Partly supersedes:** ADR-0010 — what triggers extraction, transcription and
  chunking
- **Related:** ADR-0000, ADR-0002, ADR-0007, ADR-0016, ADR-0024, ADR-0029

## Context

ADR-0000 classifies what a source layer produces, and ADR-0007 decides how text
is cut and how a passage is found. What a source is inside the index, what
extraction owes its caller, and what is stored about a chunk were left open. The
application indexes notes and nothing else, so a book in a vault is invisible to
it.

What was measured, and on what date, is in docs/performance.md. The EPUB figures
there are what the decisions below were taken against.

## Decision

### A note and a book are kinds of source

One table holds everything that has text, and a `kind` says which. A note is a
source that also carries a title, an identifier, frontmatter and a basename;
those belong to the graph and are stored beside the source rather than in it.

Chunks and vectors hang on a source, not on a note. Nothing in search asks which
kind it has.

### Deterministic extraction is a cache; a model's output is an artifact

The line is ADR-0000's and is restated because it is what divides one source
layer into two storage rules. Text that a local, deterministic extractor takes
out of a file — an EPUB's spine documents, a PDF's text layer — is cache and
lives in the index. Text that a model or a service produces — recognition of a
scan, a transcript of audio — is an artifact, is written into the vault, and is
then chunked from there like any other source.

### Discovery triggers processing

A source found in a vault is extracted, cut and embedded without anybody asking
for it. Search answers over a whole vault, so nothing in one may be waiting to be
requested. Processing runs in the background and reports what it is doing
(ADR-0018).

**ADR-0010 states the opposite** — expensive processing is "triggered by type and
on demand, not by discovery" — and the second half of that rule is replaced here.
What survives it: processing is still triggered by type, so a format no extractor
handles is never opened; it is still incremental on the fingerprint, so a scan
that finds nothing changed extracts nothing; and a source that nothing links to
is still an ordinary source rather than a problem.

### What is not stored, and what is

The text of a source is not stored. What is stored about a note beside its chunks
is its title, its headings and its frontmatter — the parts a graph is drawn from,
which ADR-0016 already decided and which are verbatim file text. Body prose is not
stored for any kind of source.

The full-text index keeps no copy of what it indexed, and is contentless. It does
keep term positions, so the sequence of words in a window can be recovered from it
without opening the file — lowercased and without punctuation. That is a property
of the index and not a copy of the source, and it is written here so that nobody
reads "the text is not stored" as more than it says.

### Extraction never refuses

An extractor that finds text returns it. **No structure is a normal outcome**, not
an error: a book with no navigation document and no headings is one span, cut into
windows by size alone. A part that will not parse is dropped and the rest is
returned.

Structure is taken at the best level the file offers, and each level is judged by
what it yields rather than by whether it is there. A navigation document with two
usable entries over a whole book is not structure.

### Extracted text is not stored

The index keeps where a passage is, not what it says. Showing a passage re-reads
the source. This is the rule ADR-0016 gives a note's body, applied to every kind
of source, and what re-reading costs is in docs/performance.md.

### A chunk has two locations

**The machine's location is `start` and `length`** — an offset and a length in the
extracted text of the source. It is always present, and it is what a passage is
read back through. Everything the application does with a chunk goes through it.

**The human's location is `location`**, a projection in the sense ADR-0016 gives
parsed frontmatter: nullable, and made only of keys that belong to the vocabulary
of the source's own format — the spine document and navigation entry of an EPUB,
the print page where the book carries one. The index defines no names for places.
Most sources give nothing, which is why the column is nullable, and nothing reads
a chunk back through it.

### What was used is recorded

**A recipe, on the source:** which extractor, at which settings, produced its
text. **A model, a number of dimensions and a kind of quantisation, with the
vectors.**

Both are staleness keys beside the fingerprint, so a source has three — the file
changed, the recipe changed, the model changed — and each has to be answerable as
a query.

**The fingerprint is the path, the size and the modification time, and nothing
else.** A warm scan opens no file, so there is nothing cheaper it could compare.
The content hash is recorded when a source is read and is not compared: it says
which file this is, for the day a moved book is to keep its chunks, and it costs a
read to compute.

That leaves a gap, and it is a real one: a file whose content changed while its
size and modification time did not is skipped. Two ordinary tools do exactly
that — an archive restored by `unzip`, which sets the time from the archive, and
`rsync -tc`, which compares content and preserves the time. The consequence is
worse than a missing result: the chunks describe text the file no longer has, so
a search answers with a passage sliced at the wrong place, or with an empty one.

There is no cheap signal that closes it. What is provided instead is a way out:
`scan --rebuild-index` reads every file whatever the index remembers, and is one
command. A person who restores a vault from an archive runs it.

The kind of quantisation is stored because a stored vector's type cannot be
recovered from the length of its blob.

### The quantisation scale is a constant

The scale that turns a float into a byte belongs to the model and is written down
with its name. It is never computed from what the corpus holds: a scale that moves
when a source is added makes every vector written before it incomparable with
every vector written after, and nothing in the stored bytes shows that it moved.

### A chunk is written before its vector

State is read from the data. What needs embedding is a chunk that has no vector,
so there is no progress column and nothing to reconcile after a crash.

The two representations of one vector, the byte one and the bit one, are written
in the same transaction as each other. A chunk holding one and not the other is
invisible to that question and stays that way.

### The full-text index keeps no copy

The lexical index is built over chunks (ADR-0007) and holds no copy of their
text. A lexical hit is a chunk, and the words come from the source, exactly as
ADR-0016 states for a note.

### Vocabulary

These words are defined here and entered in ADR-0024.

- **source** — a thing with text that the index holds. A note and a book are kinds
  of source.
- **chunk** — one window of a source's text, as a row. Both of the sizes ADR-0007
  cuts are chunks; the large one is the chunk with no parent.
- **location** — where a chunk sits, in the terms its own format uses. Nullable.
  `start` and `length` are not a location; they are the key.
- **passage** — what a search returns: the text around a hit, and where it came
  from.

## Not solved

**The same book in two formats.** Two files of one work have different content,
different extracted text and different chunks, and nothing available tells that
pair apart from two different books. Both are indexed and both answer. Which to
keep is the person's choice, and the application does not make it for them.

## Not decided

**PDF and audio.** ADR-0000 classifies their output; nothing here decides how
either is produced, nor where in the vault an artifact of one is written — that is
ADR-0004's shape. EPUB is the only extractor this decision covers.

**Why transliterated Sanskrit does not surface.** As ADR-0007 records: it is
indexed, no question in the acceptance set reaches it, and whether the cause is
the model, the transliteration or the way verse is cut is not established.

## Consequences

**Positive**

- Dropping a book into a folder is the whole gesture.
- The index stays smaller than the text it describes, and a source can be
  extracted again at any time, because nothing derived from it is authoritative.
- One table of sources gives notes and books one ranking and one code path, which
  is what ADR-0007 requires.

**Negative**

- Every result costs a read of the file it came from, on the search path, and a
  large source costs more than a small one.
- A source whose file moved or went answers nothing until the next scan, and an
  offset into text that has changed reads out the wrong place. Removing a book has
  to be as real a path as removing a note.
- Processing on discovery spends the cost of a whole library before anybody asks a
  question, so it has to be interruptible and resumable.
- A nullable `location` means the interface has to be able to say where a passage
  is with nothing but the name of its source.

## Alternatives considered

**Store the extracted text in a column.** Rejected: it is a second copy of what
is on disk, and slicing a passage out of it costs in proportion to the size of the
stored value (docs/performance.md), so the largest books are the ones it serves
worst.

**Extract when a source is first asked for.** Rejected: the first question put to
a new vault would answer over nothing, and the work is the same work moved to the
moment somebody is waiting for it.

**A table for books beside the table for notes.** Rejected: it repeats every
column that exists because a thing has text, and ADR-0007 wants one unit and one
ranking, which two tables can only express by joining themselves back together.

**A location the index defines** — chapter, page, section — with each format
mapped into it. Rejected: it either drops what the format said or names a place
the format has no name for, and the mapping is a rule per format that nobody can
check against anything.
