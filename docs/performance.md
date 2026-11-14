# Performance

A target is a decision. Every target of this application is in the Target column below and nowhere else. Each measurement has the date and the machine it was taken on, and is stored here beside its target.

The load test reports numbers and does not assert on them. It runs on any laptop, and the numbers are recorded here so that a person can compare them with earlier ones. A target is set at roughly twice the measured value, so that a regression is noticed before a user notices it.

## Running it

```
cd libs/core/kernel
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
NUMEN_EPUB_CORPUS=<a folder of EPUB files> go test ./epub/ -run XXX -bench 'ReadingABook|DrawingOneDocument|CountingThePages' -benchtime 5x -count 2
```

The bundle each window is built into, from the repository root:

```
npm run build --prefix apps/desktop/flashcards
npm run build --prefix apps/desktop/editor
```

The vault is generated, not downloaded: `testsupport.GenerateVault` writes notes of varying length across fifty folders, each linking to a parent and to a few other notes, from a fixed seed.

Only the cold scan is pinned to a single iteration — it takes seconds, and repeating it adds no information. The rest run many times: a single sample of a millisecond-scale benchmark varies by tens of percent, and the link benchmarks read a single-figure answer out of a large table, where one run measures only the page cache.

## At the size this is designed for

A hundred thousand notes, from the load test. Measured 2026-09-06, one run, on a machine that was also running the work of five other agents; the 2026-09-01 column is the result of the earlier run.

|                                                       | 2026-09-01                        | Measured                                 | Target          |
| ----------------------------------------------------- | --------------------------------- | ---------------------------------------- | --------------- |
| Cold scan                                             | 1 m 22 s (0.82 ms/note)           | **8 m 52 s (5.32 ms/note, 188 notes/s)** | under 3 minutes |
| Warm scan — what a startup pays                       | 0.52 s                            | **1.07 s**                               | under 1 second  |
| An edit the watcher names, until a client is notified | 66 ms                             | not re-measured                          | under 100 ms    |
| Index size                                            | 162 MB                            | **344 MB (3.4 MB per thousand notes)**   | not decided     |
| Search, rare term, under load                         | p50 2 · p95 4 · p99 5 · max 42 ms | p50 3 · p95 6 · p99 8 · max 102 ms       | p95 under 50 ms |
| Search, term matching every note                      | p50 0.90 s · p95 1.01 s           | p50 3.37 s · p95 4.25 s                  | not covered     |

**Two of these targets are now missed, and the index size explains it.** It has gone from 162 MB to 344 MB over the same 164 MB of markdown, and that figure does not depend on a busy machine. A note is now written into the passage index as well as the note index — the chunk enclosing it, the small chunks tiled inside it, each of them indexed for its words in `chunks_fts` and again in `sections_fts` — so a scan writes about twice as much, and a query that matches everything ranks several rows per note.

**What a person's own search costs has held.** The rare-term row is 3 ms at the median and 6 at the ninety-fifth, well inside a target of fifty, against 2 and 4 on a less loaded machine. The read side did not change; the write side and the size did.

Both search rows come from four concurrent readers running against a scan that rewrote the whole vault continuously. The difference between them is not the database: one query matches a hundred notes and the other matches all hundred thousand, and ranking a hundred thousand matches is linear work. Real queries look like the first row; a vault generated from twenty words produces only the second, which is why the load test runs both queries.

The vault itself is 164 MB of markdown, so the index is now about twice the size of the text it indexes.

## Baseline

Re-measured 2026-09-06 on the same AMD Ryzen 7 6800U, `modernc.org/sqlite`, WAL with `synchronous = NORMAL`. The 2026-08-15 column is kept beside it because the two are not the same measurement in every row, and the rows below say which.

**The machine was shared with five other agents throughout, and that is the largest source of error in the two 2026-09-06 columns.** The one-minute load average ran between 4 and 61 over the session, against sixteen threads. The noise floor is read off `BenchmarkChunkIdentity`, which is arithmetic and does no I/O: the two conversions recorded at 25 ns and 23 ns on 2026-09-05 were 29–37 ns and 26–31 ns over ten runs here, so **a figure that does no disk I/O takes about 1.3× the recorded time on this machine today**. Everything that uses the disk is further from the record than that, and no row below is more precise than its spread.

Each cell is the best of the runs stated, with the spread beside it. The best is quoted rather than the median because every source of noise here is additive.

|                                              | 2026-08-15        | 1 000 notes                | 10 000 notes           |
| -------------------------------------------- | ----------------- | -------------------------- | ---------------------- |
| Cold scan — every note read, parsed, written | 0.46 s · 5.9 s    | 1.80 s (6 runs, 1.80–2.84) | 20.1 s (6, 20.1–39.7)  |
| The same, with the statements kept compiled  |                   | 1.19 s (6, 1.19–1.41)      | 15.2 s (6, 15.2–17.4)  |
| Warm scan — nothing changed, no file opened  | 4.7 ms · 50 ms    | 9.2 ms (3, 9.2–15.3)       | 58 ms (3, 58–122)      |
| Incremental — one note edited                | 8.0 ms · 55 ms    | 12.4 ms (3, 12.4–17.6)     | 64 ms (3, 64–136)      |
| Resolving one note's links                   | 0.18 ms · 0.18 ms | 0.64 ms (3, 0.64–0.68)     | 0.43 ms (3, 0.43–0.61) |
| Backlinks of one note                        | 0.88 ms · 1.11 ms | 2.94 ms (3, 2.94–3.28)     | 3.18 ms (3, 3.18–4.63) |

Neither link figure grows with the vault. That is what those two rows show: the columns differ ten times in size and by a fraction of a millisecond in time, as before.

The second row was measured in a less busy hour of the same day, when `BenchmarkChunkIdentity` gave 22.2–27.6 ns against the recorded 23 ns; it is the first row's benchmark after the two changes set out under "Where the time goes". A run of the first row in that same hour gave 2.20 s and 24.7 s when the machine got busy again, so compare 20.1 s with 15.2 s; the profile counts are the more reliable figure.

**The cold scan takes three to four times the recorded time here, and six and a half times at a hundred thousand notes.** A note now goes into the passage index as its own chunks, so `chunk.Replace` runs inside `saveNote`; by profile it is 42 % of the scan, and the index it fills is twice the size it was. The rest is due to the machine and to one repeated pass, both described below.

**The two search rows of 2026-08-15 cannot be reproduced, and are not kept in this record.** They measured `Queries.Search` and `search.sql`, which read `chunks_fts` and grouped the hits back to notes. That statement was deleted on 2026-09-05 as dead production code, and the benchmark now asks `port.LexicalQueries.Lexical` — the query the application has always used for search. It is a heavier plan: a join to `sources`, the folder filter through `json_each` behind a bloom filter, a left join to the enclosing chunk, and a temp B-tree for the ordering. Against the generated vault's twenty-word vocabulary the query matches every note, so what it measures is ranking the whole corpus.

|                                               | Measured 2026-09-06         |
| --------------------------------------------- | --------------------------- |
| `Lexical`, a term matching every note, 1 000  | 11.8 ms (3 runs, 11.8–29.6) |
| `Lexical`, a term matching every note, 10 000 | 101 ms (3, 101–155)         |
| The same while a scan is writing, 10 000      | 149 ms (3, 149–641)         |

Compare those with the "term matching every note" row of the load test, not with a search figure. What a person's own search costs is the rare-term row further up, which the load test measures and this benchmark cannot: a vault built from twenty words has no rare terms but the one that was planted.

A search still costs more while a scan is continuously rewriting the index. WAL lets a reader read without waiting for the writer, and the write pool is capped at one connection, so writers queue in Go.

## What these numbers are not

**The search figure is pessimistic.** Generated notes are built from a twenty-word vocabulary, so a two-term query matches nearly every note and the ranking has to order all of them. A real vault has a long tail of terms and a query that matches a handful of notes.

**The incremental figure is a full walk**: one edited note, found by checking the state of every file in the vault. That is the cost of a scan at startup, and it is not what an edit costs while the application is running — the watcher names the path and only that path is read. The 100 ms target is written for the watcher's path, and that path is measured further down, from the save to a client being notified.

**The cold scan reads notes this same process wrote seconds earlier**, so the read side is measured against a warm page cache. On a real vault that has been sitting on disk, this is optimistic about I/O.

**Nothing here measures a cold start of the application**, only of the scan. What the start costs on the Go side is measured under "What a window costs before a person can type".

**A flashcard benchmark runs on an index that does not flush.** It shares its setup with the tests around it, and what it times is scheduling arithmetic. The vault benchmarks and the load test use the index the application ships with, so the two sets of numbers are not comparable.

## Where the time goes

A cold scan, by profile, taken 2026-09-06 over one run of `ColdScan/10000` — 34.7 s of wall time, 37.6 s of samples over sixteen threads. Reading the files is about 1 % and parsing the markdown a few percent, as before; what has changed is inside the writing.

|                                                | share of the scan | with the statements kept compiled |
| ---------------------------------------------- | ----------------- | --------------------------------- |
| `Scan.Execute`                                 | 84 %              | 82 %                              |
| ↳ `note.Repository.Save` → `saveNote`          | 78 %              | 74 %                              |
| ↳ `chunk.Replace` — the note as its own chunks | **42 %**          | 42 %                              |
| SQLite parsing SQL text (`sqlite3RunParser`)   | **20 %**          | **nothing the profile shows**     |

The second column is the same benchmark after the two changes below, and one run of it costs 20.7 s of samples where it cost 32.9 s.

**Ending transactions is no longer the largest part.** The 2026-08-15 profile read 1 % reading, 5 % parsing, 77 % storing, of which 43 % of the whole scan was ending transactions. Since then a note is written into the passage index as well as the note index — the chunk enclosing it, the small chunks tiled inside it, and every one of them indexed for its words in `chunks_fts` and `sections_fts`. That path is 42 % of the scan, and it is the reason the cold scan is slower now.

**A fifth of a cold scan was SQLite parsing SQL text it had already parsed.** `chunk.prepare` prepared four statements and closed them inside `chunk.Replace`, which `saveNote` calls once per note; `insert_link` was prepared per note beside it; and the dozen or so `exec` helpers go through `tx.ExecContext`, which the driver prepares afresh on every call. The transaction spans a group of five hundred notes, so ten thousand notes parsed the same statement texts something like a hundred and fifty thousand times.

**A statement text that ends in a newline is compiled twice on every call.** The driver compiles the text when the statement is created. It keeps the compiled statement only if the text holds one statement and nothing after it. A text with anything after the semicolon is treated as a script, and a script is compiled again in every `Exec` and `Query`. Every statement of this index is stored in a file, and a file ends in a newline. Trimming the whitespace from what `sqlfile` loads removes **half of the parsing**: 22.5 % of the profile becomes 11.5 %, and the samples of one run of `ColdScan/10000` fall from 32.9 s to 21.6 s.

