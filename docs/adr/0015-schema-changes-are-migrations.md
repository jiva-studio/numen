# ADR-0015: A schema change migrates the index instead of rebuilding it

- **Status:** Accepted
- **Date:** 2026-08-15
- **Applies to:** any application that keeps an index; today `modules/apps/desktop`
- **Related:** ADR-0000, ADR-0002, ADR-0014, ADR-0035
- **Partly superseded by:** ADR-0035 — what happens to an index this build cannot account for

## Context

The index is a cache: the vaults are the truth, and anything in the database can
be produced again by scanning them. Its schema will change as the application
grows — a column for a new feature, an index to make a query faster.

The question is what happens to the index already sitting on a user's machine
when that change ships. Refilling it is always available and always correct, and
it costs minutes at the sizes this is built for. Most schema changes have nothing
to do with the user: charging them that wait on every update, for a column
belonging to a feature they do not use, is the wrong default.

Being *able* to rebuild is not a reason to *always* rebuild.

## Decision

**The index carries a schema version and is brought forward by numbered
migrations.**

- Migrations are `.sql` files named `NNNN_description.sql`, applied in order,
  once each. A file that is not numbered is an error rather than something to
  skip: a schema nobody can reproduce is worse than a failed start.
- The version lives in the database itself. On start, every migration newer than
  the recorded version is applied.
- **Each migration runs in one transaction together with its version bump.** A
  failure leaves the database at the last version that fully applied, so the next
  start retries a whole migration rather than resuming half-way through one.
- **Emptying is a migration's own decision.** A numbered file may empty the
  tables it changes, and the next scan refills them. Nothing outside a migration
  file empties anything.

Adding a column, an index or a table therefore costs the user nothing. Only a
change that redefines what is stored costs a re-scan, and only for the part that
changed.

## Consequences

**Positive**

- Updating the application does not re-read the user's vaults. This is the whole
  point: a service column added for someone else's feature must not cost you an
  hour.
- Migrations are readable SQL in files, so a schema change is reviewable as a
  schema change.
- The escape hatch remains: when migrating is genuinely impossible, refilling is
  still correct, because the vaults are still the truth.

**Negative**

- Migrations are forward-only and accumulate. Every one of them is code that runs
  on someone's machine years later, and a mistake in one is not undone by a
  rebuild the way a mistake in a schema definition used to be.
- The number of migrations grows monotonically. Squashing them is possible only
  by dropping support for upgrading from before the squash.
- A migration can be wrong in a way that a rebuild never could: it can silently
  produce a schema that does not match what a fresh install would produce. Fresh
  and migrated databases must be checked against each other, not assumed equal.

## Alternatives considered

**Drop and rebuild on any schema change.** Rejected on cost, not on safety: it
is the correct fallback and the wrong default.

**Migrate the schema, keep no version** (detect the shape of the database and
adapt). Rejected: schema detection is guesswork that grows with every change, and
it fails silently when two versions happen to look alike.

**A migration framework.** Rejected for now: numbered files, one transaction
each, and a version in the database is the whole mechanism, and it fits in a
file that can be read in one sitting.
