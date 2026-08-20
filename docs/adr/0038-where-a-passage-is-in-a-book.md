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

**The number was two numbers.** pdfium returns an empty label for every page of
that book — it carries no page-label dictionary — and the reader quietly filled
in the page's position instead, stated as if it were the book's own. The page in
the person's hands prints **2**, and the pane that would open it says 33. Which
number is right depends on what is being asked, and nothing said which was being
answered.

**The heading could not be found.** It stands in the reading as
`IAYADEVA GOSVAMI'S LIFEINNABADWI`. There is no token `Jayadeva` in it, so the
lexical half cannot match; it is not language, so the dense half embeds noise.
Both halves answered honestly about what they were given.

**Nothing knew where a section began.** The layout model names a region
`doc_title` or `paragraph_title`, and the artifact writer kept the text and threw
the label away.

## Decision

### A page is called where it stands in the file, and nothing else

One page, one number, and it is the number the viewer opens at. The marks are
written one per page in order, so the number is the position and is stored
nowhere. A location says *page 33 of the file*, which is what the pane shows
while a person reads it.

What the paper printed is not kept. A viewer counts from the first sheet and a
printed book counts from wherever its body starts, so the two disagree on every
page of the front matter and by a constant after it. Told both, a person has to
work out which number is being talked about, and an agent saying "page 14" over a
pane that says 44 reads as a broken program.

This takes with it the reading of the region the layout model calls the page's
number, and the page-label dictionary a born-digital PDF may carry. Neither is
wrong; both are a second name for one page.

A book made for a screen is the exception, and it is not one: an EPUB has no
pages of its own, so the page breaks it names from the printed edition it was set
from are the only page names it has.

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

- A person is told one number for a page, and it is the one on the screen in
  front of them.
- A search reaches a heading, which is what a person asking "where does it speak
  about X" is asking for.
- A passage says which section it is in, and a section can be opened at its
  start.

**Negative**

- Every reading made before this carries a number in its marks and no parts. The
  number is passed over where an artifact is read, so an old reading loses
  nothing by it; the parts are why the book is read again, which is an hour.
- One more file beside each reading.

**Neutral**

- A person holding the paper cannot be given a citation into it, and nothing
  here will produce one. Reading the number off a scan was measured before it
  was dropped, and what it cost is in `docs/performance.md`.
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
