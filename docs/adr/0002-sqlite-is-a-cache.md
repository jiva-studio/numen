# ADR-0002: SQLite is a cache, one database for all vaults

- **Status:** Accepted
- **Date:** 2026-08-15
- **Applies to:** any application that keeps an index; today `modules/apps/desktop`
- **Related:** ADR-0000, ADR-0001, ADR-0004

## Context

ADR-0001 makes files the source of truth. That does not remove the need for a
database: graph traversal, full-text search, scheduling queries and source
highlighting cannot be served by re-reading markdown on demand at the target
scale — 100k notes, 500k edges, 50k cards, 5M review events.

So there is a database. This ADR says what it may contain, and — since the
application holds several vaults and switches between them — how many of them
there are.

## Decision

### The index is derived data only

It holds parsed links and graph edges, full-text indexes, card state, source
chunks, word coordinates and embeddings. **Deleting it must produce a full
rebuild from the vault, with no network access and no external service calls.**

It is stored outside the vault, so that syncing or committing a vault never
carries an index along.

### One database for all vaults

The application holds several vaults and switches between them. All of their
caches live in a single database file; every row carries a `vault_id`.

The alternative — a database file per vault — was considered and rejected. It
makes anything that spans vaults into a feature that has to be built: a dashboard
of what exists and in what state, "where else have I written about this", moving
a note from one vault to another. With separate files, cross-vault work means
`ATTACH`, which is capped at ten databases per connection by default and raised
only by recompiling SQLite. One file makes all of it an ordinary query.

The arguments that pointed the other way do not survive contact:

- **Full-text scoping.** Filtering is a plain `WHERE`: `vault_id` sits in the
  table as an `UNINDEXED` column. The match runs across every vault and the
  filter discards the rest, so the cost is rows visited, not correctness. BM25
  does draw its term statistics from the whole table, so weights are averaged
  across corpora and per-vault ranking shifts slightly — small between personal
  corpora, and fixable with separate full-text tables inside the same file. It
  never required separate files.
- **Vector search.** It argues the other way, in fact. Vector search is not part
  of SQLite; it comes from the `sqlite-vec` extension, which supports a
  **partition key** — the index is sharded on a column and filtered before any
  distance is computed. Its documentation names per-user and per-document indexes
  as the intended case, which is exactly what `vault_id` is. Partitioning inside
  one index is the idiom, and it is faster than an unpartitioned one.
- **Corruption.** The index is disposable by construction. Corruption costs a
  rebuild of every vault instead of one — an inconvenience, not a loss.
- **Deleting a vault.** One `DELETE ... WHERE vault_id = ?` in a transaction.

What does survive is **leakage**: a query that forgets its `vault_id` does not
crash, it silently returns another vault's notes in search, another vault's cards
in the review queue, another vault's neighbours in the graph. Invisible in any
test that uses one vault.

That is a real risk and it is handled structurally rather than by care:

- Every table that holds vault-scoped rows has a `vault_id` column, and it is
  part of the primary key wherever that is meaningful.
- Base tables are never queried directly by feature code. Access goes through a
  layer that binds `vault_id` itself, so omitting it is not expressible.
- The test suite runs with two vaults populated, not one. A single-vault suite
  cannot observe this class of bug at all.

**The authoritative list of vaults is still not the index's.** It is application
state (ADR-0000) and lives with the application's configuration. The database
holds a `vaults` table so that rows have something to point at, populated from
that configuration when a vault is opened — a cache of the registry, not the
registry. Deleting the index must not lose which vaults exist.

### One exception to "rebuild offline"

Embeddings cannot always be rebuilt without a network, because the model may be a
service. This does not weaken the guarantee, because embeddings are optional:
without them, full-text search, the graph, navigation and spaced repetition all
work; only semantic search and interference detection are unavailable.

On rebuild, the lexical index and the graph come up immediately and the vector
index fills in behind them — and may never finish. Phrased as a promise: without
a network the system is fully functional; with one it is additionally smart.

### The application ships its own SQLite build

The application links its own SQLite into its own binary and never uses the one
the platform provides.

On the desktop this is the ordinary way to embed SQLite anyway, so it costs
nothing. It is stated as a decision because the alternative is quietly available
and quietly broken: vector search is an extension rather than a part of SQLite,
and Apple's system SQLite is compiled with `SQLITE_OMIT_LOAD_EXTENSION` — it
cannot load extensions at all, and no runtime flag changes that. Anything that
links against the platform library inherits that, on macOS as much as on iOS.

