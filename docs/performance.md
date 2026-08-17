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

## The source layer: finding a passage

The numbers above are the note index. These are the source layer: text cut into
chunks, embedded, and searched. The decisions they were taken for are in
ADR-0006, ADR-0007, ADR-0029 and ADR-0030.

Recorded 2026-08-17 on the same AMD Ryzen 7 6800U, `modernc.org/sqlite` v1.56
with the bundled `sqlite-vec` v0.1.9, WAL with `synchronous = NORMAL`.

**The harness is not in the tree yet.** These were taken with standalone
programs; moving them here is outstanding work, and until it is done the numbers
below cannot be reproduced from a checkout.

### The corpus

Not generated. The Mahābhārata in three languages: the Ganguli prose
translation, the BORI critical edition in transliterated Sanskrit, the Russian
academic translation, and four smaller sources — about 65 million characters of
digital text once the scans whose recognition is wrong and the duplicate
editions are dropped.

Embedded with `bge-m3` at 1024 dimensions, through a hosted API.

| | |
| --- | --- |
| Windows read from | 37 380 |
| Windows searched | 145 800 |
| Embedding, the whole corpus | 10 minutes |
| Index | 613 MB |
| Search | 32 ms |

The same corpus embedded on this laptop's CPU instead, with a small
English-only model, runs at 10.8 chunks per second.

### The window decides what can be found

One passage, one query, four windows cut around the same sentence. The score is
against a query that paraphrases the sentence in another language.

| Window | Similarity |
| --- | --- |
| 25 words | 0.751 |
| 50 words | 0.648 |
| 100 words | 0.544 |
| 200 words | 0.517 |

The best score anything in the corpus reaches for that query is 0.639. At 25
words the passage would lead by a wide margin; at 200 it ranks **401st of
36 560**, which is not a result anybody sees.

This is the measurement ADR-0007 exists for. Nothing about the index changed
between those rows.

The window also has to fit the model. Cut on the sections a translation already
carries, in words:

| Window | Chunks | Median tokens | Over the 512-token limit |
| --- | --- | --- | --- |
| 350 words | 9 115 | 488 | 22.6 % |
| 250 words | 13 207 | 350 | 0.6 % |
| 200 words | 16 382 | 281 | 0.2 % |

Transliterated Sanskrit and Cyrillic reach the limit in fewer words than Latin
does — 1.27 tokens per word here against the usual English rate.

### Dimensions cost more than bits

145 800 vectors. Agreement is with what the same model says at full precision,
over the top twenty.

| Representation | Bytes/vector | Index | Agreement |
| --- | --- | --- | --- |
| float32, 1024 dims | 4096 | 597 MB | 1.000 |
| **int8, 1024 dims** | **1024** | **149 MB** | **0.975** |
| PCA to 512 | 2048 | 299 MB | 0.742 |
| PCA to 256 | 1024 | 149 MB | 0.725 |
| PCA to 128 | 512 | 75 MB | 0.625 |
| First 512 numbers | 2048 | 299 MB | 0.708 |
| First 256 numbers | 1024 | 149 MB | 0.475 |
| One bit per dimension | 128 | 19 MB | 0.525 |
| PCA to 64 | 256 | 37 MB | 0.367 |
| First 128 numbers | 512 | 75 MB | 0.283 |
| First 64 numbers | 256 | 37 MB | 0.133 |

Read the rows at 1024 bytes together: every dimension at one byte agrees 0.975,
the same storage spent on 256 projected dimensions agrees 0.725, and on the
first 256 numbers 0.475. This model does not order its dimensions by importance,
which is why truncating is worse than projecting.

The binary row is the coarse pass. It is not accurate and is not meant to be:
half the right answers are outside its top twenty, which is why the pass keeps a
hundred candidates for a result of six.

Replacing float32 with int8 in the working index: 1561 MB to 613 MB, load 10 s
to 5 s, search 35.7 ms to 31.8 ms, and the top six identical on every question
in the acceptance set.

### A partition key charges for every partition

The same 16 382 vectors, the same query, only the number of partitions changes.
Nothing filters by the key.

