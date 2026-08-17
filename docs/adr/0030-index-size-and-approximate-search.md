# ADR-0030: The index-size budget, and where an exact scan ends

- **Status:** Accepted
- **Date:** 2026-08-17
- **Applies to:** `modules/apps/desktop`
- **Replaces:** ADR-0019, on two of its targets — index size and search
  latency — and on nothing else
- **Related:** ADR-0002, ADR-0006, ADR-0007, ADR-0019, ADR-0029

## Context

ADR-0019's targets were set while the unit of the index was the note: one row,
one full-text entry, no vector. ADR-0007 cuts a note into the same small windows
as a book, and ADR-0006 puts books in the same index. A vault at the size ADR-0002
designs for then holds many times as many chunks as it has notes, each with a row,
a full-text entry and two vectors.

The arithmetic is in docs/performance.md, where it is labelled an extrapolation.
It breaks the index-size target, it breaks the search target, and it reaches the
size ADR-0007 named when it deferred approximate search — "every corpus measured
so far is answered by an exact scan of the coarse representation inside the
budget ADR-0019 sets".

## Decision

### The index is budgeted at ten megabytes per thousand notes

| | Target |
| --- | --- |
| Index size, a vault of notes | under 10 MB per thousand notes |
| Index size, sources | no per-note budget; proportional to the text indexed |

The first row is a third above the extrapolation rather than double it. ADR-0019
sets targets at roughly twice what is measured, and an extrapolation is not a
measurement: doubling a number nobody has observed is not a budget. The first real
measurement of a chunked vault is what revises this row.

The second row exists because a book's contribution is proportional to the book
and not to the vault it was dropped into. A library has no note count to be
divided by, and what one corpus of books costs is in docs/performance.md.

### Search keeps its target and stops being exact

| | Target |
| --- | --- |
| Search, ordinary query, while a scan is writing | p95 under 50 ms |

The number is ADR-0019's and does not move. What changes is what meets it: **an
exact scan of the coarse representation no longer does** at the number of chunks a
vault of this size holds.

**Vector search is therefore approximate**, and three rules bind whatever
structure provides it:

- **It reads from disk.** ADR-0029 disqualifies an index that has to be resident
  to be searched, and that is not reopened here.
- **It narrows before it compares, on a value every query constrains.** ADR-0007
  allows a partition key only for such a value. A cluster the query itself picks
  is one; a source, a book or a vault is not.
- **Recall is checked by rank.** ADR-0007's acceptance set is what an
  approximation has to pass. A structure that meets the latency target and moves a
  known answer out of reach has failed, and only that test sees it.

How many clusters there are, and how many a query probes, are configuration
measured against the acceptance set.

**The lexical half stays exact.** Full-text search over chunks answers over more
rows and answers them the same way.

### What is replaced in ADR-0019, and what is not

Two rows of its table. Its cold-scan and warm-scan targets stand, its statement
that a query matching the whole corpus is not covered stands, its procedure
stands, and so does its rule that a target belongs in an ADR while a measurement
belongs where it is taken — which is why the arithmetic behind this decision is in
docs/performance.md and not here.

## Consequences

**Positive**

- The budget is a number that can be checked again, against an index that will
  exist.
- Approximate search arrives with rules, so the first implementation is measured
  against them instead of choosing for itself.
- The acceptance set becomes the gate on the search structure and not only on how
  text is cut.

**Negative**

- A cache at this budget is a different kind of object from the note index alone,
  and deleting it costs embedding where it used to cost scanning. ADR-0002 keeps
  that survivable by making the vector half optional and letting it fill in behind
  the rest.
- An approximate index can lose an answer that an exact scan would return, and
  nothing but the acceptance set notices. That set is small, and its size is in
  docs/performance.md.
- The budget is set against an extrapolation from a corpus of books, applied to a
  vault of generated notes. The first measurement of the real thing may move it
  again.

## Alternatives considered

**Keep the 5 MB target and store less.** Rejected: what there is to drop is the
byte-precision vectors, which ADR-0007 reranks with, or the chunk-level full-text
index, which leaves the lexical ranking over a different unit from the dense one
and nothing to merge by rank.

**Keep the exact scan and let search take longer.** Rejected: the latency target
was set from what a person will wait through, not from what was easy to reach, and
an exact scan is the part of search that grows with every book added.

**Budget per chunk instead of per note.** Rejected: a person has notes, and how
many chunks one becomes is a window size this application chooses on their behalf.
A budget stated in chunks moves whenever that size is tuned.
