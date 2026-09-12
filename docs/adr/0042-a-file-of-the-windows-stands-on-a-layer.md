# A file of the windows stands on a layer

- **Status:** Accepted
- **Date:** 2026-09-12
- **Applies to:** `modules/apps/desktop/editor`, `modules/apps/desktop/flashcards`, `modules/libs/ui`

## Context

The window was laid out flat: a folder under `src/` was a tab, `shared/` was what every tab could take, and one rule held it — no tab reaches another. That rule was written down, checked on every push, and it worked for as long as a window was a handful of tabs.

It stopped saying enough. A tab grew a store, an adapter to the vault, a dialog its neighbour also wanted, and the only two places a shared thing could go were inside one tab, where the other could not have it, or into `shared/`, where everything could. `shared/` filled with the vocabulary of notes, decks and settings, and the rule that had held the tabs apart had nothing to say about any of it.

## Decision

### Six layers, and a layer reaches only what stands below it

Lowest first: `shared`, `entities`, `features`, `widgets`, `pages`, `app`. The component library says `screens` where the windows say `pages`, and its barrel stands where their `app` does; those are one layer under two words.

| Layer | Holds |
|---|---|
| `shared` | system types, paths, transport, base components |
| `entities` | a business thing: note, deck, media, settings, tab |
| `features` | one user scenario, whole |
| `widgets` | a composite block a page puts together |
| `pages` | one tab, whole |
| `app` | mounting, wiring, providers |

A window laid out flat is read as the same rule with one layer in it: a folder under `src/` naming no layer is a screen, and stands where the pages do. That is what `modules/apps/desktop/flashcards` is, and it is judged without being moved.

### A slice is cut into the standard segments, and its root names them

A slice's folders are `ui`, `api`, `model`, `lib`, `config` — the names the method already has, so nobody has to be told what ours mean.

What says which component a tab draws stands at the top of the slice, in `kind.ts`. A segment naming another segment is a ring: the model would name the component it draws and the component would name the state the model makes, and neither could be read first. The slice root is what may name both.

### A slice reaches no sibling slice of its own layer

`entities`, `features`, `widgets` and `pages` are each cut into slices named for what they are about. What two slices both need stands on a layer below.

Where two are bound by the domain rather than by convenience, the one asked of declares a public API for the one asking: `entities/tab/@x/media` is everything the media entity may know about a tab. A cross-import written down is a decision; one written as an import is not.

### The rules are generated from one table, and the table is the decision

`modules/tools/depgraph/layers.cjs` holds the rank of each layer and whether it is cut into slices, and builds every rule from it. A layer added or moved is a line of that table, not a new rule, and there is one place a reader goes to learn the order.

### The test harness stands on no layer

What assembles a whole window so a test can ask what it drew is drawn by the tests of every layer and draws the window itself. It points both ways by the nature of what it is. It stands in `src/testing/`, outside the layers, and no rule reads it — including the ring rule, which would otherwise find a cycle through every folder there is.

### An edge that stays is written down with why

`baseline` in `modules/tools/depgraph/modules.mjs` holds the edges the tree still draws against these rules, each under a line saying why it stands. The list only shrinks, and the check refuses an entry naming an edge nobody draws any more.

## Consequences

A thing two slices need can no longer be left in the first slice that wanted it: it goes down a layer, or the slice that owns it says so through `@x`. That is more work at the moment of writing and it is the work that keeps `shared/` from filling again.

`shared/` now holds what has no domain in it. The words the vault speaks about a file — which of four a note is, what creating one or moving one comes back with — are in `shared/file.ts`, because a deck and a stencil are created and moved exactly as a note is.

The port the window asks the vault through stands in `app/ports/`, beside what answers it. Splitting it into a port for each entity would put a composed interface across four sibling slices, which this decision refuses; whoever needs less than the whole port declares the part they call.
