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
half the right answers are outside its top twenty, which is why it keeps eight
candidates for every result and the byte-precision vectors order what it kept.

Replacing float32 with int8 in the working index: 1561 MB to 613 MB, load 10 s
to 5 s, search 35.7 ms to 31.8 ms, and the top six identical on every question
in the acceptance set.

### The similarity floor, and what the coarse pass has to keep

Taken once with a standalone program, against one index of 65 261 embedded
chunks — `baai/bge-m3` at 1024 dimensions, int8, over the Ganguli Mahābhārata
and a few hundred notes. Eighteen questions, ten the vault holds an answer to
and eight it does not. Top-1 cosine, exhaustive over every stored vector.

What the floor and the rerank do is held by fixtures in `chunks_test.go`. What
the floor should be is this measurement, and a model changed is a measurement
to take again.

| Held | | Unheld | |
| --- | --- | --- | --- |
| game of dice / Draupadi | 0.716 | `SELECT * FROM users WHERE deleted_at IS NULL` | **0.521** |
| amortised analysis of a dynamic array | 0.783 | semiconductor revenue guidance | 0.494 |
| chicken | 0.781 | mating habits of emperor penguins | 0.489 |
| death of Karna | 0.687 | changing a diesel oil filter | 0.475 |
| a king who does not protect | 0.687 | Kubernetes ingress controller | 0.458 |
| Krishna's counsel to Arjuna | 0.679 | `asdkjfh qwpoeiru zxcvbnm` | 0.443 |
| house of lac | 0.637 | best pizza toppings in Naples | 0.432 |
| dharma is subtle | 0.606 | sourdough hydration ratio | 0.413 |
| Bhishma's silence | 0.593 | | |
| Lagrangian in classical mechanics | **0.520** | | |

The two columns overlap by 0.0012: the SQL string reaches higher than the
Lagrangian note. **The floor is 0.50** — under every held question by 0.0199,
over the second-highest unheld by 0.0056. Seven of the eight unheld questions
then keep nothing at all and the eighth keeps 4 of 160; every held question
keeps its answer, from 2 passages for the Lagrangian to 160 for the narrative
ones.

The offset is per query, not per corpus: the corpus median ran from 0.207 to
0.403 across the eighteen. A floor stated against each query's own distribution
separates these two columns cleanly, and is a decision of its own.

**How many the coarse pass keeps.** Recall of a query's exact top twenty at or
above the floor, from a coarse pool of 20×f:

| f | Pool | Mean recall | Worst |
| --- | --- | --- | --- |
| 2 | 40 | 0.762 | 0.438 |
| 4 | 80 | 0.869 | 0.562 |
| 6 | 120 | 0.911 | 0.625 |
| **8** | **160** | **0.944** | **0.688** |
| 12 | 240 | 0.968 | 0.750 |
| 16 | 320 | 0.973 | 0.750 |

The curve flattens after eight while the coarse pass costs close to linear in k:
40 → 12 ms, 100 → 20 ms, 200 → 32 ms, 400 → 56 ms, 800 → 116 ms.

**What the rerank costs.** Same eighteen questions, six rounds each, warm, at a
limit of twenty.

| Meaning half | Median | p95 |
| --- | --- | --- |
| Coarse k=100, no rerank | 19.6 ms | 23.4 ms |
| Coarse k=160, rerank, floor | 26.2 ms | 30.9 ms |
| Coarse k=160 alone | 26.3 ms | 29.9 ms |

The rerank is 0.4 ms of that: one `json_each` join, 160 int8 blobs, 160×1024
dot products, a sort. The 7 ms is the wider coarse pass, less the 0.7 ms saved
by resolving 20 enclosing windows where there were 100.

The words half measures 1.1 ms median and about 39 ms p95 on a long natural
language query, where the FTS expression becomes a wide OR. A search running
both halves on such a query stands over ADR-0019's 50 ms and did before this.

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

### A save, and how long until the window knows

```
NUMEN_LOAD=1 go test ./internal/adapter/webui/ -run TestEditLoad -v -timeout 30m
```

Taken 2026-08-17, same machine. This is the path the application owns: the
write, the watcher noticing it, the refresh, and the change reaching a client.

| vault | save | until a client is told |
|---|---|---|
| 10 000 notes | 22 ms | 65 ms |
| 100 000 notes | 13 ms | 66 ms |

Neither figure grows with the vault. The watcher reports a path, the refresh
reads that one note, and what is walked is nothing — which is what says an edit
is answered by the size of the note and not the size of the library.

