# ADR-0038: Where a passage is in a book, said so a person can find it

- **Status:** Accepted
- **Date:** 2026-08-20
- **Applies to:** `modules/apps/desktop`
- **Supersedes:** two decisions of ADR-0036 — the page mark, and *"a recognised document names no parts"*
- **Related:** ADR-0006, ADR-0017, ADR-0036, ADR-0037

## Context

A person asked an agent to open a scanned book where it speaks about Jayadeva
Gosvāmī. The agent answered "page 33", opened the middle of the chapter, and lit
a run of text that crossed a heading and a footnote. Three separate things were
wrong, and each was the application knowing less than it had been told.

**The number was invented.** pdfium returns an empty label for every page of
that book — it carries no page-label dictionary — and the reader quietly filled
in the page's position instead. The page in the person's hands prints **2**. They
were given a number that appears nowhere in the book, stated as if it were the
book's own.

**The heading could not be found.** It stands in the reading as
`IAYADEVA GOSVAMI'S LIFEINNABADWI`. There is no token `Jayadeva` in it, so the
lexical half cannot match; it is not language, so the dense half embeds noise.
Both halves answered honestly about what they were given.

**Nothing knew where a section began.** The layout model names a region
`doc_title` or `paragraph_title`, and the artifact writer kept the text and threw
the label away.

## Decision

### A page is said by what it prints, or by where it stands, and never one as the other

Two facts, kept apart:

- **Where the page stands in the file** is known for every page and is stored
  nowhere: the marks are written one per page in order, so it is the position.
- **What the page prints on itself** is known sometimes, and stored when it is.

Three answers to the second, in order: what the recogniser read in the region the
layout model calls the page's number; what the document says, **when it says
something**; and nothing. The fallback that hid the difference is gone — asking
what a document calls a page and asking what to show a person are two questions.

A number read off a page is a few characters a model guessed, so it is taken only
when it looks like one: digits, or roman numerals, and short.

Where a page prints nothing, a location says so — *page 33 of the file* — and
never a bare number a person would go looking for on the paper.

### A reading names its parts

The layout model names the headings; a sidecar keeps them, one record a part:
where the heading begins in the prose, how long it runs, and how deep it is.
`Places` are built from it, so a recognised document names its parts exactly as
a book with an outline does — `location` says the section, and a passage can be
opened where its section begins.

**The structure is kept rather than inferred from the words**, and that is the
whole reason it is kept at all. `IAYADEVA GOSVAMI'S LIFEINNABADWI` is where a
section starts however badly it reads, and no search over that text would ever
find it.

A sidecar naming parts that run backwards, or past the end of the prose, is
refused whole: one written for other bytes is not a worse answer, it is another
document's answer.

### A line's boundary is measured from the line

The detector answers with the text's own outline drawn **inside** the letters,
short by a share of the line's height. What widened it again was a fixed number
of pixels — about four pixels of the page on a heading, against fifteen to twenty
of shrink. So the top of every capital and the last letter of every line were cut
away, which is where `NABADWIP` lost its P, `PADMĀVATĪ` its Ī, and where a `J`
with its serif sliced off was read as `I` or `L`.

Widened by eighteen instead of ten, the headings come back as words a search can
find. The numbers are in `docs/performance.md`; the middle of where it stops
mattering was chosen rather than the best single measurement, because one book
does not settle a default.

## Consequences

**Positive**

- A person is told a number they can find on the page, or told plainly that the
  book prints none.
- A search reaches a heading, which is what a person asking "where does it speak
  about X" is asking for.
- A passage says which section it is in, and a section can be opened at its
  start.

**Negative**

- Every reading made before this carries an invented number in its marks and no
  parts. Nothing can tell them apart from honest ones, and nothing needs to: the
  book is read again, which is an hour, and the next reading is right.
- One more file beside each reading.

**Neutral**

- The retroflex letters — `ṛ ṅ ṭ ḍ ṇ ṣ` — are absent from the alphabet this
  recogniser carries, and no boundary and no setting reaches them. A word gap in
  a display face no wider than its letter gaps is one line to any threshold, so
  that space is the recogniser's guess alone.

## Notes

The fixed widening is this binding's, not the model's: PaddleOCR's own
post-process unclips the polygon by an amount taken from its area and perimeter.
Doing that upstream would leave nothing to tune per document, and it is where the
principled fix belongs.

Two measurements guarded this from being a change tuned to one book. The body
text of twenty pages spread across it was measured against the document's own
text layer at every setting, and the heat threshold and the side a part is read
at were each swept and found to move nothing worth having.
