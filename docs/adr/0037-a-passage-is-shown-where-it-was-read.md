# ADR-0037: A passage is shown where it was read, and one shape says where that is

- **Status:** Accepted, except where noted below
- **Date:** 2026-08-20
- **Applies to:** `modules/apps/desktop`
- **Related:** ADR-0000, ADR-0006, ADR-0013, ADR-0017, ADR-0036

## Context

ADR-0036 decided that a recognition keeps where every word sat on the page: one
record per run of words, carrying the page, the run of bytes in the prose, and
the rectangle as a fraction of that page. For the 546-page book that is 51 169
records, every one checked against the words it names.

Nothing read them. The only caller outside tests was the trim that keeps the
file honest across a resume, and the window had no component that drew pixels at
all — not one image, canvas or object URL in either package. A search hit in a
document was drawn and offered nothing, because a book is not a note.

So the geometry was on disk and unusable, and a person who found a sentence in a
546-page scan was told which page it was on and left to open the book themselves.

## Decision

### A passage travels as a range of bytes, and only a viewer knows about pixels

What goes from a search, or from an agent, to the thing that shows a passage is
`{path, start, length}` — a run of the text the source's chunks are places in.
Nothing above the viewer holds a rectangle.

Each format turns that range into its own address. A document turns it into
pages and rectangles. A note is already at its offset. An EPUB would turn it into
a range in the rendered spine item.

A third format is therefore one new viewer and no change above it.

### One shape says where a run of text sits, and it has two producers

`internal/core/placed` holds it: a box is a page, a run of bytes, and a rectangle
in **fractions of the page**, so a page drawn at any size lines up by multiplying
and nothing is recomputed when the zoom changes.

Two things produce it and nothing above asks which:

- **A recognition**, kept on disk, because a model spent an hour making it and no
  machine here makes it again.
- **A document's own text layer**, kept nowhere, because it is tens of
  milliseconds a page — the same reason ADR-0036 keeps the layer's text nowhere.

Which one answers is decided by the column that already decides which *text* a
source's chunks are places in, together with one check that the file is still
the bytes that reading was made from. A document rewritten since is placed by
its own layer, which is the words that are there now. Nothing indexes a
rectangle against a text the chunks are not in.

That property is why the layer's boxes are not written down. A file would need
invalidating when the document changes, sweeping, and a rule for what happens
when a recognition arrives — and answering that last one wrongly puts a
recognition's rectangles against a layer's offsets, which lights the wrong words
and says nothing.

The shape lives in a package of its own. A layer is not what a model saw, and
`internal/core/ocr` says on its first line that it is.

### Pages are drawn here, and the document does not travel

pdfium is already in the binary and already renders pages; recognition is built
on it. The window is sent a picture of a page.

The book this was built against is 223 MB. PDF is a poor format to deliver a
piece at a time, and a scan carries no text layer at all — so a viewer in the
window would be handed the whole file to show a page of it, and would still have
nothing selectable to show for it. What makes that book's words placeable is the
recognition, which is here.

### A file in the vault is an asset, addressed as one

```
GET /assets/<id>                          what it is
GET /assets/<id>/pages/<n>?wide=W         one page, where the asset has any
GET /assets/<id>/marks?start=N&length=M   where a run of its text sits
```

`<id>` is the vault path, percent-encoded, because a file has no other name the
window holds. The handler routes on the escaped path: Go decodes before a
handler sees it, and a decoded separator runs the member and what hangs off it
together.

A recording would answer the first with a duration and grow a facet of its own.
Without a collection to hang them on, each kind of asset takes another word at
the root.

### Two caches, and they are not the same thing

An **open document** is about the worker pool, not about memory: opening takes
one of `max(NumCPU, 2)` and holds it, a recognition holds one for an hour, and
waiting for one is two minutes before it gives up. So few are held, they are let
go when idle, and a page asked for while none is free is answered busy — two
minutes of a hung request is not an answer.

A **drawn page** is in memory, keyed by the document, the page and the width it
was drawn for, and nothing goes on disk. By ADR-0000's rule it is a cache: made
here, deterministically, in the hundreds of milliseconds. A disk cache would need
the width in its key, eviction and a sweep, and has earned none of that.

> **Reversed on measurement, 2026-08-20.** Drawing a page of the 600 dpi scan is
> half a second at any width — the library decodes the page's photograph
> whatever size is asked for — and it is the same half second every time the
> page is turned back to. So a drawn page is also kept in this machine's cache
> folder, under the document's fingerprint, the page and the width, and what is
> held is swept back to half a gigabyte oldest first. A page turned back to is
> read in about a millisecond. It stays a cache by ADR-0000's rule: it is not in
> the vault, and losing it costs the drawing again. The numbers are in
> `docs/performance.md`.

## Consequences

**Positive**

- A passage in a scan can be looked at, which is the whole point of having read
  the scan.
- Selection and search inside a document become possible without another format
  or another store: the geometry for them is what this puts in place.
- A document that nobody has read still opens, pages, and lights a passage from
  its own layer. The two producers make the feature whole from the first day
  rather than only for books somebody spent an hour on.

**Negative**

- The window now draws pixels, which it never did. There was no image handling,
  no asset convention and no content policy to follow, and this had to establish
  all three.
- A viewer holds a document open, and that competes with recognition for the
  same pool. The bound is small and the wait is short, and both are numbers
  somebody will have to revisit on a machine with fewer cores.

**Neutral**

- Which sentence of a passage answered a question is not decided here. What is
  lit is the passage, and narrowing it is a separate decision about what an
  agent may say.

## Notes

pdfium reports a page's size **as it is drawn** and its characters in the space
its text is written in. On a page carrying a quarter turn the two disagree, and
dividing one by the other gives rectangles inside `[0,1]` and wrong, with nothing
saying so. The turn is asked for and the corners mapped through it. It was
settled against ink: a page was drawn, the bounding box of its dark pixels taken,
and the boxes checked against it, upright and turned.

The offsets of the two producers had to be the same offsets, and that was checked
before anything was built: the characters pdfium hands over join into exactly
what it returns as the page's text. Had they differed, the text a chunk is a
place in would have had to come from the characters, and every PDF in every index
would have been cut again. `TestTheCharactersOfAPageAreItsText` keeps it true.
