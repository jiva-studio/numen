# ADR-0032: The window saves a note as it is typed

- **Status:** Accepted
- **Date:** 2026-08-17
- **Applies to:** the vault format — every application that writes one
- **Supersedes:** ADR-0001 item 4
- **Partly supersedes:** ADR-0027, "A write that would overwrite an unseen edit refuses"
- **Related:** ADR-0009, ADR-0023, ADR-0026, ADR-0027, ADR-0033, ADR-0034

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

Closing a tab and quitting the window write what is owed and wait for it. What a
quit waits for, in what order, and for how long is ADR-0033.

### A note is embedded once the vault has been still

A save reaches the index at once: the note is parsed again, cut again (ADR-0034), and
found by word. Its vectors are asked for after the vault has been quiet for eight
seconds, and every write puts that pass off again.

The cooldown is longer than the bound above, so one sitting at one note is embedded
once. The two numbers are read together, and this pair holds in
`modules/apps/desktop`.

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
about it. This holds while the read and the rename are serialised against every other
write that reads a note and puts it back. That serialisation is one process's, and it
covers neither a create, a move's own rename nor a removal (ADR-0033).

### A save writes no identifier

ADR-0027 writes an identifier when the application changes what is in a note. This
write is the person changing what is in their own note, and it stamps nothing.

A note acquires an identifier from the operations that act on it as a note: a
create, a move, a link. Reading one and typing in it is not among them, so a note
written elsewhere keeps the frontmatter it came with (ADR-0009).

### A file that is not there is created

A note renamed or removed under an open tab leaves the tab open, showing what the
person was reading, at the name they opened. The next save puts it back there.

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
normalised text, and a note is never unsaved by being opened.

Normalised text reaches the person and never the index. Offsets into a note are
byte offsets into the file as it is on disk.

## Consequences

**Positive**

- Saving is invisible. There is no key to press and nothing to answer.
- A person can open a note of any shape up to the ceiling and type in it. A
  frontmatter this application cannot write is no longer a note it cannot save, and a
  file it did not create is returned unmarked.
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
- A note whose body holds breaks of both kinds has that body written with one of them,
  once. What is above the body keeps the breaks it had, so the file stays mixed.
- A note over a megabyte cannot be opened in a tab at all. The window says which file
  and what the bound is, and the person opens it elsewhere.
- A note just saved is not found by meaning for at least eight seconds after the
  typing stops. It is found by word at once.

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
