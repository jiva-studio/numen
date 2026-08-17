# ADR-0007: Structural chunking, and how a passage is found

- **Status:** Accepted
- **Date:** 2026-08-17
- **Extended:** 2026-08-17 — how lexical and dense search are combined
- **Applies to:** `modules/apps/desktop`
- **Related:** ADR-0000, ADR-0002, ADR-0006, ADR-0016, ADR-0019, ADR-0030

## Context

A source layer that answers "where does this idea appear" needs the text cut
into pieces small enough to embed and an index that finds the right piece. Both
halves were open: ADR-0016 stores headings and names them the boundaries
chunking will cut on, and ADR-0002 names `sqlite-vec` and its partition key
without deciding how either is used.

The decisions below were taken against measurements on the Ganguli Mahābhārata,
the BORI critical edition, and the Russian academic translation — three
languages, several sources, a corpus large enough that the wrong answer is
visible. What was measured, and on what date, is in docs/performance.md.

One finding shaped the rest and is worth stating before the decisions that
follow: **a vector describes its whole window, so how the text is cut decides
what can be found at all.** A sentence that answers a question exactly, sitting
inside a window that is mostly about something else, is not retrievable — not
because the index lost it, but because the window it belongs to is not similar
to the question. Chunking is therefore a retrieval decision and not a
formatting one.

## Decision

### The text is cut twice

**A small window is searched; the large window enclosing it is read.**

Both cuts follow the structure the file already carries — headings in a note,
the sections of a translation, the chapter its own numbering names in a verse
edition — and windows run inside a span, never across one.

The small window is what carries a vector. The large window is what a result
shows, so a hit arrives with enough text around it to be understood. Several
small windows of one large window collapse to it, keeping the best score.

Sizes are configuration, not decision, with two rules that are:

- **A window is bounded in words, never in lines or bytes.** A file that puts a
  whole book on four lines must not produce a chunk the size of the book.
- **The small window is checked against the model's input limit** and stays
  under it with margin. A window that is silently truncated indexes text it does
  not contain.

### Everything indexed is cut the same way

A note is a short document. It is cut into the same small windows as a book, and
its large window is the note itself.

This is what keeps a mixed corpus rankable. Similarity varies with the length of
what is embedded, so entries of different lengths cannot be ordered against each
other. Uniform cutting removes the problem instead of correcting for it
afterwards.

**A title is not embedded.** A title is too short to place, and the lexical
index already matches it.

### The vector keeps its dimensions and loses its precision

**Dimensions are never reduced. Bits are.** At equal storage, every dimension at
low precision retrieves far more than few dimensions at high precision, and
truncating a vector this model produces discards arbitrary directions rather
than unimportant ones.

Two representations are stored, and full precision is not one of them:

| Stored | Used for |
| --- | --- |
| One bit per dimension | The coarse pass, over everything |
| One byte per dimension | The rerank, over the candidates the coarse pass kept |

The coarse pass is not expected to be right. It is expected not to lose the
answer, which is why it keeps several times more candidates than the result
needs.

### A partition key is only for what a query filters by

An unfiltered query pays for every partition it walks, and that cost is
independent of how much each partition holds. A key that queries do not filter
by is therefore a tax on every search.

**The source a passage came from is an ordinary column, not a partition key.**
Nothing is partitioned by book, by note or by vault at the vector index. If a
partition key is introduced later it will be for a value every query constrains,
and the number of partitions is then a design parameter rather than a
consequence of how many books there are.

### There is one search, and the two indexes are parameters of it

Lexical and dense retrieval are not two searches. One search takes the query and
says which halves to run and how many candidates each keeps, and returns one kind
of result.

**Both indexes are built over the same unit: the chunk.** A lexical hit and a
dense hit name the same row, so they can be placed in one list and a hit can be
read back the same way whichever half found it.

**The two rankings are merged by rank.** A chunk's score is the sum of
`1 / (k + rank)` over the rankings that returned it. BM25 and cosine similarity
are not comparable — one is unbounded and drawn from the corpus, the other is a
bounded angle — so combining the scores means calibrating them per corpus and per
model, and combining the ranks means calibrating nothing.

**The lexical index stays.** Dense search loses the exact term: a name, a
citation, a phrase quoted the way it is written. Anything spelled precisely is a
lexical question, and no window size makes it a dense one.

### A result names one document once

Several passages of one document collapse to one result. Without this a single
document takes the whole answer, which is a failure of the answer and not of the
ranking.

### Acceptance is a rank, not a latency

Search is checked by a set of questions whose answers are known, and what is
checked is **where the known answer ranks**. A change that leaves latency and
index size untouched can still move the right passage out of reach, and only
this test sees it.

## Not decided yet

**Approximate search.** Every corpus measured so far is answered by an exact
scan of the coarse representation inside the budget ADR-0019 sets. The size at
which that stops being true is known; what replaces it is not decided, and
deciding it before it is needed would be guessing.

> **Decided by [ADR-0030](0030-index-size-and-approximate-search.md).** Cutting
> notes into chunks reaches that size, so vector search is approximate and the
> rules it obeys are stated there.

**Scripts other than Latin and Cyrillic.** Transliterated Sanskrit is indexed
and does not surface. Whether the cause is the model, the transliteration or the
way verse is cut is not established.

## Consequences

**Positive**

- What can be found is a property of one decision — how the text is cut — rather
  than of several interacting ones.
- Notes and sources share an index, a ranking and a code path.
- Merging by rank keeps the two halves independent: either ranking can change
  without the other being tuned again.
- Dropping full precision makes the vector part of the index a quarter of its
  size, with no visible change to what comes back.
- The acceptance test catches the class of regression that latency and size
  cannot see.

**Negative**

- Two cuts of the same text are stored, and the small one multiplies the number
  of vectors several times over.
- Sizes tuned against one corpus are not guaranteed on another; the acceptance
  set is what makes a bad choice visible, and it has to be maintained.
- A model change invalidates both representations, and ADR-0000 already treats
  that as a full recomputation.

## Alternatives considered

**One window size, chosen as a compromise.** Rejected on measurement: the sizes
that keep a specific sentence findable are too small to read, and the sizes that
read well bury it.

**A smaller embedding, from a model that supports it.** Not rejected — untested.
It would remove the second representation, and the model in use is not one that
allows it. Worth revisiting when the model is chosen rather than inherited.

**Correcting for length instead of cutting uniformly.** Rejected: it adds a
calibration to maintain per model and per corpus, where cutting everything the
same way removes the effect entirely.

**Partitioning by book, so that adding one touches nothing else.** Rejected for
the search this application does, which is across everything rather than within
one source. It remains the right shape for a search that always names its
source, and nothing here prevents that being reconsidered.
