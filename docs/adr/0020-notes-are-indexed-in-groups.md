# ADR-0020: Notes are indexed in groups

- **Status:** Accepted
- **Date:** 2026-08-15
- **Applies to:** `modules/apps/desktop`
- **Supersedes:** the "notes are committed one at a time" decision in ADR-0018
- **Related:** ADR-0002, ADR-0018, ADR-0019

## Context

ADR-0018 decided that a scan writes each note in its own transaction, so that a
search run while a scan is in progress sees the notes indexed so far. It said
plainly what would make that decision wrong: the measured cost of not batching,
and that the cost was affordable.

It stopped being affordable. Once notes carried links, a scan of ten thousand
notes took twenty seconds — about three and a half minutes extrapolated to the
hundred thousand this is designed for, against the three-minute target in
ADR-0019.

A profile says where the twenty seconds went:

| | share of the work |
| --- | --- |
| Reading the files from disk | 1 % |
| Parsing markdown and frontmatter | 5 % |
| Storing what was parsed | 77 % |
| — of which, ending the transaction | 43 % |

Neither the disk nor the parser is worth touching. Almost half the cost of
indexing a vault was the ten thousand transaction boundaries, and a boundary
costs the same whether one note crossed it or five hundred did.

## Decision

**A scan writes notes in groups. A group is one transaction.**

Two bounds close a group, whichever comes first:

- **five hundred notes**, which is where the measured gain flattens out;
- **eight megabytes of file content**, which bounds what is held in memory. The
  count alone is sized for ordinary notes; a folder of long transcripts would
  otherwise be read whole before any of it was written.

**A note and the record that dates it are written together.** The size and
modification time that let the next scan skip a file go into the same
transaction as the note itself. So an interrupted scan leaves whole groups, and
what it lost is simply not indexed yet — never a note the index believes is
current when it was never finished writing.

**Progress is reported per group, not per note.** A group is a fraction of a
second, which is what progress is for; a caller that needs to know about an
individual note is asking the index a question, not watching a scan.

## What this costs, and why it is still right

ADR-0018's argument was that a search during a scan should see partial results.
It still does — it now sees them in steps of up to five hundred notes instead of
one. On the vault this is sized for that is a step of half a percent, arriving
about twice a second.

The other half of that argument, that one writer holds the lock for the length
of a transaction, is the real change: the write lock is now held for the length
of a group rather than a note. It does not block readers — WAL is what ADR-0018
established — and the only writers are the scan itself and adding a vault, which
is not something done while waiting.

**A cancelled scan loses the group it was filling.** Up to five hundred notes
that were read and parsed are not stored, and the next scan reads those files
again. That is seconds of repeated work in exchange for a scan that is three
times faster every time it runs to completion.

## Consequences

**Positive**

- A cold scan of ten thousand notes went from 20.8 s to 6.0 s, which brings a
  hundred thousand inside the target in ADR-0019 with room to spare.
- Crash behaviour is easier to state, not harder: the index never holds a note
  it cannot account for.

**Negative**

- The scan holds up to eight megabytes of parsed notes that are not yet stored.
- Interrupting a scan wastes more work than it used to.
- The two bounds are numbers that were right on one machine and one vault
  shape. They are recorded here so that changing them is a decision rather than
  a tweak, and docs/performance.md is where the evidence for them lives.

## Alternatives considered

**Keeping one transaction per note and finding the time elsewhere.** Rejected on
the profile: reading and parsing together are six percent of the work, so
perfecting both — including parsing in parallel, which was measured and
dropped — could not pay for the forty-three percent spent ending transactions.

**A cgo build of SQLite.** Roughly twice as fast on inserts in published
comparisons, and it costs the cross-compilation that ADR-0014 chose the pure-Go
driver for. Grouping is worth more than the driver swap and costs nothing.

**Turning durability off entirely.** Measured at thirteen percent, and it makes
the database file corruptible rather than merely stale. The index is a cache and
a stale one is free to rebuild, but a corrupt one has to be noticed first.

**Groups of five thousand.** Measured, and worth about seven percent more than
five hundred — not enough to justify holding that many notes in memory.
