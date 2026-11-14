# Specification: Plex places each relation as a block of a grid

**Task Slug:** `plex-clusters`
**Date:** `2026-11-12`
**Intent:** `intent.md` in the task folder (three open questions, section 6; the spec takes the proposed answer for each and marks where it would change)

All paths are under `plugins/views-plex-graph/frontend/plex/`. Placement skill: [`nm-place-plugin`](../../skills/nm-place-plugin/SKILL.md). Everything new is in `lib/arrange/` (pure, no Vue, no DOM, no clock).

## 1. The pipeline after the change

`arrangePlex` (`lib/arrange/arrange.ts`) keeps its order and its `Placement` strategy seam. What changes is how much is admitted and how crowding is measured: both now ask the placement itself, through the window, instead of the per-line limits.

```mermaid
graph TD
  A[resolveOptions → applyRoom] --> R["requestNodes: group by relation, cap at maxPerCluster when no window"]
  R --> C["measureCrowding(options, canHoldAll)"]
  C --> W[createWidthLookup at the packed options]
  W --> D["admitNodes: place, drop the first box outside the window and the rest of its relation, place again"]
  D --> P["placement.place (clusters by default) at the packed gaps"]
  P --> E["routeEdges, separateTitles (unchanged)"]
```

### 1.1 Placement: `clusters` (`lib/arrange/layout/clusters.ts`)

`export const clusters: Placement = { name: 'clusters', place: placeGrid }`, the default of `ArrangeInput.placement`.

`placeGrid(nodesByRelation, focus, context)`:

1. **Blocks.** Each side of the focus holds one block: the notes of every relation whose `options.direction[relation]` is that side (so a caller who sends children up still moves them). On a side the relations with `lean: 'up'` come first, then those with no lean, then `lean: 'down'`; each relation keeps its arrival order. A side block that holds relations with `lean: 'up'` and relations without it splits in two blocks with the same cell size and the same near column: the upper block has its first row on the row of the focus and grows up, the lower block has its first row on the next row under it and grows down, so the references stand over the jumps whatever their counts.
2. **Cells.** A block's cells are as wide as its widest footprint (pill and names) and as tall as its tallest, so a block of short titles stays compact. Neighboring cells are one `gap` apart sideways and one `lineGap` apart up and down. A footprint is aligned to the top of its cell, so the pills of one row line up.
3. **Blocks over and under the focus.** The children block starts `focusGap` under the focus and the parents block ends `focusGap` over it, both centered on the focus. Their rows fill from the middle out, the row nearest the focus first, and a partial last row is centered.
4. **Side blocks.** A side block that is not split is centered on the row of the focus, ends one `lineGap` over the children block, or starts one `lineGap` under the parents block. It is placed `focusGap` from the focus, and further out past the columns of every block over or under the focus that its rows reach. Its rows fill from the focus out.
5. **Columns.** A block takes from one column up to as many as reach the edge of the window (with no window, twice the square root of its notes, rounded up), never more than it has notes. The column counts of all blocks (the two blocks of a split side each take their own) and the standing of each side block are chosen together: the most notes inside the window, then the fewest columns a block over or under the focus takes beyond five, then the least distance of the notes from the focus counted in cells.
6. **Gaps.** The gaps are the packed ones; they close in a crowded window and do not open in a roomy one. A gap of 0 is raised to 1e-6, so neighbors do not overlap on float error.
7. **Output.** Each node keeps its relation, identity and title; `width = context.width(node)`, `height = options.nodeSize.height`, `opacity = 1`, `order` = its index within its relation. The returned array is grouped by relation in `RELATION_NAMES` order, each relation in arrival order, so the drawing order and the tab order of nodes follow the relations.

### 1.2 Admission (new, `lib/arrange/layout/admission.ts`)

Replaces `canPlace` in `arrange.ts` and the per-line capacity of `limits.ts`.