| Partitions | Vectors each | Build | Unfiltered search |
| --- | --- | --- | --- |
| none | 16 382 | 0.2 s | 1.93 ms |
| 18 | 911 | 0.2 s | 2.10 ms |
| 64 | 256 | 0.3 s | 6.12 ms |
| 256 | 64 | 0.4 s | 23.93 ms |
| 1 024 | 16 | 1.0 s | 92.45 ms |
| 4 096 | 4 | 3.3 s | 275.64 ms |
| 16 382 | 1 | 48.7 s | 1067.30 ms |

About **65 µs per partition walked**, flat across the range, independent of what
each holds. A partition per book over ten thousand books would spend two thirds
of a second before comparing a single vector.

A metadata column carries no such walk, and no narrowing either:

| | Unfiltered | Narrowed to one cluster |
| --- | --- | --- |
| Cluster as partition key | 24.37 ms | 0.30 ms |
| Cluster as metadata column | 1.82 ms | 1.45 ms |

Both return the same rows. The partition key prunes; the column filters what it
has already looked at.

### Notes rank beside books

157 vectors from a vault against 145 800 from the sources — a thousand to one.
Best rank reached, over six questions:

| Indexed as | Ranks |
| --- | --- |
| A window of a source | 1, 1, 1, 8, 1, 1 |
| A note cut the same way | 2, 7, 19, 1, 91, 7865 |
| A whole note, 283–357 words | 2045, 4, 318, 3, 127, 14510 |
| A note's title | 580, 4590, 5541, 2, 510, 23808 |

Notes are not buried by the imbalance: ranking is by similarity and carries no
prior. What does bury them is being indexed whole — the same dilution the window
table shows, on a note instead of a book.

One question was answered by seven windows of a single note taking the top seven
places. That is what collapsing a document to one result is for.

### What these numbers are not

**The acceptance set is six questions.** Enough to decide between the options
here, not enough to promise a rate. Every "agreement" figure is six queries
wide.

**Agreement is with the model, not with a reader.** The full-precision ranking is
the reference, so a representation scoring 0.975 reproduces what this model
believes — including where it is wrong.

**The sizes are one corpus.** Window sizes tuned here are not guaranteed
elsewhere, which is why ADR-0007 makes the acceptance set the check rather than
the sizes.

**Sanskrit is indexed and does not surface.** No question in the set returned a
passage from the critical edition, in any configuration tried. Whether that is
the model, the transliteration or the way verse is cut is not established.

**Everything past 150 000 chunks is generated.** A corpus of six million was
measured for build time, index size and residency — 1316 MB, 148 s, 53 MB
resident — but its vectors are random, so nothing it says about accuracy means
anything. The recall figures from clustering come from the real corpus at
16 382, and the gap to six million is untested.

### What does not work

| | |
| --- | --- |
| Reducing dimensions instead of precision | Loses a quarter to a half at equal storage |
| Truncating a vector this model produces | Worse than projecting, at every size tried |
| Indexing a title | Ranks near-randomly; the lexical index already matches it |
| Indexing a note whole beside windowed sources | Erratic — 4th on one question, 14 510th on another |
| Writing chunks scattered across partitions | 1 900 rows/s against 40 000 grouped |
| Cutting a window by lines | One file put a book on four lines and produced a chunk of a million characters |

### What was invisible in review

**A batch is bounded, not an input.** The model's limit is 8192 tokens and the
provider bounds a request at 131 072 characters for the whole batch. A batch
size that holds for Latin text exceeds it in transliterated Sanskrit, where the
same word costs three times the tokens. Batches are built to a character budget.

**A stored vector's type is guessed from the length of its blob.** An int8 vector
of 1024 dimensions is 1024 bytes, which is also a float32 vector of 256, and the
extension reads it as the latter. Both sides of a comparison name their type. A
bit vector is written and matched through `vec_bit(?)` for the same reason.

**A cascade does not reach a virtual table.** Deleting a source cascades to its
chunks and stops there; the vector rows stay. They are deleted explicitly, by the
chunk number, which is the virtual table's rowid — the same rule `notes_fts`
already obeys.

