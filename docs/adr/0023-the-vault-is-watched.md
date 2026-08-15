# ADR-0023: The vault is watched

- **Status:** Accepted
- **Date:** 2026-08-15
- **Applies to:** `modules/apps/desktop`
- **Related:** ADR-0001, ADR-0002, ADR-0018

## Context

ADR-0001 chose files as the source of truth and listed what that costs. The
first item is a watcher: the application must react to changes it did not make.
ADR-0002 budgets it at under 100 ms from the event to the updated interface.

Today the only way in is a scan of the whole vault.

## Decision

### Watched as a tree, through the system's own recursion

macOS and Windows watch a tree in one call — FSEvents and
`ReadDirectoryChangesW` are recursive. Linux and BSD cannot: inotify and kqueue
take one directory each, and the tree is maintained by hand as folders come and
go.

`rjeczalik/notify` takes the system's recursion where there is one and keeps the
tree itself where there is not.

**A watch is on a directory, never on a file.** Editors save atomically — a
temporary file beside the original, renamed over the top, so a machine that dies
mid-write leaves one whole file rather than half of one. The file a watch was
placed on stops existing at the first save.

### An event names a path; the outcome decides what it meant

No rule recognises an editor's temporary file by its name. A path is either a
note that exists, and is indexed, or it is not, and is taken out of the index.

A file that appears and is renamed away costs one parse of something already
gone.

### What is not a note

- **Anything whose name begins with a dot**, as is already true of folders. This
  is what keeps `.#note.md` — the lock an editor leaves beside an open file —
  out of the index, without anyone having to ask for it.
- **Anything the vault says to ignore.** `.numen/config.json` carries a list of
  patterns in the syntax of `.gitignore`, which is the one every person opening
  this already knows. An attachments folder, an export directory, a sync
  client's scratch space: what is noise is a property of the vault, so it is
  written in the vault.

The same rules answer for the walk and for the watcher. Two lists of what to
skip would drift, and the vault would be indexed differently depending on which
one found the file.

### Events are buffered, folded by path, and acted on after a window

One save is several events, and a path arrives many times in a moment. They are
read into a buffer, held briefly, and folded so that a path is indexed once
however often it was named.

**An overflowing buffer means a scan.** A restore, a checkout, an archive
unpacked — these arrive in thousands, and no buffer is the right size for them.
Rather than work out what was missed, the scan that already exists is run.

### The watcher reports paths

What it produces is the set of notes that changed, and nothing about them.
Whoever is listening knows what they are showing and asks for what they need.

### A vault that cannot be watched says so

The number of watches an operating system grants is limited, and a large vault
can exhaust it. That failure is reported: the application keeps working, scans
as it does today, and says that changes will not appear by themselves. A window
that has quietly stopped following the vault looks exactly like one that is up
to date.

## Consequences

**Positive**

- An edit made anywhere appears without anyone asking for it.
- The 100 ms budget becomes something to measure.
- The scan stops being the only way in and becomes the way back.

**Negative**

- The index now has two ways to change, and they can disagree. Both are tested
  against the same vault.
- Between an event and the reindex the index and the disk differ. Short, and
  nothing is promised inside it.
- Watching costs a system resource a large vault can run out of, and the answer
  is a message rather than a fix.
- Echo-loop protection (ADR-0001) is not built, because nothing writes to a
  vault yet. Whatever does will see its own writes come back.

## Alternatives considered

**Polling.** The scan on a timer. Rejected by arithmetic: a walk of a hundred
thousand notes is half a second, so a timer short enough to feel immediate
spends the machine on it and one long enough to be cheap misses the budget.

**fsnotify.** The obvious library, and the one that does not do this: recursive
watching is not in its public API, and on macOS it uses kqueue rather than
FSEvents — a file descriptor per file, which a vault of this size exhausts. Both
platforms that can do this natively would be emulated instead.

**A separate ignore file.** Familiar, and rejected: the vault already carries a
configuration file, and a second one is a second place to look.

**Sending the changed note with the event.** Rejected: it puts a note's whole
shape into a message whose job is to say that something happened, usually to
someone not showing it.
