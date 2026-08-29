# ADR-0006: What the index stores

- **Status:** Accepted
- **Date:** 2026-08-25
- **Applies to:** `modules/libs/core`
- **Related:** ADR-0001, ADR-0002, ADR-0007, ADR-0008, ADR-0011, ADR-0012, ADR-0014, ADR-0017
- **Amended by:** ADR-0029 — a deck and a stencil contribute no chunk, no vector, and only the headings a person wrote

## Context

The vaults are the truth and the index is what makes them answerable. It holds one schema for every vault, and what may be put in it is what every later question is asked of. How that schema moves from one shape to the next is ADR-0007.

## Decision

### Nothing is stored without a reader today

A scan stores what something reads now. A field nothing asks for is not stored, however cheap it was to produce while a parser was already open.

### The schema

```mermaid
erDiagram
    vaults ||--o{ sources : holds
    sources ||--o| notes : "same row number"
    notes ||--o{ headings : ""
    notes ||--o{ links : ""
    notes ||--o{ problems : ""
    sources ||--o{ chunks : ""
    chunks ||--o{ chunks : "encloses"
    chunks ||--o| chunks_vec : "rowid"
    chunks ||--o| chunks_fts : "rowid"
    chunks ||--o| parts_fts : "rowid"
    notes ||--o| titles_fts : "rowid"
    headings ||--o| headings_fts : "rowid"
    chunks }o--o| vectors : "hash, under a recipe"

    vaults {
        INTEGER id PK
        TEXT identifier UK "the ULID the folder carries"
        TEXT name
        TEXT path
    }
    sources {
        INTEGER id PK
        INTEGER vault_id FK
        TEXT path UK "unique with vault_id"
        TEXT kind "note, book"
        INTEGER size "the fingerprint"
        INTEGER modified_at "the fingerprint"
        TEXT hash "null until something computes it"
        TEXT recipe "what extracted the text"
        TEXT text_from "which producer made the text, when it is not the file"
    }
    notes {
        INTEGER source_id PK "and FK to sources"
        INTEGER vault_id FK
        TEXT basename "NOCASE, what a link written by name matches"
        TEXT title
        TEXT identifier "the ULID in the file, when there is one"
        TEXT frontmatter "JSON, a projection"
        TEXT frontmatter_error
    }
    headings {
        INTEGER id PK
        INTEGER note_id FK
        INTEGER line UK "unique with note_id"
        INTEGER level
        TEXT text
    }
    links {
        INTEGER note_id PK "and FK to notes"
        INTEGER position PK
        TEXT scheme
        TEXT value "as written"
        TEXT value_base "NOCASE, the last segment without its extension"
        TEXT role
        TEXT type
        TEXT note "why the link exists"
        TEXT label
    }
    problems {
        INTEGER note_id FK
        TEXT detail
    }
    chunks {
        INTEGER id PK
        INTEGER source_id FK
        INTEGER vault_id FK
        INTEGER start "bytes into the text"
        INTEGER length
        INTEGER parent FK "null for the chunk a result shows"
        TEXT location "where it sits, in the source's own numbering"
        TEXT hash "the address its text gives it"
    }
    vectors {
        BLOB fingerprint UK "the chunk's hash"
        TEXT recipe UK
        BLOB v
    }
    chunks_vec {
        INTEGER chunk_id PK "vec0"
        INTEGER vault_id "a metadata column"
        BLOB embedding "one bit per dimension, 1024"
    }
    chunks_fts {
        TEXT text "fts5, contentless"
    }
    parts_fts {
        TEXT text "fts5, contentless, the chunk a part opens"
    }
    titles_fts {
        TEXT text "fts5"
    }
    headings_fts {
        TEXT text "fts5"
    }
```

**A file of any kind is a `sources` row.** Its path, its size and its modification time are there, and they are what the next scan answers "has this changed" with, without opening the file. `kind` says what sort of file it is; `hash` is the address the content gives it, and is what lets a moved file keep what was derived from it. `recipe` is what extracted the text and is null while nothing has. What only one kind has is a table of its own.

**A note's row number is its source's**, so a question that needs only the file joins `sources` on it. What the note table adds is the names a link reaches the note by, its title, the ULID the file carries when it carries one, and the frontmatter.

**Headings carry their level and their line**, and they are the outline of a note, the boundaries a cut falls on, and half of what the palette matches against.

**Links are stored as written**, with the last segment of the address kept beside them as the name the backwards question is answered through.

**A problem is what could not be acted on and is worth showing**: a link with no role, a target nothing understands. Frontmatter that would not parse stays on the note it broke. Both are read together, so a vault's problems are a query.

**The body is indexed over chunks.** `chunks_fts` keeps no copy of what it indexed: a chunk says where in a file its text is, and showing a passage reads the file. A lexical hit and a dense hit name the same row, so both are placed in one list and read back the same way. What a chunk is, is ADR-0012, and how a source is cut is ADR-0011; how the two passes are combined is ADR-0014.

**`titles_fts` and `headings_fts` each keep a copy of the text they indexed.** What an answer draws is the name with the run that matched marked inside it, and an index can only say where it matched over text it holds. They are two tables because a title and a heading are each ranked against their own population.

**`parts_fts` is the names of the parts a source divides into**, keyed by the chunk each part opens, so a hit on a section's name is a passage standing at the start of that section.

**A vector is addressed by the text and the recipe**, never by a chunk's row number, and it is kept where a renumbering of chunks cannot reach it (ADR-0012).

### Resolution is a query, never a column

A link is kept as it was written, and where it points is worked out when the question is asked. Adding a file mends a link that was dangling, and deleting one breaks a link that worked, with no row touched either way.

### A note is a number inside the index and a vault-and-path outside

Headings, links, problems, chunks and full-text rows are all filed under that number. What the index calls a note is not visible past it: the ports speak in vaults and paths, and the translation happens once per question.

### Parsed frontmatter is a projection

JSON has no key order, no duplicate keys and no YAML timestamps, so what the index holds is what could be represented. It is enough to query and not enough to write back: the file is the only verbatim copy, and anything editing frontmatter reads the file (ADR-0017, [the note format](../note-format.md)).

### The index knows its own shape, and a plan is asserted by its index

SQLite chooses between the ways it could answer a question from what it knows about how much is stored and how it is spread. A scan that indexed or removed something measures the database afterwards; a scan that stored nothing does not.

The measurement samples large tables and covers the whole database, including tables the connection that asks has never read from. The core states that the index has changed; how a database is measured is the adapter's.

Tests that check query plans name the index each question has to be answered through, and they measure the database the way the application does.

## Consequences

- A feature wanting something the schema does not hold costs a migration and, where it cannot be derived, a rescan.
- Resolution being a query puts the backlink question's plan on the critical path, and that plan is asserted by name.
- Frontmatter is queryable from the index and writable only through the file.
- Nothing cascades into a virtual table, so a chunk's rows in `chunks_vec`, `chunks_fts` and `parts_fts` are deleted by the code that deletes the chunk.
- A scan that stored nothing leaves the plans standing on the last measurement.

## Alternatives considered

**Store where each link resolves.** Rejected: a file appearing or disappearing changes the answer, so the column is stale as often as the vault is edited.

**Store everything a parser can cheaply extract** — tags, inline fields, word counts. Rejected: each is a parsing rule the person's files come to depend on, and none of them had a reader.

**Keep the body text beside the chunk that indexed it.** Rejected: the vault would be held twice, and a passage is read from the file it is in. Titles and headings are held because the answer marks the run that matched inside them.
