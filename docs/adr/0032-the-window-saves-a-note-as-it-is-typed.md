# ADR-0032: The window saves a note as it is typed

- **Status:** Accepted
- **Date:** 2026-08-17
- **Applies to:** the vault format — every application that writes one
- **Supersedes:** ADR-0001 item 4
- **Partly supersedes:** ADR-0027, "A write that would overwrite an unseen edit refuses"
- **Related:** ADR-0009, ADR-0023, ADR-0026, ADR-0027

## Context

ADR-0027 gave the application a write path and wrote its rules for one shape of
caller: something that reads a note, decides what to change, and asks for the
change. An agent is that caller, and so is every link tool.

A person typing in a note is not. There is no request, no decision and no moment
of asking — there is a buffer that differs from a file, and a save that has to
happen without being asked for. The rules that hold for a caller who arrives late
answer questions this caller never poses, and one of them refuses a write the
person made themselves.

Two further things separate this caller. It holds the prose and not the file: the
frontmatter is in front of the body and never on screen. And it is the only one
whose write can land while someone is looking at what it lands on.

## Decision

### A save is unasked, and it happens when the typing stops

Every change puts the write off again. One write happens once the text has been
still, and a second happens once the oldest unwritten change reaches a bound, so
that a crash costs at most that bound. Nothing changed means nothing written.

Closing a tab and quitting the window write what is owed and wait for it.

### The window's save overwrites, and is told nothing

There is no comparison. The save carries a path and a body, and it lands.

What this costs, in full:

- a body edit made under an open tab with unsaved changes — by an agent, another
  window, or a file synchroniser — is replaced by that tab's next save;
- the same note open in two places collapses to whichever saved last.

Both are silent. ADR-0001 item 4 refuses a silent winner and prescribes a conflict
copy; this chooses one and writes no copy, so item 4 no longer holds. A copy beside
the note is an ordinary note: indexed, drawn in the plex, returned by searches.

ADR-0027's refusal stands for every other caller. It is what a slow reader owes the
person, and a person typing is not a slow reader.

### A save keeps the frontmatter that is on disk when it lands

The buffer holds the body. The save reads the file, takes the frontmatter as it
then stands, puts the body on it and replaces the file.

So a link an agent adds to a note that is open survives a save that knows nothing
about it. This holds only while writes to one vault are serialised from before that
read to after the rename.

### A save writes no identifier

ADR-0027 writes an identifier when the application changes what is in a note. This
write is the person changing what is in their own note, and it stamps nothing.

A note acquires an identifier from the operations that act on it as a note: a
create, a move, a link. Reading one and typing in it is not among them, so a note
written elsewhere keeps the frontmatter it came with (ADR-0009).

### A file that is not there is created

A note renamed or removed under an open tab leaves the tab open, showing what the
person was reading, at the name they opened. The next save puts it back there.

### Line endings are decided by the whole file

A file whose endings are uniformly CRLF is written back with CRLF. Anything else is
written with LF, and mixed endings are therefore made uniform once.

The buffer is LF throughout, so what is compared for having changed is the
normalised text, and a note is never unsaved by being opened.

Normalised text reaches the person and never the index. Offsets into a note are
byte offsets into the file as it is on disk.

## Consequences

**Positive**

- Saving is invisible. There is no key to press and nothing to answer.
- A person can open any note and type in it. A frontmatter this application cannot
  write is no longer a note it cannot save, and a file it did not create is
  returned unmarked.
- A tab shows what its file holds, or what the person typed, and never a third
  thing.

**Negative**

- Concurrent edits are lost silently, which is the decision and not a defect of it.
  The product has one editor and one agent on one machine, and the loss is bounded
  by how long a person leaves a tab unsaved.
- A vault under a file synchroniser is the case this handles worst, and the
  synchroniser's own conflict copies are what a person is left with.
- The person cannot see or edit their own frontmatter here, in a product whose
  first decision is that the file is theirs. It is reached with any text editor.
- A note whose endings are mixed is rewritten end to end, once.

## Alternatives considered

**Ask.** The file changed under an edit, so the person chooses: keep mine, or take
the file's. Rejected: it interrupts typing to ask about a collision the person did
not cause and usually cannot judge, and every path out of the question is one of the
two writes it was asking about.

**Compare, and refuse.** ADR-0027's rule, applied here too. Rejected: the write it
refuses is the person's own, and there is nothing for them to do about the refusal
except make it again.

**A conflict copy**, as ADR-0001 item 4 requires. Rejected: a copy in the vault is a
note, and a vault that answers a collision by growing a second note has moved the
problem into the place the person searches.