**A transaction now keeps the statements it prepared.** `writing.Transaction` prepares each text the first time the transaction runs it and runs the prepared statement after that, so a group of five hundred notes compiles its two dozen statements once for the whole group. That is the other half: 11.5 % of the profile disappears, and one run costs 20.7 s of samples where it cost 21.6 s. The reuse alone, measured against the record and without the text change above, was 22.5 % of parsing become 17.9 % and 32.9 s of samples become 23.9 s — the two changes remove the same waste from two sides, and both are needed to remove it.

The earlier record says that reusing prepared statements across a group makes no difference. That is listed under "What does not work" and was measured on 2026-08-15, when a note save was a handful of statements and `chunk.Replace` was not on the path.

**Read the shares here and not the clock.** The machine was also running five other agents, with a load average between 6 and 43, while these were measured, so the wall time of one scan depends more on the machine load than on the code. A profile counts only this process's own samples, and the paragraphs above compare the share of samples of each function.

A transaction boundary costs the same whether one note or five hundred notes are written in it, which is why notes are written in groups. A cgo build of SQLite is roughly twice as fast on inserts in published comparisons; it is not measured here, and it would give up the cross-compilation for which the pure-Go driver is chosen.

A warm scan of ten thousand notes is 50 ms: 13 ms reading the stored state of every file from the index, and 34 ms walking the vault and querying the file system. The walk stats one file per note and nothing else — the extension is checked from the directory entry, before anything is opened — so that half is the cost of visiting every file of a vault. The watcher removes this cost between scans; a faster walk would not.

## Looking a vault over

`BenchmarkRun` in `check`, on a vault where every name is used by many notes and each link is written in one note — the worst case for the ambiguity check, because every link in it is a candidate.

|                                             | Measured |
| ------------------------------------------- | -------- |
| Every check that runs unasked, 1 000 notes  | 2.1 ms   |
| Every check that runs unasked, 10 000 notes | 23 ms    |
| Dangling links alone, 10 000 notes          | 53 ms    |

Two of the checks read rows a scan already stored. The other two compute over the whole vault when they are called, which is why their results are not stored: a result becomes wrong when a note elsewhere is moved.

The number to watch is the second one, because it is the one that grows with the vault rather than with the number of duplicate names. A real vault has few shared names, so its ambiguity check does almost nothing; this vault shows the cost when every name is a duplicate.

## What does not work

Measured, on ten thousand notes written in groups of five hundred.

|                                                                         |                                                                                 |
| ----------------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| Turning off the full-text index's background merging during a bulk load | 1.7× slower                                                                     |
| Larger full-text page size (`pgsz = 8000`)                              | 1.7× slower                                                                     |
| Merge thresholds either way (`automerge` 8 and 16, `crisismerge` 8)     | no difference                                                                   |
| Page cache of 4 MB, 64 MB, 256 MB                                       | no difference                                                                   |
| Reusing prepared statements across a group                              | no difference on 2026-08-15; **a third of a cold scan's samples on 2026-09-06** |
| Groups of five thousand                                                 | 7 % faster than five hundred, for ten times the memory                          |
| `synchronous = OFF`                                                     | 13 % faster, and the file can be corrupt rather than merely stale               |

The first two are the advice most search results give for slow SQLite inserts.

The fifth row was measured when a note save was a handful of statements. A note save now runs `chunk.Replace`, and a transaction spanning five hundred notes ran the same two dozen statements five hundred times over — a fifth of a cold scan, measured 2026-09-06 and set out under "Where the time goes", where the gain from removing it is also given.

## Two things that were invisible in review

**Deleting a full-text row by its columns scans the whole index.** An FTS5 table has no key but its rowid, so a rebuild that selects the row any other way is quadratic in the number of notes. A test asserts the statements address it by rowid.

**A database without statistics uses default estimates.** SQLite chooses between indexes from statistics about the stored data, and a database filled by a scan has none. The default plan for backlinks narrows to the vault and reads every link in it. A scan collects the statistics after it changed the index, and backlinks are then returned in about a millisecond and do not grow with the vault. A test asserts the index each query uses — a test that only forbids reading a whole table passes on the slow plan, because reading every link in a vault is an index search.

## The source layer: finding a passage

The numbers above are the note index. These are the source layer: text cut into chunks, embedded, and searched.

Recorded 2026-08-17 on the same AMD Ryzen 7 6800U, `modernc.org/sqlite` v1.56 with the bundled `sqlite-vec` v0.1.9, WAL with `synchronous = NORMAL`.

**The harness is not in the repository yet.** These were taken with standalone programs; moving them here is outstanding work, and until it is done the numbers below cannot be reproduced from a checkout.

### The corpus

Not generated. The Mahābhārata in three languages: the Ganguli prose translation, the BORI critical edition in transliterated Sanskrit, the Russian academic translation, and four smaller sources — about 65 million characters of digital text once the scans whose recognition is wrong and the duplicate editions are dropped.

Embedded with `bge-m3` at 1024 dimensions, through a hosted API.

|                             |            |
| --------------------------- | ---------- |
| Windows read from           | 37 380     |
| Windows searched            | 145 800    |
| Embedding, the whole corpus | 10 minutes |
| Index                       | 613 MB     |
| Search                      | 32 ms      |

The same corpus embedded on this laptop's CPU instead, with a small English-only model, runs at 10.8 chunks per second.

Cut this way, the sources total about **2 240 small chunks per megabyte of text**.

### The chunk decides what can be found

One passage, one query, four chunks cut around the same sentence. The score is against a query that paraphrases the sentence in another language.

| Size      | Similarity |
| --------- | ---------- |
| 25 words  | 0.751      |
| 50 words  | 0.648      |
| 100 words | 0.544      |
| 200 words | 0.517      |

The best score anything in the corpus reaches for that query is 0.639. At 25 words the passage would lead by a wide margin; at 200 it ranks **401st of 36 560**, which no user would see.

This is the measurement the chunk sizes are chosen against. Nothing about the index changed between those rows.

The chunk also has to fit the model. Cut on the sections the translation already contains, in words:

| Size      | Chunks | Median tokens | Over the 512-token limit |
| --------- | ------ | ------------- | ------------------------ |
| 350 words | 9 115  | 488           | 22.6 %                   |
| 250 words | 13 207 | 350           | 0.6 %                    |
| 200 words | 16 382 | 281           | 0.2 %                    |

Transliterated Sanskrit and Cyrillic reach the limit in fewer words than Latin does — 1.27 tokens per word here against the usual English rate.

### Dimensions cost more than bits

145 800 vectors. Agreement is with what the same model returns at full precision, over the top twenty.

| Representation        | Bytes/vector | Index      | Agreement |
| --------------------- | ------------ | ---------- | --------- |
| float32, 1024 dims    | 4096         | 597 MB     | 1.000     |
| **int8, 1024 dims**   | **1024**     | **149 MB** | **0.975** |
| PCA to 512            | 2048         | 299 MB     | 0.742     |
| PCA to 256            | 1024         | 149 MB     | 0.725     |
| PCA to 128            | 512          | 75 MB      | 0.625     |
| First 512 numbers     | 2048         | 299 MB     | 0.708     |
| First 256 numbers     | 1024         | 149 MB     | 0.475     |
| One bit per dimension | 128          | 19 MB      | 0.525     |
| PCA to 64             | 256          | 37 MB      | 0.367     |
| First 128 numbers     | 512          | 75 MB      | 0.283     |
| First 64 numbers      | 256          | 37 MB      | 0.133     |

Read the rows at 1024 bytes together: every dimension at one byte agrees 0.975, the same storage spent on 256 projected dimensions agrees 0.725, and on the first 256 numbers 0.475. This model does not order its dimensions by importance, which is why truncating is worse than projecting.

The binary row is the coarse pass. It is not accurate and is not meant to be: half the right answers are outside its top twenty, which is why it keeps eight candidates for every result and the byte-precision vectors rank the candidates it kept.

Replacing float32 with int8 in the working index: 1561 MB to 613 MB, load 10 s to 5 s, search 35.7 ms to 31.8 ms, and the top six identical on every question in the acceptance set.

### What the coarse pass has to keep

Measured once with a standalone program, against one index of 65 261 embedded chunks — `baai/bge-m3` at 1024 dimensions, int8, over the Ganguli Mahābhārata and a few hundred notes. Eighteen questions, ten the vault holds an answer to and eight it does not. Exhaustive over every stored vector.

The behavior of the floor and the rerank is tested by fixtures in `chunks_test.go`. The floor that applies is resolved for each model and vault (see `settings.md`).

**How many the coarse pass keeps.** Recall of a query's exact top twenty at or above the floor, from a coarse pool of 20×f:

| f     | Pool    | Mean recall | Worst     |
| ----- | ------- | ----------- | --------- |
| 2     | 40      | 0.762       | 0.438     |
| 4     | 80      | 0.869       | 0.562     |
| 6     | 120     | 0.911       | 0.625     |
| **8** | **160** | **0.944**   | **0.688** |
| 12    | 240     | 0.968       | 0.750     |
| 16    | 320     | 0.973       | 0.750     |

The curve flattens after eight while the coarse pass costs close to linear in k: 40 → 12 ms, 100 → 20 ms, 200 → 32 ms, 400 → 56 ms, 800 → 116 ms.

**What the rerank costs.** Same eighteen questions, six rounds each, warm, at a limit of twenty.

| Meaning half                | Median  | p95     |
| --------------------------- | ------- | ------- |
| Coarse k=100, no rerank     | 19.6 ms | 23.4 ms |
| Coarse k=160, rerank, floor | 26.2 ms | 30.9 ms |
| Coarse k=160 alone          | 26.3 ms | 29.9 ms |

The rerank is 0.4 ms of that: one `json_each` join, 160 int8 blobs, 160×1024 dot products, a sort. The 7 ms is the wider coarse pass, less the 0.7 ms saved by resolving 20 enclosing chunks where there were 100.

The words half measures 1.1 ms median and about 39 ms p95 on a long natural language query, where the FTS expression becomes a wide OR. A search running both halves on such a query is over the 50 ms target above, and was over it before this.

### A partition key charges for every partition

The same 16 382 vectors, the same query, only the number of partitions changes. Nothing filters by the key.

| Partitions | Vectors each | Build  | Unfiltered search |
| ---------- | ------------ | ------ | ----------------- |
| none       | 16 382       | 0.2 s  | 1.93 ms           |
| 18         | 911          | 0.2 s  | 2.10 ms           |
| 64         | 256          | 0.3 s  | 6.12 ms           |
| 256        | 64           | 0.4 s  | 23.93 ms          |
| 1 024      | 16           | 1.0 s  | 92.45 ms          |
| 4 096      | 4            | 3.3 s  | 275.64 ms         |
| 16 382     | 1            | 48.7 s | 1067.30 ms        |

About **65 µs per partition walked**, flat across the range, independent of what each holds. A partition per book over ten thousand books would spend two thirds of a second before comparing a single vector.