What is still not measured is the two ends a browser owns: a keystroke becoming
a request, and the picture being painted.

### What an open tab costs, and why there is no number

```
npx vitest run --project stories src/editor/Editor.stories.ts
```

Every tab of a pane is drawn and hidden, so tab count is live editor count, and
the story mounts ten in one page to find what the tenth costs.

It has no number. `performance.memory.usedJSHeapSize` is quantised by the
browser, and it reads the same 67.6 MB with one editor alive and with ten. So
what the story asserts is the part that can be checked — that ten editors are
alive at once — and the size is left unmeasured.

The instrument that would answer it is
`performance.measureUserAgentSpecificMemory()`, which needs the page to be
cross-origin isolated. Until the story runs in such a page there is no figure
here, and a figure taken from the quantised counter would be an invention.

### What an edit costs to embed

```
go test ./internal/adapter/index/ -count=1
```

These are assertions rather than timings: the count of vectors a save asks the
model for. Chunks are tiled at fifty words with ten of overlap, and a chunk is
identified by the hash of its text, so one whose text did not change keeps its
row and the vector on it.

A note's headings are its places. The chunks of one section are tiled inside that
section, and a note that names no place is one span tiled from its first word.
The two columns below are the same words cut both ways.

**A 200-word note under four headings**, one every 48 words. It holds five small
chunks over the whole body and four with the headings naming the sections.

| the edit is | no place named | the headings as places |
|---|---|---|
| a line added to the frontmatter | 0 asked · 5 of 5 kept | 0 asked · 4 of 4 kept |
| at the end of the body | 1 · 4 of 5 | 1 · 4 of 4 |
| in the middle of the body | 3 · 2 of 5 | 2 · 3 of 4 |
| at the start of the body | 5 · 0 of 5 | 2 · 3 of 4 |

**A 1000-word note under five headings**, one every 200 words. It holds 25 small
chunks either way.

| the edit is | no place named | the headings as places |
|---|---|---|
| a line added to the frontmatter | 0 asked · 25 of 25 kept | 0 asked · 25 of 25 kept |
| at the end of the body | 1 · 25 of 25 | 1 · 24 of 25 |
| in the middle of the body | 14 · 12 of 25 | 3 · 22 of 25 |
| at the start of the body | 26 · 0 of 25 | 5 · 20 of 25 |

The counts are of the chunks that carry a vector. The chunk enclosing the note
keeps its row in the frontmatter row alone: its text is the whole note, so any
edit to the body replaces it, and the chunks that were kept are pointed at the
row that is there now.

The frontmatter row is the one that says the hash is over the text and not over
the offsets: every chunk moves in the file and none of them changes, so nothing
is embedded again.

**What a heading buys is the last two rows of the second table.** With the note
as one span, an edit re-cuts every chunk after it and the worst case grows with
the note: 5 vectors at 200 words, 26 at a thousand. With a heading opening each
section the worst case is the chunks of that section at any length — 5 at a
thousand words, and 3 for an edit half way down.

**What it costs is chunk count on a note that is mostly headings.** A chunk is
never cut across a heading, so a section shorter than fifty words is a chunk of
its own. A 200-word note with a heading every five words is cut into 40 chunks
where the same words with no place named are cut into 7, and each of the 40 owes
a vector of its own. An edit anywhere in that note asks for one.

The first cut of the 200-word note asks for four vectors where one span asks for
five: a section of 48 words is one chunk, and nothing overlaps across a heading.
Every small chunk carries the name of the section it was cut inside; the chunk
enclosing the note carries the note's title, so a note is still answered by the
name it was given.

## Reading a PDF

Recorded 2026-08-19 on the same AMD Ryzen 7 6800U, `CGO_ENABLED=0`, pdfium
through WebAssembly.

The document is 546 pages of a scan at 600 dpi, 233 MB, carrying a text layer
somebody else's OCR left in it.

| | |
| --- | --- |
| The library, compiled | 5.1 s, once for the life of the process |
| Opening a document after that | 0.1–0.6 s |
| Its text layer, all of it | 10–12 s (1 701 572 characters, about 45 pages a second) |
| Naming every page | 4 ms |

The compile is the module, not the document, and it is paid by the first PDF a
run reads and by no other. The text of a smaller document is proportionally
quicker: 80 pages is under a second.

## Reading a scanned page with a model

The same machine, ONNX Runtime through `purego`, one page at 300 dpi: layout by
PP-DocLayoutV3, lines by PP-OCRv5 detection, text by PP-OCRv6 tiny.

