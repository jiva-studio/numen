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
| Frontmatter, verbatim | The user's own keys must survive round-trips (ADR-0012), and the application's own keys will live here (ADR-0009). Kept as found, including a parse error, which is reported rather than repaired. |
| Body text | What full-text search matches against. |
| Headings, with level and position | The outline of a note, and the boundaries structural chunking will cut on. |

Nothing else. In particular, a scan does not invent categories the vault format
has not defined — no tags, no inline fields, no derived collections — and does
not resolve links or anchors, whose meaning is still undecided.

**A title is derived, not stored as a fact about the note.** It is the
frontmatter `title`, else the first level-one heading, else the filename. That
order is a display convention rather than a property of the file, and it is
recorded here because the parser must not be free to change it quietly: a note
whose title moves between versions is a note the user cannot find twice.

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