- `requestNodes(nodes, options): NodesByRelation` — group by relation in arrival order. Without a viewport each relation is cut to `options.maxPerCluster` (default 20, today's 5 × 4 capacity without a window). With a viewport nothing is cut here: the window decides (open question 1; if the owner wants a cap in a window too, it is applied here and nowhere else).
- `admitNodes(requested, lay, isInside): { nodesByRelation, overflow }` where `lay(nodesByRelation) → PlacedNode[]` places at the packed options and the gaps as set, and `isInside(node)` is the window test of `arrange.ts` (`isInsideViewport` per node, less `margin`). Loop: lay the current set; for each relation, keep the prefix of its nodes (arrival order) up to the first one whose box is not inside, and halve the group when its first box is outside (a smaller block has fewer rows, so its first box may lie nearer the focus); when nothing was cut, stop. Every pass removes at least one node, so it ends. Then the cut notes are taken back (refill): one note of each relation in turn, while the window still holds every box. Without a viewport `isInside` is always true and one pass suffices (skip the lay).
- `overflow[relation]` = number arrived − number admitted, set only when above 0 — the same shape `Plex.vue` already prints ("N children not shown").
- This works for any `Placement` (the `ring` fixture too), so the strategy still does not decide how much is shown.

### 1.3 Fit to the window

- **Crowding** (`layout/crowding.ts`): `measureCrowding(options, canHoldAll: (packed: PlexOptions) => boolean)`. Same binary search over `packOptions` (7 halvings), same early returns (no viewport, `squeeze >= 1`, holds at the settings). `canRelationAll` and the import of `getLimits` go. `arrange.ts` supplies `canHoldAll`: lay every requested node at `packed` with the width lookup of `packed` and the gaps as set, and check every box is inside the window.
- **Gaps** stay as packed: the final layout is laid at the packing admission chose, and no step opens the gaps in a roomy window.
- **Tightest packing.** Admission is greedy, so where the tightest packing admits more than the crowded one, the set admitted at the tightest packing is compared with the set grown at it from what the crowded packing admitted (`refill`), and the larger one is kept.
- **Room** (#1836): `applyRoom` still widens all three gaps before crowding.

### 1.4 Options (`lib/arrange/options.ts`)

- `LimitOptions` becomes `{ maxPerCluster: number; maxHeadings: number }`. `maxPerLine` and `maxLines` are removed (no manifest setting uses them; only stories and tests do).
- `direction` stays (drop gestures and the edge axis read it, and it sets the side of each group). `isVertical` stays for routing.
- `lineGap` stays: it is the gap between boxes one above another (open question 2).

### 1.5 Routing

`edges/routing.ts` is not changed. `getAxis` still takes the axis from the side of the relation (`vertical` for parents and children, `horizontal` for jumps and siblings). Sibling edges come from the graph as parent → sibling and are routed as today. Lines that pass behind a box in a second row are accepted for this issue (open question 3).

## 2. The direction table (`lib/relation.ts`)

```ts
export type Lean = 'up' | 'down'

export interface RelationDescriptor {
  readonly grows: 'up' | 'down' | 'left' | 'right'
  /** The order of relations that share a side: `up` first, then null, then `down`. */
  readonly lean: Lean | null
  readonly plural: readonly [one: string, many: string]
}

export const RELATIONS = {
  parent: { grows: 'up', lean: null, plural: ['parent', 'parents'] },
  child: { grows: 'down', lean: null, plural: ['child', 'children'] },
  jump: { grows: 'left', lean: 'down', plural: ['jump', 'jumps'] },
  sibling: { grows: 'right', lean: null, plural: ['sibling', 'siblings'] },
} as const satisfies Record<PlexRelation, RelationDescriptor>
```

#1839 adds `reference: { grows: 'left', lean: 'up', plural: ['reference', 'references'] }` (plus the relation in the types) and nothing in the placement. Which relation a drag to the left makes once two relations share `left` (`drop.ts` takes the first) is #1839's decision.

## 3. What is removed

- `lib/arrange/layout/limits.ts` and `limits.test.ts`: deleted (`getLimits`, `Limits`, `RoleLimits`, `getViewportLimits`, `getWidestRow`, `getRelationLimits`).
- `lib/arrange/layout/placement.ts`: `rowsAndColumns`, `line`, `inRows`, `inColumns`, `lines`, `getReach`, `measureWidestWithin`, `LineContext`, `AxisPlacement` deleted. The file keeps `Placement`, `NodesByRelation`, `Widths` and `PlacementContext`, which loses `limits` (`{ options, width }`).
- `arrange.ts`: `canPlace`, the `getLimits` call and the `limits` passed to the placement.
- `crowding.ts`: `canRelationAll`.
- `options.ts`: `maxPerLine`, `maxLines`.
- `index.ts`: exports `clusters` instead of `rowsAndColumns`.

## 4. Blast radius

| Path | File | Action | Consumers |
|---|---|---|---|
| `lib` | `relation.ts` | Modify (`lean`, `Lean`) | `options.ts`, `clusters.ts`, `drop.ts`, `fixtures/build.ts` |
| `lib/arrange/layout` | `clusters.ts`, `admission.ts` | Create | `arrange.ts`, `index.ts` |
| `lib/arrange/layout` | `placement.ts`, `crowding.ts` | Modify | `arrange.ts`, `fixtures/ring.ts` |
| `lib/arrange/layout` | `limits.ts`, `limits.test.ts` | Delete | `arrange.ts`, `crowding.ts`, `crowding.test.ts` |
| `lib/arrange` | `arrange.ts`, `options.ts`, `index.ts` | Modify | `model/transition.ts`, `ui/Plex.vue`, `model/sizing/plexScale.ts` |
| `model` | `transition.ts` | Modify: track `maxPerCluster` instead of `maxPerLine`, `maxLines` | `ui/Plex.vue` |
| `ui` | `Plex.stories.ts` | Modify: `maxPerCluster` control, placement mapping `clusters` / `ring` | Storybook |
| `fixtures` | `ring.ts` | Unchanged (implements `Placement`, never read `limits`) | `arrange.test.ts`, stories |

No proto, wire, manifest or setting change. `apps/landing` has its own plex demo and is not touched.

### Tests and stories that pin rows-and-columns and must change

- `lib/arrange/layout/placement.test.ts`: every `describe` but "what arrives back" pins rows and columns ("a row", "a row too long for one line", "a column", "rows and columns together", "how far a column reaches before it clears the rows"). Replaced by `arrange.grid.test.ts` and `clusters.acceptance.test.ts`.
- `lib/arrange/layout/limits.test.ts`: deleted with `limits.ts`; its window cases ("nothing reaches past the edge", "the margin is a setting") become admission cases.
- `lib/arrange/layout/crowding.test.ts`: "the loosest packing that fits everything" uses `getLimits` and `canRelationAll`, and "leaves a window alone… twelve to a line and four lines" counts lines; rewritten against `measureCrowding(options, canHoldAll)` with a `canHoldAll` built from `arrangePlex` placement. "leaves the heights, the counts…" checks `maxPerLine`/`maxLines` → `maxPerCluster`.
- `lib/arrange/arrange.test.ts`: "clears the rows a column runs alongside, and only those"; "leaves a row packed to one gap…", "turns one edge of a column towards the focus", "starts each further column beyond the widest of the one before it"; "the lines of a relation are shared out evenly" (both); "order along a line follows the order given" ("lays a row out left to right", "puts the second line further from the focus than the first"); "counts the nodes past the last line" and "does not let a strategy decide how much is shown" (`maxPerLine × maxLines` → `maxPerCluster`); "lets the caller change how many fit on a line"; "uses rows and columns unless told otherwise" → clusters; "passes a strategy only the relations that have nodes in them" (wraps `rowsAndColumns`); "a window that holds the arrangement exactly" (row-shaped window). "puts every jump clear to the left" stays true and stays.
- `lib/arrange/look.test.ts` (#1836): "places every neighbor where the plex placed it before, within a pixel" and its `TODAY` positions are removed (positions change by design); the width/height, focus, edges (`TODAY_EDGES` stays) and "is the look the arrangement takes…" cases stay. "moves the boxes apart where no window limits them" groups children by row; it is rewritten to compare the smallest box-to-box distance with and without room. "counts the notes that still do not fit as not shown" stays.
- `model/transition.test.ts`: `rowsAndColumns` → `clusters`; "the nodes allowed on a line" and "the lines allowed" → one case "the notes allowed in a group" (`maxPerCluster`).
- `model/sizing/plexScale.test.ts`: the unscaled-key list and the two `maxPerLine`/`maxLines` expectations → `maxPerCluster`.
- `ui/Plex.stories.ts`: `rowsAndColumns` import, the placement mapping, the `maxPerLine` arg and control, and the args of the story with `maxPerLine: 9`. `MakingOne` and `DraggingThemIn` drag 220 px above the focus to make a parent; that point must still be empty (parents of `typical` stay within 220 px; the tester checks it in the play run). `SmallWindow` and `Overcrowded` are visual only.
- `ui/PlexMap.stories.ts`: no position pinned (menus, headings, line ends); re-run only.
- Acceptance `features/editor/plex/*.feature` (they replace the e2e specs of `views.plex.graph`): no scenario reads a position; they check relations. Run only.

## 5. Negative invariants

- `lib/` has no Vue, DOM, clock or text measuring; widths come in as numbers.
- `arrangePlex` stays pure and deterministic: no randomness, stable sort, ties keep candidate order.
- The `Placement` interface stays a coordinates-only strategy: admission and fitting live in `arrange.ts` / `admission.ts` / `crowding.ts`, not in `clusters`.
- No camera zoom, no change of box sizes by the placement, no "+N" node.
- Files ≤ 350 lines, functions ≤ 80 lines, complexity ≤ 10, ≤ 4 params; imperative names (`placeGrid`, `createBlocks`, `planBlocks`, `listCells`, `requestNodes`, `admitNodes`); comments only for what the code cannot show.
- Tests use invented fixtures (`fixtures/build.ts`, `fixtures/graphs.ts`), assert behavior on the frame (positions relative to the focus and each other, counts, edges), never CSS or pixels of the rendering, and do not mock the placement or admission they verify.

## 6. Acceptance criteria

Every criterion runs through `arrangePlex` unless it names a unit. "Window" is `options.viewport`; "inside" means inside the window less `margin`. Overlap is `isOverlapping` of the boxes (`layout/boxes.ts` `getBox(node, 0)`).

### AC-1: No two boxes overlap
- **Given** every fixture in `graphs`, unmeasured and measured by title, with no window and in windows 2400×1400, 1200×800, 700×600, 420×320, and the `room` setting at 0 and 20
- **Then** no two nodes of the frame overlap, the focus included.

### AC-2: Each relation on its side
- **Given** `typical` and `build('A note', { parent: 8, child: 20, jump: 8, sibling: 8 })` in a 1600×1000 window
- **Then** every parent's bottom is above the focus's top, every child's top below the focus's bottom, every sibling's left edge right of the focus's right edge, every jump's right edge left of the focus's left edge.

### AC-3: A group never comes nearer the focus than its start
- **Given** a single relation of 1, 4, 9 and 20 nodes, each of the four relations
- **Then** no node's center lies nearer the focus, measured along the group's direction, than the center of the first node of that relation placed alone.

### AC-4: A large group is a block, not a line
- **Given** `build('A note', { child: 9 })` in a 1600×1000 window (and with no window)
- **Then** the children take at least two distinct rows (center `y`), and no row holds more than five children.
- **Given** `build('A note', { sibling: 9 })` likewise
- **Then** the siblings take at least two distinct columns (center `x`).

### AC-6: A large group leaves the first note of another group in place
- **Given** `build('A note', { child: 20, jump: 1, sibling: 1, parent: 1 })` and the same graph with only the single parent, jump and sibling, no window
- **Then** the parent, the jump and the sibling are at the same position in both frames: twenty children placed alongside do not push the first note of another group away from its start.

### AC-7: A large group spreads over columns
- **Given** `build('A note', { child: 24 })` in a 1600×1000 window
- **Then** every child is shown, and some child's box lies beside the focus's row of boxes further sideways than the focus's half-width plus one box (the group spreads sideways instead of running down in one column).
- **Given** `build('A note', { child: 16, jump: 16 })` in a 1600×1000 window
- **Then** no box overlaps.

### AC-8: A long group spreads instead of running into one column
- **Given** `build('A note', { jump: 30 })` in a 1600×1000 window
- **Then** at least 20 jumps are shown, and the shown jumps take at least three distinct columns (center `x`) and at least three distinct rows (center `y`).

### AC-9: Overflow is counted, and only the last notes go
- **Given** `graphs.overcrowded` and `build('A note', { parent: 9, child: 40, jump: 9, sibling: 9 })` in a 400×400 window
- **Then** for each relation the shown nodes are the first ones in arrival order; `overflow[relation]` + shown = arrived; at least one relation has overflow; no box lies outside the window; edges to nodes not shown are dropped.
- **Given** no window and a relation of 30 nodes
- **Then** 20 are shown (`maxPerCluster`) and `overflow` is 10; with `maxPerCluster: 5`, 5 and 25.

### AC-10: Boxes narrow and gaps close before a note is left out
- **Given** `build('A note', { parent: 4, child: 12, jump: 4, sibling: 4 })` in windows 600×1000, 500×1000, 440×1000
- **Then** `measureCrowding` with the clusters `canHoldAll` returns the loosest packing (to a 128th) at which every node is inside, and one step looser does not hold them all; a window that holds all at the settings is returned unchanged.

### AC-11: Gaps stay as set in a roomy window and the arrangement stays inside
- **Given** `build('A node', { child: 3 })` in a 2400×1400 window
- **Then** the distance between neighboring children is the one at the settings.
- **Given** `build('A node', { parent: 6, child: 21, jump: 7, sibling: 7 })` in windows 2400×1400, 1970×1440, 1200×800, 700×600, 560×480
- **Then** no box lies outside the window.

### AC-12: Siblings join their parent
- **Given** `typical`
- **Then** every sibling has an edge from a shown parent and no edge to the focus; every other neighbor's edge joins it to the focus — the same pairs as `TODAY_EDGES` in `look.test.ts`.

### AC-13: The same input gives the same picture
- **Given** every fixture arranged twice, and `typical` with its relations listed in another order and the nodes of each relation kept in their order
- **Then** the two arrangements of a fixture are strictly equal, and every node of the reordered `typical` has the position it has in the original.

### AC-14: Each block is a whole grid (`arrange.grid.test.ts`)
- **Given** a note with a parent, a child, five jumps, two references and thirteen siblings in 1100×1300 and 1280×1600 windows
- **Then** every note is shown, and the siblings, and the jumps with the references, each fill the columns nearest the focus in every row, with evenly spaced rows.
- **Given** a note with thirteen children among many siblings and a few jumps in an 1825×1890 window
- **Then** the first row of children is `focusGap` under the focus, and no side note lies in the columns of the children at their height.
- **Given** a block of short titles
- **Then** its columns step by its widest title and the gap, and its rows by the height of a note and the line gap.

### AC-15: The direction table drives the sides
- **Given** `direction: { parent: 'down', child: 'up', jump: 'left', sibling: 'right' }`
- **Then** every child is above the focus and every parent below.

### AC-16: Any placement strategy is admitted by the window
- **Given** the `ring` placement and `overcrowded` in a 700×600 window
- **Then** every shown node is inside the window and overflow + shown = arrived per relation; with no window the counts equal `maxPerCluster` per relation, whatever the strategy returns.

### AC-17: The plex in the window
- **Given** `Plex.stories.ts` `Overcrowded` in Chromium and WebKit
- **Then** the status text reads "N … not shown" for the relations left out, and no two `.plex__box` rectangles intersect.
- **Given** `MakingOne` and `DraggingThemIn`
- **Then** they pass unchanged (a drag above the focus still makes a parent).

## 7. Performance check (reported, not a test)

`admitNodes` and `measureCrowding` lay the arrangement several times. The coder measures `arrangePlex` on `crowded`, `overcrowded` and AC-11's full graph in a 1200×800 window on mridanga and reports the time per call in the PR thread; above 16 ms it is raised with the owner before merge.

Measured on mridanga in a 1200×800 window at the time of #1837, with the free-spot placement that the grid of blocks replaced; not measured again since, per `arrangePlex` call: `crowded` 3.8 ms, AC-11's full graph (6/21/7/7) 3.6 ms, `overcrowded` (200 children, 40 jumps) 12.2 ms, down from 1890 ms. The placement returns its boxes up to the first one outside the window in a fit check (`measureCrowding`), and in admission up to the point where every relation has met a box outside or is fully placed (`PlacementContext.shouldStop`). The grid of blocks is not measured yet.

## 8. Amendments after #1840 (2026-11-13)

1. Notes cut by the loop are taken back, one of each relation in turn, while the window still holds every box (refill). The loop halves a group whose first box is outside.
2. A measured title is never cut by packing: a box keeps the width its title needs, up to the widest box of the settings. Unmeasured boxes narrow with the packing.
3. The numbers of section 7 were measured on the code of #1837. The footprints of #1840 (names) and the refill change them; the limit of 16 ms stands.
