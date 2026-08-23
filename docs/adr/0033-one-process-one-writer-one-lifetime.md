# ADR-0033: One process, one writer, one lifetime

- **Status:** Accepted
- **Date:** 2026-08-17
- **Applies to:** `modules/apps/desktop`
- **Related:** ADR-0018, ADR-0023, ADR-0027, ADR-0031, ADR-0032

## Context

ADR-0027 gave the core a write path that reads a note, changes one thing and puts
it back. An agent's edit is that shape, the link repair a move leaves in another
note is that shape, and so is the window's save, which reads the file for the
frontmatter it does not hold (ADR-0032). Each is correct only if its read and its
rename are one act against the others, and nothing said where that is arranged or
how far it reaches.

Two more rules were arranged in the same place and written down nowhere. ADR-0032
says a quit writes what is owed and waits for it, with no bound on the wait.
ADR-0031 names everything an agent may reach and says nothing about how long it
may run.

All three answer to one fact. The window is one operating-system process, and that
process is the boundary each of them holds inside.

## Decision

### One vault has one writer inside one process

A lock is kept per vault root, keyed by the folder resolved through its symlinks,
so every writer and every reader opened on one folder takes the same lock. A path
that cannot be resolved is its own key.

**It is taken before the read and held past the rename** by every operation that
reads a note and puts it back: the edits of ADR-0027, the window's save, and the
link repair a move leaves in another note.

**Creating a note, the rename a move itself is, and a removal take nothing.** Each
is one filesystem call over one name.

**The lock lives in this process.** Two windows on one vault hold a lock each.

### The window goes in one order, and every wait on the machine has a bound

**A tab whose save stopped is answered first.** The quit lists every overtaken tab
(ADR-0032) and waits for the person to answer each one, with no bound. The quit is
called off from that list, and the window stays as it was.

A tab put off leaves the list and stays overtaken, so the next close stops on it
again. What ends a question is one of the two answers and nothing else.

**A page that goes with a question standing is still owed.** Its work is held by a
window this process cannot reach into, and a page that comes back takes it over and
raises the question again. A page that does not come back is silence, and silence
costs the bound below.

The page is asked first, then the agents, then the scan and the follower, then the
database. The order is what each part needs from the next: the page holds text
nothing else has, an agent writes through the core, the scan and the follower write
to the index, and the database is what they write into.

**A page has three seconds to hand over what it holds.** A page whose script has
stopped answers never, and this is what that costs.

**The agents' transport has two seconds to be cut off.** A session an agent left
open holds its connection until it is closed under it. The calls already running are
waited for afterwards with no bound.

**The writes already taken are waited for with no bound.** The door is shut first,
so what is left is a fixed set of filesystem operations.

A quit that does not arrive through the window is answered on the thread the page is
served on, so the settling happens off that thread and the quit is asked for again
once it is over. It happens once, whichever way the window is asked to go.

### The runtime a page is read through is made before the window

Every model this process runs is run through one ONNX Runtime environment, made
once and kept for the life of the process. **It is made before the window is**,
and a reading is refused where it was not: a runtime made after a window reads
every page it is given as nothing, and a second one made later is the same
runtime.

Making it costs the shared library and nothing else — no model is read, so a
machine holding every model and a machine holding none open alike. A machine
holding no runtime at all is left as it is, and reading is what fetches one. The
reading that fetched it says so, and the document is the next opening's to read.

### An agent does not outlive the window

An agent runs in a process group of its own and is ended with it. A task in flight
stops where it is, and what it had already written to the vault stays written.

## Consequences

**Positive**

- Every operation that reads a note and puts it back queues on one lock in one
  file, and what that covers is one function.
- Nothing an agent started keeps running once the window is gone, and nothing it
  writes lands after the index stopped following the vault.
- A quit with no tab to ask about is bounded: a wedged page costs three seconds and
  then the window goes. A quit that does have one waits on the person, and they can
  call it off.

**Negative**

- **There is no lock between two processes on one vault.** Two windows, or a window
  and the command line, interleave their reads and renames, so a save can pass
  ADR-0032's comparison and land on a write made between the two. A file
  synchroniser is a third process and takes no part in this at all.
- **A create, a move and a removal are not serialised against a save.** A note
  renamed while a save is in the air leaves the tab open at the name it was read
  from, and ADR-0032 puts the file back there.
- **A page that does not answer inside the bound loses what only it held.** The
  bound is what the window waits, and there is nothing to wait for after it.
- **A machine that fetched its runtime during a reading reads that document on
  the next opening**, and is told so where the reading was asked for.
- **An answer in flight is lost when the window goes.** A note an agent was part of
  the way through writing is whatever its last complete write left.
- **Three bounds are three constants**, and nothing measures what any of them is a
  bound on.

## Alternatives considered

**A lock file in the vault**, so that two processes serialise. Rejected for now: a
lock a crashed process leaves behind is a vault nothing can write to, and the
recovery is a person deleting a file they were never told about. The gap it would
close is written above as a cost.

**One lock for the whole installation.** Rejected: one database holds every vault
(ADR-0002) and the writes here are to files, so two vaults have nothing to queue
behind each other for.

**No bound on the wait for the page**, however long it takes. Rejected: a webview
already torn down answers never, and the window would then never close. The wait
with no bound is the one a person is answering.

**Let an agent finish, and keep the process alive until it does.** Rejected: the
window is gone, nothing is drawing what the agent says, and the tools it writes
through are served by the process that is leaving.
