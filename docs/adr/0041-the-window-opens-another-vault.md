# ADR-0041: The window opens another vault

- **Status:** Accepted
- **Date:** 2026-08-24
- **Applies to:** `modules/apps/desktop`
- **Related:** ADR-0000, ADR-0002, ADR-0013, ADR-0031, ADR-0033, ADR-0035

## Context

ADR-0013 gave a vault an identity that travels with its folder, and put the list
of vaults in the application's own configuration. It said that list records which
vault was open last. It did not.

Nothing in the interface reached that list either. The window took the first
entry and built everything around it for the life of the process, and a person
who wanted a second vault had a registry that could hold one and no way to make
one, open one, or take one away.

ADR-0033 named the boundary that made this hard. One process, one writer, one
lifetime: the window is one operating-system process, and everything about the
vault was arranged inside it once.

## Decision

### A process holds one installation and, at a time, one vault

The two are separated. **The installation is made once**: the index, the
embedder, the list of what is being done, and everyone listening to it. **One
vault is made and unmade**: the scan, the watch, the reading of documents, the
documents held open, and what each of those has in flight.

Opening another vault unmakes the second half and makes it again. The database is
one for every vault already (ADR-0002), and the embedder is the installation's,
so neither is touched.

### Opening another vault settles first, the way closing does

Every page writes what only it holds, and **a page holding text a person has to
answer for calls the swap off**. The window stays on the vault it had, and the
question stands. This is the settling of ADR-0033, asked for a second reason.

**One settling runs at a time.** A swap and a close each ask for one, and the
second to arrive is refused in words.

### Nothing is taken down before the new vault is known to be there

The folder is opened and its identity checked against the registry first. A vault
that fails that is refused with the window exactly as it was.

**A vault that fails to come up after the old one is gone brings the old one
back.** Where that also fails the window has no vault and says so.

### The registry is the list; the index holds a row for it to point at

The list of vaults stays in the application's configuration as JSON, and now
records which vault was opened last. It is not derivable from anything and is not
a cache (ADR-0000).

The `vaults` table in the index is what every other row's foreign key points at.
**Nothing cascades into a virtual table**, so forgetting a vault clears the
vector index and the four full-text indexes by hand, before the row the cascade
hangs off is deleted.

**The vectors stay.** A vector is addressed by the text it was made from
(ADR-0034), so chunks of several vaults hold one vector, and a vault that goes
takes none of them with it.

### A vault has a unique name, and vault roots do not overlap

A name already taken is numbered. A root that lies inside a registered vault, or
that holds one, is refused: one file under two identities would be written by two
scans.

### Taking a vault away is two things

**Forgetting** takes it off the list and out of the index. The folder stays where
it is with the identity it carries, and adding it again brings back the same
vault.

**Erasing** forgets it and puts the folder in the trash this machine keeps. It is
a folder of the person's own writing, so it goes where deleted things go and
comes back from there. A machine with nowhere to put it refuses and says so.

**The vault the window is showing cannot be forgotten or erased**, and neither
can the only vault an installation has. The window always stands on something.

### An agent may add a vault and open one

The tools gain `vault_list`, `vault_add`, `vault_rename`, `vault_forget` and
`vault_open`. `vault_add` with no path puts the machine's own picker in front of
the person; with a path it makes a vault there, refusing a filesystem root and
the person's home directory.

**There is no tool that erases a vault from disk.** Taking a person's folder away
is asked for in front of them.

## Consequences

**Positive**

- A person keeps several vaults and moves between them in the window they have
  open, and the window remembers where they were.
- What belongs to the installation and what belongs to one vault is now written
  down in the shape of the code, so a new piece of per-vault work has one place
  to be stopped from.
- Forgetting a vault leaves an index that answers only for vaults that exist.

**Negative**

- **ADR-0031's consequence "the panel's agent cannot write outside the vault" is
  no longer true.** `vault_add` with a path followed by `vault_open` lets an
  agent name the folder itself, and every tool then works inside it. The two
  refusals above make it deliberate; they do not make it narrow.

- **The agents stop before the swap is known to be possible.** A swap called off
  by a page with an unanswered question, or by a folder that could not be read,
  has already ended the agent's session.

- **`vault_open` answers before the swap happens.** The endpoint the answer
  travels over is what the swap closes, so the call cannot wait for it. Whether
  the answer arrives before the transport goes is a race, and the tool says so.

- **A close asked for during a swap is refused and not asked again.** The person
  presses close a second time.

- **The registry is not locked between processes.** Each write rewrites the whole
  file, so a window and a command line writing at once lose one of the two
  entirely. Inside one process it is locked. This is the same gap ADR-0033
  accepted for the files of a vault, now reaching the list of them.

- **On Windows a folder can be deleted outright.** Where the volume has no
  recycle bin the shell deletes permanently and reports success, and no flag
  turns that into a refusal. The promise "it goes to the trash and comes back" is
  honest on Linux and is not on Windows.

- **The macOS trash is unverified.** It compiles and has never run: the
  selectors, the boolean return and the error out-parameter are all untested
  until somebody runs it on a Mac.

- **A failed reading left in the list of what is being done survives a swap.** A
  cancelled one takes itself out; one that failed stays until it is dismissed,
  and it now outlives the vault it was about.

- **The index file does not shrink when a vault is forgotten.** The space is
  reused and the file is the size it was.

- **The folder picker is the first thing here built on the desktop's own
  settings, and the library that puts it up calls `abort` where they are not
  installed.** No code of ours runs after that. The settings ship with the
  toolkit this binary is linked against, so where the machine has not put them
  on the search path the toolkit is asked where it keeps them, and a picker
  asked for on a machine where neither has them is answered in words. What is
  left uncovered is a machine holding settings without the one a file chooser
  reads: that still ends the process, and what closes it is the package naming
  what it needs.

## Alternatives considered

**Restart the process on the chosen vault**, which is what "one lifetime" read as
literally. Rejected: everything expensive in the process — the index, the model,
the runtime a page is read through — belongs to the installation and would be
built again for no reason, and the person watches their window disappear and come
back.

**A second window, in a second process, per vault.** Rejected for now: two
processes on one index is a second writer, and the registry gap above becomes the
ordinary case rather than the unlucky one. What it buys — two vaults in front of
a person at once — nobody has asked for.

**Move the list of vaults into the index.** Rejected: the index may be deleted at
any moment and rebuilt with no loss (ADR-0000, ADR-0002), and the folders a person
added are not rebuildable from anything. The list is also needed before the
database opens, and most of all when it will not open.

**Let the cascade clear a forgotten vault.** Rejected because it does not work: a
virtual table has no foreign keys, and the five would have kept every row of the
vault that went.