A metadata column has no such walk cost, and does not narrow the search either:

|                            | Unfiltered | Narrowed to one cluster |
| -------------------------- | ---------- | ----------------------- |
| Cluster as partition key   | 24.37 ms   | 0.30 ms                 |
| Cluster as metadata column | 1.82 ms    | 1.45 ms                 |

Both return the same rows. The partition key skips partitions; the column filters rows after reading them.

### Notes rank beside books

157 vectors from a vault against 145 800 from the sources — a thousand to one. Best rank reached, over six questions:

| Indexed as                  | Ranks                          |
| --------------------------- | ------------------------------ |
| A chunk of a source         | 1, 1, 1, 8, 1, 1               |
| A note cut the same way     | 2, 7, 19, 1, 91, 7865          |
| A whole note, 283–357 words | 2045, 4, 318, 3, 127, 14510    |
| A note's title              | 580, 4590, 5541, 2, 510, 23808 |

Notes are not buried by the imbalance: ranking is by similarity and has no prior weighting. What lowers their rank is being indexed whole — the same dilution the chunk table shows, on a note instead of a book.

One question was answered by seven chunks of a single note taking the top seven places. That is what collapsing a document to one result is for.

### What these numbers are not

**The acceptance set is six questions.** Enough to decide between the options here, not enough to promise a rate. Every "agreement" figure is six queries wide.

**Agreement is with the model, not with a reader.** The full-precision ranking is the reference, so a representation scoring 0.975 gives the same ranking as this model, including where the model is wrong.

**The sizes are one corpus.** Chunk sizes tuned here are not guaranteed elsewhere, which is why the acceptance set serves as the primary verification check.

**Sanskrit is indexed and does not appear in results.** No question in the set returned a passage from the critical edition, in any configuration tried. Whether that is the model, the transliteration or the way verse is cut is not established.

**Everything past 150 000 chunks is generated.** A corpus of six million was measured for build time, index size and residency — 1316 MB, 148 s, 53 MB resident — but its vectors are random, so it gives no information about accuracy. The recall figures from clustering come from the real corpus at 16 382, and the gap to six million is untested.

### What does not work

|                                                      |                                                                                |
| ---------------------------------------------------- | ------------------------------------------------------------------------------ |
| Reducing dimensions instead of precision             | Loses a quarter to a half at equal storage                                     |
| Truncating a vector this model produces              | Worse than projecting, at every size tried                                     |
| Indexing a title                                     | Ranks near-randomly; the lexical index already matches it                      |
| Indexing a note whole beside sources cut into chunks | Erratic — 4th on one question, 14 510th on another                             |
| Writing chunks scattered across partitions           | 1 900 rows/s against 40 000 grouped                                            |
| Cutting a chunk by lines                             | One file put a book on four lines and produced a chunk of a million characters |

### What a second engine would cost

Nothing measured here supports using one. Two are reachable from Go if one is ever wanted, and the numbers above are what a candidate has to beat:

| Candidate   | Reached from Go by                                 | What it would cost                          |
| ----------- | -------------------------------------------------- | ------------------------------------------- |
| `usearch`   | Official bindings; the index memory-maps from disk | cgo, and cross-compilation from one machine |
| Qdrant Edge | A Rust crate, in-process, no bindings today        | cgo, and bindings to write                  |

LanceDB has no Go bindings at all: reaching it means writing foreign-function bindings by hand, or running a second process beside the application.

An index that has to be held in memory to be searched is excluded regardless of its speed. At the size this is designed for it needs gigabytes of memory, which competes with everything else on the person's machine. That is why quantisation and staged retrieval are the approach measured above.

### What was invisible in review

**A batch needs its own size limit.** The model's limit is 8192 tokens and the provider bounds a request at 131 072 characters for the whole batch. A batch size that holds for Latin text exceeds it in transliterated Sanskrit, where the same word costs three times the tokens. Batches are built to a character budget.

**A stored vector's type is guessed from the length of its blob.** An int8 vector of 1024 dimensions is 1024 bytes, which is also a float32 vector of 256, and the extension reads it as the latter. Both sides of a comparison state their type. A bit vector is written and matched through `vec_bit(?)` for the same reason.

**A cascade does not apply to a virtual table.** Deleting a source cascades to its chunks and stops there; the vector rows remain. They are deleted explicitly, by the chunk number, which is the virtual table's rowid — the same rule `chunks_fts` and `sections_fts` already follow.

**`INSERT OR REPLACE` is not supported by `vec0`.** It fails with `UNIQUE constraint failed`, so a vector that is being replaced is deleted first.

**Re-saving a row does not trigger a cascade.** `save_source.sql` conflicts on `(vault_id, path)` and updates, so the row number never changes and no parent row is deleted. Every derived table is therefore cleared explicitly — headings, links, problems, and now chunks. A test saves the same note twice and asserts the counts did not double, because no other test would detect it.

**A query plan names the alias, not the table.** A check that forbade `SCAN <table>` passed on `SCAN c`, so reading every chunk was not detected. The check now forbids every `SCAN` step except a virtual table and `vaults`.

**One populated vault makes a vault filter appear to cost nothing.** With a single vault in the fixture, the planner chose the primary key over the vault index and read every vault's chunks. The fixture holds two vaults of 1500 notes for this reason, which is the same lesson as measuring the database before trusting its plans.

**A semicolon inside a migration comment breaks the migration.** `migrate.go` splits a file on semicolons without stripping comments. The convention is to avoid them; the splitter has not been changed.

### Getting the text out of an EPUB

Recorded 2026-08-17 on the same AMD Ryzen 7 6800U. The timings below were taken with a Python prototype; the structure figures further down come from the Go extractor that ships. Where the two disagree, the Go figure is the one that describes the application.

Forty EPUBs from `resources/mahabharata`: 85.5 MB of archives, 59 MB of text once extracted.

The corpus is not in git. The test that measures it skips when the directory is absent.

|                       |        |
| --------------------- | ------ |
| Extraction, all forty | 2.4 s  |
| Per book              | 59 ms  |
| Slowest book          | 0.26 s |

Reading a passage back, which is what a result costs when the extracted text is not stored:

|                                            |        |
| ------------------------------------------ | ------ |
| One book, extracted again                  | 42 ms  |
| One spine document of it                   | 5.9 ms |
| Slicing out of a stored 1.5 MB text column | 0.9 ms |
| Slicing out of a stored 60 MB text column  | 66 ms  |

`substr` costs about 1.1 ms per megabyte of the value it reads, so the largest books are the ones a stored column serves worst. Re-reading one spine document is faster than both.

What structure the forty have, **measured with the Go extractor**:

| Where it came from      | Books |
| ----------------------- | ----- |
| The navigation document | 16    |
| Heading tags            | 4     |
| Nothing at all          | 20    |

Half the corpus has nothing to cut on, which extraction must handle.

A tier is used only when it names at least two parts. One named part is a cover or a book's own title, and it gives no cut point that the caller does not already have, because the offset of every spine document is reported separately. Four books depend on that rule: each has a single heading, one of them across 853 spine documents. Counting a lone heading as structure puts those four in the heading tier and gives 16 / 8 / 16.

Cover-only navigation is not rare: eighteen books have a single navPoint labelled `Начать` pointing at the title page, which is why a tier is judged by its usable entries rather than by whether the file exists.

Five of the forty have the page numbers of a print edition, 54 to 311 targets each, all through `.ncx pageList`. No book in this corpus uses EPUB 3's `epub:type="pagebreak"`.

### Embedding, locally and through a service

Recorded 2026-08-17 on the same AMD Ryzen 7 6800U — 16 threads, AVX2 and no AVX-512 — with `CGO_ENABLED=0`, `intfloat/multilingual-e5-small`, real tokenisation and 256-token inputs.

| Local, pure Go      | Chunks/s | 400 000 chunks |
| ------------------- | -------- | -------------- |
| default build       | 1.38     | about 80 hours |
| `GOEXPERIMENT=simd` | 3.12     | about 36 hours |

The flag gives 2.2×. It must be set when the binary is built, and nothing in the repository sets it. It should be set in the release build.

Window size changes the rate as much as the flag does: the same model answers 6.0/s at 128 tokens and 1.06/s at 512.

For comparison, a hosted service embedded 145 800 chunks of the source corpus in **10 minutes** for about ten cents.

What that means for the default: a personal vault of a few thousand notes is ten to twenty thousand chunks, which finishes locally in one to two hours. A hundred thousand notes is four hundred thousand chunks, and local is then a day and a half of background work. The vector index is filled after the lexical one and may never finish, so neither figure blocks anything — but only the service embeds a corpus of that size within a session.

### Multi-session inference pool and sequence bucketing

Recorded 2026-10-09 on AMD Ryzen 7 6800U (16 threads). Embedding batches of variable-length passages with ONNX Runtime:

| Optimization                                      | Batch Latency (p50) | Throughput (passages/s) | Speedup |
| ------------------------------------------------- | ------------------- | ----------------------- | ------- |
| Baseline (single session, naive padding)          | 142 ms              | 56.3                    | 1.0×    |
| Multi-session parallel pool                       | 48 ms               | 166.7                   | 3.0×    |
| Pool + Sequence bucketing (length-sorted batches) | 31 ms               | 258.1                   | 4.6×    |

Bucketing groups candidate chunks by token length into contiguous ranges before dispatching tensors, which minimizes padding waste.

### Search pipeline caching and deduplication

Recorded 2026-10-09 on AMD Ryzen 7 6800U:

| Search Component              | Baseline            | With Deduplication & Cache                                                                               | Speedup       |
| ----------------------------- | ------------------- | -------------------------------------------------------------------------------------------------------- | ------------- |
| Repeated query embedding      | 18.2 ms             | 0.04 ms (memory cache)                                                                                   | 455×          |
| Identical chunk indexing      | Full embedding pass | Vector reused by SHA-256 content hash; the chunk still gets its own coarse row, written from that vector | No model call |
| Hybrid search query execution | 32.4 ms (serial)    | 12.1 ms (parallel lexical, named, dense)                                                                 | 2.7×          |

### What naming a chunk costs the walk

The core identifies a chunk by an opaque `domain.ChunkID` and resumes the walk from an opaque `port.ChunkCursor`. The index builds both from the row number it keeps. That is one `FormatInt` per row read and one `ParseInt` per vector written, over every chunk of a vault. The total cost was assumed before and is now measured.

Recorded 2026-09-05 on the same AMD Ryzen 7 6800U, from `BenchmarkUnembedded`, `BenchmarkSaveVectors` and `BenchmarkChunkIdentity` in `adapter/index`. Ten runs of each; the two conversions vary by under 10 % across them, and the pages by two- to fivefold, which is why the conversions are negligible by comparison.

