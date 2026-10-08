# One database for all vaults, outside them

- **Status:** Accepted
- **Date:** 2026-08-25
- **Applies to:** `modules/libs/core`
- **Related:** [Files on disk are the source of truth](0001-files-are-the-source-of-truth.md), [A vault carries its identity, and application state lives with the application](0003-a-vault-carries-its-identity.md), [A hexagonal core in Go](0004-a-hexagonal-core-in-go.md), [What the index stores](0006-what-the-index-stores.md), [A schema change is a numbered migration](0007-a-schema-change-is-a-numbered-migration.md), [The vector index stays inside SQLite](0013-the-vector-index-stays-inside-sqlite.md), [How this application is tested](0025-how-this-application-is-tested.md)

## Context

Everything the application works out about a vault is a cache, and a cache does not live in the vault it describes. The application holds several vaults and switches between them, and one file holds all of their rows. A query that forgets its vault answers with another vault's notes and reports nothing wrong.

## Decision

### The index is one file, outside every vault

```
<cache dir>/numen/index.db
```

The platform's cache directory is where a system keeps what it may delete, which is what the index is. One installation has one index, and it holds every vault that installation knows.

### Every row belongs to a vault

A note carries the vault's identifier. Everything else stored about a note — its headings, its links, its chunks, its problems, its rows in the full-text indexes — belongs to the note, and reaches the vault through it.

The database holds a `vaults` table so that rows have something to point at, filled from the registry when a vault is scanned. The authoritative list of vaults is the registry's.

### Vault scope is structural

Feature code never queries base tables. Access goes through a layer that binds the vault itself, so a query that reaches the database is a query already scoped. The test suite runs with two vaults populated, and every query is answered out of a database holding notes it must not return.

### The application links its own SQLite

The application links SQLite into its own binary and never uses the one the platform provides. Full-text search and the vector index are in the build the application ships. The driver is pure Go and sits behind the index port.

## Consequences

- Vault scope is an obligation on the access layer, and the database does not enforce it.
- A question that spans vaults is one query on one connection.
- Removing a vault is a delete inside a file the other vaults are still using.
- Every vault's cache is lost together, and a scan puts it back.
- The version and build of SQLite are the application's, the same on every platform the application ships to.

## Alternatives considered

**A database file per vault.** Rejected: every question that spans vaults then needs `ATTACH`, which is capped per connection, and deleting a vault is the only thing it makes simpler.

**The index inside the vault it describes.** Rejected: a database in the vault is synced, diffed and backed up as though it were the person's work, and a vault opened on two machines carries a file two builds have written.

**The SQLite the platform provides.** Rejected: what the search is built on is compiled into the build the application ships, and a platform library is whichever version that machine happens to carry.
