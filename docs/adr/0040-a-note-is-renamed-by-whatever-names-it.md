# ADR-0040: A note is renamed by whatever names it

- **Status:** Accepted
- **Date:** 2026-08-24
- **Applies to:** the vault format — every application that writes one
- **Partly supersedes:** ADR-0012 and ADR-0027 — the `title` key, for a rename
  alone
- **Related:** ADR-0009, ADR-0011, ADR-0012, ADR-0024, ADR-0025, ADR-0026,
  ADR-0027, ADR-0032, ADR-0033, ADR-0035

## Context

A note is shown by its `title`, else by its first level-one heading, else by its
filename (ADR-0012). A note is also named by the file it is in, and ADR-0027
renames one by moving that file: the bytes are identical on the other side,
because a move is not an edit.

The two hold together for a note whose filename is the whole of its naming, and
for no other. A note with a level-one heading is shown by that heading. Moving
its file changes the path and changes nothing a person sees — the list says what
it said before, the tab says what it said before, and the note is somewhere else
on disk.

Some of the notes this application creates land in that class by its own hand.
Where a title cannot survive as a filename, ADR-0027 reduces the name to what
will fit and opens the body with the exact title as a level-one heading. So a
note numen made from a title a filename could not carry is one numen cannot
rename.

`note_rename` (ADR-0026) has moved the file and nothing else since it was first
served, and it worked out the file it was moving to inside the adapter, so the
rule about what a note is called lived outside the core that owns every other
one.

## Decision

### Renaming brings into line whichever of the three names the note

One title is asked for, and one of these is written:

- the note carries a non-empty `title` — the key is given the new title;
- the note has a level-one heading — the text of the first one is rewritten;
- the note has neither, and the title cannot survive as a filename — the body
  opens with the exact title as a level-one heading;
- the note has neither, and the title can — nothing is written, and the filename
  says it.

The reduction from a title to a filename is ADR-0027's, unchanged: the same
characters are refused, the same length is the ceiling, and the same answer says
whether the title survived. **Creating a note and renaming one are the same
convention read in two directions**, so a note arrives named the way a rename
would name it.

**A title that leaves nothing to name a file after is refused.** The reduction
drops what a filename cannot hold, and a title made of nothing else comes out
empty. There is no name to file the note under, so the note is not opened and
nothing is written. Creating a note refuses such a title for the same reason,
which keeps the two directions of the convention the same one.

A title that only fails to survive whole is a different case: it is reduced, the
body carries it, and the rename goes through.

**The file keeps the extension it had.** What a vault files new notes under is a
setting about creating one and says nothing about a note that already exists.

### The note is brought into line before the file is moved

A move can be refused — something already sits where the note would go
(ADR-0027) — and the two halves land in the order that makes a refusal cheap. It
leaves the title right and the filename behind, which is a rename asked for again
once the name is free.

The other order leaves a note showing its old name under a filename that claims
a new one, and a refusal there is undone only by moving the file back.

### `title` is written into a note that already carries it, and never added

ADR-0012 claims the key and charges nothing for it: the application adds no
`title` to a note that has none. That still holds. What changes is one case — a
note the person named with that key is renamed by writing that key, because it is
what the note is shown by and nothing below it is read while it is there.

A vault of notes without the key stays a vault of notes without the key.

### A note its filename names keeps its bytes

Nothing in such a note has to be brought into line, so nothing is written to it
and the rename is a move. It takes no identifier, which is ADR-0027 and ADR-0009
unchanged: a move is not an edit.

A rename that writes the key or the heading **is** an edit, and stamps an
identifier the way every other edit does (ADR-0027).

### The file half is a move, whole

The move is ADR-0027's, called and not reimplemented. A link written by the old
name that stopped resolving is repaired by name; a link that now reaches a
different note is reported and never repaired; whoever is showing the note at the
name it had is told where it went (ADR-0032). A rename answers with all of it.

### A note whose frontmatter cannot be read is not renamed

The order above cannot be walked without reading the frontmatter, and a block
that does not parse is one the application refuses to read past (ADR-0027). It
cannot know whether the note carries a `title`, so it cannot know what renaming
the note means. It says so and changes nothing.

## Consequences

**Positive**

- A rename changes what the person sees, for every note and not only for the ones
  their filenames name.
- A note this application created with its title in a heading can be renamed by
  this application.
- One convention names a note, whether it is being made or being renamed, and
  there is one place to change it.
- A note named by its filename is still moved and not edited: the bytes are what
  they were and no identifier appears.
- A refusal leaves a note whose name is right and whose filename is behind, and
  nothing that has to be undone.

**Negative**

- **A note whose frontmatter does not parse cannot be renamed at all**, where
  moving its file was possible before. Its name is changed by renaming the file
  in a file manager, and by repairing the block so the application can read it.
- Renaming a note named by a heading or a `title` writes that note: an identifier
  appears on one that had none, and the write lands in the tab of whoever is
  reading it.
- **A setext heading is not a heading here.** ADR-0012 resolves a name through
  the first level-one heading, and what reads one recognises `#` and not a line
  underlined with `=`. A note titled that way is taken to be named by its
  filename, and moving its file is the whole of its rename. That shows the new
  name, because the same reading names the note in the list. Where the title
  cannot survive as a filename the body opens with a `#` heading above the
  underlined one, and the file carries two. This decision inherits that reading
  and does not improve it.
- A rename is two acts where a move is one. A machine that dies between them
  leaves the note titled and the file where it was — the same state a refusal
  leaves, and one nothing else produces.
- **The word `rename` now names an operation that edits a note.** ADR-0027 uses
  it for the filesystem call a move makes, and both readings are in the record.
  ADR-0024 settles the two apart, and the sentences written before it stand as
  they were written: a reader meeting one of them translates.

## Alternatives considered

**Move the file and leave the note alone**, which is ADR-0027 as it stands.
Rejected: it renames nothing a person sees for any note with a heading or a
title, and the notes this application creates are in that class.

**Write `title` into every note that is renamed.** Rejected: it puts an owned key
into notes that never had one, on an operation people run often, and the cost
ADR-0012 declined to charge is exactly that.

**Rewrite the heading and never the `title` key.** Rejected: `title` outranks the
heading, so a note carrying both would show its old name over a heading nobody
reads.

**Move the file first and bring the note into line afterwards.** Rejected: the
move is the half that can be refused, and a refusal there leaves the worse of the
two halves standing.

**Refuse a title that cannot survive whole as a filename.** Rejected: creating a
note with one is allowed and the body carries the title there, so refusing it
here would make a note that can be created and not renamed. A title that reduces
to nothing at all is refused, and is refused by creating too.
