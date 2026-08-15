# ADR-0021: The index measures itself after a scan

- **Status:** Accepted
- **Date:** 2026-08-15
- **Applies to:** `modules/apps/desktop`
- **Related:** ADR-0002, ADR-0011, ADR-0017, ADR-0019

## Context

SQLite chooses between the ways it could answer a question from what it knows
about how much is stored and how it is spread. Knowing nothing, it chooses by
rule of thumb, and the rule of thumb for "which notes point at this one" is to
narrow to the vault and then read every link in it.

A database filled by a scan knows nothing. Nothing in the application ever asked
it to measure itself, because nothing had decided that it should.

That failure is invisible in a query plan unless the plan is read closely: it is
not a table scan, it is a search through a real index, and it looks reasonable.

## Decision

**A scan that changed the index measures it afterwards.**

The core states the need — the index has changed wholesale — and the adapter
knows how a database is measured. A scan that indexed and removed nothing does
not measure, because an unchanged vault is scanned at every startup and that
path has a budget of its own (ADR-0019).

The measurement samples large tables rather than reading them, and covers the
whole database rather than only the tables the scanning connection read from. A
scan writes and asks nothing, on whichever pooled connection was free, so what
that connection has read is an accident.

**A plan is asserted by the index it uses, not by the absence of a scan.** Tests
that check query plans name the index each question has to be answered through,
and they measure the database the way the application does. A test that measures
on its own behalf certifies a plan that never reaches a user.

## Consequences

**Positive**

- Backlinks are answered in about a millisecond and do not grow with the vault.
- The indexes that already existed are used, which is worth more than adding
  another would have been.

**Negative**

- A scan ends with something that has nothing to do with what it found, so a use
  case depends on a port that exists for speed rather than for meaning.
- The measurement is a write, and queues behind the same single writer as
  everything else.
- It is a sample, so a vault whose shape changes without its size changing can
  go unnoticed until the next scan that stores something.
- Which parts of the measurement are requested is a constant no test can defend:
  several spellings produce the same plans, and only a measurement tells the
  cheap one from the expensive one.

## Alternatives considered

**Measuring when the database is opened.** Rejected: the last scan's changes are
already in place and unmeasured by then, so the first session after a rebuild
would answer badly for the whole of its life.

**Measuring on a timer.** Rejected: it makes a background thread out of
something with one natural moment.

**Another index.** Rejected, and this is the point of the ADR: the indexes were
right, and what was missing was the knowledge that they help.

**Storing where each link resolves.** Rejected elsewhere and not by this ADR:
resolution is a query because a file appearing or disappearing changes the
answer (ADR-0011). That is what makes the plan for this question matter.
