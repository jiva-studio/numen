# The vector index stays inside SQLite

- **Status:** Accepted
- **Date:** 2026-08-25
- **Applies to:** `modules/libs/core`
- **Related:** [One database for all vaults, outside them](0002-one-database-for-all-vaults.md), [A hexagonal core in Go](0004-a-hexagonal-core-in-go.md), [What the index stores](0006-what-the-index-stores.md), [A schema change is a numbered migration](0007-a-schema-change-is-a-numbered-migration.md), [A source is text in one table](0010-a-source-is-text-in-one-table.md), [Text is cut twice](0011-text-is-cut-twice.md), [A chunk is identified by its text](0012-a-chunk-is-identified-by-its-text.md), [One search, three rankings, merged by rank](0014-one-search-three-rankings.md)

## Context

Everything else the index holds is a reading of files on disk, and a scan puts it back for the price of reading them. A vector is not: it is bought, once, from a model. Where the vectors live, what shape they are stored in and what makes one stale are one decision, and it is taken on a desktop machine that is running everything else the person is doing.

## Decision

### The vector index lives in the one database

Vectors are searched through the SQLite vector extension, in the same file as every other table, under the same transaction.

**A vector index this application uses reads from disk.** Any index that must be resident in memory to be searched is disqualified, whatever its speed.

### The driver carries the vector extension

`modernc.org/sqlite`, at or above **v1.50.0**, which is where the bundled vector extension arrives; `go.mod` names v1.56.0. The floor is a requirement: below it the vector index is not in the build, and the failure is a missing SQL function.

### Dimensions are kept and precision is dropped

Dimensions are never reduced. Bits are. Two representations are stored: one bit per dimension for the coarse pass over everything, and one byte per dimension for the rerank over the candidates the coarse pass kept. The coarse pass is expected not to lose the answer, so it keeps several times more candidates than the result needs.

The scale that turns a float into a byte belongs to the model and is written down with its name.

The coarse representation is addressed by the chunk's own row number, which is the only key the vector index has, and it is written in the same transaction as the vector it belongs to.

### A vector is kept by its text and its recipe, and it outlives the chunk

The `vectors` table is keyed by the hash of the text that was embedded and the recipe it was embedded under. The recipe names everything that decides what the vector is: where it was made, the model, its width, where the text is cut off, how the model's output becomes one vector, and how the numbers are stored. Change any of them and the old rows are not found, and they are found again if the setting goes back — unless a migration emptied the table in the meantime.

Where it was made is the placement that fills the index: a model run on this machine, addressed by the weights that are run, or a service, addressed by the base URL and the name asked for there. One model name is run here and served by more than one place, and the numbers each of them gives for one text are its own. A question placed elsewhere is answered from the rows the index was filled with, and what holds the two together is that they are compared as one model when both are open.

A vector is kept where no renumbering of chunks and no rebuilding of the index reaches it. A chunk whose text was embedded before is not asked for again, whichever row now holds that text.

**A vector is forgotten only where a source was cut again and no chunk holds that text any more.** A source whose folder could not be read takes nothing with it, so an unreadable folder and a deleted one leave the same thing behind, and a vault on a detached drive loses nothing bought.

### A partition key is only for what a query filters by

The source a passage came from is an ordinary column. Nothing is partitioned by book, by note or by vault. The vault is a metadata column constrained inside the query.

### Three staleness keys, and each is a query

The file changed; the recipe that produced the text changed; the recipe the vectors were made under changed. A source owes its text when its recipe is not one now in use. A chunk owes a vector when no row holds one for its text under the recipe in use.

## Consequences

- A model change invalidates both representations of every vector.
- The coarse pass decides what the rerank can see, and a candidate it drops is lost.
- A vector kept past its chunk is storage nothing points at until the same text is cut again.
- The driver floor is a version the build carries, and going below it is a missing SQL function at run time.
- What the two representations cost to store and to search is in [performance](../performance.md).

## Alternatives considered

**A separate vector service holding an in-memory approximate index.** Rejected: it holds its index in memory to answer at all, and it asks that of the machine somebody is working on; it is also a second process and a second lifetime.

**Compute the quantisation scale from what the corpus holds.** Rejected: a scale that moved would make every vector written before it incomparable with every vector written after, and nothing in the stored bytes would show that it moved.

**Partition the vector index by vault.** Rejected: an unfiltered query pays for every partition it walks, whatever each partition holds.
