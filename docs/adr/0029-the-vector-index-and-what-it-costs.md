# ADR-0029: The vector index stays inside SQLite, and what that costs

- **Status:** Accepted
- **Date:** 2026-08-17
- **Applies to:** `modules/apps/desktop`
- **Related:** ADR-0002, ADR-0007, ADR-0014, ADR-0019
- **Narrows:** ADR-0002, on which engine is the fallback and on what the
  extension costs

## Context

ADR-0002 decided that vector search comes from `sqlite-vec` inside the one
database, and named a dedicated vector store — "LanceDB and similar" — as the
move to make if the extension hit a wall. Neither half had been tried: the
extension had never been built or measured, and the fallback had never been
checked for reachability from Go.

Both have now been. What was measured is in docs/performance.md; two of the
findings change what ADR-0002 says, and a third makes a version a requirement
rather than an assumption.

## Decision

### The engine stays `sqlite-vec`

Nothing measured argues for a second engine at the sizes this application is
designed for. ADR-0002's reasoning holds and is not restated here.

### The fallback named in ADR-0002 is not reachable, and is replaced

**LanceDB has no Go bindings.** Reaching it means writing foreign-function
bindings by hand or running a second process beside the application, and ADR-0014
rules out neither in principle but both are a different decision from "swap the
vector engine".

The fallbacks that are actually reachable, if one is ever needed:

| Candidate | Reached from Go by | What it would cost |
| --- | --- | --- |
| `usearch` | Official bindings; the index memory-maps from disk | cgo, so ADR-0014's cross-compilation is given up |
| Qdrant Edge | A Rust crate, in-process, no bindings today | cgo, and bindings to write |

Both are recorded so that the next person does not re-derive them. Neither is
adopted.

### An index that must be resident is disqualified

A vector index this application can use **reads from disk**. An index that has
to be held in memory to be searched competes with everything else on the user's
machine, and at the sizes ADR-0002 designs for it asks for gigabytes.

This rules out the in-memory approximate indexes, whatever their speed. It is a
property of the application being a desktop tool rather than a server, and it is
the reason the measured direction was quantisation and staged retrieval
(ADR-0007) rather than a graph index.

### The driver must be one that carries the extension

`modernc.org/sqlite` gained the bundled extension in **v1.50.0**. ADR-0014 states
that the driver supports it; that is true only from this version, which is the
floor the module is held above. `go.mod` names v1.56.0.

**The pin is a requirement, not a detail:** the vector index does not exist in a
build made with an older driver, and the failure is a missing SQL function
rather than a build error.

### The cost of the index is stated where it is measured

Index size, the size at which an exact scan stops meeting ADR-0019's budget, and
what each representation costs are measurements with a date on them. They live
in docs/performance.md. This ADR records only that they are the numbers a
decision about a second engine has to beat.

## Consequences

**Positive**

- The fallback in ADR-0002 is now a real option with a name and a price, instead
  of a plausible one that turns out not to exist.
- The residency rule disqualifies a whole class of candidate without measuring
  each one.
- A build that silently lacks vector search is prevented by a version floor.

**Negative**

- Every reachable fallback costs cgo, so leaving `sqlite-vec` and keeping
  ADR-0014's one-machine cross-compilation are now known to be mutually
  exclusive. That is a smaller escape hatch than ADR-0002 assumed.
- Raising the driver floor takes whatever else changed with it.

## Alternatives considered

**Keep LanceDB as the named fallback and write the bindings when needed.**
Rejected as a fallback, because a fallback whose cost is unknown is not one. It
remains available as a project in its own right.

**Adopt an embedded engine now, before the wall is reached.** Rejected: the
measurements say the wall is further away than the corpus this is being built
for, and every candidate costs the cross-compilation ADR-0014 chose deliberately.

**Leave the driver pinned and load the extension at runtime.** Rejected: it is
the pure-Go driver that makes the bundled extension possible at all, and a
runtime-loaded one returns to the platform-library problem ADR-0002 already
settled.
