# ADR-0036: A PDF's text layer is a cache; reading it with a model is an artifact a person asks for

- **Status:** Accepted
- **Date:** 2026-08-19
- **Applies to:** `modules/apps/desktop`
- **Answers:** ADR-0006 — "PDF and audio" under *Not decided*
- **Related:** ADR-0000, ADR-0002, ADR-0006, ADR-0013, ADR-0015, ADR-0017

## Context

ADR-0006 decided what a source is and how one is cut, and closed with **"PDF and
audio. ADR-0000 classifies their output; nothing here decides how either is
produced."** EPUB was the only format an extractor handled. A PDF in a vault was
invisible: the fixture vault's `assets/paper.pdf` existed to be asserted absent.

A PDF is two things wearing one extension. Some carry a text layer a machine can
take out, deterministically and for free. Some are photographs of paper and carry
nothing. And some carry a text layer that is somebody else's recognition, which
looks like the first and behaves like the second.

The book this was worked out against is the third kind: 546 pages, 600 dpi, with
an OCR text layer from 2023. It reads well enough in English and destroys the
transliterated Sanskrit the book is set in — `pāṅti` comes out `paiiti`, and not
one of its 1.7 million characters carries a diacritic.

## Decision

### A PDF is a book, and what read it is its recipe's business

One kind of source covers every file with text that nobody typed here. What
takes the text out is decided from the file's name and recorded in the recipe,
which already had to name the extractor.

The recipe was compared against one string. With two readers that is a bug and
not a detail: every PDF would differ from the EPUB recipe on every scan and be
re-read forever. **The staleness question takes the set of recipes now in use**,
and a source carrying none of them owes its text.

### The text layer is a cache; a recognition is an artifact

Straight from ADR-0000, and restated because it is what divides one format into
two paths. A text layer is deterministic local extraction, so it is read on every
scan and stored nowhere. A model's reading is not reproducible, so it is written
into the vault and the source is cut from it afterwards.

### Nothing decides between them automatically

**The layer is used, always. Recognition is something a person asks for.**

There is no reliable test of a text layer. A page of Chinese inside an English
book is ordinary; a page of English recognised as Chinese is a defect; and
nothing in the words says which. A heuristic here would either re-read documents
that were fine — an hour each — or leave broken ones broken while reporting that
it had checked.

So there is a command and there are tools, and no trigger. What an agent needs to
use them is not the reading but the listing: **which documents have been read**
is a question the index answers, and without it an agent reads one document twice
and another never.

### Where a source's text is, is a column

`sources.text_from` names the producer that made the text a source's chunks are
places in. Null is the ordinary case, and the only case for a note, an EPUB, or a
PDF nobody has read.

It holds a producer and not a name. The four files one reading writes are all
composed from the producer and the hash, and the hash is a column here already,
so a name is composed where it is needed and nothing takes an extension off a
stored string.

It is not in the recipe. The recipe is compared against the strings the running
binary produces, so a per-source value in it makes every recognised source
permanently unequal to all of them. **A recipe names a procedure.**

Three rules hold it together:

1. The column is authoritative for reading a passage back. A source naming a
   producer reads from that producer's files or reads nothing — falling back to
   the document would slice one text at another text's offsets, which is a wrong
   answer given confidently.
2. It is written by the same statement that writes the chunks cut from that text,
   and cleared by the same write that clears `hash` and `recipe`. A row naming a
   text its chunks are not offsets into answers with the wrong words, so the two
   are one fact and one write.
3. **A reading is named by the hash of the bytes it was made from.** A document
   renamed or moved keeps its recognition, two copies of one document share one
   reading, and a fingerprint that moved without the content — `unzip`,
   `rsync -t` — costs one read rather than an hour.

Rule 3 is what makes rule 2 safe. Without it, a sync client touching a file
throws a recognition away.

Rule 3 is also what a sweep has to ask. A source that leaves the vault takes the
files of its reading with it, and the question is whether **any** source still
names that reading — not whether the path that named it went. Asked by path, a
rename deletes the reading it was meant to keep.

### One reader answers for every kind of source

Three places turned a path into text and all three were hardcoded to EPUB: the
extractor cutting a source, a search showing a passage, an embedder re-slicing a
window. They now ask one type. A chunk keeps an offset into the text a reader
produced, and a second reader producing other text at other offsets reads the
wrong place and says so with confidence.

### The artifact lives in the service folder, in an area of its own

ADR-0013 decided the shape: one folder per vault, subfolders inside, and it
rejected a hidden folder per feature. A recognition goes to `.numen/ocr/`.

