# ADR-0019: Performance targets for indexing and search

- **Status:** Accepted, except where noted below
- **Date:** 2026-08-15
- **Applies to:** `modules/apps/desktop`
- **Partly superseded by:** ADR-0030 — two rows of the table below, index size
  and search latency, and nothing else
- **Related:** ADR-0002, ADR-0015, ADR-0018

## Context

ADR-0002 designs for a hundred thousand notes and states budgets: two seconds to
interactive, a hundred milliseconds for an incremental update, single-digit
minutes for a full rebuild. Those numbers were written before there was code,
and for a while nothing measured them — which makes a budget a wish.

There is now a load test at that size, so the budgets can be replaced by
measurements, targets, and a way for anyone to check both.

## Decision

### The targets

| | Target |
| --- | --- |
| Cold scan — every note read, parsed, indexed | under 3 minutes |
| Warm scan — nothing changed; what a startup pays | under 1 second |
| Search, ordinary query, while a scan is writing | p95 under 50 ms |
| Index size | under 5 MB per thousand notes |

> **Two of these rows are superseded by
> [ADR-0030](0030-index-size-and-approximate-search.md).** Index size is
> rebudgeted, and the search target keeps its value while an exact scan of the
> vector index stops meeting it. The other two rows are unaffected.

**What these currently measure is in docs/performance.md, and only there.** The
numbers that justified these targets were written here as well, and by the
second time the load test ran they were wrong in both directions — the scan
faster than recorded, the index larger. A target is a decision and belongs in an
ADR; a measurement has a date on it and belongs where it is taken.

The targets are deliberately looser than the measurements. A target that equals
today's number turns every ordinary change into a failure, and one that is
double leaves room to notice a regression before a user does.

### What is not promised

**A query that matches the entire corpus is not covered.** The same load test
measures 2.1 seconds for one, and that is honest: ranking a hundred thousand
matches takes as long as it takes. Nothing about search is slow — the same
database answers a query matching a hundred notes in 4 ms while being written
to — but a query with no selectivity does linear work, and no target will change
that.

This distinction matters more than it looks. A synthetic vault built from twenty
words makes every query match everything, so a benchmark that uses one measures
ranking and reports it as search. The load test asks both questions on purpose.

### How it is checked

```
cd modules/apps/desktop
NUMEN_LOAD=1 go test ./internal/core/usecase/vault/ -run TestLoad -v -timeout 40m
```

Roughly two minutes and a gigabyte of temporary files. A smaller rehearsal:

```
NUMEN_LOAD=1 NUMEN_LOAD_NOTES=10000 go test ./internal/core/usecase/vault/ -run TestLoad -v
```

**It reports; it does not assert.** A threshold that fails on a slower machine
teaches people to ignore the test, and this one runs on whatever laptop is at
hand. The numbers go in docs/performance.md, where a person compares them with
what was there before.

Run it before a release, and after any change to how notes are read, stored or
queried. Every finding that has moved these numbers so far was invisible in
review and obvious in a measurement; docs/performance.md keeps the list, and the
two most recent were a transaction boundary per note and a database nothing had
ever asked to measure itself.

## Consequences

**Positive**

- "Fast enough" is a number that can be checked rather than an opinion.
- The budgets in ADR-0002 now have evidence behind them, and one of them —
  single-digit minutes — turns out to have been generous rather than tight.
- A regression has somewhere to show up before a user finds it.

**Negative**

- The load test is opt-in, so it is only as useful as the discipline of running
  it. Nothing enforces that, and this ADR is the only thing that says when.
- The measurements come from one machine. They are a baseline for comparison
  with itself, not a specification of what a user's laptop will do.
- Generated notes are not real notes. They are the same length distribution and
  the wrong vocabulary, which is why the search figures need two rows rather
  than one.

## Alternatives considered

**Assert the targets in CI.** Rejected for now: the runners are shared and
variable, and a performance test that fails for reasons unrelated to the change
gets disabled within a month. Worth revisiting with a dedicated machine and
comparison against a stored baseline rather than a fixed threshold.

**Keep the budgets in ADR-0002 and add measurements beside them.** Rejected:
two places would state what fast enough means, and they would disagree. ADR-0002
keeps the reasoning; the numbers live here.
