# ADR-0027: The application writes to the vault

- **Status:** Accepted, except where noted below
- **Date:** 2026-08-16
- **Applies to:** the vault format — every application that writes one
- **Partly superseded by:** ADR-0032 — the identifier, for the window's save alone
- **Partly superseded by:** ADR-0040 — `title` is written when a note already
  carrying it is renamed
- **Related:** ADR-0001, ADR-0009, ADR-0011, ADR-0012, ADR-0022, ADR-0023

## Context

Nothing in this repository has ever written a note. The application gives a
folder an identity, keeps a list of vaults, and otherwise reads: a scan walks,
a refresh reparses, the index answers. Every rule about writing lives in
ADR-0012 and in the format specification as a promise about a writer that does
not exist yet.

Two things need one now — the window, where a person makes a note and draws a
link, and an agent (ADR-0026). They want the same operations, so the rules
belong to the core rather than to either of them.

The difficulty is not the writing. It is that the application is a guest.
ADR-0001 made the file the source of truth and the person its author; the
application arrives afterwards, changes one thing, and must leave everything
else exactly as it found it — including what it does not understand, and
including whitespace nobody would notice was gone.

## Decision

### A write happens completely or not at all

A note is written to a temporary file beside it and renamed over the top. A
machine that dies mid-write leaves the previous note whole rather than half of
the next one, which is the same reason every editor already does this, and the
reason `.#note.md` is excluded from the index by name (ADR-0023).

The vault registry is written this way already; the writer is the same act with
a different target.

### Owned keys are written; everything else is carried across untouched

The frontmatter is shared, not owned (ADR-0012). A writer that parses YAML into
a map and serialises it back cannot honour that: a map has no order, so the keys
come back rearranged, comments are gone, and the diff the person did not ask for
is the whole file.

So the frontmatter is round-tripped through its parse tree. The application
reads the keys it owns, writes the ones it changed, and everything else — order,
comments, quoting style, the key it will own in a future version — arrives on
the other side as the bytes it went in as.

The same holds outside the frontmatter: line endings are kept per file, the body
is not reflowed, and a file that arrives with a final newline keeps one.

**A note whose frontmatter did not parse is never written.** The format
specification says a broken block is reported rather than repaired, and a writer
is where that promise is kept or broken: adding a link to a note whose YAML is
malformed means rewriting a block the application could not read, which is
guessing at what the person wrote.

### The application writes what it was asked to write, and nothing else

Three rules, all the same rule seen from different sides.

**An identifier is written when the application creates a note or edits its
contents, and never backfilled** (ADR-0009). A note written in vim has no
identifier and is a note in full; it acquires one when the application itself
changes what is in it.

> **Partly superseded by
> [ADR-0032](0032-the-window-saves-a-note-as-it-is-typed.md).** A save carrying what
> a person typed writes no identifier: what it puts in the note is theirs. Every
> other write from this application stamps as above.

**Moving and removing do not change a note.** They are renames: the bytes are
identical on the other side, so a note that had no identifier still has none
afterwards. A move is not an edit, and an operation that quietly promoted every
file it touched would be one.

**Repairing a link in another note is not an edit of that note either.** When a
repair is due (below), what changes is the address inside one link. The note it
lives in receives no identifier: its author did not open it, did not change it,
and is not the reason the repair is happening.

### A note is created with its title in its name

`note_create` is given a title. The file is named after it, and that is the
whole mechanism: a note is shown by its `title`, else by its first level-one
heading, else by its filename (ADR-0012), so a file named after the title is
already named correctly.

Where a title cannot survive as a filename — a slash in it, a leading dot, more
characters than a filesystem will take — the name is reduced to what will fit
and the body opens with the exact title as a level-one heading. The order of
resolution does the rest.

**`title` is still never written.** It is read and not written (ADR-0012), and
nothing here needs it to be otherwise.

> **Partly superseded by [ADR-0040](0040-a-note-is-renamed-by-whatever-names-it.md).**
> Renaming a note needs it. A rename brings into line whichever of the `title`, the
> first level-one heading and the filename names the note, and does that before the
> file is moved. The key is written into a note that already carries it and is never
> added to one that does not. A note its filename names keeps the bytes it had.

### Removing a note moves it to `.trash/`

A removed note is renamed into `.trash/` inside the vault, keeping the path it
had underneath so that two notes of the same name do not land on each other and
so that putting one back is obvious. Anything under a dot-folder is not a note
(ADR-0023), so the note leaves the index, the search and the plex by the
machinery that already exists, without being destroyed.

Destroying it outright is available and is asked for explicitly.

The reason is narrow and worth naming. Everything the index knows is rebuilt
from the file, so losing the index costs a scan; a review log is not, and cannot
be recomputed from anything (ADR-0009). Spaced repetition is not built yet, and
when it is, removal becomes the only irreversible act in the product. The
behaviour is settled now, while nothing depends on it.

