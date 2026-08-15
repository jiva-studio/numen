# Performance

The budgets are in [ADR-0002](adr/0002-sqlite-is-a-cache.md). This page is what
they measure to, so that a change which quietly costs a second is visible before
it ships rather than after someone complains that the application feels slow.

## Running it

```
cd modules/apps/desktop
go test ./internal/core/usecase/vault/ -run XXX -bench ColdScan -benchtime 1x
go test ./internal/core/usecase/vault/ -run XXX -bench 'WarmScan|Incremental|Search'
go test ./internal/core/usecase/vault/ -run XXX -bench 'Links|Backlinks' -benchtime 300x
```

The vault is generated, not downloaded: `testsupport.GenerateVault` writes notes
of varying length across fifty folders, from a fixed seed, so two runs measure
the same work.

Only the cold scan is pinned to a single iteration — it takes seconds, and
repeating it measures patience. The rest cost milliseconds and are run many
times, which is what makes their figures worth quoting: a single sample of a
millisecond-scale benchmark varies by tens of percent, and the link benchmarks
in particular read a single-figure answer out of a large table, where one run
measures little but the page cache.

## At the size this is designed for

A hundred thousand notes, from the load test. The targets these are compared
against are in [ADR-0019](adr/0019-performance-targets.md).

| | Measured | Target | |
| --- | --- | --- | --- |
| Cold scan | 1 m 24 s (0.84 ms/note, 1196 notes/s) | under 3 minutes | ✅ |
| Warm scan — what a startup pays | 0.72 s | under 1 second | ✅ |
| Index size | 577 MB (5.8 MB per thousand notes) | under 5 MB per thousand | ❌ |
| Search, rare term, under load | p50 5 ms · p95 7 ms · p99 8 ms · max 65 ms | p95 under 50 ms | ✅ |
| Search, term matching every note | p50 2.15 s · p95 2.86 s | not covered | |

Both search rows come from four concurrent readers running against a scan that
rewrote the whole vault continuously. The difference between them is not the
database: it is that one query matches a hundred notes and the other matches all
hundred thousand, and ranking a hundred thousand matches is linear work. Real
queries look like the first row; a vault generated from twenty words produces
only the second, which is why the load test asks both. The 65 ms worst case in
the first row is one search out of twelve thousand landing while the writer
committed; the p99 is 8 ms.

**The index is over its size budget, and this is the first honest measurement
of it.** 577 MB against 316 MB before notes carried links: the links themselves,
two indexes over them, and the fact that the full-text table holds the only copy
of every note's text. The scan and search figures beside it were extrapolated
from ten thousand notes until now and turned out to be pessimistic; this one was
extrapolated too, and turned out to be optimistic. It is left failing rather
than quietly re-targeted — what to do about it is a decision, and the cheapest
lead is that the full-text table stores the text a second time.

## What links cost, and what paid for them

Generated notes carry links — a parent in frontmatter and two or three
references in prose — because a vault of notes that link to nothing measures a
parser and an index that are faster than the real ones. Adding them made a cold
scan of ten thousand notes go from 7.6 s to 20.8 s, and put a hundred thousand
over the target in [ADR-0019](adr/0019-performance-targets.md).

| 10 000 notes | with links, before | after |
| --- | --- | --- |
| Cold scan | 20.8 s | **6.0 s** |
| Resolving one note's links | 0.17 ms | 0.25 ms |
| Backlinks of one note | 67 ms | **1.8 ms** |

Backlinks also stopped growing with the vault: 1.71 ms on a thousand notes
against 1.76 ms on ten thousand. Before, the same pair was 5.3 ms and 58 ms.

Two changes did this, and each is an ADR because each gave something up.

**Notes are written in groups** ([ADR-0020](adr/0020-notes-are-indexed-in-groups.md)).
A profile of the twenty seconds put 1 % of it in reading files, 5 % in parsing
them, and 43 % in ending transactions — ten thousand of them, one per note,
each costing the same whether one note crossed it or five hundred did. What was
given up is that a cancelled scan now loses the group it was filling.

**The index measures itself afterwards**
([ADR-0021](adr/0021-the-index-measures-itself.md)). A database that has never
been measured answers by rule of thumb, and the rule of thumb for "what points
at this note" was to narrow to the vault and read every link in it — seventy
thousand rows to find twenty. The indexes for the fast answer already existed;
what was missing was the knowledge that they helped.

