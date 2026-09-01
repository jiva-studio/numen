# ADR-0008: A vault is scanned in the background

- **Status:** Accepted
- **Date:** 2026-08-25
- **Applies to:** `modules/apps/desktop`
- **Related:** ADR-0004, ADR-0006, ADR-0009, ADR-0020

## Context

The vault is the truth and the index is level with it only for as long as nothing has been edited. A walk of the whole vault is what brings it level, and it runs while the person is using the application: a window is drawing, a search is being answered, and the same database is being written to throughout.

## Decision

### A scan returns immediately, says what it is doing, and stops when asked

Starting a scan does not block. It reports what it has done as it goes, and it ends when its context does. The core declares what it needs for that — a progress port and a context — and knows nothing of threads, windows or terminals. A cancelled scan leaves an index that is correct as far as it got.

The command line waits for its own scan, because a command invoked to scan has nothing else to do. That is a property of that adapter.

### Newest first

The walk is collected before anything is read, so the order can be chosen. Sources are read newest modification time first: a vault has a working set and an archive, and what the person is looking for while the scan runs is what they touched last.

### Notes are written in groups

A group is one transaction. Two constants in `usecase/vault/groups.go` close one, whichever is reached first: `notesPerWrite`, five hundred notes, and `bytesPerWrite`, eight megabytes of file content, which bounds what a vault of long files holds in memory before any of it is written.

A note and the fingerprint that dates it — its size and its modification time — go into the same transaction. An interrupted scan therefore leaves whole groups, and never a note the index believes is current.

Progress is reported per group. A caller that needs to know about one note asks the index a question.

The same grouping serves a watcher's refresh (ADR-0009). One event can name a whole folder — a checkout, a restore, a sync client unpacking an archive — so a refresh is handed as many paths as a walk is.

### The index's single write connection

**Amended by [ADR-0035](0035-the-review-window-writes-the-index.md).** Two processes hold a writer, and a write transaction takes its lock at BEGIN so the busy timeout is the thing that covers a collision.

The write pool is capped at one connection, and writes queue in Go. The read pool is unrestricted: under write-ahead logging a reader never waits, which is what lets a search answer while a scan is still running. A busy timeout covers a collision. Every connection opens with those pragmas in its connection string, foreign keys included.

The per-vault file lock that guards a read-change-write of a note is a different lock, and it is ADR-0020's.

## Consequences

- A cancelled scan loses the group it was filling, and those files are read again by the next one.
- A scan holds a group's worth of parsed notes before any of it is stored.
- A search is answered while a vault is being read.
- A file whose size and modification time did not change with its content is skipped — an archive restored, a tree brought over by `rsync -tc` — and a rebuild is the way out.
- A file that vanishes between the walk and the read is counted, and the scan goes on: the vault is edited while it is being read.
- Progress arrives at the granularity of a transaction.

## Alternatives considered

**Block the caller until the scan finishes.** Rejected: the window has a vault to draw before it has one indexed. Where waiting is right it belongs to the adapter, as it does on the command line.

**A transaction per note.** Rejected: a vault's worth of commits is paid once per file, and a note would still have to land with its fingerprint, so the group is where that pairing is enforced anyway.

**Let the write pool grow.** Rejected: SQLite takes one write at a time whatever the pool holds, and the queue moves out of Go and into a busy timeout.