The area is named for what made the files. Their format, the fields recorded
beside them and what a place in one is called all belong to the thing that wrote
them; a transcript of audio would share the storage and none of the rest, and
**no shared shape is defined here for a second producer that does not exist**.

Getting in required a way past two guarantees worth keeping — the walk skips the
service folder, and every path in it is refused. Neither is loosened. `within`
splits into the containment rule and two complements: the vault as the person's,
and the folder as the application's. A path belongs to exactly one of them, and
that is asserted as a property rather than trusted. A separate port and a
separate type write there; `VaultWriter`, which an agent can reach, cannot.

### What the artifact holds

Plain text, with each page marked by a form feed, its printed label, and another
form feed. The marks are taken out when it is read, so an offset in what comes
back is an offset in the prose.

That is what keeps `location` meaningful: the pages come back from the marks, and
a result says which printed page it came from. **Places do not survive.** A
recognised document names no parts — what a layout model calls a heading is a
shape on a page, not an entry in an outline — so it is one span, which ADR-0006
already calls an ordinary outcome.

A page with nothing on it is still written; a document with nothing on it writes
nothing at all. An empty artifact would stand in for a text layer that worked,
and rule 1 leaves nothing to fall back to.

### It is written in batches and it resumes

A document is an hour. ADR-0006 requires processing to be interruptible and
resumable, and a run stopped part way keeps what it read: pages are appended to a
partial file, and the count of how far it got is appended after them. The count
is what makes the batch in front of it count, so a batch that did not land whole
is one no count claims and the next run reads those pages again.

**A partial is the source's text while it is being written.** After every batch
the source is cut again from what has been read, so a book answers questions
about the pages that have been read while the rest of it is still being read. The
whole book waiting on the last page is an hour of finished pages on disk that
answer nothing.

What that costs is stated rather than discovered: while a run is on, the source
is cut from the recognised prefix only, so the part of the book not yet read
drops out of search until the run finishes.

Reading it back tries the finished file and then the partial. The rename to the
final name is still the last act, and the two names are one question because both
are composed from the producer and the hash.

### Where a passage was read is kept, and only here

A reading is written beside a second file holding, per run of words, where on the
page it was read: the page, the run of bytes in the text, and the rectangle as a
fraction of that page. Fractions, so a viewer multiplies by whatever it rendered
into and needs neither the dpi nor the page size. Fixed-width records in page
order, so a viewer wanting one page seeks to it and reads no more.

**It is kept because a model made it and this machine remakes it in an hour.**
Every other map from a text offset to a format's own address is milliseconds
away and is not kept: a PDF's own layer answers per word on demand, an EPUB's
spine item is re-parsed, and a note's offset is already its address.

What travels between a search and a viewer is a range of bytes in the source's
text, and nothing above the viewer knows about pixels. Each format turns that
range into its own address, so a third format is one new viewer and no change
above it.

Which range is a question this does not answer. It is decided where the passage
is chosen, and the coordinates make no more claim than the offsets do.

## Consequences

**Positive**

- Dropping a PDF into a folder is the whole gesture, and it costs no models, no
  weights and no network.
- A recognition survives losing the index, which is what ADR-0000's "rebuild the
  cache from artifacts alone, offline" demands.
- The reader that produced a text is named in the recipe, so changing it re-cuts
  what it produced and nothing else.

**Negative**

- The service folder now holds one thing that is not disposable. Until now it
  held an identity whose loss costs a re-scan.
- A recognition is only as good as the models, and choosing them is a person's
  problem: a recogniser that cannot spell a script writes plausible nonsense, and
  the only thing that catches it is `window.legible` refusing to index the worst
  of it.
- Two paths through one format is two paths to keep working, and only one of them
  is exercised by a document that carries text.

## Alternatives considered

**A heuristic on the text layer** — unknown words, missing marks, a language
guess. Rejected: every measure of "is this text good" is a measure of "is this
text like what I expected", and a vault holds what it holds.

**The artifact beside the document** (`book.pdf.md`). Rejected: it is a file in
the vault, so it is walked, indexed and linkable as a note, and moving the
document leaves it behind. The application's own folder is where the application
writes.

**The artifact as a source of its own row.** Rejected: the walk skips the service
folder, so the row could only be conjured out of band; a search hit would name
the artifact instead of the document; and the two rows would need joining back
together by a column pointing the wrong way for reading a passage.

**A flag that lets a writer into the service folder.** Rejected: it turns a
structural refusal into a caller's argument, true only for the call sites that
exist today.
