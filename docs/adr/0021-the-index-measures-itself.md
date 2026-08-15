# ADR-0021: The index measures itself after a scan

- **Status:** Accepted
- **Date:** 2026-08-15
- **Applies to:** `modules/apps/desktop`
- **Related:** ADR-0002, ADR-0011, ADR-0017, ADR-0019

## Context

Asking which notes point at a given one took 58 ms on a vault of ten thousand,
and grew with the vault. The indexes for it exist and the query is written as
single-index lookups, deliberately, because the first version was slow for
exactly the reason a chain of `OR`s is slow.

The plans said the indexes were being used. They were being read from a database
that had been measured — and a database filled by a scan never has been.

SQLite chooses between the ways it could answer a question from what it knows
about how much is stored and how it is spread. Knowing nothing, it falls back on
rules of thumb, and the rule of thumb here picks the index that narrows to the
vault and then reads every link in it — seventy thousand rows to find twenty.
That is not a full table scan and does not look like one: it reports itself as
a search, through a real index, and is invisible in a plan unless the index is
named.

Nothing in the application ever measured the database. Nothing had decided to.

## Decision

**A scan that changed the index measures it afterwards.**

The core states the need — the index has changed enough to be worth measuring
again — and the adapter knows how a database is measured. A scan that indexed
and removed nothing does not measure, because an unchanged vault is scanned at
every startup and that path has a budget of its own (ADR-0019).

The measurement covers the whole database rather than only the tables the
scanning connection happened to read from. A scan writes and does not ask
questions, so a decision scoped to what this connection has read would find
nothing worth measuring and leave the index exactly as slow as before.

**A plan is asserted by the index it uses, not by the absence of a scan.** The
failure this decision exists to prevent passes any test that only forbids
reading a whole table. Tests that check query plans name the index each question
has to be answered through, and they measure the database the way the
application does rather than measuring it themselves — a test that measures on
its own behalf certifies a plan that never reaches a user.

## Consequences

**Positive**

- Backlinks went from 58 ms to 1.8 ms on ten thousand notes, and stopped growing
  with the vault: 1.7 ms on one thousand, 1.8 ms on ten.
- The indexes that already existed are now actually used, which is worth more
  than adding another one would have been.

**Negative**

- A scan does something at the end that has nothing to do with what it found, so
  a use case now depends on a port that exists for the sake of speed rather than
  meaning.
- The measurement is a write, so it queues behind the same single writer as
  everything else.
- It is a sample, not a survey, and a vault whose shape changes without its size
  changing can go unnoticed until the next scan that stores something.

## Alternatives considered

**Measuring when the database is opened.** Rejected: at that moment the last
scan's changes are already in place and unmeasured, so the first session after a
rebuild would answer badly for the whole of its life.

**Measuring on a timer.** Rejected: it makes a background thread out of
something that has one natural moment — the end of the scan that caused it.

**Another index.** Rejected as the first thing to try, and this is the point of
the ADR. The indexes were right; what was missing was the knowledge that they
helped. Two earlier guesses at this problem made it slower, and the fix was
found by reading plans rather than by adding structure.

**Storing where each link resolves instead of resolving on the way in.**
Rejected here, and not by this ADR: resolution is a query rather than stored
state because a file appearing or disappearing changes the answer (ADR-0011).
That decision holds, and it is what makes the plan for this query matter.
