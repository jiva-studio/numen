# Performance

The targets are in [ADR-0019](adr/0019-performance-targets.md). This page is
what they measure to.

## Running it

```
cd modules/apps/desktop
go test ./internal/core/usecase/vault/ -run XXX -bench ColdScan -benchtime 1x
go test ./internal/core/usecase/vault/ -run XXX -bench 'WarmScan|Incremental|Search'
go test ./internal/core/usecase/vault/ -run XXX -bench 'Links|Backlinks' -benchtime 300x
NUMEN_LOAD=1 go test ./internal/core/usecase/vault/ -run TestLoad -v -timeout 40m
```

The vault is generated, not downloaded: `testsupport.GenerateVault` writes notes
of varying length across fifty folders, each naming a parent and pointing at a
few others, from a fixed seed.

Only the cold scan is pinned to a single iteration — it takes seconds, and
repeating it measures patience. The rest run many times: a single sample of a
millisecond-scale benchmark varies by tens of percent, and the link benchmarks
read a single-figure answer out of a large table, where one run measures the
page cache.

## At the size this is designed for

A hundred thousand notes, from the load test.

| | Measured | Target |
| --- | --- | --- |
| Cold scan | 1 m 22 s (0.82 ms/note, 1216 notes/s) | under 3 minutes |
| Warm scan — what a startup pays | 0.52 s | under 1 second |
| Index size | 162 MB (1.6 MB per thousand notes) | under 5 MB per thousand |
| Search, rare term, under load | p50 2 ms · p95 4 ms · p99 5 ms · max 42 ms | p95 under 50 ms |
| Search, term matching every note | p50 0.90 s · p95 1.01 s | not covered |

Both search rows come from four concurrent readers running against a scan that
rewrote the whole vault continuously. The difference between them is not the
database: one query matches a hundred notes and the other matches all hundred
thousand, and ranking a hundred thousand matches is linear work. Real queries
look like the first row; a vault generated from twenty words produces only the
second, which is why the load test asks both.

The vault itself is 164 MB of markdown, so the index is about the size of the
text it describes.

## Baseline

Recorded 2026-08-15 on an AMD Ryzen 7 6800U, `modernc.org/sqlite`, WAL with
`synchronous = NORMAL`.

| | 1 000 notes | 10 000 notes |
| --- | --- | --- |
| Cold scan — every note read, parsed, written | 0.46 s (1 run) | 5.9 s (3 runs) |
| Warm scan — nothing changed, no file opened | 4.7 ms (231 runs) | 50 ms (10) |
| Incremental — one note edited | 8.0 ms (139) | 55 ms (19) |
| Search — two terms, twenty results | 2.5 ms (300) | 20 ms (300) |
| Search while a scan is writing | — | 60 ms (21) |
| Resolving one note's links | 0.18 ms (300) | 0.18 ms (300) |
| Backlinks of one note | 0.88 ms (300) | 1.11 ms (300) |
| Index size | 1.6 MB | 15.5 MB |

Neither link figure grows with the vault: the two columns are ten times apart in
size and within a fraction of a millisecond of each other.

A search costs about a third more while a scan is continuously rewriting the
index. That is the number ADR-0018 exists to keep honest: WAL lets a reader
answer without waiting for the writer, and the write pool is capped at one
connection so writers queue in Go rather than collide in SQLite.

## What these numbers are not

**The search figure is pessimistic.** Generated notes are built from a
twenty-word vocabulary, so a two-term query matches nearly every note and the
ranking has to order all of them. A real vault has a long tail of terms and a
query that matches a handful of notes.

**The incremental figure is a full walk**: one edited note, found by asking
every file in the vault what it looks like. That is what a scan at startup pays,
and it is not what an edit costs while the application is running — the watcher
names the path and only that path is read. The 100 ms budget is written for the
watcher's path, which nothing here measures yet.

**The cold scan reads notes this same process wrote seconds earlier**, so the
read side is measured against a warm page cache. On a real vault that has been
sitting on disk, this is optimistic about I/O.

**Nothing here measures a cold start of the application**, only of the scan.

## Where the time goes

A cold scan, by profile: 1 % reading the files, 5 % parsing them, 77 % storing
what was parsed — of which the transaction boundaries are the largest single
part. Parsing in parallel is therefore worth at most a few percent, and has been
measured and left alone.

The full-text index accounts for about 45 % of what a write costs.

A warm scan of ten thousand notes is 50 ms: 13 ms asking the index what it knows
about every file, and 34 ms walking the vault and asking the file system. The
walk stats one file per note and nothing else — the extension is checked from
the directory entry, before anything is opened — so that half is the cost of
looking at a whole vault at all. It is what the watcher removes between scans
rather than what a faster walk would.

## Looking a vault over

`BenchmarkRun` in `internal/core/lint`, on a vault where every name answers for
many notes and every link is written by one — the worst case the ambiguity check
has, because every link in it is a candidate.

| | Measured |
| --- | --- |
| Every check that runs unasked, 1 000 notes | 2.1 ms |
| Every check that runs unasked, 10 000 notes | 23 ms |
| Dangling links alone, 10 000 notes | 53 ms |

Two of the checks read rows a scan already stored. The other two work the whole
vault out when they are asked, which is why they are not stored: what they
answer stops being true when a note somewhere else moves.

The number to watch is the second one, because it is the one that grows with the
vault rather than with the number of duplicate names. A real vault has few
shared names, so its ambiguity check does almost nothing; this one is what the
shape costs when the answer is "all of them".

## What does not work

Measured, on ten thousand notes written in groups of five hundred.

| | |
| --- | --- |
| Turning off the full-text index's background merging during a bulk load | 1.7× slower |
| Larger full-text page size (`pgsz = 8000`) | 1.7× slower |
| Merge thresholds either way (`automerge` 8 and 16, `crisismerge` 8) | no difference |
| Page cache of 4 MB, 64 MB, 256 MB | no difference |
| Reusing prepared statements across a group | no difference |
| `synchronous = OFF` | 13 % faster, and the file can be corrupt rather than merely stale |

The first two are what a search returns for "slow SQLite inserts".

## Two things that were invisible in review

**Deleting a full-text row by its columns scans the whole index.** An FTS5 table
has no key but its rowid, so a rebuild that addresses the row any other way is
quadratic in the number of notes. A test asserts the statements address it by
rowid.

**A database that has never been measured answers by rule of thumb.** SQLite
picks between indexes from what it knows about how much is stored, and a
database filled by a scan has never been asked. The rule of thumb for backlinks
is to narrow to the vault and read every link in it. A scan measures the index
when it changed it, and a test asserts the index each question is answered
through — a test that only forbids reading a whole table passes on the slow
plan, because reading every link in a vault is an index search.
