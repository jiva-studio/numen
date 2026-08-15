# Performance

The budgets are in [ADR-0002](adr/0002-sqlite-is-a-cache.md). This page is what
they measure to, so that a change which quietly costs a second is visible before
it ships rather than after someone complains that the application feels slow.

## Running it

```
cd modules/apps/desktop
go test ./internal/core/usecase/vault/ -run XXX -bench ColdScan -benchtime 1x
go test ./internal/core/usecase/vault/ -run XXX -bench 'WarmScan|Incremental|Search'
```

The vault is generated, not downloaded: `testsupport.GenerateVault` writes notes
of varying length across fifty folders, from a fixed seed, so two runs measure
the same work.

Only the cold scan is pinned to a single iteration — it takes seconds, and
repeating it measures patience. The other three cost milliseconds and run to the
default duration, which is what makes their figures worth quoting: a single
sample of a millisecond-scale benchmark varies by tens of percent.

## At the size this is designed for

A hundred thousand notes, from the load test. The targets these are compared
against are in [ADR-0019](adr/0019-performance-targets.md).

| | Measured | Target |
| --- | --- | --- |
| Cold scan | 1 m 32 s (0.92 ms/note, 1090 notes/s) | under 3 minutes |
| Warm scan — what a startup pays | 0.78 s | under 1 second |
| Index size | 316 MB (3.2 MB per thousand notes) | under 5 MB per thousand |
| Search, rare term, under load | p50 4 ms · p95 7 ms · p99 9 ms · max 14 ms | p95 under 50 ms |
| Search, term matching every note | p50 2.08 s · p95 2.27 s | not covered |

Both search rows come from four concurrent readers running against a scan that
rewrote the whole vault continuously. The difference between them is not the
database: it is that one query matches a hundred notes and the other matches all
hundred thousand, and ranking a hundred thousand matches is linear work. Real
queries look like the first row; a vault generated from twenty words produces
only the second, which is why the load test asks both.

## Baseline

Recorded 2026-08-15 on an AMD Ryzen 7 6800U, `modernc.org/sqlite`, WAL with
`synchronous = NORMAL`.

| | 1 000 notes | 10 000 notes | per note |
| --- | --- | --- | --- |
| Cold scan — every note read, parsed, written | 0.71 s (1 run) | 7.58 s (1 run) | 0.76 ms |
| Warm scan — nothing changed, no file opened | 5.1 ms (268 runs) | 55 ms (24) | 5 µs |
| Incremental — one note edited | 5.6 ms (235) | 50 ms (24) | — |
| Search — two terms, twenty results | 8.2 ms (156) | 55 ms (19) | — |
| Search while a scan is writing | — | 64 ms (57) | — |

A search costs about a third more while a scan is continuously rewriting the
index — 64 ms against 48 ms for the same query on a quiet database. That is the
number ADR-0018 exists to keep honest: WAL lets a reader answer without waiting
for the writer, and the write pool is capped at one connection so writers queue
in Go rather than collide in SQLite.

Extrapolated to the 100k notes ADR-0002 designs for: a cold scan is about 75
seconds, against a budget of single-digit minutes. A warm scan is under half a
second, against two seconds to interactive.

## What these numbers are not

**The search figure is pessimistic.** Generated notes are built from a
twenty-word vocabulary, so a two-term query matches nearly every note and the
ranking has to order all of them. A real vault has a long tail of terms and a
query that matches a handful of notes.

**The incremental figure is a full walk**, because that is all there is today:
one edited note still costs a `stat` of every file in the vault. A file watcher
would reindex the one path it was told about, and the 100 ms budget in ADR-0002
is written for that path rather than this one.

**The cold scan reads notes this same process wrote seconds earlier**, so the
read side is measured against a warm page cache. On a real vault that has been
sitting on disk, the extrapolation to 100k is optimistic about I/O.

**Nothing here measures a cold start of the application**, only of the scan.

## What was learned changing them

Two findings account for every large number that has moved so far, and both are
worth remembering because neither was visible in the code.

**Deleting a full-text row by its columns scans the whole index.** An FTS5 table
has no key but its rowid, and `Save` deletes before it inserts, so the cost of
writing one note grew with the size of the index: a rebuild was quadratic. Fixed
by addressing the row by the note's own rowid, which is now asserted by a test.

**An fsync per note dominated everything else.** With `synchronous = FULL`, a
cold scan of a thousand notes took 5.8 seconds; with `NORMAL` it takes 0.73. The
index is a cache, so the durability being paid for protects nothing: a crash
costs a rescan of files that are still on disk, which is the same thing that
happens when the index is deleted on purpose.
