# An EPUB is reflowed in the window, and addressed by byte offset

- **Status:** Accepted
- **Date:** 2026-09-06
- **Applies to:** `modules/libs/core`, `modules/libs/ui`, `modules/apps/desktop`
- **Related:** [A book's text is a cache or an artifact](0015-a-books-text-is-a-cache-or-an-artifact.md), [A passage is a range of bytes](0016-a-passage-is-a-range-of-bytes.md), [Text is cut twice](0011-text-is-cut-twice.md), [One service to a subject](0034-one-service-to-a-subject.md), [How an interface component is built](0023-how-an-interface-component-is-built.md)

## Context

A vault holds two kinds of book under one word. A PDF is paginated by whoever made it: a page is a fixed rectangle, it is the same rectangle on every screen, and what a reader shows is a picture of it. An EPUB is markup, and it has no pages at all until something lays it out. The same file is forty pages on a phone and twelve on a monitor, and eighteen again when the reader sets the text larger.

Both are `book` to the index, and both are read at offsets into one text stream. What differs is everything about how a person is shown one.

## Decision

### A reflowing book is set in the window, not drawn into pictures

The application answers with the book's markup and the window sets it. Nothing renders a page of an EPUB into pixels.

The markup is the book's own, taken down to the elements a book is drawn from. The book's stylesheets do not survive: how a book looks belongs to the window it is read in, which carries the person's theme and the size they read at.

### The markup carries the offsets, and the window never counts one

Reading a document as text and reading it as elements are one walk of the same markup, so every run of text stands at the offset the index cut its chunks at. The window reads that number off an attribute.

Offsets are bytes. A string in a browser is UTF-16, and a book in Devanagari or Cyrillic spends two or three bytes a letter, so a window that counted its own offsets would light the wrong words in exactly the books this is for.

### A place in such a book is an offset

Where a person is reading is an offset into the book's text, and never a page. A page here is a property of the window: it changes when the pane is dragged, when the text is set larger, and when a second column appears. An offset is the book's own and survives all three.

This is the same address the index, a search result and a passage already use.

### Its pages are counted over the text, not laid out

How many pages a book has is arithmetic over the one text stream, taken without laying anything out. A page is a number of letters, and how many bytes that comes to is measured in the book's own script, so two books of one length are the same number of pages whatever they are written in.

### The two kinds of book are two readers

A reader of pictures and a reader of text share a toolbar and nothing else. What each is built on — a strip of pages fetched at a width, against columns laid out from markup — has no common part worth naming.

## Consequences

- A book opens without anything being rendered first, and a book of five thousand pages opens as fast as one of fifty.
- The number of pages is stable while the window is not, and it is the same number on every machine, because it is read off the text.
- A person reads every book in the window's own typography, and a book cannot dictate how it looks.
- A book of fixed layout cannot be shown this way, and says so rather than being reflowed into nonsense.
- The markup crosses the schema, reduced to what a book is drawn from, so the window trusts it. The escaping and the allowlist stand in one place, between somebody else's file and the window's DOM.
- A book's pictures are bytes, and bytes are what the window's one address for bytes answers: the place the picture has in the archive is the address's name for it, and the allowlist is what keeps every other entry of the archive off the window's origin.

## Alternatives considered

**Draw an EPUB into pictures, as a PDF is drawn.** Rejected: it would fix a page size the book does not have, and every change of pane or text size would be a book's worth of pixels drawn again. Nothing about a book made for a screen has a page in it to draw.

**Show the text stream the index already holds.** Rejected: it is the right offsets and the wrong book. Italics, headings, verse and pictures are how a book is read, and a reader that dropped them would be a reader nobody uses on the books this is for.

**Give each document of the spine an iframe, as other reading systems do.** Rejected: the isolation an iframe buys is isolation from the book's stylesheets and scripts, and neither crosses to the window. What it costs is selection, quoting and marking a passage, which are the things this reader is for.

**Count pages by laying every document out ahead.** Rejected: it is what makes other readers slow to open a long book, and it is measured in seconds on the engine the window runs in. The text is already one stream, and counting over it needs no layout at all.

**Address a place by an EPUB canonical fragment identifier.** Rejected: numen already addresses a place in every source by a range of bytes, and a second scheme for one format would be a second thing to be wrong about. A fragment identifier is what a reader without a text stream uses; this one has the stream.
