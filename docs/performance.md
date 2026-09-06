# Performance

A target is a decision, and every one this application holds itself to is in the Target column below and nowhere else. A measurement carries the date it was taken and the machine it was taken on, and lives here beside the target it answers.

The load test reports and does not assert. It runs on whatever laptop is at hand, and the numbers come here, where a person compares them with what was there before. A target is set at roughly twice what is measured, which leaves room to notice a regression before a user does.

## Running it

```
cd modules/libs/core
go test ./usecase/vault/ -run XXX -bench ColdScan -benchtime 1x
go test ./usecase/vault/ -run XXX -bench 'WarmScan|Incremental|Search'
go test ./usecase/vault/ -run XXX -bench 'Links|Backlinks' -benchtime 300x
NUMEN_LOAD=1 go test ./usecase/vault/ -run TestLoad -v -timeout 40m
go test ./usecase/flashcards/ -run XXX -bench Vault -benchtime 5x -benchmem -timeout 40m
go test ./usecase/flashcards/ -run XXX -bench PresetCurve -benchtime 3x -benchmem -timeout 40m
go test ./usecase/flashcards/ -run XXX -bench CurveCards -benchtime 3x -count 2 -benchmem -timeout 180m
go test ./adapter/window/flashcards/ -run XXX -bench FrontDoor -benchtime 5x -count 2 -timeout 40m
go test ./internal/adapter/filesystem/ -run XXX -bench Derived -benchmem -count 3
go test ./adapter/index/ -run XXX -bench Unembedded -benchmem -count 10 -benchtime 300x
go test ./adapter/index/ -run XXX -bench SaveVectors -benchmem -count 10 -benchtime 50x
go test ./adapter/index/ -run XXX -bench ChunkIdentity -benchmem -count 10
```

The bundle each window is built into, from the repository root:

```
npm run build --prefix modules/apps/desktop/flashcards
npm run build --prefix modules/apps/desktop/editor
```

The vault is generated, not downloaded: `testsupport.GenerateVault` writes notes of varying length across fifty folders, each naming a parent and pointing at a few others, from a fixed seed.

Only the cold scan is pinned to a single iteration — it takes seconds, and repeating it measures patience. The rest run many times: a single sample of a millisecond-scale benchmark varies by tens of percent, and the link benchmarks read a single-figure answer out of a large table, where one run measures the page cache.

## At the size this is designed for

A hundred thousand notes, from the load test. Measured 2026-09-06, one run, on a machine carrying five other agents' work; the 2026-09-01 column is what it answered then.

| | 2026-09-01 | Measured | Target |
| --- | --- | --- | --- |
| Cold scan | 1 m 22 s (0.82 ms/note) | **8 m 52 s (5.32 ms/note, 188 notes/s)** | under 3 minutes |
| Warm scan — what a startup pays | 0.52 s | **1.07 s** | under 1 second |
| An edit the watcher names, until a client is told | 66 ms | not re-measured | under 100 ms |
| Index size | 162 MB | **344 MB (3.4 MB per thousand notes)** | not decided |
| Search, rare term, under load | p50 2 · p95 4 · p99 5 · max 42 ms | p50 3 · p95 6 · p99 8 · max 102 ms | p95 under 50 ms |
| Search, term matching every note | p50 0.90 s · p95 1.01 s | p50 3.37 s · p95 4.25 s | not covered |

**Two of these targets are now missed, and the index size says why.** It has gone from 162 MB to 344 MB over the same 164 MB of markdown, and that figure owes nothing to a busy machine. A note is now written into the passage index as well as the note index — the chunk enclosing it, the small chunks tiled inside it, each of them indexed for its words in `chunks_fts` and again in `sections_fts` — so a scan writes about twice as much, and a query that matches everything ranks several rows a note instead of one.

**What a person's own search costs has held.** The rare-term row is 3 ms at the median and 6 at the ninety-fifth, well inside a target of fifty, against 2 and 4 on a quieter machine. The read side did not move; the write side and the size did.

Both search rows come from four concurrent readers running against a scan that rewrote the whole vault continuously. The difference between them is not the database: one query matches a hundred notes and the other matches all hundred thousand, and ranking a hundred thousand matches is linear work. Real queries look like the first row; a vault generated from twenty words produces only the second, which is why the load test asks both.

The vault itself is 164 MB of markdown, so the index is now about twice the size of the text it describes.

## Baseline

Re-measured 2026-09-06 on the same AMD Ryzen 7 6800U, `modernc.org/sqlite`, WAL with `synchronous = NORMAL`. The 2026-08-15 column is kept beside it because the two are not the same measurement everywhere, and the rows below say where.

**The machine was shared with five other agents throughout, and that is the largest single fact about the two 2026-09-06 columns.** The one-minute load average ran between 4 and 61 over the session, against sixteen threads. The noise floor is read off `BenchmarkChunkIdentity`, which is arithmetic and touches nothing: the two conversions recorded at 25 ns and 23 ns on 2026-09-05 answer 29–37 ns and 26–31 ns over ten runs here, so **a figure that touches no disk runs about 1.3× the record on this machine today**. Everything that touches the disk runs further out than that, and no row below is quoted closer than its spread.

Each cell is the best of the runs stated, with the spread beside it. The best is quoted rather than the median because every source of noise here is additive.

| | 2026-08-15 | 1 000 notes | 10 000 notes |
| --- | --- | --- | --- |
| Cold scan — every note read, parsed, written | 0.46 s · 5.9 s | 1.80 s (6 runs, 1.80–2.84) | 20.1 s (6, 20.1–39.7) |
| Warm scan — nothing changed, no file opened | 4.7 ms · 50 ms | 9.2 ms (3, 9.2–15.3) | 58 ms (3, 58–122) |
| Incremental — one note edited | 8.0 ms · 55 ms | 12.4 ms (3, 12.4–17.6) | 64 ms (3, 64–136) |
| Resolving one note's links | 0.18 ms · 0.18 ms | 0.64 ms (3, 0.64–0.68) | 0.43 ms (3, 0.43–0.61) |
| Backlinks of one note | 0.88 ms · 1.11 ms | 2.94 ms (3, 2.94–3.28) | 3.18 ms (3, 3.18–4.63) |

Neither link figure grows with the vault, which is the claim those two rows carry: the columns are ten times apart in size and a fraction of a millisecond apart in answer, as they were.

**The cold scan is three to four times the record here, and six and a half times it at a hundred thousand.** A note now goes into the passage index as its own chunks, so `chunk.Replace` runs inside `saveNote`; by profile it is 42 % of the scan, and the index it fills is twice the size it was. What is left over is the machine and one repeated pass, both below.

**The two search rows of 2026-08-15 cannot be reproduced, and are not carried forward.** They measured `Queries.Search` and `search.sql`, which read `chunks_fts` and grouped the hits back to notes. That statement was deleted on 2026-09-05 as dead production code, and the benchmark now asks `port.PassageQueries.Lexical` — the query the application has been searching through all along. It is a heavier plan: a join to `sources`, the folder filter through `json_each` behind a bloom filter, a left join to the enclosing chunk, and a temp B-tree for the ordering. Against the generated vault's twenty-word vocabulary the query matches every note, so what it measures is ranking the whole corpus.

| | Measured 2026-09-06 |
| --- | --- |
| `Lexical`, a term matching every note, 1 000 | 11.8 ms (3 runs, 11.8–29.6) |
| `Lexical`, a term matching every note, 10 000 | 101 ms (3, 101–155) |
| The same while a scan is writing, 10 000 | 149 ms (3, 149–641) |

Those belong beside the "term matching every note" row of the load test, not beside a search figure. What a person's own search costs is the rare-term row further up, which the load test measures and this benchmark cannot: a vault built from twenty words has no rare terms but the one that was planted.

A search still costs more while a scan is continuously rewriting the index. WAL lets a reader answer without waiting for the writer, and the write pool is capped at one connection, so writers queue in Go.

## What these numbers are not

**The search figure is pessimistic.** Generated notes are built from a twenty-word vocabulary, so a two-term query matches nearly every note and the ranking has to order all of them. A real vault has a long tail of terms and a query that matches a handful of notes.

**The incremental figure is a full walk**: one edited note, found by asking every file in the vault what it looks like. That is what a scan at startup pays, and it is not what an edit costs while the application is running — the watcher names the path and only that path is read. The 100 ms target is written for the watcher's path, and that path is measured further down, from the save to a client being told.

**The cold scan reads notes this same process wrote seconds earlier**, so the read side is measured against a warm page cache. On a real vault that has been sitting on disk, this is optimistic about I/O.

**Nothing here measures a cold start of the application**, only of the scan. What the start costs on the Go side is measured under "What a window costs before a person can type".

**A flashcard benchmark runs on an index that does not flush.** It shares its setup with the tests around it, and what it times is scheduling arithmetic. The vault benchmarks and the load test ask for the index the application ships with, so those two numbers are not one another's.

## Where the time goes

A cold scan, by profile, taken 2026-09-06 over one run of `ColdScan/10000` — 34.7 s of wall time, 37.6 s of samples over sixteen threads. Reading the files is about 1 % and parsing the markdown a few percent, as before; what has moved is inside the writing.

| | share of the scan | with the statements kept compiled |
| --- | --- | --- |
| `Scan.Execute` | 84 % | 82 % |
| ↳ `note.Repository.Save` → `saveNote` | 78 % | 74 % |
| ↳ `chunk.Replace` — the note as its own chunks | **42 %** | 42 % |
| SQLite parsing SQL text (`sqlite3RunParser`) | **20 %** | **nothing the profile shows** |

The second column is the same benchmark after the two changes below, and one run of it costs 20.7 s of samples where it cost 32.9 s.

