# ADR-0022: Notes are indexed in groups

- **Status:** Accepted
- **Date:** 2026-08-15
- **Applies to:** `modules/apps/desktop`
- **Supersedes:** the "notes are committed one at a time" decision in ADR-0018
- **Related:** ADR-0002, ADR-0018, ADR-0019

## Context

ADR-0018 wrote each note in its own transaction so that a search run during a
scan sees the notes indexed so far. It named the condition that would make that
wrong: the cost of not batching, while it stays affordable.

A profile of a cold scan puts 1 % of the work in reading the files, 5 % in
parsing them, and 43 % in ending transactions. A transaction boundary costs the
same whether one note crossed it or five hundred did, and there was one per
note.

## Decision

**A scan writes notes in groups. A group is one transaction.**

Two bounds close a group, whichever comes first:

- **five hundred notes**, where the measured gain flattens out;
- **eight megabytes of file content**, which bounds what is held in memory. The
  count alone is sized for ordinary notes; a folder of long transcripts would
  otherwise be read whole before any of it was written.

**A note and the record that dates it are written together.** The size and
modification time that let the next scan skip a file go into the same
transaction as the note. An interrupted scan therefore leaves whole groups, and
what it lost is not indexed yet — never a note the index believes is current
when it was never finished writing.

**Progress is reported per group.** A group is a fraction of a second, which is
what progress is for. A caller that needs to know about an individual note is
asking the index a question rather than watching a scan.

## What this gives up

A search during a scan sees results arrive in steps of up to five hundred notes.
On the vault this is sized for that is half a percent at a time, about twice a
second.

The write lock is held for the length of a group. It does not block readers, and
the only other writer is adding a vault, which is not done while waiting.

**A cancelled scan loses the group it was filling.** Up to five hundred notes
were read and parsed and are not stored, so the next scan reads those files
again.

## Consequences

**Positive**

- A cold scan of a hundred thousand notes is inside the target in ADR-0019 with
  room to spare.
- Crash behaviour is simple to state: the index never holds a note it cannot
  account for.

**Negative**

- The scan holds up to eight megabytes of parsed notes that are not yet stored.
- Interrupting a scan wastes more work.
- Both bounds are numbers measured on one machine and one vault shape. They are
  here so that changing them is a decision rather than a tweak;
  docs/performance.md holds the evidence.

## Alternatives considered

**Keeping one transaction per note and finding the time elsewhere.** Rejected on
the profile: reading and parsing together are six percent of the work, so
perfecting both — including parsing in parallel, which was measured and
dropped — cannot pay for the forty-three percent spent ending transactions.

**A cgo build of SQLite.** Roughly twice as fast on inserts in published
comparisons, and it costs the cross-compilation that ADR-0014 chose the pure-Go
driver for. Grouping is worth more and costs nothing.

**Turning durability off entirely.** Thirteen percent, in exchange for a
database file that can be corrupt rather than merely stale. The index is a cache
and a stale one is free to rebuild; a corrupt one has to be noticed first.

**Groups of five thousand.** Seven percent better than five hundred, for ten
times the memory.