**`INSERT OR REPLACE` is not honoured by `vec0`.** It raises `UNIQUE constraint
failed`, so a vector that is being replaced is deleted first.

**Re-saving a row does not cascade anything.** `save.sql` conflicts on
`(vault_id, path)` and updates, so the row number never changes and nothing hangs
off a deleted parent. Every derived table is therefore cleared by hand — headings,
links, problems, and now chunks. A test saves the same note twice and asserts the
counts did not double, because nothing else would notice.

**A query plan names the alias, not the table.** A check that forbade
`SCAN <table>` passed on `SCAN c`, so reading every chunk went unnoticed. The
check now forbids every `SCAN` step except a virtual table and `vaults`.

**One populated vault makes a vault filter look free.** With a single vault in the
fixture, the planner chose the primary key over the vault index and read every
vault's chunks. The fixture holds two vaults of 1500 notes for this reason, which
is the same lesson as measuring the database before trusting its plans.

**A semicolon inside a migration comment breaks the migration.** `migrate.go`
splits a file on semicolons without stripping comments. The convention is to
avoid them; the splitter has not been changed.

### Getting the text out of an EPUB

Recorded 2026-08-17 on the same AMD Ryzen 7 6800U. The timings below were taken
with a Python prototype; the structure figures further down come from the Go
extractor that ships. Where the two disagree, the Go figure is the one that
describes the application.

Forty EPUBs from `resources/mahabharata`: 85.5 MB of archives, 59 MB of text once
extracted.

The corpus is not in git. The test that measures it skips when the directory is
absent.

| | |
| --- | --- |
| Extraction, all forty | 2.4 s |
| Per book | 59 ms |
| Slowest book | 0.26 s |

Reading a passage back, which is what a result costs when the extracted text is
not stored (ADR-0006):

| | |
| --- | --- |
| One book, extracted again | 42 ms |
| One spine document of it | 5.9 ms |
| Slicing out of a stored 1.5 MB text column | 0.9 ms |
| Slicing out of a stored 60 MB text column | 66 ms |

`substr` costs about 1.1 ms per megabyte of the value it reads, so the largest
books are the ones a stored column serves worst. Re-reading one spine document
beats both.

What structure the forty carry, **measured with the Go extractor**:

| Where it came from | Books |
| --- | --- |
| The navigation document | 16 |
| Heading tags | 4 |
| Nothing at all | 20 |

Half the corpus offers nothing to cut on, which is what ADR-0006 requires
extraction to survive.

A tier answers only when it names at least two places. One named place is a
cover or a book's own title, and it carries no cut a caller does not already
have, because the offset of every spine document is reported separately. Four
books turn on that rule: they carry a single heading each, one of them across 853
spine documents. Counting a lone heading as structure puts those four in the
heading tier and reads 16 / 8 / 16.

Cover-only navigation is not a curiosity: eighteen books carry a single navPoint
labelled `Начать` pointing at the title page, which is why a tier is judged by
its usable entries rather than by whether the file exists.

Five of the forty carry the page numbers of a print edition, 54 to 311 targets
each, all through `.ncx pageList`. No book in this corpus uses EPUB 3's
`epub:type="pagebreak"`.

### Embedding, locally and through a service

Recorded 2026-08-17 on the same AMD Ryzen 7 6800U — 16 threads, AVX2 and no
AVX-512 — with `CGO_ENABLED=0`, `intfloat/multilingual-e5-small`, real
tokenisation and 256-token windows.

| Local, pure Go | Chunks/s | 400 000 chunks |
| --- | --- | --- |
| default build | 1.38 | about 80 hours |
| `GOEXPERIMENT=simd` | 3.12 | about 36 hours |

The flag is worth 2.2×, has to be set when the binary is built, and nothing in
the repository records it. It belongs wherever releases are built.

Window size moves it as much as the flag does: the same model answers 6.0/s at
128 tokens and 1.06/s at 512.

