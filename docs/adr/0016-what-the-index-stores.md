# ADR-0016: What a scan stores about a note

- **Status:** Accepted
- **Date:** 2026-08-15
- **Applies to:** `modules/apps/desktop`
- **Related:** ADR-0002, ADR-0012, ADR-0015, ADR-0024

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

A scan stores what has a consumer today, and nothing else:

| Stored | Why it exists |
| --- | --- |
| Path, size, modification time | The invalidation key. Without it every scan reads every file. |
| Frontmatter, parsed | The application's own keys will live here (ADR-0009), and a query needs them without reopening the file. A parse error is stored rather than repaired. |
| The body, indexed and not kept | What full-text search matches against. The index holds no copy of the text: a result is a title and a path, and the text is on disk where it was read from. |
| Headings, with their level and the line they stand on | The outline of a note, and the boundaries structural chunking will cut on. |
| The note identifier, when the file carries one | What a link written as `note://` points at, across vaults (ADR-0009, ADR-0011). An identifier that is not a ULID is a reported problem rather than a target. |
| The filename without its extension | What a link written by name is matched against. Stored rather than computed, because resolution asks for it on every link. |
| Links, as written | The edges of the graph (ADR-0003). Stored as written; where each one points is worked out when asked, so adding a file resolves a link that was dangling without touching a row. |
| What could not be acted on | A link with no role, a role nobody decided on, frontmatter that will not parse. Read by `numen problems`, which is what makes storing it allowed. |

**The parsed frontmatter is a projection, not the record.** JSON has no key
order, no duplicate keys and no YAML timestamps, so what the index holds is what
could be represented rather than what was written. That is enough for querying
and not enough for writing back: the file remains the only verbatim copy, and
anything that edits frontmatter reads the file rather than the index. ADR-0012
requires unknown keys to survive a round-trip through the *file*, which this
does not weaken and does not satisfy either.

Nothing else. In particular, a scan does not invent categories the vault format
has not defined — no tags, no inline fields, no derived collections — and does
not read anchors, which are decided (ADR-0009) and have nothing attaching to them
yet.

**Resolution is not stored.** A link is kept as it was written, and what it
points at is worked out when the question is asked. That is what lets adding a
file mend a link that was dangling, and it is why there is no "resolved" column
to go stale.

**The title is stored, not decided here.** What a note is called and how that is
worked out belongs to the note format (ADR-0012); the index keeps the answer so
that a list of results does not have to reopen every file to label itself.

### A note is addressed by a number

Inside the index a note is a number. Its headings, its links, what could not be
acted on and its place in the full-text index are all stored against that
number. A vault is a number too, and the ULID it carries in the world is stored
once.

The path is one of the things stored about a note. It is what the vault calls
the note, not what the index calls it.

What the index calls a note is not visible outside it: the ports speak in vaults
and paths, and the translation happens once per question asked.

### A search result is a title and a path

> **Partly superseded by ADR-0006 and ADR-0007.** A search answers with passages:
> the text around a hit and where it came from. The rule that survives is the
> second clause — the full-text index keeps no copy — and it is what makes showing
> a passage a read of the file it belongs to.

Nothing quotes the matching text back, so the full-text index keeps no copy of
what it indexed. Anything that wants a fragment of a matched note reads the
file.

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
