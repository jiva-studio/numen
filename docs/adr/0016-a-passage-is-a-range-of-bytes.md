# ADR-0016: A passage is a range of bytes

- **Status:** Accepted
- **Date:** 2026-08-25
- **Applies to:** `modules/libs/core`, `modules/apps/desktop` — the asset routes
- **Related:** ADR-0004, ADR-0005, ADR-0011, ADR-0014, ADR-0015

## Context

A search, and an agent, answer with a place in a book, and the window has to open that book and light it. A PDF addresses a place by page and rectangle, a plain file by an offset in its bytes, and the next format by something of its own.

What travels from the core to the window is settled once, because every producer of a passage and every viewer of one is written on one side of it.

## Decision

### A passage travels as a range of bytes

What goes from a search, or from an agent, to the thing that shows a passage is `{path, start, length}` — a run of the text the source's chunks are places in. **Nothing above the viewer holds a rectangle.** Each format turns that range into its own address, so a third format is one new viewer and no change above it. The schema that carries it to the window is ADR-0005's.

### One shape says where a run of text sits, and it has two producers

`highlight` holds it: a box is a page, a run of bytes, and a rectangle in **fractions of the page**, so a page drawn at any size lines up by multiplying.

Two things produce it and nothing above asks which: a recognition, kept on disk because a model made it and no machine here remakes it cheaply, and a document's own text layer, kept nowhere because it answers per word on demand.

Which one answers is decided by `text_from` together with a check that the file is still the bytes the reading was made from. A document rewritten since is lit from its own layer, which is the words that are there now.

**The layer's boxes are never written down.** A file of them would need invalidating, sweeping, and a rule for the day a recognition arrives, and the wrong answer to that last one puts one producer's rectangles against another producer's offsets, which lights the wrong words and says nothing.

### Pages are drawn in this process, and the document never travels

The window is sent a picture of a page. A scan is hundreds of megabytes, PDF is a poor format to deliver a piece at a time, and what says where such a book's words sit is the recognition, which is here.

### A vault file is an asset

```
GET    /assets/<id>                          what it is
GET    /assets/<id>/pages/<n>?wide=W         one page drawn, where the asset has any
GET    /assets/<id>/marks?start=N&length=M   where a run of its text sits
GET    /assets/<id>/cues[?start=N&length=M]  the transcript, as JSON
PUT    /assets/<id>/cues                     put the transcript right
DELETE /assets/<id>/cues                     take the transcript away
POST   /assets/<id>/recognise                read the pages
POST   /assets/<id>/transcribe               write down what is said
POST   /assets/<id>/proofread                put a reading or a transcript right
```

A recording's bytes are not here. They are served ranged, from a loopback port, at an address the answer to `GET /assets/<id>` carries: a media element speaks the protocols of the world and not the scheme one application serves its window under.

```
```

`<id>` is the vault path, percent-encoded, because a file has no other name the window holds. **The handler routes on the escaped path**: Go decodes before a handler sees it, and a decoded separator runs the member and what hangs off it together. These routes are served by the same adapter that serves the generated handler (ADR-0005).

### Two bounds on what is held

**Open documents** are bounded by a worker pool. Opening takes one and holds it, a recognition holds one for its whole run, and a page asked for while none is free is answered busy: a hung request is not an answer.

**A drawn page** is kept in this machine's cache folder, keyed by the document's fingerprint, the page and the width it was drawn for, and what is held is swept oldest first. It is not in the vault and not beside the document; losing it costs the drawing again.

What a page is called, and what a location says to a person, is [`../reading.md`](../reading.md).

## Consequences

- A format arrives as one viewer, and everything that produces a passage is left alone.
- Every rectangle is multiplied by the width the page was drawn at, on both sides of the wire.
- A viewer holds a document open, and that competes with recognition for the same pool.
- A page asked for with the pool full comes back busy, and the window has to show that.
- A document rewritten since its recognition is lit by its own layer, and the two disagree about where a word is.
- Drawn pages are lost with the cache folder, and cost the drawing again.

## Alternatives considered

**A rectangle carried down from the search.** Rejected: everything above the viewer would then hold the geometry of every format, and a format with no pages has no rectangle to hand over.

**The text layer's boxes written to disk beside the recognition.** Rejected: they are cheap to ask again and expensive to keep true, and the day a recognition arrives for the same document there are two files claiming the same words.

**The document streamed to the window and drawn there.** Rejected: a scan is hundreds of megabytes over a loopback port for one page, and where a scanned book's words sit on the page is worked out in this process.

## Notes

pdfium reports a page's size **as it is drawn** and its characters in the space its text is written in. On a page carrying a quarter turn the two disagree, and dividing one by the other gives rectangles inside `[0,1]` and wrong, with nothing saying so. The turn is asked for and the corners mapped through it.

The offsets of the two producers are the same offsets: the characters pdfium hands over join into exactly what it returns as the page's text.
