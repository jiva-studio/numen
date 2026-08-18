# ADR-0032: The window saves a note as it is typed, and stops at an unseen edit

- **Status:** Accepted
- **Date:** 2026-08-18
- **Applies to:** `modules/apps/desktop`
- **Partly supersedes:** ADR-0001 — item 3, and the conflict copy of item 4
- **Partly supersedes:** ADR-0009 and ADR-0027 — the identifier, for the window's
  save alone
- **Related:** ADR-0009, ADR-0023, ADR-0024, ADR-0026, ADR-0027, ADR-0033, ADR-0034

## Context

ADR-0027 gave the application a write path and wrote its rules for one shape of
caller: something that reads a note, decides what to change, and asks for the
change. An agent is that caller, and so is every link tool.

A person typing in a note is not. There is no request, no decision and no moment
of asking — there is a buffer that differs from a file, and a save that has to
happen without being asked for.

Two further things separate this caller. It holds the prose and not the file: the
frontmatter is in front of the body and never on screen. And it is the only one
whose write can land while someone is looking at what it lands on.

The note underneath is not still. An agent, a second window and a file
synchroniser write into the same vault, and the save that follows a keystroke
carries a body read minutes ago.

## Decision

### A save is unasked, and it happens when the typing stops

Every change puts the write off again. One write happens once the text has been
still, and a second happens once the oldest unwritten change reaches a bound, so
that a crash costs at most that bound. Nothing changed means nothing written.

`Ctrl+S` writes what is owed at the moment it is pressed, under the rules below.

Closing a tab and quitting the window write what is owed and wait for it. What a
quit waits for, in what order, and for how long is ADR-0033.

### A note is embedded once the vault has been still

A save reaches the index at once: the note is parsed again, cut again (ADR-0034), and
found by word. Its vectors are asked for after the vault has been quiet for eight
seconds, and every write puts that pass off again.

The cooldown is longer than the bound above, so one sitting at one note is embedded
once. The two numbers are read together, and this pair holds in
`modules/apps/desktop`.

### A save that would overwrite an unseen edit stops

The save reads the file it is replacing and compares the prose there with the prose
the tab read. Equal prose is not a change, and the save lands. The frontmatter is
not compared; it is carried across (below).

Prose the tab has not read stops it. Nothing is written, the tab carries the mark
`overtaken` (ADR-0024), and the unasked save stops for that tab: what the person
typed stays in the buffer, and every keystroke after that leaves it there.

The person answers with one of two, and the unasked save runs again afterwards:

- **keep** writes the tab's prose over the file;
- **take** replaces the tab's prose with the file's.

`Ctrl+S` stops here too. A tab that is overtaken is answered before it closes and
before the window quits (ADR-0033).

ADR-0027 refuses a write that would overwrite an unseen edit. A caller that read a
note presents the fingerprint it was given; this caller holds prose, and the prose
it read is what it presents.

### A save keeps the frontmatter that is on disk when it lands

The buffer holds the body. The save reads the file, takes the frontmatter as it
then stands, puts the body on it and replaces the file. The read and the rename are
one act against every other write that reads a note and puts it back (ADR-0033).

### A save writes no identifier

ADR-0027 writes an identifier when the application changes what is in a note, and
ADR-0009 counts the person opening a file and changing it among those. A save from a
tab carries what the person wrote in their own note, and it stamps nothing.

A note acquires an identifier from the operations that act on it as a note: a
create, a move, a link. Reading one and typing in it is not among them, so a note
written elsewhere keeps the frontmatter it came with.

### A file that is not there is created

A note renamed or removed under an open tab leaves the tab open, showing what the
person was reading, at the name they opened. A name with no file behind it holds no
prose, so the next save puts the note back there.

### A note has a ceiling, and a file over it is not opened here

A megabyte is the most a note may be and still be read. The size is asked of the file
before it is opened, so a file over the bound is refused with none of its bytes read,
and the tab says which file and what the bound is. A body handed back over the same
number is refused by it too.

A megabyte of prose is a quarter of a million words. The bound is the core's, so what
is refused to the window is refused to an agent.

### Line endings are decided by the whole file, and the body alone is written

Every break in the file is looked at. A file whose breaks are all CRLF has its body
written with CRLF, and any other file has its body written with LF.

The frontmatter arrives on the other side as the bytes it went in as (ADR-0027). So a
file whose breaks are mixed above the body keeps that mixture, and a save that changes
no text leaves the file byte for byte as it was. A body of mixed breaks is written with
one break throughout.

The buffer is LF throughout, so what is compared for having changed is the
normalised text, and a note is never unsaved by being opened. The prose read back
from disk is normalised the same way before the comparison above.

Normalised text reaches the person and never the index. Offsets into a note are
byte offsets into the file as it is on disk.

## Consequences

**Positive**

- Saving is invisible while nothing else writes the note: nothing has to be pressed
  and nothing is asked.
- Prose the tab never read is never written over, and a collision is settled by the
  person whose note it is.
- A person can open a note of any shape up to the ceiling and type in it. A
  frontmatter this application cannot write is no longer a note it cannot save, and a
  file it did not create is returned unmarked.
- A tab shows what its file holds, or what the person typed, and never a third
  thing.

**Negative**

- A collision the person did not cause reaches them: a tab stops saving over
  somebody else's write, in the middle of their typing.
- A tab that stopped holds the only copy of what is in it, and a crash costs all of
  it. The bound above stops applying the moment the tab stops.
- **keep** writes over what the file held, and nothing keeps what was there.
- A vault under a file synchroniser stops tabs often, and the person answers for
  writes they did not make.
- Two processes on one vault each read, compare and write in turn, so a save can
  pass the comparison and land on a write made since it (ADR-0033).
- The person cannot see or edit their own frontmatter here, in a product whose
  first decision is that the file is theirs. It is reached with any text editor.
- A note whose body holds breaks of both kinds has that body written with one of them,
  once. What is above the body keeps the breaks it had, so the file stays mixed.
- A note over a megabyte cannot be opened in a tab at all. The window says which file
  and what the bound is, and the person opens it elsewhere.
- A note just saved is not found by meaning for at least eight seconds after the
  typing stops. It is found by word at once.

## Alternatives considered

**A conflict copy**, as ADR-0001 item 4 requires. Rejected: a copy in the vault is a
note, and a vault that answers a collision by growing a second note has moved the
problem into the place the person searches.

**A modal over the window.** Rejected: it stops every tab for one tab's collision,
and the prose the question is about is behind it.