For comparison, a hosted service embedded 145 800 chunks of the source corpus in
**10 minutes** for about ten cents.

What that means for the default: a personal vault of a few thousand notes is ten
to twenty thousand chunks, which finishes locally in one to two hours. A hundred
thousand notes is four hundred thousand chunks, and local is then a day and a
half of background work. ADR-0002 already says the vector index fills in behind
the lexical one and may never finish, so neither figure blocks anything — but only
the service answers a corpus of that size in a sitting.

The default model is a 470 MB fp32 ONNX file, fetched on first use. An int8 export
was not tried.

### Why the markup is not parsed as XML

Four books hold documents that `encoding/xml` refuses — 125 documents in all,
failing with `element <p> closed by </html>`. `golang.org/x/net/html` reads every
one of them. That is the whole case for the dependency: a strict parser drops a
tenth of this corpus, and ADR-0006 does not allow extraction to refuse.

Two more shapes in the same corpus would have cost whole books. Three books
declare their spine documents as `media-type="text/html"`, so filtering the spine
by media type loses them entirely. Five carry no `dc:title` at all, so an empty
title is a correct answer rather than a parse failure.

### What a chunked index will cost

**An extrapolation, not a measurement.** The chunk rate is from the source corpus
above; the vault it is applied to is generated markdown of a different shape, and
nothing has yet built this index.

The sources cut about **2 240 small windows per megabyte of text**. The load-test
vault is 164 MB of markdown, so a hundred thousand notes cut the same way
(ADR-0007) come to **370 000 – 400 000 chunks**.

| | |
| --- | --- |
| int8 vectors, 1024 B each | ~400 MB |
| Bit vectors, 128 B each | ~50 MB |
| Chunk rows | ~30 MB |
| Full-text index over chunks | ~100 MB |
| The note index as it stands | 162 MB |
| | **about 750 MB** |

ADR-0019's target of 5 MB per thousand notes allows 500 MB. This is the
arithmetic ADR-0030 rebudgets against.

## Editing a note in a tab

```
go test ./internal/core/usecase/note/ -run XXX -bench 'Read|Save' -benchtime 50x
```

Taken 2026-08-17, AMD Ryzen 7 6800U, NVMe, `-benchtime 50x`.

| | |
|---|---|
| Read a note, vault of 1 000 | 98 µs |
| Read a note, vault of 100 000 | 99 µs |
| Save, 500-word body | 10.06 ms |
| Save, 5 000-word body | 10.07 ms |

**A read does not grow with the vault.** The two figures are within a percent of
each other, which is what says the read addresses one file and looks at nothing
else. This is the cost a window pays per clean tab per change that names its
path: ten open tabs are about a millisecond of reading per change.

**A save costs the same whatever is in it.** Both bodies land in the same 10 ms,
so what is being measured is not the text. A save syncs the temporary file and
then the folder it is renamed into, and those two are the whole figure. Against
a quiet interval of 800 ms it is not a wait a person can notice; it is worth
recording because it says where a faster save would have to come from.

### Not measured yet

- keystroke to picture redrawn, at a hundred thousand notes. It crosses the
  webview, the schema, the write, the watcher and the index, and nothing here
  drives that whole path.
- what one open tab costs resident. Every tab of a pane is drawn and hidden, so
  tab count is live editor count, and only a browser can answer it.

### What an edit costs to embed

```
go test ./internal/adapter/index/ -run Recut -count=1
```

These are assertions rather than timings: the count of vectors a save asks the
model for. A 200-word note cuts into one large chunk and five small ones, tiled
at fifty words with ten of overlap.

| the edit is | vectors asked for | chunks that keep their row |
|---|---|---|
| a line added to the frontmatter | 0 | 6 of 6 |
| at the end of the body | 1 | 4 of 5 small |
| at the start of the body | 5 | 0 |

The first row is the one that says the hash is over the text and not over the
offsets: every chunk moves in the file and none of them changes, so nothing is
embedded again. The third is the shape of the cost that remains, and what would
have to be spent to bound it is places — a heading dividing the body into spans
that tile separately, so an edit re-cuts one of them.