The second one had been hiding behind a test. The plan test measured the
database itself before reading the plans, so it certified a plan the application
never got — and its assertion, that no query reads a whole table, was true of
the slow plan as well: reading every link in a vault is a *search*, through a
real index, and looks entirely reasonable. Plans are now asserted by the index
each question is answered through, on a database measured the way the
application measures it.

## Baseline

Recorded 2026-08-15 on an AMD Ryzen 7 6800U, `modernc.org/sqlite`, WAL with
`synchronous = NORMAL`.

| | 1 000 notes | 10 000 notes | per note |
| --- | --- | --- | --- |
| Cold scan — every note read, parsed, written | 0.46 s (1 run) | 6.01 s (1 run) | 0.60 ms |
| Warm scan — nothing changed, no file opened | 4.7 ms (231 runs) | 51 ms (24) | 5 µs |
| Incremental — one note edited | 8.0 ms (139) | 55 ms (19) | — |
| Search — two terms, twenty results | 6.6 ms (300) | 44 ms (300) | — |
| Search while a scan is writing | — | 60 ms (21) | — |
| Resolving one note's links | 0.33 ms (300) | 0.25 ms (300) | — |
| Backlinks of one note | 1.71 ms (300) | 1.76 ms (300) | — |

A search costs about a third more while a scan is continuously rewriting the
index — 60 ms against 44 ms for the same query on a quiet database. That is the
number ADR-0018 exists to keep honest: WAL lets a reader answer without waiting
for the writer, and the write pool is capped at one connection so writers queue
in Go rather than collide in SQLite.

Neither link figure grows with the vault, which is the point of them: the two
rows are ten times apart in size and within a fraction of a millisecond of each
other. Resolution is faster on ten thousand notes than on one thousand, which is
noise rather than a finding.

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

Four findings account for every large number that has moved so far, and all four
are worth remembering because none was visible in the code.

**Almost half of indexing was the boundary, not the work.** A profile of a cold
scan put 1 % of it in reading files, 5 % in parsing them and 43 % in ending
transactions. Nothing about the code says which of those it is spending its
time on, and the two that look expensive — the disk and the parser — were six
percent together. Writing notes in groups took the scan from 20.8 s to 6.0 s on
ten thousand notes.

**A database that has never been measured answers by rule of thumb.** SQLite
picks between indexes from what it knows about how much is stored, and a
database filled by a scan has never been asked. The rule of thumb for backlinks
was to narrow to the vault and read every link in it. Measuring after a scan took
that from 58 ms to 1.8 ms and stopped it growing. The indexes were already
right; nothing had told the database they helped.

Both of those are also a lesson about tests. The plan test measured the database
itself and asserted that no query read a whole table — and passed, while the
application ran the slow plan, because reading every link in a vault *is* an
index search and reports itself as one. A test that sets up a condition the
application never reaches proves nothing about the application.

**Deleting a full-text row by its columns scans the whole index.** An FTS5 table
has no key but its rowid, and `Save` deletes before it inserts, so the cost of
writing one note grew with the size of the index: a rebuild was quadratic. Fixed
by addressing the row by the note's own rowid, which is now asserted by a test.

**An fsync per note dominated everything else.** With `synchronous = FULL`, a
cold scan of a thousand notes took 5.8 seconds; with `NORMAL` it takes 0.73. The
index is a cache, so the durability being paid for protects nothing: a crash
costs a rescan of files that are still on disk, which is the same thing that
happens when the index is deleted on purpose.

## What was tried and did not work

Kept because it is cheaper to read this list than to measure these again, and
because most of them are the advice one finds when looking for it.

| Tried, on ten thousand notes written in groups of 500 | Result |
| --- | --- |
| Turning off the full-text index's background merging during a bulk load — the usual advice | 7.3 s against 4.4 s: **1.7× worse** |
| Larger full-text page size (`pgsz = 8000`) | 7.6 s: worse |
| Merge thresholds either way (`automerge` 8 and 16, `crisismerge` 8) | no difference |
| Page cache of 4 MB, 64 MB, 256 MB | no difference; the 64 MB currently configured earns nothing |
| Reusing prepared statements | 9 % without grouping, nothing with it |
| `synchronous = OFF` | 13 %, in exchange for a file that can be corrupt rather than merely stale |

Two of these were measured only because they are what a search engine returns
for "slow SQLite inserts". The one about merging is worse than doing nothing,
which is a good reason to measure advice before taking it.

Removing the full-text table entirely takes the same work from 4.4 s to 2.4 s.
Nothing follows from that except a bound: search costs 45 % of what it takes to
write a note, and no amount of tuning around it will find that back.