|                                                            | Measured |
| ---------------------------------------------------------- | -------- |
| A row spelled as a `domain.ChunkID`                        | 25 ns    |
| A `port.ChunkCursor` read back as a row                    | 23 ns    |
| One chunk read out of the index, in a page of 200          | 3.9 µs   |
| A row turned into a `domain.SearchResult`, the whole of it | 0.67 µs  |
| One vector written, in a page of 200                       | 200 µs   |

**It is below the noise level.** Over 400 000 chunks — the vault a hundred thousand notes cut into — both conversions together total **19 ms**. Reading those rows out of SQLite is 1.6 s, writing their vectors is 80 s, and the embedding itself is 27 minutes through a service or 36 hours on this laptop's CPU. The conversion is a thousandth of the cheapest adjacent operation, and the row a walk resumes from is serialized once per page of 200, not on each row.

The two pages are measured at a fixed number of iterations because a laptop with other work on it makes them vary fivefold between runs, and a conversion two orders of magnitude smaller than the page cannot be derived from a difference between pages. It is measured separately, on a row number of nine digits — longer than anything a vault of this size produces.

### Transcribing a recording

Recorded 2026-09-01 on the same AMD Ryzen 7 6800U — 8 threads of it — with the models quantised to int8 where the column says so. The figure is wall time over audio time, so 0.10 is ten minutes of speech transcribed in one.

**The target: a recording is transcribed faster than it plays.** With anything slower, a vault of lectures is never fully transcribed.

|                                   | ru        | en        | Peak RSS |
| --------------------------------- | --------- | --------- | -------- |
| parakeet-tdt-0.6b-v3 int8, ONNX   | 0.08–0.15 | 0.08–0.11 | 1.2 GB   |
| whisper tiny, ggml                | 0.08      | 0.06      | 0.3 GB   |
| whisper small, ggml               | 0.33      | 0.22      | 0.9 GB   |
| whisper small int8, ONNX          | 0.39      | 0.26      | 1.4 GB   |
| whisper large-v3-turbo int8, ONNX | 0.41      | 0.39      | 2.2 GB   |
| whisper large-v3-turbo, ggml      | 0.92      | 0.84      | 1.9 GB   |

Quantising is what makes the largest model usable: the same weights through ggml at fp16 take 0.92, and through ONNX Runtime at int8 take 0.41. The ggml build of `small` quantised to q5\_1 is **slower** than the same model unquantised — 0.73 against 0.38 — so measure quantisation for each build instead of assuming a gain.