**Ending transactions is no longer the largest part.** The 2026-08-15 profile read 1 % reading, 5 % parsing, 77 % storing, of which 43 % of the whole scan was ending transactions. Since then a note is written into the passage index as well as the note index — the chunk enclosing it, the small chunks tiled inside it, and every one of them indexed for its words in `chunks_fts` and `sections_fts`. That path is 42 % of the scan and it is why the cold scan costs what it now costs.

**A fifth of a cold scan was SQLite reading SQL text it had read before.** `chunk.prepare` prepared four statements and closed them inside `chunk.Replace`, which `saveNote` calls once per note; `insert_link` was prepared per note beside it; and the dozen or so `exec` helpers go through `tx.ExecContext`, which the driver prepares afresh on every call. The transaction spans a group of five hundred notes, so ten thousand notes parsed the same statement texts something like a hundred and fifty thousand times.

**A statement text that ends in a newline is compiled twice on every call.** The driver compiles the text when the statement is made, and keeps what it compiled only where the text held one statement and nothing after it; a text with anything past the semicolon is a script to it, and a script is compiled again inside every `Exec` and `Query`. Every statement of this index is a file, and a file ends in a newline. Taking the whitespace off what `sqlfile` loads is **half of the parsing**: 22.5 % of the profile becomes 11.5 %, and the samples one run of `ColdScan/10000` costs fall from 32.9 s to 21.6 s.

**A transaction now keeps the statements it prepared.** `writing.Transaction` prepares each text the first time the transaction runs it and runs the prepared statement after that, so a group of five hundred notes compiles its two dozen statements once between them. That is the other half: 11.5 % of the profile becomes nothing the profile shows, and one run costs 20.7 s of samples where it cost 21.6 s. The reuse alone, measured against the record and without the text change above, was 22.5 % of parsing become 17.9 % and 32.9 s of samples become 23.9 s — the two answer the same waste from either end, and both are needed to take it out.

The record's line that reusing prepared statements across a group makes no difference is under "What does not work", and it was measured on 2026-08-15, when a note save was a handful of statements and `chunk.Replace` was not on the path.

**Read the shares here and not the clock.** The machine was carrying five other agents and a load average between 6 and 43 while these were measured, so the wall time of one scan says more about the hour than about the code. What a profile counts is this process's own samples, and the share of them one function holds is what the paragraphs above compare.

A transaction boundary costs the same whether one note crossed it or five hundred did, which is why notes are written in groups. A cgo build of SQLite is roughly twice as fast on inserts in published comparisons; it is not measured here, and it costs the cross-compilation the pure-Go driver is chosen for.

A warm scan of ten thousand notes is 50 ms: 13 ms asking the index what it knows about every file, and 34 ms walking the vault and asking the file system. The walk stats one file per note and nothing else — the extension is checked from the directory entry, before anything is opened — so that half is the cost of looking at a whole vault at all. It is what the watcher removes between scans rather than what a faster walk would.

## Looking a vault over

`BenchmarkRun` in `check`, on a vault where every name answers for many notes and every link is written by one — the worst case the ambiguity check has, because every link in it is a candidate.

| | Measured |
| --- | --- |
| Every check that runs unasked, 1 000 notes | 2.1 ms |
| Every check that runs unasked, 10 000 notes | 23 ms |
| Dangling links alone, 10 000 notes | 53 ms |

Two of the checks read rows a scan already stored. The other two work the whole vault out when they are asked, which is why they are not stored: what they answer stops being true when a note somewhere else moves.

The number to watch is the second one, because it is the one that grows with the vault rather than with the number of duplicate names. A real vault has few shared names, so its ambiguity check does almost nothing; this one is what the shape costs when the answer is "all of them".

## What does not work

Measured, on ten thousand notes written in groups of five hundred.

| | |
| --- | --- |
| Turning off the full-text index's background merging during a bulk load | 1.7× slower |
| Larger full-text page size (`pgsz = 8000`) | 1.7× slower |
| Merge thresholds either way (`automerge` 8 and 16, `crisismerge` 8) | no difference |
| Page cache of 4 MB, 64 MB, 256 MB | no difference |
| Reusing prepared statements across a group | no difference on 2026-08-15; **a third of a cold scan's samples on 2026-09-06** |
| Groups of five thousand | 7 % faster than five hundred, for ten times the memory |
| `synchronous = OFF` | 13 % faster, and the file can be corrupt rather than merely stale |

The first two are what a search returns for "slow SQLite inserts".

The fifth row was measured when a note save was a handful of statements. A note save now runs `chunk.Replace`, and a transaction spanning five hundred notes ran the same two dozen statements five hundred times over — a fifth of a cold scan, measured 2026-09-06 and set out under "Where the time goes", where what taking it out was worth is also written.

## Two things that were invisible in review

**Deleting a full-text row by its columns scans the whole index.** An FTS5 table has no key but its rowid, so a rebuild that addresses the row any other way is quadratic in the number of notes. A test asserts the statements address it by rowid.

**A database that has never been measured answers by rule of thumb.** SQLite picks between indexes from what it knows about how much is stored, and a database filled by a scan has never been asked. The rule of thumb for backlinks is to narrow to the vault and read every link in it. A scan measures the index when it changed it, and backlinks are then answered in about a millisecond and do not grow with the vault. A test asserts the index each question is answered through — a test that only forbids reading a whole table passes on the slow plan, because reading every link in a vault is an index search.

## The source layer: finding a passage

The numbers above are the note index. These are the source layer: text cut into chunks, embedded, and searched. The decisions they were taken for are in [A source is text in one table](adr/0010-a-source-is-text-in-one-table.md) and the four decisions after it.

Recorded 2026-08-17 on the same AMD Ryzen 7 6800U, `modernc.org/sqlite` v1.56 with the bundled `sqlite-vec` v0.1.9, WAL with `synchronous = NORMAL`.

**The harness is not in the tree yet.** These were taken with standalone programs; moving them here is outstanding work, and until it is done the numbers below cannot be reproduced from a checkout.

### The corpus

Not generated. The Mahābhārata in three languages: the Ganguli prose translation, the BORI critical edition in transliterated Sanskrit, the Russian academic translation, and four smaller sources — about 65 million characters of digital text once the scans whose recognition is wrong and the duplicate editions are dropped.

Embedded with `bge-m3` at 1024 dimensions, through a hosted API.

| | |
| --- | --- |
| Windows read from | 37 380 |
| Windows searched | 145 800 |
| Embedding, the whole corpus | 10 minutes |
| Index | 613 MB |
| Search | 32 ms |

The same corpus embedded on this laptop's CPU instead, with a small English-only model, runs at 10.8 chunks per second.

Cut this way, the sources come to about **2 240 small chunks per megabyte of text**.

### The chunk decides what can be found

One passage, one query, four chunks cut around the same sentence. The score is against a query that paraphrases the sentence in another language.

| Size | Similarity |
| --- | --- |
| 25 words | 0.751 |
| 50 words | 0.648 |
| 100 words | 0.544 |
| 200 words | 0.517 |

The best score anything in the corpus reaches for that query is 0.639. At 25 words the passage would lead by a wide margin; at 200 it ranks **401st of 36 560**, which is not a result anybody sees.

This is the measurement the chunk sizes are chosen against. Nothing about the index changed between those rows.

The chunk also has to fit the model. Cut on the sections a translation already carries, in words:

| Size | Chunks | Median tokens | Over the 512-token limit |
| --- | --- | --- | --- |
| 350 words | 9 115 | 488 | 22.6 % |
| 250 words | 13 207 | 350 | 0.6 % |
| 200 words | 16 382 | 281 | 0.2 % |

Transliterated Sanskrit and Cyrillic reach the limit in fewer words than Latin does — 1.27 tokens per word here against the usual English rate.

### Dimensions cost more than bits

145 800 vectors. Agreement is with what the same model says at full precision, over the top twenty.

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

Read the rows at 1024 bytes together: every dimension at one byte agrees 0.975, the same storage spent on 256 projected dimensions agrees 0.725, and on the first 256 numbers 0.475. This model does not order its dimensions by importance, which is why truncating is worse than projecting.

The binary row is the coarse pass. It is not accurate and is not meant to be: half the right answers are outside its top twenty, which is why it keeps eight candidates for every result and the byte-precision vectors order what it kept.

Replacing float32 with int8 in the working index: 1561 MB to 613 MB, load 10 s to 5 s, search 35.7 ms to 31.8 ms, and the top six identical on every question in the acceptance set.

### The similarity floor, and what the coarse pass has to keep

Taken once with a standalone program, against one index of 65 261 embedded chunks — `baai/bge-m3` at 1024 dimensions, int8, over the Ganguli Mahābhārata and a few hundred notes. Eighteen questions, ten the vault holds an answer to and eight it does not. Top-1 cosine, exhaustive over every stored vector.

What the floor and the rerank do is held by fixtures in `chunks_test.go`. What the floor should be is this measurement, and a model changed is a measurement to take again.

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

The two columns overlap by 0.0012: the SQL string reaches higher than the Lagrangian note. **The floor is 0.50** — under every held question by 0.0199, over the second-highest unheld by 0.0056. Seven of the eight unheld questions then keep nothing at all and the eighth keeps 4 of 160; every held question keeps its answer, from 2 passages for the Lagrangian to 160 for the narrative ones.

The offset is per query, not per corpus: the corpus median ran from 0.207 to 0.403 across the eighteen. A floor stated against each query's own distribution separates these two columns cleanly, and is a decision of its own.

**How many the coarse pass keeps.** Recall of a query's exact top twenty at or above the floor, from a coarse pool of 20×f:

| f | Pool | Mean recall | Worst |
| --- | --- | --- | --- |
| 2 | 40 | 0.762 | 0.438 |
| 4 | 80 | 0.869 | 0.562 |
| 6 | 120 | 0.911 | 0.625 |
| **8** | **160** | **0.944** | **0.688** |
| 12 | 240 | 0.968 | 0.750 |
| 16 | 320 | 0.973 | 0.750 |