### A backlink is repaired only when it stopped resolving

A note that moves does not usually break anything. A name resolves by an exact
path, then by a path relative to the note the link is written in, then by a
single file of that name anywhere in the vault (ADR-0011) — so a link keeps
finding a note that moved, whichever of the two forms it was written in.

The application therefore does not rewrite links after a move as a matter of
course. It resolves the backlinks it knew about before the move, resolves them
again after, and repairs only those that now resolve to nothing, by writing the
name. Nothing else in anyone's file changes.

Two consequences are accepted rather than fixed. A link that now resolves to a
*different* note is not repaired, because it is not broken — it is ambiguous,
which is a property of two notes sharing a name and not of the move. And an
identifier is not written in its place: `note://` is the auxiliary form, needed
only where a name cannot pick a target (ADR-0011), and a repair that reached for
it would be answering a question nobody asked.

### A write that would overwrite an unseen edit refuses

A caller may present the fingerprint it was given when it read the note. If the
note on disk no longer matches it, the write is refused and says so.

This is the fingerprint the index already keeps — path, size and modification
time (ADR-0024) — used for the other thing it is good for. Without it, a slow
reader that thinks between reading and writing silently discards whatever
happened in between, and the person watching their own note revert has no way to
know why.

### A batch of files is not a transaction

Notes are indexed in groups and a group either arrives or does not (ADR-0022).
The filesystem offers nothing of the kind: fifty renames are fifty renames, and
the twenty-ninth can fail on its own.

So an operation over many notes reports what happened to each, and never
presents itself as all-or-nothing. Pretending otherwise would make a partial
failure look like a total one, and the recovery would start by undoing work that
succeeded.

### What is shown rather than fixed

A vault accumulates things the application will not act on and will not guess
at, and none of them stops a scan. They are gathered by a set of named checks —
one to a file, so that the next thing worth noticing is a new file rather than a
change to what the old ones say.

**A problem belongs to one note: the file somebody would open to settle it.** For
a link that reaches two notes that is the note the link is written in, and
neither of the notes it could mean — those two have nothing to answer for, and
the person disambiguates by editing the link.

Two of the checks read what a scan already stored while parsing one file. Two
work the whole vault out at the moment they are asked, and **must not be
stored**: a link is ambiguous, or reaches nothing, only as of now (ADR-0011), so
a flag written down would be wrong the moment a note somewhere else moved, and
nothing would say so.

**A check whose findings are ordinary is left out unless it is named.** A link
to a note not yet written is how people work — the link is a note to themselves
that the note is owed — and a vault under construction holds hundreds. Reported
alongside the rest they would bury it.

Creating a note whose name is already taken is allowed and said out loud in the
answer. Refusing it would be an invariant the application cannot hold — the
person can make the second file in vim a second later — and an invariant that
only binds the honest half of the writers is a false one.

## Consequences

**Positive**

- The window and an agent get one set of rules, and a rule is enforced by the
  writer rather than by everyone remembering it.
- A person's file survives contact with the application: their keys, their
  order, their comments, their line endings.
- The irreversible operation is reversible by default.
- Nothing acquires an identifier by being moved past.

**Negative**

- Round-tripping a parse tree is more work than marshalling a struct, and the
  cost is paid on every write forever.
- `.trash/` is the application putting a folder in the person's vault. It is a
  convention other tools use for the same reason, and it is visible, which is
  the best that can be said for it.
- Repairing links after a move means writing to files the person did not touch.
  Doing it only for links that broke keeps that to the smallest set that still
  leaves the vault working; it does not make it none.
- A duplicate name stays possible, so the ambiguity it causes stays possible.
  What changes is that both are now said out loud.

## Alternatives considered

**Rewrite every inbound link after a move**, as editors that own their library
do. Rejected: most of those links were not broken, the ones that were are found
exactly, and an application that rewrites a hundred files to move one has made
itself the only safe way to move a file — in a product whose first decision was
that the files are yours (ADR-0001).

**Never repair anything.** Tempting, and nearly right: dangling is a legitimate
state, reported rather than fatal. Rejected because the person did not break the
link — the application did, on their behalf, in the same breath. Leaving the
damage for the problems view is correct for damage the application did not
cause.

**Write the identifier form when repairing.** Rejected: it is durable against
every future move, unreadable to the person whose file it is, and would spread
`note://` through a vault as the ordinary form, which ADR-0011 decided against
on purpose.

**Delete a note outright.** Rejected as the default, kept as an option. See the
review log above; the day that argument bites is the day it is too late to
change the behaviour.

**Marshal the frontmatter from a struct and write it back.** Rejected: it is the
straightforward implementation and it silently destroys everything the
application does not have a field for.