| | |
| --- | --- |
| One page, ten regions | 8.6 s |
| A document of 546 pages | about 75 minutes |

Two settings are measured rather than guessed.

**Threads is four.** Two is slower (9.7 s a page), eight is slower (10.7), and
sixteen is much slower (23.2). The lines of a page are small, and spreading one
of them over sixteen threads costs more than it saves.

**One page at a time.** A second worker holds a second copy of every model, 124
MB of layout weights among them, and on a 13 GB machine the pair spent their time
in swap: 18.4 s a page against 8.5.

The recogniser is chosen for what it keeps rather than for its size. Measured
over three pages of a book set in transliterated Sanskrit, against the document's
own text layer with every mark folded away from both:

| | size | a page | keeps the letter | marks |
| --- | --- | --- | --- | --- |
| PP-OCRv6 tiny | 4.5 MB | 1.2 s | 92.6% | 142 |
| PP-OCRv6 small | 21 MB | 2.6 s | 81.5% | 156 |
| PP-OCRv6 medium | 77 MB | 8.1 s | 81.5% | 157 |
| PP-OCRv5 server | 85 MB | 7.9 s | 90.8% | 67 |

No model in the family can write a consonant with a dot below it. Small and
medium delete the letter they cannot spell — `ṭuṭaba hṛdayaka` becomes `uaba
hdayaka` — and one Sanskrit word in five loses a letter. Tiny writes the plain
letter instead, so the word keeps its length and a search still reaches it. The
smallest is also the fastest and, on this book, the best.

A recogniser that cannot spell a script at all writes plausible nonsense: the
same models over a Russian document return Latin gibberish, and what keeps most
of it out of the index is `window.legible` refusing to cut it.

## Where a document's own words sit

Recorded 2026-08-20 on the same machine, over the 546-page scan and its text
layer.

| | |
| --- | --- |
| Reading the whole text | 9.5 s, 1 702 107 bytes over 546 pages |
| The words of one page | 239 ms |
| The words of three pages | 213 ms, 71 ms a page |
| The words of ten pages | 403 ms, 40 ms a page |

**Almost all of it is opening the document.** Ten pages cost twice what one
does, not ten times, because the 233 MB file is read and parsed once and the
pages after the first are tens of milliseconds each. That is what a cache of
open documents is for, and it is why the pages wanted are asked for together.

The layer yields 572 boxes a page, 312 357 over the book, against the 51 169 a
recognition of the same pages produced. The gap is punctuation: this layer puts
a space before a comma, so 43 523 of the boxes are one byte standing alone in
the text.

## What a line's boundary costs

Recorded 2026-08-20 over the same scan: the four headings of pages 31 and 32,
and the body of twenty pages spread across the book, against the document's own
text layer. Error is the mean share of characters wrong.

The detector answers with the text's own outline drawn inside the letters, and
the amount it falls short is a share of the line's height. What widens it again
was a fixed number of pixels, measured on the part after it has been scaled — on
a heading that is about four pixels of the page against fifteen to twenty of
shrink. The top of every capital and the last letter of every line were cut off.

| the line widened by | headings | body |
| --- | --- | --- |
| 10 pixels | 0.074 | 0.0644 |
| 14 | 0.031 | 0.0633 |
| 18 | 0.025 | 0.0639 |
| 20 | 0.017 | 0.0648 |
| 24 | 0.035 | 0.0667 |

**Eighteen**, the middle of where it stops mattering rather than the edge. The
body is flat from ten to twenty and begins to pay after that.

```
"IAYADEVA GOSVAMI'S LIFEINNABADWI"         became "JAYADEVA GOSVAMI'S LIFE INNABADWIP"
"LAYADEVA GOSVAMI'S MARRIAGE TO PADMAVAT"  became "JAYADEVA GOSVAMI'S MARRIAGE TO PADMAVATI"
"THELORD HELPS IAYADEVA GOSVAMIWRITE…"     became "THE LORD HELPS JAYADEVA GOSVÁMI WRITE GITA GOVINDA"
```

Two other settings were measured and neither is worth moving. The heat a pixel
carries to be part of a line gives the same boxes at 0.15, 0.20 and 0.30 — the
map is as good as binary here. The longest side a part is read at costs small
type when lowered and costs everything when raised, and 300 dpi beats 150, 200,
400 and 600 because detection is scaled to that side whatever the page was drawn
at.

