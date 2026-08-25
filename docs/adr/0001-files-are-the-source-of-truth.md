# ADR-0001: Files on disk are the source of truth

- **Status:** Accepted
- **Date:** 2026-08-25
- **Applies to:** the product — every application in this repository
- **Related:** ADR-0002, ADR-0008, ADR-0009, ADR-0011, ADR-0012, ADR-0015,
  ADR-0017, ADR-0018

## Context

A tool of this shape accumulates many kinds of data: note text, link markup, a
link table, a full-text index, headings, chunks, a reading of a scanned page,
vectors. Each of them has to be somewhere, and one rule answers for the ones
nobody has thought of yet. This is a tool people put years into, so the rule
starts from what is left of their work when the application is gone.

## Decision

### The vault is an ordinary directory of ordinary files

Notes are `.md` files with YAML frontmatter, in whatever folder structure the
person likes, and the application neither imposes nor rearranges that layout.
The link markup lives in the note file — see [the note
format](../note-format.md). The application stores nothing unique in its own
database, and deleting the index in full loses nothing.

Third-party tools therefore work on live data: `rg`, `git`, vim, another
markdown editor, a sync client, the person's own scripts, with the application
closed. What that costs is a watcher (ADR-0009), a scan that reads a moving
vault (ADR-0008), rows that outlive the files they describe, and a write path
that puts a file back whole (ADR-0017).

### Three classes, and one question that sorts them

> **Can this be reproduced locally, deterministically, and for free?**

- **Yes → it is a cache.** It lives in SQLite, outside the vault (ADR-0002), and
  deleting it costs a scan.
- **No, a person made it → it is an artifact.** It lives in the vault, in an
  open text format.
- **No, a machine was paid for it → it is bought.** A vector is bought: minutes
  of a machine, or money and a network.

"For free" means without a network call, without a paid service, and without a
model whose output is not byte-reproducible. "Deterministically" means the same
input gives the same output on any machine, on any day.

```mermaid
graph TD
    Q{"can this be reproduced locally,<br/>deterministically, and for free?"}

    Q -->|"yes"| C
    Q -->|"no — a person made it"| A
    Q -->|"no — a machine was paid for it"| B

    subgraph art["artifact — in the vault, open text"]
        A["notes: text, frontmatter, link markup<br/>readings of documents that carry no text<br/>the identity a vault carries"]
        AX(["deleted: the work is gone"])
    end

    subgraph cache["cache — in SQLite, outside the vault"]
        C["sources, notes, headings, links, problems,<br/>chunks, full-text indexes"]
        CX(["deleted: a scan puts it back"])
    end

    subgraph bought["bought — in SQLite, outside the vault"]
        B["vectors"]
        BX(["deleted: bought again, with a machine<br/>or with money"])
    end

    A -.-> AX
    C -.-> CX
    B -.-> BX
```

### A vector is kept, and dropped at one moment

A vector lives in SQLite beside the cache and is kept. It is addressed by the
text it was made from and the recipe it was made under — where a model runs, its
width, where the text was cut off, how its output becomes one vector, how the
numbers are stored. It is dropped at one moment: a source is cut again and the
text the vector was made from is held by no chunk (ADR-0011, ADR-0012). Nothing
else deletes one.

### Anything producing artifacts rebuilds its cache from them

Whatever writes an artifact into a vault can rebuild every cache derived from
that artifact out of the vault alone, offline. The text a reading takes out of a
document that carries none is an artifact for this reason (ADR-0015).

## Consequences

- A cold rebuild reads every file in every vault, and has to report progress and
  stay usable while it runs.
- Rows outlive the files they describe, and startup has to find them.
- Storing something new starts by answering the question above, and "artifact"
  means writing a file format.
- A vector survives a rebuild of the cache, and the recipe it was made under is
  stored beside it.
- "You are not locked in" is checkable, with the application closed.

## Alternatives considered

**A database as the source of truth, with files exported on demand.** Rejected:
it kills what motivates the design — other tools operating on live data.

**Files as the source of truth, editable only while the application is closed.**
Rejected: lock-in through the back door, and nothing enforces it.

**Two classes, kept and rebuildable.** Rejected: a vector is rebuildable in the
sense the code cares about and not in the sense the person's bill cares about,
and one word covering both is the word that throws a vector away.