Two things follow:

- **Full-text search is always present**, because it is in the build we ship
  rather than in the platform's. No client has to degrade to substring matching.
- **The vector extension is a per-platform build option, not a requirement.** The
  desktop links it in. A client that leaves it out loses semantic search and
  nothing else, because the vector index is optional by the rule above.

A client without the vector index is a normal client, not a broken one.

**Whether the mobile client carries vectors is not decided here.** The extension
publishes prebuilt libraries for Android and iOS, so the path is not blocked —
but "not blocked" is not the same as "works", and the answer depends on things
that do not exist yet: which stack the mobile client is built on (Ionic and
Capacitor, Kotlin Multiplatform, or native), whether that stack's SQLite driver
tolerates a custom build with extensions linked in at all, what the index costs
in memory, storage and battery on a real device, and whether semantic search is
even wanted on a phone. That is a spike, not a paragraph in an ADR. This decision
only guarantees that dropping it is a supported outcome rather than a defect.

## Consequences

**Positive**

- A model change can always be answered by dropping the tables and replaying from
  the vault and its artifacts, because nothing here is a source of truth. How a
  schema change is actually applied is ADR-0015.
- Corruption is a rebuild, not a loss.
- The schema can be denormalised for query speed without anybody worrying about
  the truth drifting, because it is not the truth.

**Negative**

- A cold rebuild is expensive and has to be engineered for: progress reporting,
  and the application staying usable while it runs.
- Anything wanting to store something here must first answer the ADR-0000
  question, and "artifact" means writing a submodule instead of adding a column.
  The friction is intentional.
- Vault scoping is a correctness obligation on every query, enforced by the
  access layer rather than by the database.

**Budgets**

Stated here as design constraints; the numbers they are measured against, and
the procedure, are ADR-0019.

| Operation | Budget |
| --- | --- |
| Startup on a warm cache | under 2 s to interactive, independent of vault size |
| Incremental update after one edit | under 100 ms, watcher event to updated UI |
| Full rebuild, 100k notes | single-digit minutes, with progress, usable meanwhile |
| 2-hop graph traversal | under 50 ms |
| PDF page open with highlights | under 200 ms |

These force three things: invalidation on `(path, mtime, size)` so only changed
files are reparsed; a batched startup reconciliation — one directory walk and one
batch query, never a per-file round trip; and neighbourhood queries by recursive
CTE, because a 100k-node graph is neither loadable into memory nor readable on
screen.

## Alternatives considered

**Database as the source of truth.** Covered and rejected in ADR-0001.

**No database; search and traversal over files on demand.** Rejected on the
numbers — full-text search over 100k notes and 2-hop traversal over 500k edges
are not achievable per keystroke without an index.

**A database file per vault.** Rejected — see the decision above. It removes the
leakage risk by construction, but turns every cross-vault view into an `ATTACH`
against a ten-database limit, and the ranking and corruption arguments that
seemed to support it do not hold.

**An embedded graph database** (Kuzu, Cozo and the like), on the grounds that the
notes form a graph. Rejected on two counts.

The workload is not graph-shaped. The graph is the smallest part of the system —
500k edges against 5M review events, plus chunks, page coordinates and
embeddings — and the queries are shallow: neighbours two hops out, children of a
parent, a topological order over `requires`, cycle detection. A recursive CTE
covers all of it inside the budget. Graph engines earn their keep on
variable-length paths and pattern matching across millions of edges, which is
not what a navigational graph a human reads two hops at a time asks for.
Choosing one would optimise the smallest part of the system at the expense of
the largest.

And the ground moves. Kuzu — embedded, Cypher, vector and full-text indexes
built in, the closest fit on paper — was archived in October 2025 after its
company was acquired, leaving community forks. Cozo, relational-graph-vector
with HNSW and full-text on every platform including phones, went quiet after
December 2024 and was forked onwards. Both died inside eighteen months. This is
a tool people keep for a decade; SQLite will be on every phone on the planet
when this decision is old.

**A dedicated vector store beside SQLite** (LanceDB and similar). Not rejected
so much as deferred: it would be the better vector engine and a second engine to
bundle, and the relational half stays regardless. Worth revisiting only if the
extension hits a wall — and cheaply, because the vector index is optional cache.
Nothing in the vault or the rest of the schema depends on which engine produced
it, so this choice does not have to be right forever.

**A separate database per subsystem.** Rejected: the interesting
queries cross subsystems — card to chunk to asset to note to graph neighbourhood
— and those should be joins, not application-level merges.