The curve flattens after eight while the coarse pass costs close to linear in k: 40 → 12 ms, 100 → 20 ms, 200 → 32 ms, 400 → 56 ms, 800 → 116 ms.

**What the rerank costs.** Same eighteen questions, six rounds each, warm, at a limit of twenty.

| Meaning half | Median | p95 |
| --- | --- | --- |
| Coarse k=100, no rerank | 19.6 ms | 23.4 ms |
| Coarse k=160, rerank, floor | 26.2 ms | 30.9 ms |
| Coarse k=160 alone | 26.3 ms | 29.9 ms |

The rerank is 0.4 ms of that: one `json_each` join, 160 int8 blobs, 160×1024 dot products, a sort. The 7 ms is the wider coarse pass, less the 0.7 ms saved by resolving 20 enclosing chunks where there were 100.

The words half measures 1.1 ms median and about 39 ms p95 on a long natural language query, where the FTS expression becomes a wide OR. A search running both halves on such a query stands over the 50 ms target above, and did before this.

### A partition key charges for every partition

The same 16 382 vectors, the same query, only the number of partitions changes. Nothing filters by the key.

| Partitions | Vectors each | Build | Unfiltered search |
| --- | --- | --- | --- |
| none | 16 382 | 0.2 s | 1.93 ms |
| 18 | 911 | 0.2 s | 2.10 ms |
| 64 | 256 | 0.3 s | 6.12 ms |
| 256 | 64 | 0.4 s | 23.93 ms |
| 1 024 | 16 | 1.0 s | 92.45 ms |
| 4 096 | 4 | 3.3 s | 275.64 ms |
| 16 382 | 1 | 48.7 s | 1067.30 ms |

About **65 µs per partition walked**, flat across the range, independent of what each holds. A partition per book over ten thousand books would spend two thirds of a second before comparing a single vector.

A metadata column carries no such walk, and no narrowing either:

| | Unfiltered | Narrowed to one cluster |
| --- | --- | --- |
| Cluster as partition key | 24.37 ms | 0.30 ms |
| Cluster as metadata column | 1.82 ms | 1.45 ms |

Both return the same rows. The partition key prunes; the column filters what it has already looked at.

### Notes rank beside books

157 vectors from a vault against 145 800 from the sources — a thousand to one. Best rank reached, over six questions:

| Indexed as | Ranks |
| --- | --- |
| A chunk of a source | 1, 1, 1, 8, 1, 1 |
| A note cut the same way | 2, 7, 19, 1, 91, 7865 |
| A whole note, 283–357 words | 2045, 4, 318, 3, 127, 14510 |
| A note's title | 580, 4590, 5541, 2, 510, 23808 |

Notes are not buried by the imbalance: ranking is by similarity and carries no prior. What does bury them is being indexed whole — the same dilution the chunk table shows, on a note instead of a book.

One question was answered by seven chunks of a single note taking the top seven places. That is what collapsing a document to one result is for.

### What these numbers are not

**The acceptance set is six questions.** Enough to decide between the options here, not enough to promise a rate. Every "agreement" figure is six queries wide.

**Agreement is with the model, not with a reader.** The full-precision ranking is the reference, so a representation scoring 0.975 reproduces what this model believes — including where it is wrong.

**The sizes are one corpus.** Chunk sizes tuned here are not guaranteed elsewhere, which is why the acceptance set is the check rather than the sizes.

**Sanskrit is indexed and does not surface.** No question in the set returned a passage from the critical edition, in any configuration tried. Whether that is the model, the transliteration or the way verse is cut is not established.

**Everything past 150 000 chunks is generated.** A corpus of six million was measured for build time, index size and residency — 1316 MB, 148 s, 53 MB resident — but its vectors are random, so nothing it says about accuracy means anything. The recall figures from clustering come from the real corpus at 16 382, and the gap to six million is untested.

### What does not work

| | |
| --- | --- |
| Reducing dimensions instead of precision | Loses a quarter to a half at equal storage |
| Truncating a vector this model produces | Worse than projecting, at every size tried |
| Indexing a title | Ranks near-randomly; the lexical index already matches it |
| Indexing a note whole beside sources cut into chunks | Erratic — 4th on one question, 14 510th on another |
| Writing chunks scattered across partitions | 1 900 rows/s against 40 000 grouped |
| Cutting a chunk by lines | One file put a book on four lines and produced a chunk of a million characters |

### What a second engine would cost

Nothing measured here argues for one. Two are reachable from Go if one is ever wanted, and the numbers above are what a candidate has to beat:

| Candidate | Reached from Go by | What it would cost |
| --- | --- | --- |
| `usearch` | Official bindings; the index memory-maps from disk | cgo, and cross-compilation from one machine |
| Qdrant Edge | A Rust crate, in-process, no bindings today | cgo, and bindings to write |

LanceDB has no Go bindings at all: reaching it means writing foreign-function bindings by hand, or running a second process beside the application.

An index that has to be held in memory to be searched is out whatever its speed. At the size this is designed for it asks for gigabytes, and it competes with everything else on the person's machine. That is what left quantisation and a staged retrieval as the direction measured above.

### What was invisible in review

**A batch is bounded, not an input.** The model's limit is 8192 tokens and the provider bounds a request at 131 072 characters for the whole batch. A batch size that holds for Latin text exceeds it in transliterated Sanskrit, where the same word costs three times the tokens. Batches are built to a character budget.

**A stored vector's type is guessed from the length of its blob.** An int8 vector of 1024 dimensions is 1024 bytes, which is also a float32 vector of 256, and the extension reads it as the latter. Both sides of a comparison name their type. A bit vector is written and matched through `vec_bit(?)` for the same reason.

**A cascade does not reach a virtual table.** Deleting a source cascades to its chunks and stops there; the vector rows stay. They are deleted explicitly, by the chunk number, which is the virtual table's rowid — the same rule `notes_fts` already obeys.

**`INSERT OR REPLACE` is not honoured by `vec0`.** It raises `UNIQUE constraint failed`, so a vector that is being replaced is deleted first.

**Re-saving a row does not cascade anything.** `save_source.sql` conflicts on `(vault_id, path)` and updates, so the row number never changes and nothing hangs off a deleted parent. Every derived table is therefore cleared by hand — headings, links, problems, and now chunks. A test saves the same note twice and asserts the counts did not double, because nothing else would notice.

**A query plan names the alias, not the table.** A check that forbade `SCAN <table>` passed on `SCAN c`, so reading every chunk went unnoticed. The check now forbids every `SCAN` step except a virtual table and `vaults`.

**One populated vault makes a vault filter look free.** With a single vault in the fixture, the planner chose the primary key over the vault index and read every vault's chunks. The fixture holds two vaults of 1500 notes for this reason, which is the same lesson as measuring the database before trusting its plans.

**A semicolon inside a migration comment breaks the migration.** `migrate.go` splits a file on semicolons without stripping comments. The convention is to avoid them; the splitter has not been changed.

### Getting the text out of an EPUB

Recorded 2026-08-17 on the same AMD Ryzen 7 6800U. The timings below were taken with a Python prototype; the structure figures further down come from the Go extractor that ships. Where the two disagree, the Go figure is the one that describes the application.

Forty EPUBs from `resources/mahabharata`: 85.5 MB of archives, 59 MB of text once extracted.

The corpus is not in git. The test that measures it skips when the directory is absent.

| | |
| --- | --- |
| Extraction, all forty | 2.4 s |
| Per book | 59 ms |
| Slowest book | 0.26 s |