What no setting reaches: a word gap in a display face no wider than its letter
gaps is one line to any threshold, so the space is the recogniser's own guess.
And the alphabet the model carries has `ā ī ū ñ ś` and none of the letters with
a dot under them, so `Lakṣmaṇa` and `Kṛṣṇa` cannot be written whatever the boxes
are.

## A part too small to find a line in

Recorded 2026-08-20 over the same scan, pages 44 to 73 of the file, which print
14 to 43. The layout model finds the `number` region on every one of them — the
number stands alone at the foot between two ornaments — and it is some fifty
pixels across at 300 dpi.

Nothing is read in it. The detector draws a boundary around dark pixels and
shrinks it by a share of the line's height; a line filling the image it is
looked for in leaves no boundary to draw, so the region comes back blank.

| what was read | pages right of 30 |
| --- | --- |
| the region as it stands | 0 |
| the region magnified 3× to 20× | 0 |
| the crop widened to 150, 250, 400, 640, 960 pixels | 0 to 6 |

Widening the crop takes in the ornaments either side, and what they read as is
not a number. Keeping only the lines whose middle falls in the region gets 30 of
30 at a margin of 150 pixels and above — and that margin is a pixel count
against one book at one dpi, so it is not a default anything can carry.

So nothing reads it. A page is called where it stands in the file, which is known
for every page, costs nothing, and is the number the pane in front of the person
is showing.

The margin that works is the measurement worth keeping here: it says the failure
is the detector needing background around a line, not the models being unable to
read two digits.

## Drawing a page of a scan

Recorded 2026-08-20 over the same 546-page scan, five pages averaged, on an
idle machine. The recognition run competes for the same cores and for the same
pool of pdfium workers; under one, every number below is roughly doubled.

| width asked | pdfium | resample | jpeg | total | over the wire |
| --- | --- | --- | --- | --- | --- |
| 800 | 443 ms | 43 | 17 | **503 ms** | 234 kB |
| 1200 | 498 | 95 | 34 | **628** | 417 kB |
| 1600 | 553 | 163 | 52 | **768** | 625 kB |
| 2400 | 616 | 377 | 107 | **1101** | 1094 kB |

**Drawing barely moves with the resolution** — 104 dpi costs 443 ms and 311 dpi
costs 616. So the cost is not rasterising: each page of this book holds one
large photograph, and the library decodes the whole of it whatever size is
asked for. Half a second a page is what this document costs, and no setting
here reaches it.

**The resample was for a pixel or two.** The resolution is a whole number
rounded up, so a page asked for at 800 comes back 804 across. Scaling those
four pixels off cost between a tenth and four tenths of a second, and the window
lays the page out at the width it asked for anyway. It is gone: the page goes as
it was drawn, and the browser takes the four pixels off in the compositor.

That leaves the half second, and it is the same half second every time a page is
turned back to. So the drawn pages are kept, in this machine's cache folder and
not in the vault — a page turned back to is read from disk in about a
millisecond, and a book read through once costs its half second a page and never
again.

## Finding the section a question is about

Recorded 2026-08-20 over the same 546-page scan. 700 questions drawn from the
book's own parts: for each of 350 parts whose heading the recogniser read as
words, one question is that heading — a person asking where the book speaks
about a thing — and one is seven words from the middle of the part. The right
answer is that part either way.

| | right section | opened at its start |
| --- | --- | --- |
| the words half as it stands | 307/350 | 298/350 |
| the section's name in every chunk | 340/350 | 306/350 |
| the sections findable by name | 346/350 | 346/350 |
| both | 348/350 | 348/350 |

The questions quoting a phrase from the middle of a part moved by one either
way — 331, 330, 331, 330 — so none of these costs anything on a question that
names no section.

**The second column is the one that decided it.** Putting the name into every
chunk lifts the right chapter but still answers with whatever paragraph of it
ranks best; making a section findable by its own name answers with the section,
at its heading.

What the words half was doing is visible in one query. Asked for
`Madhavendra Puri`, BM25 scored the chapter that *is* about him at −12.74 and
the subsections inside it between −17.8 and −19.7 — and lower is better. The
chapter's opening says his name once, in its heading; a paragraph in the
subsection after it says it four times.

**The name in every chunk was left.** It is worth two points on top of the
other, and it changes the text a vector is made from: the recipe changes, and
every chunk of every recognised document is embedded again. Only the words half
was measured here — what a vector that knows its chapter is worth cannot be
known without buying those vectors. The number to beat is 346 and 346.
