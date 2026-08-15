# ADR-0016: What a scan stores about a note

- **Status:** Accepted
- **Date:** 2026-08-15
- **Applies to:** `modules/apps/desktop`
- **Related:** ADR-0002, ADR-0012, ADR-0015

## Context

A scan reads a note and puts something in the index. What that something is has
never been decided, and the first implementation chose for itself — which is how
a parser ends up with rules nobody asked for and a schema nobody can justify.

The rule this ADR exists to enforce is the one ADR-0003 states for link types and
ADR-0012 states for syntax: **a thing is stored together with the code that reads
it, not in anticipation of code that might.** Anything extracted "because it is
easy while we are here" becomes a column that must be maintained, migrated and
explained, and a parsing rule that becomes permanent the moment vaults are full
of what it accepts.

## Decision

A scan stores exactly four things, and each has a consumer today:

| Stored | Why it exists |
| --- | --- |
| Path, size, modification time | The invalidation key. Without it every scan reads every file. |
| Frontmatter, parsed | The application's own keys will live here (ADR-0009), and a query needs them without reopening the file. A parse error is stored rather than repaired. |
| Body text | What full-text search matches against. |
| Headings, with level and position | The outline of a note, and the boundaries structural chunking will cut on. |

**The parsed frontmatter is a projection, not the record.** JSON has no key
order, no duplicate keys and no YAML timestamps, so what the index holds is what
could be represented rather than what was written. That is enough for querying
and not enough for writing back: the file remains the only verbatim copy, and
anything that edits frontmatter reads the file rather than the index. ADR-0012
requires unknown keys to survive a round-trip through the *file*, which this
does not weaken and does not satisfy either.

Nothing else. In particular, a scan does not invent categories the vault format
has not defined — no tags, no inline fields, no derived collections — and does
not resolve links or anchors, whose meaning is still undecided.

**The title is stored, not decided here.** What a note is called and how that is
worked out belongs to the note format (ADR-0012); the index keeps the answer so
that a list of results does not have to reopen every file to label itself.

### Headings are the one entry admitted early

Headings have no consumer in the first tool beyond a count shown after a scan.
They are stored because they are the structure the note already has, and both
things that will need it — an outline view, and chunking on structure rather than
on token count — read the same rows. Storing them now costs one table and no
parsing rule beyond the one markdown already defines.

That is the whole argument, and it is deliberately a weak one. It is admitted
because the risk it carries is small and reversible: dropping a derived table
costs a migration and a rescan of a vault that is already on disk. Anything whose
mistake would live in the user's files instead — a syntax, a key, a convention —
does not get the same benefit of the doubt.

## Consequences

**Positive**

- The schema can be justified row by row, which is what makes it possible to say
  no to the next addition.
- A scan reads a file once and writes what search needs; there is nothing to
  recompute later because nothing speculative was computed.

**Negative**

- Features that arrive later will need a migration and, where the data is not
  derivable from what is stored, a rescan. That is the cost of not guessing, and
  ADR-0015 makes it a bounded one.
- The line between "structure the note already has" and "a category we invented"
  is a judgement, and headings sit close to it.

## Alternatives considered

**Store everything a parser can cheaply extract** — tags, inline fields, link
occurrences, word counts. Rejected: each one is a parsing rule that the user's
files then depend on, and none of them had a reader. The first version of this
scanner did exactly this, and every one of those fields was removed before it
shipped.

**Store only the fingerprint and the body**, deriving headings on demand.
Rejected: the derivation would run on every query that needs an outline, over
text the index already holds, to avoid one table.
