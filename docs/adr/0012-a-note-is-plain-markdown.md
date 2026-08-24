# ADR-0012: A note is plain markdown any editor can open

- **Status:** Accepted, except where noted below
- **Date:** 2026-08-15
- **Applies to:** the vault format — every application that reads or writes one
- **Partly superseded by:** ADR-0040 — `title` is written when a note already
  carrying it is renamed
- **Related:** ADR-0000, ADR-0001

## Context

The note file is the one artifact the user actually touches. ADR-0001 makes it
the source of truth, which means everything the system knows about a note has to
survive inside that file — identity, links, hierarchy, cards, anchors.

That is a lot to express, and it puts the same question every note tool faces:
invent a format that says exactly what you need, or restrict yourself to what
other tools already understand. Inventing is easier at every individual step and
irreversible in aggregate: once a vault is full of syntax only one application
can read, "your files are yours" is no longer true, whatever the licence says.

The number is unusual for something this fundamental. ADR numbers are identity,
not reading order — this decision was simply taken after the storage ones. See
the index for the order to read them in.

## Decision

**A note is a UTF-8 markdown file. Everything the application adds must stay
inside what a third-party markdown editor renders without complaint and a human
reads without a manual.**

### Which files count as notes

The extensions treated as notes are a setting, and the default is `.md` alone.
It is a setting because markdown is written under several names — `.markdown`,
and whatever a user's other tools produce — and because the answer is a
preference rather than a property of the format. It defaults to one extension
because a default that guesses widely is a default that indexes things the user
did not mean.

Two places are never notes, whatever the setting says: the service folder, which
is the application's own (ADR-0013), and any directory whose name begins with a
dot, which holds some tool's state rather than anything a person wrote. Both are
skipped whole, without descending into them.

The complete list of permitted additions:

| Addition | Where | Purpose |
| --- | --- | --- |
| YAML frontmatter | top of file | machine-owned fields |
| `[[wikilink]]` | body | link to another note |
| `^anchor` | end of a line | a name for that line |

Nothing else. No custom fences, no HTML comments carrying data, no sidecar
files, no private file extension.

### Frontmatter is where machine-owned fields live

The body is prose a human wrote. The frontmatter is a place for fields the
application owns. Anything the application needs to record about the note as a
whole goes there, not into the text.

That includes the note's identifier, if it has one. **This ADR settles only that
the identifier belongs in frontmatter** — what it looks like, when it is written,
and what happens to it when the file is renamed are ADR-0009's, and nothing here
should be read as prejudging them. The reason to fix the placement early is that
it is a property of the file format rather than of identity: an identifier in the
body would be a token in the middle of prose, and every reader of that note would
have to look at it forever.

**`title` is the first owned key.** A note is shown by a name, and the choice is
between deciding that name from the file and letting the user state it. Both are
needed: a note usually has a level-one heading and sometimes deliberately does
not, and a file called `2026-08-15.md` has a name nobody wants to read in a list.

So the name of a note is the frontmatter `title` if there is one, else the first
level-one heading, else the filename. The order matters more than the choice —
a note whose displayed name moves between versions is a note the user cannot
find twice.

The key is **read and never written**. The application does not add a `title` to
a note that has none and does not rewrite one it finds, so claiming this key
costs the user nothing except the word itself, which they were already using for
this purpose in every other tool.

> **Partly superseded by [ADR-0040](0040-a-note-is-renamed-by-whatever-names-it.md).**
> A note carrying this key is renamed by writing it. Nothing adds the key to a note
> that has none, and nothing else rewrites one it finds.

The frontmatter is shared with the user, not owned by the application, and three
rules follow:

- **Keys the application does not own are preserved verbatim** — including their
  order, and including keys it may want to own in a future version. It reads what
  it knows and leaves the rest alone.
- **Owned keys are a closed, documented set.** Each one is introduced by an ADR
  that says what it means. A key nobody decided on does not get written.
- **A collision is the user's win.** If a user's own key has the same name as one
  the application wants, the application does not overwrite it and does not
  silently reinterpret it: it reports the conflict, the same way ADR-0003 treats
  contradictory data as something to show rather than resolve.

### The test to apply to anything proposed later

Open the file in an editor that has never heard of this application. If the
addition renders as broken markup, or the note stops being readable prose, it is
rejected. If it renders as slightly unusual but plain text, it is allowed.

That is the whole line, and it is deliberately drawn at *legibility*, not at
*standards compliance*. `^anchor` is not part of CommonMark; in a plain renderer
it appears as literal text at the end of a line. Literal text a human can read is
acceptable. Markup that renders as garbage is not.

## Consequences

**Positive**

- The vault stays portable, and leaving stays possible — which is the promise
  ADR-0001 exists to keep.
- Other tools can create notes for us without knowing anything about us: a
  markdown file dropped into the vault is a valid note from the first byte.
- Compatibility with the conventions Obsidian users already have in their
  fingers comes along for free, since the permitted set is close to theirs.

**Negative**

- The format cannot express everything the system might want, and that limit is
  real, not theoretical. Anything that does not fit goes into the service folder
  as an artifact instead — which is what ADR-0000 already prescribes, but this
  ADR is what forces the question to be asked.
- Frontmatter is shared with the user: keys the application owns can collide with
  keys the user invented. A reserved-key policy is required, and taking a
  common English word as a reserved key is a decision with a cost, not a
  formality.
- Parsing is harder than parsing a format we designed. Markdown in the wild is
  irregular, and the parser has to tolerate what other editors produce rather
  than only what we produce.

## Alternatives considered

**A custom syntax or DSL for what markdown cannot express.** Rejected: it is the
one choice that cannot be walked back, because it is the vault contents that
become unreadable, not the application.

**Sidecar metadata files** (`note.md` plus `note.meta.json`). Rejected: the pair
splits on any external move, rename or copy — precisely the operations ADR-0001
promises are safe — and it doubles what the user sees in every folder.

**A private file extension.** Rejected outright: no other editor opens it, which
inverts the entire premise.

**HTML comments as a data channel** (`<!-- id: 01J8… -->`). Tempting, because
comments render as nothing and cannot break a document. Rejected for exactly that
reason: data the human cannot see is a binary format wearing a text costume, and
it makes the file dishonest — what you read is not what is there.
