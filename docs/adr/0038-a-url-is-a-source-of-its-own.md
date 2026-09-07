# A url is a source of its own

- **Status:** Accepted
- **Date:** 2026-09-07
- **Applies to:** `modules/libs/core`, `modules/apps/desktop`
- **Related:** [A source is text in one table](0010-a-source-is-text-in-one-table.md), [The note file](0018-the-note-file.md), [A recording is a source of its own](0030-a-recording-is-a-source-of-its-own.md), [What comes back from an address is named by it](0039-what-comes-back-from-an-address-is-named-by-it.md)

## Context

A person watches a lecture, reads an article, and wants what is in it where the rest of their thinking is. What they have is an address.

A vault holds files. An address is not one, and what is at it is somebody else's and may go away.

## Decision

### An address is a file of its own, and it holds nothing else

```
02 лекция. Бхагавад-Гита. Введение.url
```

```ini
[InternetShortcut]
URL=https://www.youtube.com/watch?v=A4OZ4L9TCpM
```

`.url` is what this file is called everywhere and `[InternetShortcut]` is what it holds, so a file manager and a browser open it too.

`sources.kind` takes a fourth value. A file is a url from its name alone, as every other kind is.

### It holds one text, and that text is what was fetched

The words a site published with a video, or the prose a page is written around. There is no body to write in, so nothing has to decide which of two texts a write is for: an agent asked to put a transcript right has one place to put it, and a person editing the words in the tab writes to the same place.

The text is named by the address ([0039](0039-what-comes-back-from-an-address-is-named-by-it.md)), so two urls on one video share it and renaming one keeps it.

### It is named the way a book is

What it is called is the name of the file. A fetch that learns what is at the address renames the file; a person who renames it has named it.

There is no frontmatter, so no title key, no identifier, and no typed link leading out of it. A book leads nowhere either.

### What a person writes about it is a note, and it points here

A person writes about a lecture the way they write about a book: a note of their own, pointing at the file.

### The tab is the recording's tab

The player heads the pane and the words stand under it, one line a cue, the time in the gutter, and choosing a line plays from the moment it was said. It is the same tab a recording opens in, because it holds the same thing.

## Consequences

- The kind decides everything downstream, so nothing reads a file's name a second
time to find out what it is.
- A person cannot type into a url, and writes a note beside it instead.
- A `.md` file carrying `type: link` means nothing now, and is an ordinary note
until it is converted.
- A url carries no typed link out of itself. What points at it is a question
for whatever resolves a link's target, which today reaches notes alone.
- A search hit is the url, and what a person wrote about it is a second hit
in their own note.

## Alternatives considered

**A link is a fifth kind of note, `type: link` with `url` beside it.** Rejected after it was built: the note has a body, so a video stood in a file holding two texts — the prose a person writes and the words fetched from the address — with no rule saying which a write is for. The window drew the fetched words and left the prose nowhere on the screen, and an agent asked to put the transcript right wrote into the body, because the body was the only thing it could write.

**A link note with the prose shown beside the transcript.** Rejected: the ambiguity stays. Two texts in one file is two answers to "write this here", and every tool, every agent and every person has to hold the difference in their head.

**A note whose body is refused.** Rejected: a note whose every rule is suspended is not a note, and the file format would carry an exception nobody reading it would expect.