Reading a passage back, which is what a result costs when the extracted text is not stored ([A book's text is a cache or an artifact](adr/0015-a-books-text-is-a-cache-or-an-artifact.md)):

| | |
| --- | --- |
| One book, extracted again | 42 ms |
| One spine document of it | 5.9 ms |
| Slicing out of a stored 1.5 MB text column | 0.9 ms |
| Slicing out of a stored 60 MB text column | 66 ms |

`substr` costs about 1.1 ms per megabyte of the value it reads, so the largest books are the ones a stored column serves worst. Re-reading one spine document beats both.

What structure the forty carry, **measured with the Go extractor**:

| Where it came from | Books |
| --- | --- |
| The navigation document | 16 |
| Heading tags | 4 |
| Nothing at all | 20 |

Half the corpus offers nothing to cut on, which is what extraction has to survive.

A tier answers only when it names at least two parts. One named part is a cover or a book's own title, and it carries no cut a caller does not already have, because the offset of every spine document is reported separately. Four books turn on that rule: they carry a single heading each, one of them across 853 spine documents. Counting a lone heading as structure puts those four in the heading tier and reads 16 / 8 / 16.

Cover-only navigation is not a curiosity: eighteen books carry a single navPoint labelled `Начать` pointing at the title page, which is why a tier is judged by its usable entries rather than by whether the file exists.

Five of the forty carry the page numbers of a print edition, 54 to 311 targets each, all through `.ncx pageList`. No book in this corpus uses EPUB 3's `epub:type="pagebreak"`.

### Embedding, locally and through a service

Recorded 2026-08-17 on the same AMD Ryzen 7 6800U — 16 threads, AVX2 and no AVX-512 — with `CGO_ENABLED=0`, `intfloat/multilingual-e5-small`, real tokenisation and 256-token inputs.

| Local, pure Go | Chunks/s | 400 000 chunks |
| --- | --- | --- |
| default build | 1.38 | about 80 hours |
| `GOEXPERIMENT=simd` | 3.12 | about 36 hours |

The flag is worth 2.2×, has to be set when the binary is built, and nothing in the repository records it. It belongs wherever releases are built.

Window size moves it as much as the flag does: the same model answers 6.0/s at 128 tokens and 1.06/s at 512.

For comparison, a hosted service embedded 145 800 chunks of the source corpus in **10 minutes** for about ten cents.

What that means for the default: a personal vault of a few thousand notes is ten to twenty thousand chunks, which finishes locally in one to two hours. A hundred thousand notes is four hundred thousand chunks, and local is then a day and a half of background work. The vector index fills in behind the lexical one and may never finish, so neither figure blocks anything — but only the service answers a corpus of that size in a session.

The default model is a 470 MB fp32 ONNX file, fetched on first use. An int8 export was not tried.

### What naming a chunk costs the walk

The core addresses a chunk by an opaque `domain.ChunkID` and resumes the walk on an opaque `port.ChunkCursor`, and the index spells both out of the row number it keeps. That is one `FormatInt` per row read and one `ParseInt` per vector written, over every chunk of a vault. What it comes to was assumed and is now measured.

Recorded 2026-09-05 on the same AMD Ryzen 7 6800U, from `BenchmarkUnembedded`, `BenchmarkSaveVectors` and `BenchmarkChunkIdentity` in `adapter/index`. Ten runs of each; the two conversions vary by under 10 % across them, and the pages by two- to fivefold, which is the shape of the answer.

| | Measured |
| --- | --- |
| A row spelled as a `domain.ChunkID` | 25 ns |
| A `port.ChunkCursor` read back as a row | 23 ns |
| One chunk read out of the index, in a page of 200 | 3.9 µs |
| A row turned into a `domain.Passage`, the whole of it | 0.67 µs |
| One vector written, in a page of 200 | 200 µs |

**It is lost in the noise, and the numbers say so plainly.** Over 400 000 chunks — the vault a hundred thousand notes cut into — both conversions together come to **19 ms**. Reading those rows out of SQLite is 1.6 s, writing their vectors is 80 s, and the embedding itself is 27 minutes through a service or 36 hours on this laptop's CPU. The conversion is a thousandth of the cheapest thing it sits beside, and the row a walk resumes from is spelled once per page of 200 rather than once per row.

The two pages are measured at a fixed number of iterations because a laptop with other work on it moves them by fivefold between runs, and a conversion two orders of magnitude smaller than the page cannot be read out of a difference of pages. It is measured on its own instead, and the row it is measured on is nine digits — longer than anything a vault of this size hands out.

### Hearing a recording

Recorded 2026-09-01 on the same AMD Ryzen 7 6800U — 8 threads of it — with the models quantised to int8 where the column says so. The figure is wall time over audio time, so 0.10 is ten minutes of speech heard in one.

**The target: a recording is heard faster than it plays.** Anything slower and a vault of lectures never catches up with itself.

| | ru | en | Peak RSS |
| --- | --- | --- | --- |
| parakeet-tdt-0.6b-v3 int8, ONNX | 0.08–0.15 | 0.08–0.11 | 1.2 GB |
| whisper tiny, ggml | 0.08 | 0.06 | 0.3 GB |
| whisper small, ggml | 0.33 | 0.22 | 0.9 GB |
| whisper small int8, ONNX | 0.39 | 0.26 | 1.4 GB |
| whisper large-v3-turbo int8, ONNX | 0.41 | 0.39 | 2.2 GB |
| whisper large-v3-turbo, ggml | 0.92 | 0.84 | 1.9 GB |

Quantising is what makes the largest model usable: the same weights through ggml at fp16 take 0.92, and through ONNX Runtime at int8 take 0.41. The ggml build of `small` quantised to q5\_1 is **slower** than the same model unquantised — 0.73 against 0.38 — so quantisation is worth measuring per build and not assumed.

Words wrong, against [FLEURS](https://huggingface.co/datasets/google/fleurs), 60 utterances a language, lowercased and stripped of punctuation:

| | ru | en |
| --- | --- | --- |
| parakeet-tdt-0.6b-v3 int8 | 7.1% | 8.6% |
| whisper large-v3-turbo, ggml | 4.6% | 9.2% |
| whisper small, ggml | 10.3% | 19.7% |

The two are not measured the same way and the numbers are not directly comparable: parakeet was given one utterance at a time, as a segmenter feeds it, and whisper was given the utterances joined into one file, which is how it does its best work. Read the table as two separate answers to "is this good enough", not as a race.

What settles it is the pair. Parakeet is better than `small` on Russian and five times quicker than it, and a third of the time of the turbo model that beats it. Half an hour of Russian speech is two and a half to four and a half minutes of one core.

One more thing decides it on real recordings, and no table above shows it. Whisper's encoder always reads a window of exactly 30 seconds, whatever is in it, so speech cut at the silences — which is how a recording is fed to a model — costs three times what its length says. Parakeet pays by the second.

The weights are 641 MB, fetched when the first recording is heard.

### Why the markup is not parsed as XML

Four books hold documents that `encoding/xml` refuses — 125 documents in all, failing with `element <p> closed by </html>`. `golang.org/x/net/html` reads every one of them. That is the whole case for the dependency: a strict parser drops a tenth of this corpus, and extraction is not allowed to refuse.

Two more shapes in the same corpus would have cost whole books. Three books declare their spine documents as `media-type="text/html"`, so filtering the spine by media type loses them entirely. Five carry no `dc:title` at all, so an empty title is a correct answer rather than a parse failure.

## Editing a note in a tab

```
go test ./usecase/note/ -run XXX -bench 'Read|Save' -benchtime 50x
```

Taken 2026-08-17, AMD Ryzen 7 6800U, NVMe, `-benchtime 50x`.

| | |
|---|---|
| Read a note, vault of 1 000 | 98 µs |
| Read a note, vault of 100 000 | 99 µs |
| Save, 500-word body | 10.06 ms |
| Save, 5 000-word body | 10.07 ms |

**A read does not grow with the vault.** The two figures are within a percent of each other, which is what says the read addresses one file and looks at nothing else. This is the cost a window pays per clean tab per change that names its path: ten open tabs are about a millisecond of reading per change.

**A save costs the same whatever is in it.** Both bodies land in the same 10 ms, so what is being measured is not the text. A save syncs the temporary file and then the folder it is renamed into, and those two are the whole figure. Against a quiet interval of 800 ms it is not a wait a person can notice; it is worth recording because it says where a faster save would have to come from.

### A save, and how long until the window knows

```
NUMEN_LOAD=1 go test ./adapter/window/editor/ -run TestEditLoad -v -timeout 30m
```

Taken 2026-08-17, same machine. This is the path the application owns: the write, the watcher noticing it, the refresh, and the change reaching a client.

| vault | save | until a client is told |
|---|---|---|
| 10 000 notes | 22 ms | 65 ms |
| 100 000 notes | 13 ms | 66 ms |

Neither figure grows with the vault. The watcher reports a path, the refresh reads that one note, and what is walked is nothing — which is what says an edit is answered by the size of the note and not the size of the library.

What is still not measured is the two ends a browser owns: a keystroke becoming a request, and the picture being painted.

### What an open tab costs, and why there is no number

```
npx vitest run --project stories src/editor/Editor.stories.ts
```

Every tab of a pane is drawn and hidden, so tab count is live editor count, and the story mounts ten in one page to find what the tenth costs.

It has no number. `performance.memory.usedJSHeapSize` is quantised by the browser, and it reads the same 67.6 MB with one editor alive and with ten. So what the story asserts is the part that can be checked — that ten editors are alive at once — and the size is left unmeasured.

The instrument that would answer it is `performance.measureUserAgentSpecificMemory()`, which needs the page to be cross-origin isolated. Until the story runs in such a page there is no figure here, and a figure taken from the quantised counter would be an invention.

### What an edit costs to embed

```
go test ./adapter/index/ -count=1
```

These are assertions rather than timings: the count of vectors a save asks the model for. Chunks are tiled at fifty words with ten of overlap, and a chunk is identified by the hash of its text, so one whose text did not change keeps its row and the vector on it.

A note's headings are its parts. The chunks of one section are tiled inside that section, and a note that names no part is one division tiled from its first word. The two columns below are the same words cut both ways.

**A 200-word note under four headings**, one every 48 words. It holds five small chunks over the whole body and four with the headings naming the sections.

| the edit is | no part named | the headings as parts |
|---|---|---|
| a line added to the frontmatter | 0 asked · 5 of 5 kept | 0 asked · 4 of 4 kept |
| at the end of the body | 1 · 4 of 5 | 1 · 4 of 4 |
| in the middle of the body | 3 · 2 of 5 | 2 · 3 of 4 |
| at the start of the body | 5 · 0 of 5 | 2 · 3 of 4 |

**A 1000-word note under five headings**, one every 200 words. It holds 25 small chunks either way.

| the edit is | no part named | the headings as parts |
|---|---|---|
| a line added to the frontmatter | 0 asked · 25 of 25 kept | 0 asked · 25 of 25 kept |
| at the end of the body | 1 · 25 of 25 | 1 · 24 of 25 |
| in the middle of the body | 14 · 12 of 25 | 3 · 22 of 25 |
| at the start of the body | 26 · 0 of 25 | 5 · 20 of 25 |

The counts are of the chunks that carry a vector. The chunk enclosing the note keeps its row in the frontmatter row alone: its text is the whole note, so any edit to the body replaces it, and the chunks that were kept are pointed at the row that is there now.

The frontmatter row is the one that says the hash is over the text and not over the offsets: every chunk moves in the file and none of them changes, so nothing is embedded again.

**What a heading buys is the last two rows of the second table.** With the note as one span, an edit re-cuts every chunk after it and the worst case grows with the note: 5 vectors at 200 words, 26 at a thousand. With a heading opening each section the worst case is the chunks of that section at any length — 5 at a thousand words, and 3 for an edit half way down.

**What it costs is chunk count on a note that is mostly headings.** A chunk is never cut across a heading, so a section shorter than fifty words is a chunk of its own. A 200-word note with a heading every five words is cut into 40 chunks where the same words with no place named are cut into 7, and each of the 40 owes a vector of its own. An edit anywhere in that note asks for one.

The first cut of the 200-word note asks for four vectors where one span asks for five: a section of 48 words is one chunk, and nothing overlaps across a heading. Every small chunk carries the name of the section it was cut inside; the chunk enclosing the note carries the note's title, so a note is still answered by the name it was given.

## Reading a PDF

Recorded 2026-08-19 on the same AMD Ryzen 7 6800U, `CGO_ENABLED=0`, pdfium through WebAssembly.

The document is 546 pages of a scan at 600 dpi, 233 MB, carrying a text layer somebody else's OCR left in it.

| | |
| --- | --- |
| The library, compiled | 5.1 s, once for the life of the process |
| Opening a document after that | 0.1–0.6 s |
| Its text layer, all of it | 10–12 s (1 701 572 characters, about 45 pages a second) |
| Naming every page | 4 ms |

The compile is the module, not the document, and it is paid by the first PDF a run reads and by no other. The text of a smaller document is proportionally quicker: 80 pages is under a second.

## Reading a scanned page with a model

The same machine, ONNX Runtime through `purego`, one page at 300 dpi: layout by PP-DocLayoutV3, lines by PP-OCRv5 detection, text by PP-OCRv6 tiny.

| | |
| --- | --- |
| One page, ten regions | 8.6 s |
| A document of 546 pages | about 75 minutes |

Two settings are measured rather than guessed.

**Threads is four.** Two is slower (9.7 s a page), eight is slower (10.7), and sixteen is much slower (23.2). The lines of a page are small, and spreading one of them over sixteen threads costs more than it saves.

**One page at a time.** A second worker holds a second copy of every model, 124 MB of layout weights among them, and on a 13 GB machine the pair spent their time in swap: 18.4 s a page against 8.5.

The recogniser is chosen for what it keeps rather than for its size. Measured over three pages of a book set in transliterated Sanskrit, against the document's own text layer with every mark folded away from both:

| | size | a page | keeps the letter | marks |
| --- | --- | --- | --- | --- |
| PP-OCRv6 tiny | 4.5 MB | 1.2 s | 92.6% | 142 |
| PP-OCRv6 small | 21 MB | 2.6 s | 81.5% | 156 |
| PP-OCRv6 medium | 77 MB | 8.1 s | 81.5% | 157 |
| PP-OCRv5 server | 85 MB | 7.9 s | 90.8% | 67 |

No model in the family can write a consonant with a dot below it. Small and medium delete the letter they cannot spell — `ṭuṭaba hṛdayaka` becomes `uaba hdayaka` — and one Sanskrit word in five loses a letter. Tiny writes the plain letter instead, so the word keeps its length and a search still reaches it. The smallest is also the fastest and, on this book, the best.

A recogniser that cannot spell a script at all writes plausible nonsense: the same models over a Russian document return Latin gibberish, and what keeps most of it out of the index is `chunking.legible` refusing to cut it.

## Where a document's own words sit

Recorded 2026-08-20 on the same machine, over the 546-page scan and its text layer.

| | |
| --- | --- |
| Reading the whole text | 9.5 s, 1 702 107 bytes over 546 pages |
| The words of one page | 239 ms |
| The words of three pages | 213 ms, 71 ms a page |
| The words of ten pages | 403 ms, 40 ms a page |

**Almost all of it is opening the document.** Ten pages cost twice what one does, not ten times, because the 233 MB file is read and parsed once and the pages after the first are tens of milliseconds each. That is what a cache of open documents is for, and it is why the pages wanted are asked for together.

The layer yields 572 boxes a page, 312 357 over the book, against the 51 169 a recognition of the same pages produced. The gap is punctuation: this layer puts a space before a comma, so 43 523 of the boxes are one byte standing alone in the text.

## What a line's boundary costs

Recorded 2026-08-20 over the same scan: the four headings of pages 31 and 32, and the body of twenty pages spread across the book, against the document's own text layer. Error is the mean share of characters wrong.

The detector answers with the text's own outline drawn inside the letters, and the amount it falls short is a share of the line's height. What widens it again was a fixed number of pixels, measured on the part after it has been scaled — on a heading that is about four pixels of the page against fifteen to twenty of shrink. The top of every capital and the last letter of every line were cut off.

| the line widened by | headings | body |
| --- | --- | --- |
| 10 pixels | 0.074 | 0.0644 |
| 14 | 0.031 | 0.0633 |
| 18 | 0.025 | 0.0639 |
| 20 | 0.017 | 0.0648 |
| 24 | 0.035 | 0.0667 |

**Eighteen**, the middle of where it stops mattering rather than the edge. The body is flat from ten to twenty and begins to pay after that.

```
"IAYADEVA GOSVAMI'S LIFEINNABADWI"         became "JAYADEVA GOSVAMI'S LIFE INNABADWIP"
"LAYADEVA GOSVAMI'S MARRIAGE TO PADMAVAT"  became "JAYADEVA GOSVAMI'S MARRIAGE TO PADMAVATI"
"THELORD HELPS IAYADEVA GOSVAMIWRITE…"     became "THE LORD HELPS JAYADEVA GOSVÁMI WRITE GITA GOVINDA"
```

Two other settings were measured and neither is worth moving. The heat a pixel carries to be part of a line gives the same boxes at 0.15, 0.20 and 0.30 — the map is as good as binary here. The longest side a part is read at costs small type when lowered and costs everything when raised, and 300 dpi beats 150, 200, 400 and 600 because detection is scaled to that side whatever the page was drawn at.

What no setting reaches: a word gap in a display face no wider than its letter gaps is one line to any threshold, so the space is the recogniser's own guess. And the alphabet the model carries has `ā ī ū ñ ś` and none of the letters with a dot under them, so `Lakṣmaṇa` and `Kṛṣṇa` cannot be written whatever the boxes are.

## A part too small to find a line in

Recorded 2026-08-20 over the same scan, pages 44 to 73 of the file, which print 14 to 43. The layout model finds the `number` region on every one of them — the number stands alone at the foot between two ornaments — and it is some fifty pixels across at 300 dpi.

Nothing is read in it. The detector draws a boundary around dark pixels and shrinks it by a share of the line's height; a line filling the image it is looked for in leaves no boundary to draw, so the region comes back blank.

| what was read | pages right of 30 |
| --- | --- |
| the region as it stands | 0 |
| the region magnified 3× to 20× | 0 |
| the crop widened to 150, 250, 400, 640, 960 pixels | 0 to 6 |

Widening the crop takes in the ornaments either side, and what they read as is not a number. Keeping only the lines whose middle falls in the region gets 30 of 30 at a margin of 150 pixels and above — and that margin is a pixel count against one book at one dpi, so it is not a default anything can carry.

So nothing reads it. A page is called where it stands in the file, which is known for every page, costs nothing, and is the number the pane in front of the person is showing.

The margin that works is the measurement worth keeping here: it says the failure is the detector needing background around a line, not the models being unable to read two digits.

## Drawing a page of a scan

Recorded 2026-08-20 over the same 546-page scan, five pages averaged, on an idle machine. The recognition run competes for the same cores and for the same pool of pdfium workers; under one, every number below is roughly doubled.

| width asked | pdfium | resample | jpeg | total | over the wire |
| --- | --- | --- | --- | --- | --- |
| 800 | 443 ms | 43 | 17 | **503 ms** | 234 kB |
| 1200 | 498 | 95 | 34 | **628** | 417 kB |
| 1600 | 553 | 163 | 52 | **768** | 625 kB |
| 2400 | 616 | 377 | 107 | **1101** | 1094 kB |

**Drawing barely moves with the resolution** — 104 dpi costs 443 ms and 311 dpi costs 616. So the cost is not rasterising: each page of this book holds one large photograph, and the library decodes the whole of it whatever size is asked for. Half a second a page is what this document costs, and no setting here reaches it.

**The resample was for a pixel or two.** The resolution is a whole number rounded up, so a page asked for at 800 comes back 804 across. Scaling those four pixels off cost between a tenth and four tenths of a second, and the window lays the page out at the width it asked for anyway. It is gone: the page goes as it was drawn, and the browser takes the four pixels off in the compositor.

That leaves the half second, and it is the same half second every time a page is turned back to. So the drawn pages are kept, in this machine's cache folder and not in the vault — a page turned back to is read from disk in about a millisecond, and a book read through once costs its half second a page and never again.

## Finding the section a question is about

Recorded 2026-08-20 over the same 546-page scan. 700 questions drawn from the book's own parts: for each of 350 parts whose heading the recogniser read as words, one question is that heading — a person asking where the book speaks about a thing — and one is seven words from the middle of the part. The right answer is that part either way.

| | right section | opened at its start |
| --- | --- | --- |
| the words half as it stands | 307/350 | 298/350 |
| the section's name in every chunk | 340/350 | 306/350 |
| the sections findable by name | 346/350 | 346/350 |
| both | 348/350 | 348/350 |

The questions quoting a phrase from the middle of a part moved by one either way — 331, 330, 331, 330 — so none of these costs anything on a question that names no section.

**The second column is the one that decided it.** Putting the name into every chunk lifts the right chapter but still answers with whatever paragraph of it ranks best; making a section findable by its own name answers with the section, at its heading.

What the words half was doing is visible in one query. Asked for `Madhavendra Puri`, BM25 scored the chapter that *is* about him at −12.74 and the subsections inside it between −17.8 and −19.7 — and lower is better. The chapter's opening says his name once, in its heading; a paragraph in the subsection after it says it four times.

**The name in every chunk was left.** It is worth two points on top of the other, and it changes the text a vector is made from: the recipe changes, and every chunk of every recognised document is embedded again. Only the words half was measured here — what a vector that knows its chapter is worth cannot be known without buying those vectors. The number to beat is 346 and 346.

## Putting a reading right

Recorded 2026-08-21 over the same 546-page scan, through a hosted model.

**The unit is the printed line.** The reply carries only the lines that changed, and what comes back is 31.2 % of the book when the page is asked as whole blocks, 14.3 % as sentences, and **3.8 % as the printed line**. A line of the recogniser's own is one run of words read in one go — 31 characters on average, 50 841 of them in this book — and about a fifth of blocks carry a misread word.

**Two models, over 58 pages spread through the book.**

| | pages the gates refused | output tokens | a book |
| --- | --- | --- | --- |
| `gemini-2.5-flash-lite` | 67 % | 25 005 per 10 pages | $0.14 |
| `gemini-2.5-flash` | 8.6 % | 6 919 per 20 pages | $0.29 |

The cheap model reports lines it did not change, which is where its output goes. A chain running it first pays $0.14 for a reply it throws away and $0.29 for the good model after it. The good model alone is $0.29, about 1 165 000 tokens in and 194 000 out. A refused page asked again answers the same.

**Where `max_edit_distance` comes from.** Over 931 corrections the distribution has a hole in it: 36 stand further apart than 0.50, 47 than 0.30, 49 than 0.20, 70 than 0.10. Above 0.30 every correction read was damage — text dragged in from the next line, or one corrected word in place of a whole line — and below it every one was a correction. The threshold is the hole and not a round number, and it is in the settings file because it was measured on one book.

**A reply row is written two ways.** Over one batch of 40 pages the model answered 15 pages with `2544|the line` and 25 with `2544 the line`, each page in one style throughout.

A queue is half the price of asking a page at a time and waiting.

Folding diacritics before embedding measures +0.08 to +0.11 of cosine for a person who types plainly. It is not general: it eats `q̇`, a vector arrow and a bar, which in a book of mathematics are the content.

## Changing how large the window is drawn

Recorded 2026-08-25 in Chrome 149, over a window holding a 400-line editor and three thousand rows of chrome.

| the multiplier changed | forced layout |
| --- | --- |
| `--numen-interface-scale` | 27–39 ms |
| `--numen-text-scale` | 4–6 ms |

The interface is the root's font size, so every length written in `rem` is measured again and the whole document is laid out. The reading size is written in `rem` as well, so the interface carries the editor and marked-up text along with everything else: what the reading size moves stands inside what the interface moves, and not beside it.

The editor draws only the lines that are on screen — thirty-six of the four hundred — so what it adds to either is a screen of text, whatever the note is worth. Marked-up text draws all of it. The same window with four hundred blocks of a note on screen, over the whole stylesheet rather than the tokens alone:

| the multiplier changed | with the note | without it |
| --- | --- | --- |
| `--numen-interface-scale` | 45–71 ms | 32–43 ms |
| `--numen-text-scale` | 9–13 ms | 7–11 ms |

Those two columns are read against each other. A page carrying the whole stylesheet costs more to recalculate than one carrying the tokens, which is why neither column meets the table above.

A held arrow key crosses a row of the size list every 40 ms, and a size is worn once the keyboard has stood on a row for 150 ms.

## What a name of the derived store costs

Recorded 2026-09-01 and re-measured 2026-09-06 on the same AMD Ryzen 7 6800U, from `BenchmarkDerived` in `internal/adapter/filesystem`. Every name the store takes is answered where the vault still is, and the check that it is asks the folder what identity it carries.

| | Before | After | 2026-09-06 | 2026-09-06, resolved once |
| --- | --- | --- | --- | --- |
| One name read | 92.2 µs · 5504 B · 57 allocs | 67.0 µs · 4584 B · 48 allocs | 64.9 µs · 6496 B · **76 allocs** | 36.4 µs · 3584 B · **41 allocs** |
| One folder listed | 79.6 µs · 4895 B · 60 allocs | 66.9 µs · 3989 B · 51 allocs | 51.9 µs · 5672 B · **71 allocs** | 20.8 µs · 2745 B · **36 allocs** |
| One line appended | 2.43 ms · 5334 B · 58 allocs | 2.24 ms · 4413 B · 49 allocs | 2.29 ms · 6400 B · **80 allocs** | 1.80 ms · 3355 B · **45 allocs** |

Each column is the median of three runs; the last two name the best of three on the clock, and the same allocations on all three. The allocation columns are the ones these are read on: a run of the read row spread by 10 % on the clock and not at all on the allocations. The last two columns were measured one after the other on a machine carrying a load average of 6.

**The allocations have gone past where the change above started, and the reason is a containment fix.** On 2026-09-04 a name was held to the area it says it is in, because a link written into an area — by a sync client, by another tool — pointed back at the vault's identity and a transcript appended down it landed on `config.json`. The check is right and the hole was real.

**What was not owed was doing it three times.** `DerivedStore.at` resolved symlinks on the target, then on the store's area, then on the store's own root. The last two are paths fixed for the life of the store, and `deepest` is a `filepath.EvalSymlinks` walk — one `lstat` per segment of an absolute path, every time. By allocation profile **60.5 % of one `Read` was inside `EvalSymlinks`**, which was the whole of the difference between 48 allocations and 76.

**The store's own folders are resolved when it is opened**, and the last column is what that is worth: a read is 41 allocations against 76 and 36.4 µs against 64.9, and a listing 36 against 71 and 20.8 µs against 51.9 — both below the row the containment fix started from. The check itself is unchanged, and a name still lands inside its area or is refused.

**A folder that moved is resolved again.** A name the resolved folders refuse is judged a second time against where those folders are now, so a service folder put there after the store was opened — a sync client keeping the application's files on another disk — is written into as it always was; and `still()`, where it reads the identity again, asks where the folders are with it. The refusal is what costs the second resolution, so nothing a person does twice pays for it.

This is a name a person pays per card answered and per run file read. A history screen over 180 run files pays it 180 times, twice over each.

**The identity was read out of the file at every name.** `config.json` opened, its bytes parsed and the identity in it checked, which is 19.9 µs and 12 allocations on its own — a fifth of a read and a quarter of a listing. It is now a stat of that file, and a file of the same length and the same age carries the identity already read out of it. A file that moved is read again, so a folder carrying another vault's identity is still refused at the first name after it arrives.

**An append is what a person answering a card pays**, one file of the log to a session, and it is unchanged in every way that shows: the `fsync` at the end of it is two milliseconds of the two and a quarter.

**A vault carrying no identity is stated by its folder.** A store opened on a folder that never held one has none to lose, and the check now asks whether that folder is there.

## What a window asks of one vault

Recorded 2026-09-01 and re-measured 2026-09-06 on the same AMD Ryzen 7 6800U, from `BenchmarkVault` in `usecase/flashcards`. The vault is generated: fifty thousand card faces over twenty decks, answered a hundred and fifty times a day for six months — 27 000 answers in 180 run files.

Each row is one request through its use case. The first three are warm, which is a person's second question of an evening: the schedule cache and the day counts are filled before the clock starts. The fourth is the history screen on a build that keeps no counting, which is what the first question of a launch pays.

| | Before | After | 2026-09-06 |
| --- | --- | --- | --- |
| The front door, one vault counted | 0.82 s · 605 MB | 0.85 s · 605 MB | 1.06 s · 617 MB |
| Starting a session | 1.25 s · 836 MB | 1.19 s · 793 MB | 1.46 s · 824 MB |
| The history screen | 0.71 s · 536 MB | 0.71 s · 493 MB | 0.83 s · 502 MB |
| The history screen, nothing counted yet | 0.78 s · 554 MB | 0.68 s · 490 MB | 0.82 s · 498 MB |

Every column is the median of two runs of five. The memory column is what a request allocates, which is the steadier of the two: the times of repeated runs of one build spread by about 5 %, and the memory by under 0.1 %.

**The four rows hold.** Read on the memory column, which is what these are read on, every one is between 1.7 % and 3.9 % of where the change above left it. The clock is 17 % to 23 % slower and the machine carried five other agents while it was taken, so the clock says nothing here that the memory does not say better.

**The front door is unchanged, and is here as the thing the others are read against.** It was already asking the schedule cache first, and nothing was taken off its path.

What was counted, per warm request:

| | Before | After |
| --- | --- | --- |
| Starting a session: the schedule cache read | 0 | 1 |
| Starting a session: the schedule cache written | 1 | 0 |
| Opening the history: the schedule cache read | 0 | 1 |
| Opening the history: the schedule cache written | 1 | 0 |
| The history screen, nothing counted yet: run files opened | 360 | 180 |

A session and the history screen each worked the whole log out again and wrote what it came to, whatever the cache held; both now ask it first. The history screen read the log twice — once for the days behind and once for the days ahead — and now reads it once and hands the reading on.

**A reading of the answers is put in order once.** De-duplicating the log and sorting it by when is what every question of a history begins with, and four of them were each doing it: where the answers leave each card face, how much came back, what a day spent, and how long an answer takes. A reading now carries that order, worked out at the first asking. Two of the four read it. The other two are `SpentUnder` and `Faced` in `usecase/flashcards/budget.go` and `usecase/flashcards/curve.go`, and `Costed` and `CostedUnder` in `flashcards/review/cost.go`, which still put the answers in order for themselves; each is a one-line change to the reading's order, and 27 000 answers sorted is what each of them costs.

**Nothing is shared between two requests.** Opening the window and then one preset tab walks the whole vault twice and reads the whole log twice: what a vault holds is read from its deck and stencil files at every request, and so are its answers. That is most of what the table above measures and none of what it changed.

By profile, over the front door on this vault: **57 % of the request is `ListCardFaces.Execute`** — every deck read and parsed, most of it in the markdown parser — and 7 % is reading the log. Working out the day's budgets is 8 %, of which the cost of an answer is 4 %. What a memo for the life of a window would take off a second request is those first two.

## The curve of one preset

Recorded 2026-09-01 on the same AMD Ryzen 7 6800U, from `BenchmarkPresetCurve` in `usecase/flashcards`. The vault is the one above — fifty thousand card faces over twenty decks, 27 000 answers in 180 run files — with its decks pointing at two presets: one deck points at the first and the other nineteen at the second. Each row is the curve of one of those presets, drawn under a goal of minutes, with the schedule cache filled before the clock starts: opening the window and then a preset tab.

| | Before | After |
| --- | --- | --- |
| A preset holding one deck of the twenty | 2.84 s · 1.21 GB | 2.07 s · 0.83 GB |
| A preset holding nineteen of them | 44.7 s · 13.31 GB | 42.4 s · 13.30 GB |

Both columns are the median of two runs of three. The memory column is the one these are read on: the times of the second row spread by a quarter between runs, and its memory by under 0.1 %.

**A curve reads the decks pointing at its preset.** The index answers what points at one note, so a preset of one deck opens one deck file and the other nineteen are never read. The second row is the same request where there is nothing to leave out — nineteen decks of twenty — and it stands here as what the first is read against.

**What the projection costs stands on the card faces the preset holds.** Twenty-five places of the grid, each of them a run of the scheduler, and a second run at each place for the day the material is learned. It is the whole of the second row and most of the first, and it is what the section below takes off.

**A curve does not ask the schedule cache.** That cache is filed under the assignment a whole vault stands at, and a curve holds the card faces of one preset, so the answers are replayed for the faces it is drawn over. Reading the log is on this path in any case: what an answer costs and what the day has already spent are read from the answers themselves, cache or no cache.

**The curve of the defaults reads every deck.** Nothing points at a preset that stands in no note, so which decks name none is a question only the deck files answer, and that one curve pays what the front door pays.

## What a curve costs as the preset grows

Recorded 2026-09-01 on the same AMD Ryzen 7 6800U, from `BenchmarkCurveCards` in `usecase/flashcards`. Twenty decks with every one of them pointing at one preset, so the curve is drawn over the whole vault; the log grows with the vault at about half an answer a card face, in 180 run files. The schedule cache is filled before the clock starts.

| card faces | Before | After |
| --- | --- | --- |
| 500 | 0.42 s · 158 MB | 0.31 s · 145 MB |
| 5 000 | 3.90 s · 1.43 GB | 2.67 s · 1.29 GB |
| 20 000 | 15.5 s · 5.51 GB | 9.9 s · 4.86 GB |
| 50 000 | 40.4 s · 14.08 GB | 23.4 s · 11.68 GB |

Both columns are the median of two runs of three on an idle machine, where the two runs of a row landed within 7 % of each other on the clock and within 0.01 % on the memory. The memory column is the one these are read on: on a vault where a change cannot help, a row's time moves by a quarter between runs while its memory moves by tenths of a per cent. The last row is 105.1 million allocations before and 92.2 million after.

**A curve is fifty-one runs of the scheduler.** One at each of twenty-five places of the grid, one more at each place for the day the material is learned, and one at the top of the range to find how far the range reaches. Each of them walks ninety days, and each of those days used to go over every card face of the preset four times: to find what was due, to count what stood learned, to count what the day left behind, and to count what comes back. At fifty thousand card faces that is nine hundred million card visits in one request.

**Three of the four are gone.** A card face is filed under the day of review its schedule falls in, so a day takes what fell in it: what the day before did not reach and what falls due in this one, put into one run in the order the debt fell, and the material behind them is never looked at. What the day left standing is then what fell due in it less what it reached, which is a subtraction. What stands learned under a rule of an interval is a fact about a card face's schedule, so the count is carried from day to day and asked again only of the card faces the day answered; under a rule of a chance of recall it is a fact about the instant, and is read off the same number as the share that comes back.

**The fourth stands, and is 32 % of the request** at twenty thousand card faces by profile. The share of the material that comes back is a fact about every card face at every instant, so it is the one thing a day still counts over the whole preset. A curve reads one day of that series out of each run and the run fills ninety; filling only the days a caller asks for is a change to what a projection promises rather than a faster way to keep it, and it is the section below.

**Both endings of an answer come from one reckoning of the card.** A projection weighs the ending where the card came back against the ending where it did not, and asked the scheduler for each of them separately. A card face the scheduler has put into review is settled at every rating in one working out, so the two are asked for together and the second costs nothing. A card face it is still putting into memory is settled a rating at a time and is asked a rating at a time: asking for all four there costs more than it saves, and measured 5.40 GB against 4.86 GB at twenty thousand card faces.

**What is left is the scheduler's own arithmetic.** `Simulation.Run` is 97 % of the request, and half of that is the answers themselves. Nine tenths of what a request allocates is allocated inside that call by the scheduling library, which builds a table of every rating for every card it is asked about.

**What does not change over the twenty-five places was left where it stands.** The order the card faces are in, how loaded each day already is, how many have had their day and were not answered on it, and how many stand learned as the run opens are worked out once a run and could be worked out once a curve. They are 2.7 % of the request together, and a reading handed in from outside is a second path to an answer there is one path to.

## What a projection is asked to answer for

Recorded 2026-09-01 on the same AMD Ryzen 7 6800U, from `BenchmarkCurveCards` in `usecase/flashcards`, over the vault of the section above and with the same schedule cache filled before the clock starts.

| card faces | Before | After |
| --- | --- | --- |
| 500 | 0.29 s · 145 MB | 0.23 s · 145 MB |
| 5 000 | 2.59 s · 1.29 GB | 2.30 s · 1.29 GB |
| 20 000 | 9.5 s · 4.86 GB | 6.6 s · 4.86 GB |
| 50 000 | 24.4 s · 11.68 GB | 15.5 s · 11.68 GB |

Both columns are the median of two runs of three on an idle machine. **The clock is the column these are read on**, which is the other way round from every row above: what is taken off is arithmetic over a slice that is already there, so it allocates nothing, and the memory of every row moved by under a hundredth of a per cent — 92.17 million allocations at fifty thousand card faces either way.

The two runs of a row landed within 8 % of each other on the clock, and the last two rows within 1 %. Those two are what this table rests on; the 5 000 row's two runs after the change are a fifth apart, and it is the row that says least.

**A projection is asked which days it must answer for.** A run is given the days whose returning share it works out, and the projection holds a share under those days and under no other. A day nobody named is answered as a day the run does not answer for, and there is no number to read off it.

**A curve names one day a run**, which is the day the place is read on: the last day of the horizon under a goal of minutes or of retention, and the day the place stands for under a goal of a date. The twenty-five second runs that ask when the material is learned name no day at all. Fifty-one runs of ninety days were filling four and a half thousand days of it; they now fill twenty-five.

**What the days that are asked for come to is unmoved.** The projections written down over four materials and eight settings — every scalar and every series, day by day — are byte-identical, their runs asking for every day of themselves.

**Where a card face counts as learned by a chance of recall, the walk stands.** That count is a fact about the instant and is read off the same numbers as the share, so such a day goes over the whole material whether the share was asked for or not. The saving is a preset counting by an interval, where the count is carried from day to day and the walk is made only on a day that was named.

**What is left is the scheduler's own arithmetic.** By profile at twenty thousand card faces, `Simulation.Run` is 96 % of the request and the reckoning that closes a day is 0.2 % of it. That is the pass that was 32 %.

## What the scheduler is asked to work out

Recorded 2026-09-01 on the same AMD Ryzen 7 6800U, from `BenchmarkCurveCards` in `usecase/flashcards`, over the vault of the section above and with the same schedule cache filled before the clock starts.

| card faces | Before | After |
| --- | --- | --- |
| 500 | 0.27 s · 145 MB | 0.12 s · 19.6 MB |
| 5 000 | 2.17 s · 1.29 GB | 1.01 s · 150 MB |
| 20 000 | 7.4 s · 4.86 GB | 3.5 s · 592 MB |
| 50 000 | 17.6 s · 11.68 GB | 8.8 s · 1.52 GB |

Both columns are the median of two runs of three on an idle machine. **The allocation is the column these are read on.** It is what the change takes away, and it is steady: the two runs of a row are within 0.05 % of each other everywhere. The clock halves at every size, which is far outside the spread of the runs behind it — 3 % or less at every size but 5 000, whose two runs after the change are a sixth apart. **That row says nothing on the clock** and is here for its memory.

**The Before column is this machine on this day.** Its 50 000 row reads 17.6 s where the section above wrote 15.5 s, on the same build and the same allocation to a hundredth of a per cent. It is what the After beside it is read against and not a figure to carry anywhere else.

**A card is worked out, not tabulated.** The scheduling library settles a card by building a scheduler, a map, and an entry for each of the four ratings, and formats a string for a fuzz seed on every call — these parameters carry no fuzz and nothing reads the seed. A projection asks about two ratings. The arithmetic now stands beside the port, on the library's own weights: at fifty thousand card faces a request allocated 92.2 million times and now allocates 3.3 million.

**87 % was the estimate and 87 % is the measurement.** A profile before the change put 87 % of every byte a request allocates inside the library. The four rows came to 86.5, 88.3, 87.8 and 87.0 per cent.

**Good is worked out from hard.** The interval a good answer names is held a day past the one a hard answer names, so the two endings a projection weighs cost the hard ending's stability as well. Easy is worked out only where an easy answer is asked for.

**A day count does not go below none.** The days a card face stood away are the whole days gone by. A card face asked about at an instant no later than the answer it already holds stood none of them, and the forgetting curve is read at none.

That is the one place the arithmetic here parts from the library, which counts those days into a whole number carrying no sign and reads such a card face as one nobody could recall. What that count comes to is the machine's, so the same card face read one way on one architecture and the other way on another.

**No schedule a vault can hold moves with it.** A due day is always worked out forward from the instant of an answer, so a card face is never asked about before its own last answer: over a curve of five thousand card faces the days away were read 602 102 times and not once fell below none, and the same vault with twenty answers timestamped a week ahead by a wrong clock reads the same 0. The state needs a due day standing behind the answer that produced it, which is what the written-down projections build by hand — 96 of their 10 523 readings, 36 of them where the phase makes it count.

**What is left is arithmetic.** By profile at twenty thousand card faces, `Simulation.Run` is 77 % of the request, the answers themselves are 49 %, and the scheduler is 30 %. `math.Pow` and `math.Exp` together are 22 %, and choosing the day a card lands on is 15 %. Reading the vault, which was 4 %, is now 7 % of a shorter request.

## Every place of a curve at once

Recorded 2026-09-01 on the same AMD Ryzen 7 6800U, which has eight cores and sixteen threads, from `BenchmarkCurveCards` in `usecase/flashcards`, over the vault of the section above and with the same schedule cache filled before the clock starts.

| card faces | Before | After |
| --- | --- | --- |
| 500 | 0.12 s · 19.6 MB | 0.08 s · 19.7 MB |
| 5 000 | 1.01 s · 150 MB | 0.29 s · 150 MB |
| 20 000 | 3.5 s · 592 MB | 1.06 s · 592 MB |
| 50 000 | 8.8 s · 1.52 GB | 2.5 s · 1.52 GB |

Both columns are the median of two runs of three on an idle machine. **The clock is the column these are read on, and the allocation is the control.** The same work on other cores allocates the same bytes, and three of the four rows moved by under 0.05 %; a memory figure that had moved would have said something changed that was not meant to. The 500 row's allocation is 0.4 % higher, which is the stacks of the goroutines a request of eighty milliseconds now starts.

**The 500 row says least.** Its two runs after the change are a sixth apart, and what it is here for is that the smallest preset did not get slower, which was the thing worth checking. The factor beside it is not a figure to quote.

**Three and a half times, on eight cores.** The estimate before the measurement was 2.8, from the 77 % of a request that the runs are and eight cores to carry it. The measurement is 3.4 at twenty thousand card faces and 3.5 at fifty thousand, so the threads past the eight cores are worth something to arithmetic that waits for no memory.

**Twenty-five places over sixteen threads is two waves, and the second is smaller.** A curve is twenty-five places and each is two runs of the scheduler, handed out a thread at a time. That unevenness, and the reading of the vault that no place shares, are most of what stands between three and a half and eight.

**A place holds the preset's cards while it runs.** Peak memory at fifty thousand card faces is 164 MB with one place running at a time and 307 MB with all of them, sampled from the kernel's own high-water mark over the whole benchmark. A run holds about four and a half megabytes of cards at that size, and a request now holds as many of those as the machine has threads.

**What a person waits for.** A preset tab over fifty thousand card faces answers in between two and a half and three seconds: four runs of three on this machine spread from 2.47 s to 2.99 s, and allocated the same 1.52 GB to within 0.02 % every time. The work is the same work and the spread is the clock, so three seconds is the figure to hold this to. It was 8.8 s before this change, 17.6 s before the one above it, and 40.4 s at the top of the section three above. Twenty thousand card faces is a second, and five thousand is under a third of one.

## What a window asks of every vault

Recorded 2026-09-01 on the same AMD Ryzen 7 6800U, from `BenchmarkFrontDoor` in `adapter/window/flashcards`. The installation is generated: four vaults, each of five thousand card faces over twenty decks, answered a hundred times a day for sixty days — 6 000 answers in 60 run files a vault. The schedule caches are filled before the clock starts, which is a person's second opening of a day.

| | Measured 2026-09-01 | Measured 2026-09-06 |
| --- | --- | --- |
| Every vault counted inside one answer — what the window opened on before | 0.52 s | 0.52 s (6, 0.52–0.66) |
| The list of vaults on screen | 1.3 ms | 1.0 ms (6, 1.0–3.3) |
| The last of the four counts landing | 0.21 s | 0.20 s (6, 0.20–0.29) |

The first column is the median of two runs of five. The second is six runs of five taken on a machine carrying five other agents' work, quoted at the best of the six with the spread beside it.

**Nothing here has moved.** Every row's best run lands on the figure recorded, and the whole of the spread above it is the machine: the list on screen varies threefold between runs because it is a millisecond of work, and the two counts vary by a quarter. Read the three together — they hold.

The first row is the old front door, run here as it stood: the four vaults counted one after another before anything was handed over.

**What a person waits for is a reading of the registry.** The list is names and paths, which the registry answers before any database is opened, so it is on screen in about a millisecond whatever the vaults hold. Everything after that arrives at its own row.

**Four counts at once cost the slowest, not the sum.** The four vaults here are the same size, so the last count lands in about a quarter of what counting them in turn took, less what they spend competing for the same cores. An installation of one vault waits exactly as long as it did.

Nothing here says what a vault of fifty thousand card faces does to the row beside it: these four are alike on purpose, and the claim being measured is the shape and not the spread.

## What a window costs before a person can type

Recorded 2026-09-06 on the same AMD Ryzen 7 6800U, over the flashcards window's own start: the settings read, the registry opened, the index opened, the use cases assembled, the themes read, the pages handler built, and the first page fetched from it. Three runs of each part; the spread is quoted because the machine carried other work throughout.

**A window is not measured here, and cannot be.** What a person waits for is the process starting, the Go side assembling, WebKit creating a webview, and the bundle being parsed and run — and the three that are not Go need a window on a screen. What is below is the Go side alone, which is the part a change to the composition root could move.

| | Measured | Allocations |
| --- | --- | --- |
| The settings read | 0.13 ms (0.13–0.15) | 14.9 kB · 164 |
| The registry opened | 50 ns (49–63) | 24 B · 1 |
| The index opened, no file there yet | 13.7 ms (13.7–16.4) | 64 kB · 608 |
| The index opened, the file already there | 1.07 ms (1.07–1.32) | 36 kB · 164 |
| The themes read | 13.0 µs (13.0–18.3) | 3.0 kB · 20 |
| The pages handler built | 0.55 µs (0.55–0.96) | 120 B · 6 |
| The use cases assembled | 0.26 ms (0.26–0.44) | 19 kB · 270 |
| The first page served | 0.73 ms (0.73–1.13) | 24 kB · 138 |
| **All of it, first launch on a new machine** | **31.9 ms (31.9–41.0)** | 236 kB · 2 553 |

**The Go side is thirty milliseconds and the schema is most of it.** Opening an index that is not there yet runs the migrations, and that is 14 ms of the 32; on every launch after the first it is one millisecond. Assembling every use case the window serves is a quarter of a millisecond, and building the handler that serves the pages is half a microsecond. Nothing the night's moves touched is measurable from here.

**So what a person waits for is not this.** A launch on an existing installation spends under 20 ms in Go before the first byte of the page. The rest of the wait is the webview and the bundle the section below measures, and neither is reached by anything in `container`.

The harness for this is not in the tree. It was taken with a benchmark written against `container` and the flashcards window package and then removed, so the table cannot be reproduced from a checkout as it stands. Every path it named — the index, the registry, the settings, the themes, the schedules — was pointed at a temporary folder, so nothing on the machine was read.

## What the two windows are built into

Recorded 2026-09-06, `vite build` in each window's own folder, into the Go package that embeds it. The figure is the one file the window loads.

| | JavaScript | gzip | CSS | modules |
| --- | --- | --- | --- | --- |
| flashcards | 507 kB | 176 kB | 100 kB | 726 |
| editor | 1 631 kB | 562 kB | 111 kB | 2 601 |

**The flashcards window's fall held.** It was 1 180 kB before `preserveModules` and `sideEffects` landed together in the interface library and 506 kB after; it builds at 507 kB today, a kilobyte of the day's own work.

**The editor did not move, as it was expected not to.** It was 1 628 kB and is 1 631 kB. It draws more of the library and more of its own, and the tree shaking that took two thirds off the smaller window takes almost nothing off this one: what the editor imports, it uses.

The editor's `vue-tsc` step does not pass in this tree, so its figure comes from `vite build` alone. Two copies of `@vue/runtime-core` are installed — one under the editor and one under the interface library — and every component's props typecheck against the wrong one. That is a state of `node_modules` and not of the source; the bundle is unaffected, since Vite resolves one copy.

### The same two, off one install

Recorded 2026-09-06, the same command, with `modules/` an npm workspace root.

| | JavaScript | gzip | CSS | modules |
| --- | --- | --- | --- | --- |
| flashcards | 432 kB | 155 kB | 101 kB | 571 |
| editor | 1 556 kB | 543 kB | 112 kB | 2 476 |

**Each window carried the protobuf runtime twice and now carries it once.** Eleven installs meant `@numen/protocol` resolved `@bufbuild/protobuf` out of its own `node_modules` and the window resolved another out of its own, so both were bundled: `google/protobuf/descriptor.proto` appears once in each bundle now and appeared twice in each before. It is 76 kB off both windows, which is the whole of the fall — nothing else about either window changed.

The two copies of `@vue/runtime-core` went the same way, and the editor's `vue-tsc` passes.

## A recipe that moved, and whether it settles

The quantisation scale now stands in the recipe a vector is kept under, so every vector made before it is owed again. Read 2026-09-06, from the code rather than the clock.

**It settles.** `port.EmbeddingModel.Recipe` is a `Sprintf` over the model's five fields, the name of the quantisation and `port.Int8Scale`. The five fields come from the settings file; the other two are constants. Nothing in it is read from the clock, from a random source, or from a path that differs between launches, so two runs of one installation ask for the same recipe and the second finds every vector the first bought. The cost is once, and `TestAScaleThatMovedBuysTheVectorsAgain` is what holds it there.

**What it does not settle is the space.** `vectors` is keyed by the text's hash and the recipe together, and `forget_vector` takes a row out only where no chunk holds that text at all. A chunk whose text did not change keeps its old-recipe row for ever beside its new one, so a scale that moves doubles what the vectors cost on disk and leaves it doubled. At the corpus measured under "Dimensions cost more than bits" that is 149 MB become 298 MB. Nothing sweeps the rows of a recipe no longer in force.