Words wrong, against [FLEURS](https://huggingface.co/datasets/google/fleurs), 60 utterances a language, lowercased and stripped of punctuation:

|                              | ru    | en    |
| ---------------------------- | ----- | ----- |
| parakeet-tdt-0.6b-v3 int8    | 7.1%  | 8.6%  |
| whisper large-v3-turbo, ggml | 4.6%  | 9.2%  |
| whisper small, ggml          | 10.3% | 19.7% |

The two are not measured the same way and the numbers are not directly comparable: parakeet was given one utterance at a time, as a segmenter feeds it, and whisper was given the utterances joined into one file, which gives the best accuracy for whisper. Read the table as two separate results for "is this good enough", not as a ranking.

The decision rests on both tables together. Parakeet is more accurate than `small` on Russian and five times faster than it, and it takes a third of the time of the turbo model, which is more accurate. Half an hour of Russian speech is two and a half to four and a half minutes of one core.

One more factor matters for real recordings, and no table above shows it. Whisper's encoder always processes a window of exactly 30 seconds, whatever is in it, so speech cut at the silences — which is how a recording is fed to a model — takes three times as long as its duration suggests. Parakeet's cost is proportional to the audio length.

The weights are 641 MB, fetched when the first recording is transcribed.

### Resampling audio to the model rate

Recorded 2026-10-08 on AMD Ryzen 7 6800U (16 threads) and Apple M4 (10 cores), from `BenchmarkResampledAMinute` in `internal/adapter/transcription`. Resampling one minute of 44.1 kHz audio to 16 kHz:

| Platform                       | Sequential | Parallel (`GOMAXPROCS`) | Speedup |
| ------------------------------ | ---------- | ----------------------- | ------- |
| Linux (x86_64)                 | 328 ms     | 98 ms                   | 3.3×    |
| macOS (Apple Silicon M4 arm64) | 83.4 ms    | 18.5 ms                 | 4.5×    |

Each output sample is computed independently, so the samples are chunked across available worker goroutines.

### Extracting log-mel spectrogram features

Recorded 2026-10-08 on AMD Ryzen 7 6800U (16 threads) and Apple M4 (10 cores), from `BenchmarkLogMel` in `internal/adapter/transcription`. Computing 128-band log-mel features for 10 seconds of 16 kHz audio:

| Platform                       | Sequential STFT & dense bank | Parallel STFT & sparse bank | Speedup |
| ------------------------------ | ---------------------------- | --------------------------- | ------- |
| Linux (x86_64)                 | 42.8 ms                      | 5.3 ms                      | 8.0×    |
| macOS (Apple Silicon M4 arm64) | 24.3 ms                      | 1.4 ms                      | 17.2×   |

Spectrogram FFTs and mel filter bands are chunked across available worker goroutines, and triangular filters evaluate only non-zero frequency bins.

### Why the markup is not parsed as XML

Four books hold documents that `encoding/xml` rejects — 125 documents in all, failing with `element <p> closed by </html>`. `golang.org/x/net/html` reads every one of them. That is the reason for the dependency: a strict parser drops a tenth of this corpus, and extraction must not reject a document.

Two more shapes in the same corpus would have lost whole books. Three books declare their spine documents as `media-type="text/html"`, so filtering the spine by media type loses them entirely. Five carry no `dc:title` at all, so an empty title is a correct answer rather than a parse failure.

## Editing a note in a tab

```
go test ./usecase/note/ -run XXX -bench 'Read|Save' -benchtime 50x
```

Taken 2026-08-17, AMD Ryzen 7 6800U, NVMe, `-benchtime 50x`.

|                               |          |
| ----------------------------- | -------- |
| Read a note, vault of 1 000   | 98 µs    |
| Read a note, vault of 100 000 | 99 µs    |
| Save, 500-word body           | 10.06 ms |
| Save, 5 000-word body         | 10.07 ms |

**A read does not grow with the vault.** The two figures are within a percent of each other, so a read opens one file and no other. This is the cost a window pays per clean tab per change that names its path: ten open tabs are about a millisecond of reading per change.

**A save costs the same whatever is in it.** Both bodies take the same 10 ms, so the text is not what is measured. A save syncs the temporary file and then the folder it is renamed into, and those two are the whole figure. Against a quiet interval of 800 ms a person cannot notice it; it is recorded because it shows what a faster save would have to change.

### A save, and how long until the window knows

```
NUMEN_LOAD=1 go test ./adapter/window/editor/ -run TestEditLoad -v -timeout 30m
```

Taken 2026-08-17, same machine. This path is the application's own code: the write, the watcher detecting it, the refresh, and the change reaching a client.

| vault         | save  | until a client is notified |
| ------------- | ----- | -------------------------- |
| 10 000 notes  | 22 ms | 65 ms                      |
| 100 000 notes | 13 ms | 66 ms                      |

Neither figure grows with the vault. The watcher reports a path, the refresh reads that one note, and no directory is walked. So the cost of an edit depends on the size of the note and not on the size of the library.

What is still not measured is the two steps done in the browser: a keystroke becoming a request, and the page being painted.

### What an open tab costs, and why there is no number

```
npx vitest run --project stories src/editor/Editor.stories.ts
```

Every tab of a pane is drawn and hidden, so the tab count is the live editor count. The story mounts ten editors in one page to measure the cost of the tenth.

There is no number. `performance.memory.usedJSHeapSize` is quantised by the browser, and it reads the same 67.6 MB with one editor alive and with ten. So the story asserts only what can be checked, that ten editors are alive at once, and the size is not measured.

The API that would measure it is `performance.measureUserAgentSpecificMemory()`, which needs the page to be cross-origin isolated. Until the story runs in such a page there is no figure here, and a figure taken from the quantised counter would not be a measurement.

### What an edit costs to embed

```
go test ./adapter/index/ -count=1
```

These are assertions rather than timings: the count of vectors a save requests from the model. Chunks are tiled at fifty words with ten of overlap, and a chunk is identified by the hash of its text, so a chunk whose text did not change keeps its row and its vector.

A note's headings divide it into parts. The chunks of one section are tiled inside that section. A note with no headings is one part, tiled from its first word. The two columns below use the same words cut both ways.

**A 200-word note under four headings**, one every 48 words. It has five small chunks when tiled over the whole body and four when the headings divide it into sections.

| the edit is                     | no part named         | the headings as parts |
| ------------------------------- | --------------------- | --------------------- |
| a line added to the frontmatter | 0 asked · 5 of 5 kept | 0 asked · 4 of 4 kept |
| at the end of the body          | 1 · 4 of 5            | 1 · 4 of 4            |
| in the middle of the body       | 3 · 2 of 5            | 2 · 3 of 4            |
| at the start of the body        | 5 · 0 of 5            | 2 · 3 of 4            |

**A 1000-word note under five headings**, one every 200 words. It holds 25 small chunks either way.

| the edit is                     | no part named           | the headings as parts   |
| ------------------------------- | ----------------------- | ----------------------- |
| a line added to the frontmatter | 0 asked · 25 of 25 kept | 0 asked · 25 of 25 kept |
| at the end of the body          | 1 · 25 of 25            | 1 · 24 of 25            |
| in the middle of the body       | 14 · 12 of 25           | 3 · 22 of 25            |
| at the start of the body        | 26 · 0 of 25            | 5 · 20 of 25            |

The counts are of the chunks that have a vector. The chunk enclosing the note keeps its row only in the frontmatter edit: its text is the whole note, so any edit to the body replaces it. The kept chunks are linked to the row that exists now.

The frontmatter row shows that the hash is over the text and not over the offsets: every chunk moves in the file, none of them changes, and nothing is embedded again.

**Headings reduce the cost in the last two rows of the second table.** With the note as one span, an edit re-cuts every chunk after it and the worst case grows with the note: 5 vectors at 200 words, 26 at a thousand. With a heading opening each section the worst case is the chunks of that section at any length — 5 at a thousand words, and 3 for an edit half way down.

**The cost is a higher chunk count on a note that is mostly headings.** A chunk is never cut across a heading, so a section shorter than fifty words is a chunk of its own. A 200-word note with a heading every five words is cut into 40 chunks where the same words with no headings are cut into 7, and each of the 40 needs a vector of its own. An edit anywhere in that note requests one vector.

The 200-word note tiled by sections requests four vectors where one span requests five: a section of 48 words is one chunk, and nothing overlaps across a heading. Every small chunk has the name of the section it was cut from; the chunk enclosing the note has the note's title, so a search by a note's name still finds it.

## Reading a PDF

Recorded 2026-08-19 on the same AMD Ryzen 7 6800U, `CGO_ENABLED=0`, pdfium through WebAssembly.

The document is 546 pages of a scan at 600 dpi, 233 MB, with a text layer from earlier OCR.

|                               |                                                         |
| ----------------------------- | ------------------------------------------------------- |
| The library, compiled         | 5.1 s, once per process                                 |
| Opening a document after that | 0.1–0.6 s                                               |
| Its text layer, all of it     | 10–12 s (1 701 572 characters, about 45 pages a second) |
| Naming every page             | 4 ms                                                    |

The compile applies to the module, not the document. Only the first PDF a run reads pays it. The text of a smaller document is proportionally quicker: 80 pages is under a second.

## Opening a book made for a screen

Recorded 2026-09-06 on the same AMD Ryzen 7 6800U. The corpus is not in the repository; these tests are skipped without it.

The book is the largest of the forty: 19.8 MB on disk, 4 032 243 bytes of text over 96 spine documents, with 99 parts and 2 177 pages. Its longest document is 88 707 bytes of text.

|                                                                |                             |
| -------------------------------------------------------------- | --------------------------- |
| Reading the whole file                                         | 105–122 ms (about 175 MB/s) |
| Converting the longest document to the markup a window renders | 1.5–2.0 ms                  |
| Counting the pages of the book                                 | 100–200 ns                  |

No layout is done to count the pages, so the last row is in nanoseconds and not seconds: the count is arithmetic over the one text stream. A reader that paginated by laying out every document would process the whole book before showing any of it. This matters in the engine the window runs in: WebKit lays out the pages on two of the three platforms.

The file is read once when the book is opened and not again while the book is open; opening a chapter costs the second row.

## Recognising a scanned page with a model

The same machine, ONNX Runtime through `purego`, one page at 300 dpi: layout by PP-DocLayoutV3, lines by PP-OCRv5 detection, text by PP-OCRv6 tiny.

|                         |                  |
| ----------------------- | ---------------- |
| One page, ten regions   | 8.6 s            |
| A document of 546 pages | about 75 minutes |

Two settings are measured rather than guessed.

**Four threads.** Two is slower (9.7 s a page), eight is slower (10.7), and sixteen is much slower (23.2). The lines of a page are small, and spreading one of them over sixteen threads costs more than it saves.

**One page at a time.** A second worker holds a second copy of every model, 124 MB of layout weights among them, and on a 13 GB machine the two workers spent their time in swap: 18.4 s a page against 8.5.

The recogniser is chosen by how many letters it preserves, not by its size. Measured over three pages of a book set in transliterated Sanskrit, against the document's own text layer with every mark folded away from both:

|                 | size   | a page | keeps the letter | marks |
| --------------- | ------ | ------ | ---------------- | ----- |
| PP-OCRv6 tiny   | 4.5 MB | 1.2 s  | 92.6%            | 142   |
| PP-OCRv6 small  | 21 MB  | 2.6 s  | 81.5%            | 156   |
| PP-OCRv6 medium | 77 MB  | 8.1 s  | 81.5%            | 157   |
| PP-OCRv5 server | 85 MB  | 7.9 s  | 90.8%            | 67    |

No model in the family can write a consonant with a dot below it. Small and medium drop the letter they cannot output — `ṭuṭaba hṛdayaka` becomes `uaba hdayaka` — and one Sanskrit word in five loses a letter. Tiny writes the plain letter instead, so the word keeps its length and a search still finds it. The smallest is also the fastest and, on this book, the best.

A recogniser that does not support a script outputs plausible nonsense: the same models over a Russian document return Latin gibberish, and `chunking.legible` keeps most of it out of the index by rejecting it.

## Reading the words of a document's text layer

Recorded 2026-08-20 on the same machine, over the 546-page scan and its text layer.

|                          |                                       |
| ------------------------ | ------------------------------------- |
| Reading the whole text   | 9.5 s, 1 702 107 bytes over 546 pages |
| The words of one page    | 239 ms                                |
| The words of three pages | 213 ms, 71 ms a page                  |
| The words of ten pages   | 403 ms, 40 ms a page                  |

**Almost all of it is opening the document.** Ten pages cost twice what one does, not ten times, because the 233 MB file is read and parsed once and the pages after the first are tens of milliseconds each. That is the purpose of a cache of open documents, and the reason the pages needed are requested together.

The layer yields 572 boxes a page, 312 357 over the book, against the 51 169 a recognition of the same pages produced. The gap is punctuation: this layer puts a space before a comma, so 43 523 of the boxes are a single byte.

## What a line's boundary costs

Recorded 2026-08-20 over the same scan: the four headings of pages 31 and 32, and the body of twenty pages spread across the book, against the document's own text layer. Error is the mean share of characters wrong.

The detector returns an outline that lies inside the letters, and the shortfall is a share of the line's height. The widening that corrects it was a fixed number of pixels, measured on the part after it has been scaled: on a heading that is about four pixels of the page against fifteen to twenty of shrink. The top of every capital and the last letter of every line were cut off.

| the line widened by | headings | body   |
| ------------------- | -------- | ------ |
| 10 pixels           | 0.074    | 0.0644 |
| 14                  | 0.031    | 0.0633 |
| 18                  | 0.025    | 0.0639 |
| 20                  | 0.017    | 0.0648 |
| 24                  | 0.035    | 0.0667 |

**Eighteen** is used, the midpoint where the improvement levels off. The body is flat from ten to twenty and begins to degrade after that.

```
"IAYADEVA GOSVAMI'S LIFEINNABADWI"         became "JAYADEVA GOSVAMI'S LIFE INNABADWIP"
"LAYADEVA GOSVAMI'S MARRIAGE TO PADMAVAT"  became "JAYADEVA GOSVAMI'S MARRIAGE TO PADMAVATI"
"THELORD HELPS IAYADEVA GOSVAMIWRITE…"     became "THE LORD HELPS JAYADEVA GOSVÁMI WRITE GITA GOVINDA"
```

Two other settings were measured and neither is worth changing. The activation threshold for a pixel to count as part of a line gives the same boxes at 0.15, 0.20 and 0.30 — the map is effectively binary here. The longest side a part is read at loses small type when lowered and degrades all text when raised, and 300 dpi gives better results than 150, 200, 400 and 600 because detection is scaled to that side whatever the page was drawn at.

No setting fixes this: a word gap in a display face no wider than its letter gaps is one line to any threshold, so the recogniser has to guess the space. And the model's alphabet has `ā ī ū ñ ś` and none of the letters with a dot under them, so `Lakṣmaṇa` and `Kṛṣṇa` cannot be output whatever the boxes are.

## A part too small to find a line in

Recorded 2026-08-20 over the same scan, pages 44 to 73 of the file, which print 14 to 43. The layout model finds the `number` region on every one of them — the number is alone at the foot of the page between two ornaments — and it is some fifty pixels across at 300 dpi.

Nothing is read from it. The detector outlines dark pixels and shrinks the outline by a share of the line's height. A line that fills the whole image leaves no outline, so the region is returned blank.

| what was read                                      | pages right of 30 |
| -------------------------------------------------- | ----------------- |
| the unmodified region                              | 0                 |
| the region magnified 3× to 20×                     | 0                 |
| the crop widened to 150, 250, 400, 640, 960 pixels | 0 to 6            |

Widening the crop includes the ornaments on either side, and they are read as something other than a number. Keeping only the lines whose middle is inside the region gives 30 of 30 at a margin of 150 pixels and above. That margin is a pixel count for one book at one dpi, so it cannot be used as a default.

So the printed number is not read. A page is referenced by its sequential position in the file, which is known for every page, costs nothing, and is the number the pane shows.

The margin that works is kept here because it shows the cause of the failure: the detector needs background around a line, and the models can read two digits.

## Drawing a page of a scan

Recorded 2026-08-20 over the same 546-page scan, five pages averaged, on an idle machine. The recognition run competes for the same cores and for the same pool of pdfium workers; under one, every number below is roughly doubled.

| width asked | pdfium | resample | jpeg | total      | over the wire |
| ----------- | ------ | -------- | ---- | ---------- | ------------- |
| 800         | 443 ms | 43       | 17   | **503 ms** | 234 kB        |
| 1200        | 498    | 95       | 34   | **628**    | 417 kB        |
| 1600        | 553    | 163      | 52   | **768**    | 625 kB        |
| 2400        | 616    | 377      | 107  | **1101**   | 1094 kB       |

**Drawing time barely changes with the resolution** — 104 dpi costs 443 ms and 311 dpi costs 616. So the cost is not rasterising: each page of this book holds one large photograph, and the library decodes the whole of it whatever size is asked for. Half a second a page is what this document costs, and no setting here lowers it.

**The resample only removed a pixel or two.** The resolution is a whole number rounded up, so a page requested at 800 is returned 804 wide. Scaling away those four pixels cost between a tenth and four tenths of a second, and the window lays the page out at the width it asked for anyway. It is removed: the page is sent as drawn, and the browser scales off the four pixels in the compositor.

That leaves the half second, which is spent again every time the person returns to a page. So the drawn pages are cached in this machine's cache folder and not in the vault. A cached page is read from disk in about a millisecond, and a book read through once costs half a second a page only the first time.

## Finding the section a question is about

Recorded 2026-08-20 over the same 546-page scan. 700 questions drawn from the book's own parts: for each of 350 parts whose heading the recogniser read as words, one question is that heading — a person asking where the book discusses a thing — and one is seven words from the middle of the part. The right answer is that part either way.

|                                   | right section | opened at its start |
| --------------------------------- | ------------- | ------------------- |
| the words half in original form   | 307/350       | 298/350             |
| the section's name in every chunk | 340/350       | 306/350             |
| the sections findable by name     | 346/350       | 346/350             |
| both                              | 348/350       | 348/350             |

The questions quoting a phrase from the middle of a part moved by one either way — 331, 330, 331, 330 — so none of these lowers the result for a question that names no section.

**The second column decided the choice.** Putting the name into every chunk ranks the right chapter higher but still returns whichever paragraph of it ranks best; making a section findable by its own name returns the section, at its heading.

One query shows how the words half behaves. Asked for `Madhavendra Puri`, BM25 scored the chapter that _is_ about him at −12.74 and the subsections inside it between −17.8 and −19.7 — and lower is better. The chapter's opening contains his name once, in its heading; a paragraph in the subsection after it contains it four times.

**The name in every chunk was not adopted.** It adds two points on top of the other, but it changes the text a vector is made from: the recipe changes, and every chunk of every recognised document is embedded again. Only the words half was measured here — what a vector that includes its chapter name gains cannot be known without computing those vectors. The number to beat is 346 and 346.

## Correcting a recognition

Recorded 2026-08-21 over the same 546-page scan, through a hosted model.

**The unit is the printed line.** The reply contains only the lines that changed, and the share returned is 31.2 % of the book when the page is asked as whole blocks, 14.3 % as sentences, and **3.8 % as the printed line**. A line of the recogniser's own is one run of words read in one go — 31 characters on average, 50 841 of them in this book — and about a fifth of blocks contain a misread word.

**Two models, over 58 pages spread through the book.**

|                         | pages rejected by gates | output tokens       | a book |
| ----------------------- | ----------------------- | ------------------- | ------ |
| `gemini-2.5-flash-lite` | 67 %                    | 25 005 per 10 pages | $0.14  |
| `gemini-2.5-flash`      | 8.6 %                   | 6 919 per 20 pages  | $0.29  |

The cheap model lists lines it did not change, which is where its output tokens go. A chain that runs it first pays $0.14 for a reply that is discarded and then $0.29 for the better model. The good model alone is $0.29, about 1 165 000 tokens in and 194 000 out. A rejected page that is queried again gives the same result.

**Where `maxEditDistance` comes from.** Over 931 corrections the distribution has a gap: 36 differ by more than 0.50, 47 by more than 0.30, 49 by more than 0.20, 70 by more than 0.10. Above 0.30 every correction read was damage — text taken from the next line, or one corrected word in place of a whole line — and below it every one was a correction. The threshold is set at the gap and not at a round number, and it is in the settings file because it was measured on one book.

**A reply row is written two ways.** Over one batch of 40 pages the model replied to 15 pages with `2544|the line` and 25 with `2544 the line`, each page in one style throughout.

A queue costs half as much as requesting a page at a time and waiting.

Folding diacritics before embedding measures +0.08 to +0.11 of cosine for a person who types plainly. It does not apply generally: it removes `q̇`, a vector arrow and a bar, which in a book of mathematics carry meaning.

## Changing how large the window is drawn

Recorded 2026-08-25 in Chrome 149, over a window holding a 400-line editor and three thousand rows of chrome.

| the multiplier changed    | forced layout |
| ------------------------- | ------------- |
| `--numen-interface-scale` | 27–39 ms      |
| `--numen-text-scale`      | 4–6 ms        |

The interface scale sets the root font size, so every length written in `rem` is recalculated and the whole document is laid out again. The reading size is also written in `rem`, so the interface scale also scales the editor and the marked-up text: everything the reading size changes is also changed by the interface scale.

The editor draws only the lines that are on screen — thirty-six of the four hundred — so what it adds to either is a screen of text, however long the note is. Marked-up text renders all of it. The same window with four hundred blocks of a note on screen, over the whole stylesheet rather than the tokens alone:

| the multiplier changed    | with the note | without it |
| ------------------------- | ------------- | ---------- |
| `--numen-interface-scale` | 45–71 ms      | 32–43 ms   |
| `--numen-text-scale`      | 9–13 ms       | 7–11 ms    |

Compare the two columns with each other. A page with the whole stylesheet takes longer to recalculate than one with only the tokens, so neither column matches the table above.

A held arrow key moves across a row of the size list every 40 ms, and a size is applied once the selection has stayed on a row for 150 ms.

## What a name of the derived store costs

Recorded 2026-09-01 and re-measured 2026-09-06 on the same AMD Ryzen 7 6800U, from `BenchmarkDerived` in `internal/adapter/filesystem`. Every name the store accepts is resolved in the place where the vault is, and the check that the vault is still there reads the identity of the folder.

|                   | Before                       | After                        | 2026-09-06                       | 2026-09-06, resolved once        |
| ----------------- | ---------------------------- | ---------------------------- | -------------------------------- | -------------------------------- |
| One name read     | 92.2 µs · 5504 B · 57 allocs | 67.0 µs · 4584 B · 48 allocs | 64.9 µs · 6496 B · **76 allocs** | 36.4 µs · 3584 B · **44 allocs** |
| One folder listed | 79.6 µs · 4895 B · 60 allocs | 66.9 µs · 3989 B · 51 allocs | 51.9 µs · 5672 B · **71 allocs** | 20.8 µs · 2745 B · **39 allocs** |
| One line appended | 2.43 ms · 5334 B · 58 allocs | 2.24 ms · 4413 B · 49 allocs | 2.29 ms · 6400 B · **80 allocs** | 1.80 ms · 3355 B · **48 allocs** |

Each column is the median of three runs; the last two name the best of three on the clock, and the same allocations on all three. The allocation columns are the ones to compare: a run of the read row varied by 10 % on the clock and not at all in allocations. The last two columns were measured one after the other on a machine with a load average of 6.

**The allocations are higher than at the start of the change above, because of a containment fix.** On 2026-09-04 a path was constrained to its declared directory, because a link written into an area — by a sync client, by another tool — pointed to the vault's identity file and a transcript appended through it was written to `config.json`. The check is correct and the vulnerability was real.

**Resolving the path three times was not required.** `DerivedStore.at` resolved symlinks on the target, then on the store's area, then on the store's own root. The last two are paths that do not change while the store exists, and `deepest` is a `filepath.EvalSymlinks` walk — one `lstat` per segment of an absolute path, every time. By allocation profile **60.5 % of one `Read` was spent in `EvalSymlinks`**, which accounts for the whole difference between 48 allocations and 76.

**The store's folder is resolved when it is opened, and each area is placed inside it**, and the last column shows the result: a read is 44 allocations against 76 and 36.4 µs against 64.9, and a listing 39 against 71 and 20.8 µs against 51.9 — both below the row the containment fix started from. The check itself is unchanged, and a name is still resolved inside its area or is refused.

**A folder that moved is resolved again.** A name that the resolved folder refuses is checked a second time against the folder's current location, so a service folder created after the store was opened — a sync client keeping the application's files on another disk — is still written to as before; and `still()`, which reads the identity again, also resolves the folders again. Only a refusal triggers the second resolution, so repeated operations by a person do not pay for it.

This cost applies to each card answered and each run file read. A history screen over 180 run files incurs it 180 times, twice for each.

**The identity was read out of the file at every name.** `config.json` opened, its bytes parsed and the identity in it checked, which is 19.9 µs and 12 allocations on its own — a fifth of a read and a quarter of a listing. It is now a stat of that file, and a file with the same length and the same age reuses the identity already read from it. A file that changed is read again, so a folder with another vault's identity is still refused at the first name after it appears.

**An append is the cost of reviewing a card**, writing one log file per session, and it is unchanged: the `fsync` at the end takes two milliseconds of the two and a quarter.

**A vault without an identity is checked by its folder.** A store opened on a folder that never had an identity has none to lose, and the check now tests whether that folder exists.

## What a window requests from one vault

Recorded 2026-09-01 and re-measured 2026-09-06 on the same AMD Ryzen 7 6800U, from `BenchmarkVault` in `usecase/flashcards`. The vault is generated: fifty thousand review cards over twenty decks, answered a hundred and fifty times a day for six months — 27 000 answers in 180 run files.

Each row is one request through its use case. The first three are warm, which is a person's second request of an evening: the schedule cache and the day counts are filled before the clock starts. The fourth is the history screen on a build that keeps no counting, which is what the first request after launch pays.

|                                         | Before          | After           | 2026-09-06      |
| --------------------------------------- | --------------- | --------------- | --------------- |
| Initial overview, one vault counted     | 0.82 s · 605 MB | 0.85 s · 605 MB | 1.06 s · 617 MB |
| Starting a session                      | 1.25 s · 836 MB | 1.19 s · 793 MB | 1.46 s · 824 MB |
| The history screen                      | 0.71 s · 536 MB | 0.71 s · 493 MB | 0.83 s · 502 MB |
| The history screen, nothing counted yet | 0.78 s · 554 MB | 0.68 s · 490 MB | 0.82 s · 498 MB |

Every column is the median of two runs of five. The memory column is what a request allocates, which is the steadier of the two: the times of repeated runs of one build spread by about 5 %, and the memory by under 0.1 %.

**The four rows are stable.** In the memory column, which is the one to compare, every row differs by 1.7 % to 3.9 % from the value after the change above. The clock is 17 % to 23 % slower and five other agents were running on the machine while it was taken, so the clock adds nothing that the memory does not show better.

**The initial overview benchmark is unchanged, and serves as the baseline.** It already queried the schedule cache first, and nothing was removed from its path.

What was counted, per warm request:

|                                                           | Before | After |
| --------------------------------------------------------- | ------ | ----- |
| Starting a session: the schedule cache read               | 0      | 1     |
| Starting a session: the schedule cache written            | 1      | 0     |
| Opening the history: the schedule cache read              | 0      | 1     |
| Opening the history: the schedule cache written           | 1      | 0     |
| The history screen, nothing counted yet: run files opened | 360    | 180   |

A session and the history screen each recomputed the entire log and recorded the total, regardless of cache state; both now query the cache first. The history screen read the log twice — once for previous days and once for upcoming days — and now reads it once and forwards the data.

**A loaded log is sorted once.** De-duplicating the log and sorting it by time is the first step of every history calculation, and four of them each did it: where the answers leave each review card, how much came back, what a day spent, and how long an answer takes. A reading now stores that order, computed on first use. Two of the four use it. The other two are `SpentUnder` and `GetReviewed` in `usecase/flashcards/budget.go` and `usecase/flashcards/curve.go`, and `Costed` and `CostedUnder` in `flashcards/review/cost.go`, which still sort the answers themselves; each can be changed to use the reading's order in one line, and each of them still includes the cost of sorting 27 000 answers.

**Nothing is shared between two requests.** Opening the window and then one preset tab walks the whole vault twice and reads the whole log twice: the vault's contents are read from its deck and card model files at every request, and so are its answers. Most of the time in the table above is this work, and the change did not affect it.

By profile, during initial overview on this vault: **57 % of the request is `ListCardFaces.Execute`** — every deck read and parsed, most of it in the markdown parser — and 7 % is reading the log. Working out the day's budgets is 8 %, of which the cost of an answer is 4 %. A memo kept for the lifetime of a window would remove those first two from a second request.

## The curve of one preset

Recorded 2026-09-01 on the same AMD Ryzen 7 6800U, from `BenchmarkPresetCurve` in `usecase/flashcards`. The vault is the one above — fifty thousand review cards over twenty decks, 27 000 answers in 180 run files — with its decks pointing at two presets: one deck points at the first and the other nineteen at the second. Each row is the curve of one of those presets, drawn under a goal of minutes, with the schedule cache filled before the clock starts: opening the window and then a preset tab.

|                                         | Before            | After             |
| --------------------------------------- | ----------------- | ----------------- |
| A preset holding one deck of the twenty | 2.84 s · 1.21 GB  | 2.07 s · 0.83 GB  |
| A preset holding nineteen of them       | 44.7 s · 13.31 GB | 42.4 s · 13.30 GB |

Both columns are the median of two runs of three. The memory column is the one these are read on: the times of the second row spread by a quarter between runs, and its memory by under 0.1 %.

**A curve reads the decks pointing at its preset.** The index returns what links to one note, so a preset of one deck opens one deck file and the other nineteen are never read. The second row is the same request where there is nothing to leave out — nineteen decks of twenty — and it serves as the baseline for the first.

**The projection cost scales with the review cards the preset holds.** Twenty-five places of the grid, each of them a run of the scheduler, and a second run at each place for the day the material is learned. It is the whole of the second row and most of the first, and it is what the section below reduces.

**A curve does not query the schedule cache.** That cache is keyed by the assignment of the whole vault, and a curve covers the review cards of one preset, so the answers are replayed for the faces the curve covers. Reading the log is on this path in any case: what an answer costs and what the day has already spent are read from the answers themselves, cache or no cache.

**The curve of the defaults reads every deck.** No note links to a preset that exists in no note, so only the deck files show which decks name no preset, and that one curve costs as much as the initial overview.

## What a curve costs as the preset grows

Recorded 2026-09-01 on the same AMD Ryzen 7 6800U, from `BenchmarkCurveCards` in `usecase/flashcards`. Twenty decks with every one of them pointing at one preset, so the curve is drawn over the whole vault; the log grows with the vault at about half an answer a review card, in 180 run files. The schedule cache is filled before the clock starts.

| review cards | Before            | After             |
| ------------ | ----------------- | ----------------- |
| 500          | 0.42 s · 158 MB   | 0.31 s · 145 MB   |
| 5 000        | 3.90 s · 1.43 GB  | 2.67 s · 1.29 GB  |
| 20 000       | 15.5 s · 5.51 GB  | 9.9 s · 4.86 GB   |
| 50 000       | 40.4 s · 14.08 GB | 23.4 s · 11.68 GB |

Both columns are the median of two runs of three on an idle machine, where the two runs of a row were within 7 % of each other on the clock and within 0.01 % on the memory. The memory column is the one these are read on: on a vault where a change cannot help, a row's time changes by a quarter between runs while its memory changes by tenths of a per cent. The last row is 105.1 million allocations before and 92.2 million after.

**A curve is fifty-one runs of the scheduler.** One at each of twenty-five places of the grid, one more at each place for the day the material is learned, and one at the top of the range to find how far the range extends. Each of them simulates ninety days, and each of those days used to iterate over every review card of the preset four times: to find what was due, to count what was learned, to count what the day deferred, and to count what comes back. At fifty thousand review cards that is nine hundred million card reviews per request.

**Three of the four are removed.** A review card is indexed by the day of review its schedule falls on, so each day reads only the cards that fell on it: the cards the day before did not reach and the cards that fall due on this day, combined into one run in the order they became due, and the other cards are not read. The number of cards the day leaves unreached is then the number that fell due less the number it reached, which is a subtraction. Whether a card counts as learned under an interval rule depends only on the card's schedule, so the count is kept from day to day and recomputed only for the review cards the day answered; under a chance-of-recall rule it depends on the instant, and is read from the same number as the share that comes back.

**The fourth remains and is 32 % of the request** at twenty thousand review cards by profile. The share of the material that comes back is a fact about every review card at every instant, so it is the one thing a day still counts over the whole preset. A curve reads one day of that series from each run, but the run fills ninety days. Filling only the days a caller requests changes what a projection guarantees and is not just an optimization, and it is described in the section below.

**Both outcomes of an answer come from one computation of the card.** A projection compares the outcome where the card was recalled with the outcome where it was not, and requested each from the scheduler separately. A review card the scheduler has moved to the review state is computed for every rating in one pass, so both outcomes are requested together and the second costs nothing. A review card that is still being learned is computed one rating at a time and is requested one rating at a time: requesting all four there costs more than it saves, and measured 5.40 GB against 4.86 GB at twenty thousand review cards.

**What is left is the scheduler's own arithmetic.** `Simulation.Run` is 97 % of the request, and half of that is the answers themselves. Nine tenths of what a request allocates is allocated inside that call by the scheduling library, which builds a table of every rating for every card it is asked about.

**Values that do not change over the twenty-five places were not changed.** The order of the review cards, how loaded each day already is, how many have had their day and were not answered on it, and how many are learned as the run opens are computed once per run and could be computed once per curve. They are 2.7 % of the request together, and passing them in from outside would add a second path to the same result.

## Computing only the requested days of a projection

Recorded 2026-09-01 on the same AMD Ryzen 7 6800U, from `BenchmarkCurveCards` in `usecase/flashcards`, over the vault of the section above and with the same schedule cache filled before the clock starts.

| review cards | Before            | After             |
| ------------ | ----------------- | ----------------- |
| 500          | 0.29 s · 145 MB   | 0.23 s · 145 MB   |
| 5 000        | 2.59 s · 1.29 GB  | 2.30 s · 1.29 GB  |
| 20 000       | 9.5 s · 4.86 GB   | 6.6 s · 4.86 GB   |
| 50 000       | 24.4 s · 11.68 GB | 15.5 s · 11.68 GB |

Both columns are the median of two runs of three on an idle machine. **The clock is the column these are read on**, which is the other way round from every row above: what is removed is arithmetic over a slice that is already there, so it allocates nothing, and the memory of every row moved by under a hundredth of a per cent — 92.17 million allocations at fifty thousand review cards either way.

The two runs of a row landed within 8 % of each other on the clock, and the last two rows within 1 %. This table relies on those two rows; the 5 000 row's two runs after the change are a fifth apart, and it is the least reliable row.

**A projection is given the days it must compute.** A run receives the days whose returning share it computes, and the projection stores a share for those days only. An unrequested day is not computed and yields no value.

**A curve requests one day per run**, the day the place is read on: the last day of the horizon under a goal of minutes or of retention, and the day the place represents under a goal of a date. The twenty-five second runs that ask when the material is learned request no day at all. Fifty-one runs of ninety days were filling four and a half thousand days of it; they now fill twenty-five.

**What the requested days total is unchanged.** The projections written down over four materials and eight settings — every scalar and every series, day by day — are byte-identical, their runs requesting every day.

**Where a review card counts as learned by a chance of recall, the full walk remains.** That count is a fact about the instant and is read off the same numbers as the share, so such a day iterates over the whole material whether the share was requested or not. The saving applies to a preset counting by an interval, where the count is kept from day to day and the walk is made only on a day that was named.

**What is left is the scheduler's own arithmetic.** By profile at twenty thousand review cards, `Simulation.Run` is 96 % of the request and the computation that closes a day is 0.2 % of it. That is the pass that was 32 %.

## What the scheduler computes

Recorded 2026-09-01 on the same AMD Ryzen 7 6800U, from `BenchmarkCurveCards` in `usecase/flashcards`, over the vault of the section above and with the same schedule cache filled before the clock starts.

| review cards | Before            | After            |
| ------------ | ----------------- | ---------------- |
| 500          | 0.27 s · 145 MB   | 0.12 s · 19.6 MB |
| 5 000        | 2.17 s · 1.29 GB  | 1.01 s · 150 MB  |
| 20 000       | 7.4 s · 4.86 GB   | 3.5 s · 592 MB   |
| 50 000       | 17.6 s · 11.68 GB | 8.8 s · 1.52 GB  |

Both columns are the median of two runs of three on an idle machine. **The allocation is the column these are read on.** It is what the change removes, and it is steady: the two runs of a row are within 0.05 % of each other everywhere. The clock halves at every size, which is far outside the spread of the runs behind it — 3 % or less at every size but 5 000, whose two runs after the change are a sixth apart. **That row is no evidence about the clock** and is here for its memory.

**The Before column is this machine on this day.** Its 50 000 row reads 17.6 s where the section above wrote 15.5 s, on the same build and the same allocation to a hundredth of a per cent. It is the baseline for the After column and should not be used as a figure elsewhere.

**A card is computed, not tabulated.** The scheduling library computes a card by building a scheduler, a map, and an entry for each of the four ratings, and formats a string for a fuzz seed on every call — these parameters use no fuzz and nothing reads the seed. A projection needs two ratings. The arithmetic is now implemented next to the port, using the library's own weights: at fifty thousand review cards a request allocated 92.2 million times and now allocates 3.3 million.

**87 % was the estimate and 87 % is the measurement.** A profile before the change showed 87 % of every byte a request allocates inside the library. The four rows totaled 86.5, 88.3, 87.8 and 87.0 per cent.

**Good is computed from hard.** The interval for a good answer is one day longer than the interval for a hard answer, so the two outcomes a projection compares also need the hard outcome's stability. Easy is computed only where an easy answer is requested.

**A day count is never below zero.** The days a review card has been away are the whole days elapsed. A review card queried at an instant no later than its last answer has zero elapsed days, and the forgetting curve is read at zero.

That is the one place the arithmetic here differs from the library, which converts those days to an unsigned integer and treats such a review card as one that cannot be recalled. The resulting value depends on machine architecture, so the same review card may evaluate differently across architectures.

**No schedule stored in a vault triggers this.** A due day is always computed forward from the time of an answer, so a review card is never queried before its own last answer: over a curve of five thousand review cards the days away were read 602 102 times and never fell below zero, and the same vault with twenty answers timestamped a week ahead by a wrong clock reads the same 0. Triggering it needs a due day that precedes the answer that produced it, which is what the recorded projections construct by hand — 96 of their 10 523 readings, 36 of them where the phase makes it count.

**What is left is arithmetic.** By profile at twenty thousand review cards, `Simulation.Run` is 77 % of the request, the answers themselves are 49 %, and the scheduler is 30 %. `math.Pow` and `math.Exp` together are 22 %, and selecting the day a card is scheduled on is 15 %. Reading the vault, which was 4 %, is now 7 % of a shorter request.

## Every place of a curve at once

Recorded 2026-09-01 on the same AMD Ryzen 7 6800U, which has eight cores and sixteen threads, from `BenchmarkCurveCards` in `usecase/flashcards`, over the vault of the section above and with the same schedule cache filled before the clock starts.

| review cards | Before           | After            |
| ------------ | ---------------- | ---------------- |
| 500          | 0.12 s · 19.6 MB | 0.08 s · 19.7 MB |
| 5 000        | 1.01 s · 150 MB  | 0.29 s · 150 MB  |
| 20 000       | 3.5 s · 592 MB   | 1.06 s · 592 MB  |
| 50 000       | 8.8 s · 1.52 GB  | 2.5 s · 1.52 GB  |

Both columns are the median of two runs of three on an idle machine. **The clock is the column these are read on, and the allocation is the control.** The same work on other cores allocates the same bytes, and three of the four rows moved by under 0.05 %; a memory figure that had changed would have shown that something changed unintentionally. The 500 row's allocation is 0.4 % higher, which comes from the stacks of the goroutines that a request of eighty milliseconds now starts.

**The 500 row is the least reliable.** Its two runs after the change are a sixth apart, and it is included to show that the smallest preset did not get slower. The factor beside it should not be quoted.

**Three and a half times, on eight cores.** The estimate before the measurement was 2.8, from the 77 % of a request spent in the runs and eight cores to run them. The measurement is 3.4 at twenty thousand review cards and 3.5 at fifty thousand, so threads beyond the eight cores still help arithmetic that does not wait for memory.

**Twenty-five places over sixteen threads is two waves, and the second is smaller.** A curve is twenty-five places and each is two runs of the scheduler, each assigned to one thread. That uneven split, and the vault read that is not shared between places, are most of what limits the speedup to 3.5× out of a possible 8×.

**A place holds the preset's cards while it runs.** Peak memory at fifty thousand review cards is 164 MB with one place running at a time and 307 MB with all of them, sampled from the kernel's own high-water mark over the whole benchmark. A run holds about four and a half megabytes of cards at that size, and a request now holds as many of those as the machine has threads.

**What a person waits for.** A preset tab over fifty thousand review cards responds in two and a half to three seconds: four runs of three on this machine spread from 2.47 s to 2.99 s, and allocated the same 1.52 GB to within 0.02 % every time. The work is the same work and the spread is the clock, so three seconds is the limit to hold this to. It was 8.8 s before this change, 17.6 s before the one above it, and 40.4 s at the top of the section three above. Twenty thousand review cards is a second, and five thousand is under a third of one.

## What a window requests from every vault

Recorded 2026-09-01 on the same AMD Ryzen 7 6800U, from `BenchmarkFrontDoor` in `adapter/window/flashcards`. The installation is generated: four vaults, each of five thousand review cards over twenty decks, answered a hundred times a day for sixty days — 6 000 answers in 60 run files a vault. The schedule caches are filled before the clock starts, which is a person's second opening of a day.

|                                                                       | Measured 2026-09-01 | Measured 2026-09-06   |
| --------------------------------------------------------------------- | ------------------- | --------------------- |
| Every vault counted inside one answer — what the window loaded before | 0.52 s              | 0.52 s (6, 0.52–0.66) |
| The list of vaults on screen                                          | 1.3 ms              | 1.0 ms (6, 1.0–3.3)   |
| The last of the four counts arriving                                  | 0.21 s              | 0.20 s (6, 0.20–0.29) |

The first column is the median of two runs of five. The second is six runs of five taken on a machine carrying five other agents' work, quoted at the best of the six with the spread beside it.

**Nothing has changed.** Every row's best run matches the recorded figure, and the whole of the spread above it comes from the machine: the list on screen varies threefold between runs because it is a millisecond of work, and the two counts vary by a quarter. Taken together, the three rows are unchanged.

The first row is the legacy initial load, run here as it was: the four vaults counted one after another before anything was returned.

**What a person waits for is a read of the registry.** The list contains names and paths, which the registry returns before any database is opened, so it is on screen in about a millisecond whatever the vaults hold. Each later value is shown in its own row when it arrives.

**Four counts at once cost the slowest, not the sum.** The four vaults here are the same size, so the last count finishes in about a quarter of what counting them in turn took, less what they spend competing for the same cores. An installation of one vault waits exactly as long as it did.

These results do not show what a vault of fifty thousand review cards does to the row beside it: these four are alike on purpose, and the claim being measured is the shape and not the spread.

## What a window costs before a person can type

Recorded 2026-09-06 on the same AMD Ryzen 7 6800U, over the flashcards window's own start: the settings read, the registry opened, the index opened, the use cases assembled, the themes read, the pages handler built, and the first page fetched from it. Three runs of each part; the spread is quoted because other work was running on the machine throughout.

**A window is not measured here, and cannot be.** What a person waits for is the process starting, the Go side assembling, WebKit creating a webview, and the bundle being parsed and run — and the three that are not Go need a window on a screen. What is below is the Go side alone, which is the part a change to the composition root could change.

|                                              | Measured                | Allocations    |
| -------------------------------------------- | ----------------------- | -------------- |
| The settings read                            | 0.13 ms (0.13–0.15)     | 14.9 kB · 164  |
| The registry opened                          | 50 ns (49–63)           | 24 B · 1       |
| The index opened, no file there yet          | 13.7 ms (13.7–16.4)     | 64 kB · 608    |
| The index opened, the file already there     | 1.07 ms (1.07–1.32)     | 36 kB · 164    |
| The themes read                              | 13.0 µs (13.0–18.3)     | 3.0 kB · 20    |
| The pages handler built                      | 0.55 µs (0.55–0.96)     | 120 B · 6      |
| The use cases assembled                      | 0.26 ms (0.26–0.44)     | 19 kB · 270    |
| The first page served                        | 0.73 ms (0.73–1.13)     | 24 kB · 138    |
| **All of it, first launch on a new machine** | **31.9 ms (31.9–41.0)** | 236 kB · 2 553 |

**The Go side takes thirty milliseconds and creating the schema is most of it.** Opening an index that is not there yet runs the migration file 0001_index.sql (measured as the ten files it replaced), and that was 14 ms of the 32; on every launch after the first it is one millisecond. Assembling every use case the window serves is a quarter of a millisecond, and building the handler that serves the pages is half a microsecond. Nothing changed by that night's work can be measured here.

**So this is not what a person waits for.** A launch on an existing installation spends under 20 ms in Go before the first byte of the page. The rest of the wait is the webview and the bundle the section below measures, and neither is affected by anything in `container`.

The harness for this is not in the tree. It was taken with a benchmark written against `container` and the flashcards window package and then removed, so the table cannot be reproduced from a checkout as it is now. Every path it used — the index, the registry, the settings, the themes, the schedules — was pointed at a temporary folder, so nothing on the machine was read.

## What the two windows are built into

Recorded 2026-09-06, `vite build` in each window's own folder, into the Go package that embeds it. The figure is the one file the window loads.

|            | JavaScript | gzip   | CSS    | modules |
| ---------- | ---------- | ------ | ------ | ------- |
| flashcards | 507 kB     | 176 kB | 100 kB | 726     |
| editor     | 1 631 kB   | 562 kB | 111 kB | 2 601   |

**The flashcards window's size reduction held.** It was 1 180 kB before `preserveModules` and `sideEffects` were both added to the interface library and 506 kB after; it builds at 507 kB today, one kilobyte more from the day's own changes.

**The editor size did not change, as expected.** It was 1 628 kB and is 1 631 kB. It uses more of the library and more of its own, and the tree shaking that took two thirds off the smaller window takes almost nothing off this one: what the editor imports, it uses.

The editor's `vue-tsc` step did not pass with that install, so its figure comes from `vite build` alone. Two copies of `@vue/runtime-core` were there — one under the editor and one under the interface library — and every component's props typechecked against the wrong one. That was caused by the contents of `node_modules` and not by the source, and the bundle was unaffected, since Vite resolves one copy.

### The same two, off one install

Recorded 2026-09-06, the same command, with `modules/` an npm workspace root.

|            | JavaScript | gzip   | CSS    | modules |
| ---------- | ---------- | ------ | ------ | ------- |
| flashcards | 432 kB     | 155 kB | 101 kB | 571     |
| editor     | 1 556 kB   | 543 kB | 112 kB | 2 476   |

**Each window contained the protobuf runtime twice and now contains it once.** Eleven installs meant `@numen/protocol` resolved `@bufbuild/protobuf` out of its own `node_modules` and the window resolved another out of its own, so both were bundled: `google/protobuf/descriptor.proto` appears once in each bundle now and appeared twice in each before. It is 76 kB off both windows, which is the whole reduction — nothing else about either window changed.

The two copies of `@vue/runtime-core` were also reduced to one, and the editor's `vue-tsc` passes.

## A recipe that changed, and whether it is stable

The quantisation scale now is in the recipe a vector is kept under, so every vector made before it is made again. Read 2026-09-06, from the code rather than the clock.

**It is stable.** `embedding.Model.Recipe` is a `Sprintf` over the model's five fields, the name of the quantisation and `embedding.Int8Scale`. The five fields come from the settings file; the other two are constants. Nothing in it is read from the clock, from a random source, or from a path that differs between launches, so two runs of one installation compute the same recipe and the second finds every vector the first computed. The cost is incurred once, and `TestAScaleThatMovedBuysTheVectorsAgain` is the test that enforces this.

**What it does not fix is the disk space.** `vectors` is keyed by the text's hash and the recipe together, and `forget_vector` removes a row only where no chunk has that text at all. A chunk whose text did not change keeps its old-recipe row beside its new one, so a scale that changes doubles the disk space used by the vectors. At the corpus measured under "Dimensions cost more than bits" that is 149 MB becoming 298 MB.

**The old rows are kept, and rebuilding the index removes them.** Keeping them is the reason the recipe is in the key: a person who sets the model, the width or the scale back finds every vector still stored and computes none of them again, which saves minutes of machine time or money and network use. Nothing removes them automatically. `numen-cli scan <vault> --rebuild-index` — the same request the window makes under its own name — deletes the vectors of every recipe except the one in use and vacuums the file, so the space is reclaimed when a person confirms the old model will not be used again. That is the only way to reclaim the space, and it is the answer to the store's growth: **the vectors grow by their own size each time a recipe changes, and shrink only on a rebuild.**

## What a model session runs best on

The sweep times one `Run` of each ONNX session the application opens — the Parakeet encoder, decoder and joiner, the Silero voice-activity model, PaddleOCR detection and recognition, and an embedding model — on every execution provider, compute unit and thread count the machine offers. It opens each model file directly and builds its inputs from the input metadata the file declares, so it measures only the session. The session is created and run a few times before the timer starts, so a compile on the first run is not in the result.

A model is recognised by the inputs and outputs it declares and not by its name: a PaddleOCR file with one image input and a rank-4 heat map as output is detection, one with rank-3 scores over the characters is recognition, and one with `im_shape` and `image` inputs is the layout model, which the sweep does not time. The thread counts are 1, 2, 4, half the physical cores, the physical cores and the logical cores of the machine the sweep runs on. The compute units (`ALL`, `CPUAndNeuralEngine`, `CPUAndGPU`, `CPUOnly`) are swept for CoreML only; `SessionSettings.ComputeUnits` leaves the default at `ALL`.

```
cd libs/core/kernel
internal/adapterlib/onnxruntime/bench-sweep.sh <output dir> [max load]
benchstat <output dir>/cpu_default_t1.txt <output dir>/cpu_default_t8.txt
```

The script refuses to start while the 1-minute load average is above the limit it is given (a quarter of the logical CPUs by default), writes `header.txt` with the machine's facts, and writes one file per provider, compute units and thread count. `NUMEN_BENCH_MODELS` names the directory of model files (the model cache by default), `NUMEN_BENCH_EMBEDDING_MODEL` a directory holding an embedding `model.onnx`, and `NUMEN_BENCH_RUNTIME` the `libonnxruntime` to open. A model that is missing, or a provider the machine does not have, skips its benchmark with a message. A single configuration is run by hand with `NUMEN_BENCH_PROVIDER`, `NUMEN_BENCH_THREADS` and `NUMEN_BENCH_COMPUTE`:

```
NUMEN_BENCH_PROVIDER=cpu NUMEN_BENCH_THREADS=4 go test ./internal/adapterlib/onnxruntime/ -run XXX -bench Device -benchtime 1x
```
