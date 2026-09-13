# A file of the windows stands on a layer

- **Status:** Accepted
- **Date:** 2026-09-12
- **Applies to:** `modules/apps/desktop/editor`, `modules/apps/desktop/flashcards`, `modules/libs/ui`

## Context

The window was laid out flat: a folder under `src/` was a tab, `shared/` was what every tab could take, and one rule held it — no tab reaches another. That rule was written down, checked on every push, and it worked for as long as a window was a handful of tabs.

It stopped saying enough. A tab grew a store, an adapter to the vault, a dialog its neighbour also wanted, and the only two places a shared thing could go were inside one tab, where the other could not have it, or into `shared/`, where everything could. `shared/` filled with the vocabulary of notes, decks and settings, and the rule that had held the tabs apart had nothing to say about any of it.

## Decision

### Six layers, and a layer reaches only what stands below it

A file of a window stands on one of six layers, and each holds one kind of thing. The component library says `screens` where the windows say `pages`, and its barrel stands where their topmost layer does; those are one layer under two words.

| Layer | Holds |
|---|---|
| `shared` | system types, paths, transport, base components |
| `entities` | a business thing: note, deck, media, settings, tab |
| `features` | one user scenario, whole |
| `widgets` | a composite block a page puts together |
| `pages` | one tab, whole |
| `app` | mounting, wiring, providers |

A window laid out flat is read as the same rule with one layer in it: a folder under `src/` naming no layer is a screen, and stands where the pages do. The review window is laid out that way, and it is judged without being moved.

### A slice holding more than one kind of file is cut into the standard segments

A slice's folders are `ui`, `api`, `model`, `lib`, `config` — the names the method already has, so nobody has to be told what ours mean.

A segment tells one kind of file from another. A slice whose files are all one kind reads as well flat, and a slice that is a component and its test is one. A slice holding two kinds is cut, and cut for every kind in it at once: a component beside a wire mapper, a store beside a pure reducer.

What says which component a tab draws stands at the root of the slice and in no segment. A segment naming another segment is a ring: the model would name the component it draws and the component would name the state the model makes, and neither could be read first. The slice root is what may name both.

### A slice reaches no sibling slice of its own layer

A layer that is cut into slices has them named for what they are about. What two slices both need stands on a layer below.

Where two are bound by the domain rather than by convenience, the one asked of declares a public API for the one asking, under `@x` and named for the slice it is for: what the media entity may know about a tab is declared by the tab. A cross-import written down is a decision; one written as an import is not.

### The rules are generated from one table, and the table is the decision

`modules/tools/depgraph/layers.cjs` holds the rank of each layer and whether it is cut into slices, and builds every rule from it. A layer added or moved is a line of that table, not a new rule, and there is one place a reader goes to learn the order.

### The test harness stands on no layer

What assembles a whole window so a test can ask what it drew is drawn by the tests of every layer and draws the window itself. It points both ways by the nature of what it is. It stands outside the layers, and no rule reads it — the ring rule included, which would find a cycle through every folder there is.

### A slice that hands out a door is reached through it

A slice's `index.ts` names what the rest of the tree may take from it, and it is what the rest of the tree reaches. The surface is what is asked for, not everything there is.

Which slices have a door is read off the tree being cruised rather than written down: a slice is covered the day it declares one, and a module that has declared none is judged by the direction rules alone. Nothing has to be added to a list when a file is added to a slice.

### An edge that stays is written down with why

`baseline` in `modules/tools/depgraph/modules.mjs` holds the edges the tree still draws against these rules, each under a line saying why it stands. The list only shrinks, and the check refuses an entry naming an edge nobody draws any more.

## Consequences

A thing two slices need can no longer be left in the first slice that wanted it: it goes down a layer, or the slice that owns it says so through `@x`. That is more work at the moment of writing and it is the work that keeps `shared/` from filling again.

`shared/` now holds what has no domain in it. The words the vault speaks about a file — which of four a note is, what creating one or moving one comes back with — stand on the lowest layer, because a deck and a stencil are created and moved exactly as a note is.

The port the window asks the vault through stands on the topmost layer, beside what answers it. Splitting it into a port for each entity would put a composed interface across four sibling slices, which this decision refuses; whoever needs less than the whole port declares the part they call.
