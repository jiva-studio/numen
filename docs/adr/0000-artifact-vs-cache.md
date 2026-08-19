| Embeddings | bought | A model made them, and the system works without them |
# ADR-0000: Data is classified as either artifact or cache

- **Status:** Accepted
- **Date:** 2026-08-15
- **Applies to:** the product — every application in this repository
- **Related:** ADR-0001, ADR-0002, ADR-0004, ADR-0005, ADR-0006

## Context

The product is a hybrid of three tools: an Obsidian-style markdown note base, a
TheBrain-style typed navigational DAG, and an Anki-style spaced repetition
system, plus a source layer (PDF, audio, books) with text extraction, search and
highlighting inside the original document.

A system of this shape accumulates many different kinds of data: note text, link
markup, a link table, a full-text index, a review log, a card schedule, word
coordinates on a PDF page, OCR output, transcripts, embeddings. Deciding
case-by-case where each of them lives produces an inconsistent system and an
endless stream of "and where do we put *this*?" questions as the product grows.

We want one rule that answers the question mechanically, including for data types
nobody has thought of yet.

## Decision

Every piece of data in the system is classified by exactly one question:

> **Can this be reproduced locally, deterministically, and for free?**

- **Yes → it is a cache.** It lives in SQLite, outside the vault. It can be
  deleted at any moment and rebuilt with no loss.
- **No → it is an artifact.** It lives in the vault, in an open text format.

This criterion is applied to any new data type as the system evolves. It ranks
above every individual decision in the other ADRs — all of them are derived from
it. If a later decision contradicts this rule, the later decision is wrong.

"For free" means: without a network call, without a paid service, without a model
whose output is not byte-reproducible. "Deterministically" means: the same input
produces the same output on any machine, on any day.

## Worked examples

| Data | Class | Why |
| --- | --- | --- |
| Note body text | artifact | A human wrote it |
| Link markup (`links:` block) | artifact | A human wrote it |
| Link table, graph edges | cache | Derived by parsing notes |
| Full-text index | cache | Derived from notes |
| Card review log | artifact | Immutable events; cannot be recomputed |
| Card schedule (`due`, `ease`, `stability`) | cache | Pure function of the review log |
| Word coordinates in a PDF with a text layer | cache | Deterministic local extraction |
| OCR output for a scanned page | artifact | Needs a service/model; not byte-reproducible |
| Audio transcript | artifact | Same |
| Embeddings | cache, optional | Bound to a model; fully recomputed when the model changes, and the system works without them |

## Scope: what the rule does not cover

The rule classifies **data in the system** — things that belong to a vault. There
is a third kind of state that belongs to neither class, and pretending otherwise
sends it to the wrong place.

**Application state** is what the installed application knows about itself: which
vaults exist and where they are, which one was open last, window geometry, the
selected theme, the chosen OCR engine.

Take the vault list. The application can hold several vaults and switch between
them, so it has to know the set. That set cannot be derived from any vault — ask
which vault would contain it and the question answers itself: none of them can,
because the answer spans all of them and none owns it. Nor is it an artifact of a
vault, for the same reason. It is not a cache either, since nothing regenerates
it.

But it is also not irreplaceable in the way a review log is. Lose it and no
knowledge is gone: the vaults are still folders on disk, and the user points the
application at them again. It is **user-recoverable but not machine-derivable**,
and that is precisely the category the artifact/cache question does not answer.

Application state never lives in a vault. Its loss is neither a rebuild nor data
loss but a reconnect by hand, and that cost is accepted — the alternative is a
vault holding a list it does not own.

Two things follow from this and are decided elsewhere: where that state is kept,
and how a vault stays recognisable once its folder is moved. Neither is settled
here.

## The third class: what was bought

A vector is not written by a person and is not reproduced for free. It comes
from a model, and a model is either minutes of a machine or money and a network.
Neither answer the rule offers fits it: it does not belong in the vault, and it
is not something to delete and make again.

**A vector is bought.** It lives in SQLite beside the cache, and it is kept:

- It is addressed by the text it was made from and the recipe it was made
  under — never by the row that pointed at it. Chunks are renumbered by every
  cut; the words a window holds and the model that read them are what the vector
  is about.
- It outlives what asked for it. A source the vault no longer offers takes
  nothing with it: a folder that could not be read looks the same as one whose
  files were deleted.
- It is forgotten where a source is cut again and the text it held is gone, and
  no other chunk holds that text. That is the one moment the answer is known.
- Nothing else deletes one. Emptying the cache is free; emptying this is a bill.


## Consequences

**Positive**

- The vault contains only data that would be genuinely lost — nothing else. That
  makes the vault small, greppable and diffable, and makes backups cheap.
- Schema migrations disappear as a category (see ADR-0004): when the SQLite model
  changes, tables are dropped and replayed from artifacts.
- "Delete the cache if something looks wrong" is a legitimate, safe support
  answer. What was bought is kept apart from it and outlives it.
- New feature areas have a decision procedure instead of a debate.

**Negative / costs**

- Everything classified as cache has to be recomputed after it is thrown away, so
  a cold rebuild is expensive by construction. What that costs and what it is
  allowed to cost is ADR-0002's problem.
- The boundary is occasionally uncomfortable: OCR of a scan is an artifact even
  though it is "just extracted text", because a different engine version produces
  different bytes.
- The rule is binary, and the application-state category above shows it does not
  cover everything. That exception has to be stated explicitly rather than
  discovered, or state ends up in whichever vault happened to be open.

**Implementation requirement**

Anything that produces artifacts must be able to rebuild its cache tables from
those artifacts alone, offline. That is the whole obligation this rule imposes;
what the artifacts look like is the business of whoever writes them (ADR-0004).

## Alternatives considered

**Classify by "is it user-visible?"** — Rejected. A review log is not
user-visible in any meaningful sense, but it is irreplaceable. A note title is
user-visible and also trivially derivable from the file. The criterion cuts
across the useful line.

**Classify by "is it expensive to compute?"** — Rejected. Expense is a moving
target (hardware, model size), and it produces unstable answers: an index that is
expensive today is cheap next year, but its *class* must not change, because the
storage layout depends on it.

**Put application state in a vault** (for example, the vault list in the vault
that happens to be open). Rejected: it makes one vault authoritative over the
others, breaks the moment that vault is not opened, and turns a folder that is
supposed to be self-contained into a control point for the whole installation.
