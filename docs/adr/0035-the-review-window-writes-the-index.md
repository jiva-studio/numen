# ADR-0035: The review window writes the index

- **Status:** Accepted
- **Date:** 2026-08-30
- **Applies to:** the two desktop applications and the index they share
- **Amends:** ADR-0008 (a write transaction takes its lock at BEGIN)
- **Supersedes:** ADR-0030 — "It reads the index and never writes it", and its consequence "The editor's index goes stale when a mark is minted here"; ADR-0032 — "The reviewer opens the index for reading, and the reading half of it is a type of its own"
- **Related:** ADR-0002, ADR-0020, ADR-0029, ADR-0034

## Context

ADR-0030 gave the review window a read-only index. The index answered one question there — which files are decks — and the one thing the window wrote into a vault was a mark, which changes no heading. A stale index cost nothing, and a second writer over the one database every vault shares was the price not worth paying.

Both halves of that have gone.

The reviewer's agent writes cards (ADR-0032), the window mints marks, and cards are edited during a sitting. Every one of those writes leaves the index behind: the window draws from an index it cannot bring up to date, so what it has just written to a vault is invisible to it until another application has been started and has scanned.

Which preset schedules a deck is now among what the index answers (ADR-0034), so the same staleness reaches how a day is planned and not only what a heading says.

## Decision

### The review window opens the index for writing

**Superseded in part by [ADR-0045](0045-the-review-window-reads-a-vault-it-does-not-carry.md).** A vault the index does not carry is walked whole here; a vault it carries is still levelled by the paths that changed and no more. What follows in this section is the decision as it stood.

Both pools, the schema put in place, and the same connection settings the editor opens with. Every write the window makes brings the paths it touched up to date before it returns, through the same refresh the editor uses and over the same cutting sizes, so neither application re-cuts what the other wrote.

Only the paths that changed are read again. Nothing here rescans a vault.

### A write transaction takes its lock at BEGIN

Two processes now hold a writer each. A transaction that reads before it writes is a snapshot upgraded under lock, and SQLite refuses that with `SQLITE_BUSY` **without calling the busy handler** — so the busy timeout, which ADR-0008 relied on, never runs and the write is simply refused.

The write pool therefore begins every transaction immediately. A writer whose turn has not come waits out the timeout and then fails, and a write is never lost quietly.

### An index nobody has built is built here

**Superseded in part by [ADR-0045](0045-the-review-window-reads-a-vault-it-does-not-carry.md).** A vault the index does not carry is walked into it here, over the same scan the editor walks one with. What follows in this section is the decision as it stood.

ADR-0030 left the file to the application that scans. Opening for writing makes it, so a machine that has only ever run its cards now has an index file. What a person is told is unchanged: a vault nothing has scanned still reads as one nothing has read.

## Consequences

- **What the review window writes, it can see.** A mark it minted and a card its agent corrected answer the next question it asks, and so does anything it is given to write later.
- **Two processes write to one index.** Their writers queue in SQLite, and a write that runs out of the timeout comes back as an error rather than as silence.
- **The read-only handle is gone.** The reading half ADR-0032 gave the reviewer's sources, and the read-only opening ADR-0030 named, have no callers left and are removed with this record.
- **A machine that has only run its cards holds an index file.** It is empty until something scans.

## Alternatives considered

**Leaving the index to the editor's next scan.** Rejected: what this window writes is what it draws next, and a person would correct a card and watch the old one come round until they opened another application.

**Asking the editor to rescan.** Rejected: it makes the daily act depend on a process that may not be running, and it is a second way to do what a write already knows how to do.

**Keeping the reader-only handle for everything but the writes.** Rejected: two handles over one file, told apart by which questions they answer, is a rule to remember at every call site for no gain once the file is open for writing anyway.
