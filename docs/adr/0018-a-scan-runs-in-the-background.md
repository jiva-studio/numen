# ADR-0018: A scan runs in the background, and there is one writer

- **Status:** Accepted, except where noted below
- **Date:** 2026-08-15
- **Applies to:** `modules/apps/desktop`
- **Partly superseded by:** ADR-0022 — everything here about the size of a
  transaction, in the decision, the consequences and the alternatives
- **Related:** ADR-0001, ADR-0002, ADR-0014

## Context

A scan of a large vault takes about a minute and a half — measured, not
estimated (docs/performance.md). Today it is synchronous: the caller waits for
it to finish. That is tolerable for a command that exists to run a scan and
unacceptable for the application that will use the same code, where the failure
mode is well known and has a name: an editor that spends the first minute after
launch unusable while it reads its own library.

At the same time the index is one SQLite database. SQLite in WAL mode allows any
number of readers alongside a writer, but only **one writer at a time**. While a
scan is running it is that writer, so anything else that wants to write —
adding a vault today, recording a review later — queues behind it.

Both questions are the same question: what happens to the person using the
application while it is busy with itself.

## Decision

### A scan never blocks the person

Starting a scan returns immediately. It reports what it is doing as it goes, and
it stops when it is asked to. The core declares what it needs for that — a way
to report progress and a context to be cancelled by — and knows nothing about
threads, windows or terminals.

The command line waits for its own scan, because a command invoked to scan has
nothing else to do. That is a property of that adapter, not of the use case.

### Partial results are usable, so notes are committed one at a time

> **Superseded by [ADR-0022](0022-notes-are-indexed-in-groups.md).** The
> condition this section named — that the cost of not batching is affordable —
> stopped holding once notes carried links. Notes are now written in groups.

Each note is written in its own transaction. A search run while a scan is in
progress therefore returns the notes indexed so far, rather than nothing until
the end.

This is already true, and it is recorded here so that it survives the next
optimisation: batching notes into larger transactions would make a rebuild
faster and would take this away, hiding results until a batch commits and
holding the write lock for the length of it. The measured cost of not batching
is the number in docs/performance.md, and it is affordable.

### Scanning order follows modification time, newest first

A vault has a working set and an archive, and they are not the same size. Notes
touched this week are what the person is looking for while the scan runs, so
they are indexed first; the archive follows.

### One writer, short transactions

Writes go through a single connection. Readers are unrestricted, which is what
WAL is for.

The rule that makes this liveable is that no write transaction is long: one note
is a transaction, so a user action waits for one note rather than for a batch,
a file, or a vault. `busy_timeout` covers the collision rather than failing it.

## Consequences

**Positive**

- The application is usable from the moment it starts, on a vault of any size.
- A scan can be cancelled, and cancelling it leaves an index that is correct as
  far as it got — the next scan continues from what is on disk.
- The write path is simple enough to reason about: one writer, short
  transactions, no lock held across a file read.

**Negative**

- Progress reporting is a port the core declares and every adapter must satisfy
  or ignore, including the ones that do not care.
- Per-note transactions cost measurably more than batching; that cost is now
  a decision rather than an accident.
- A scan that runs while the vault is edited sees a moving target — already
  accepted by ADR-0001, and the reason a vanished file is counted rather than
  fatal.

## Alternatives considered

**Batch notes into one transaction per N.** Faster, and rejected: it holds the
single write lock for the length of the batch and hides everything in it from
search until it commits. The thing being optimised is the rebuild, which is the
rarest event; the thing being harmed is every other moment.

**A separate writer process.** Rejected by ADR-0014: no daemon until something
outside the binary needs the core.

**Show a progress bar and wait.** Rejected: it is the failure this decision
exists to avoid, dressed up.

## How this is measured

`BenchmarkSearchDuringScan` searches while a scan rewrites the same index
continuously. Measured: 64 ms against 48 ms for the same query on a quiet
database at ten thousand notes — a third more, not a multiple.

A concurrency decision with no measurement is a hope, and this one is now in
docs/performance.md beside the rest.
